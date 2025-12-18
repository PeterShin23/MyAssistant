package key

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	hook "github.com/robotn/gohook"

	"github.com/PeterShin23/MyAssistant/backend/internal/audio"
	"github.com/PeterShin23/MyAssistant/backend/internal/openai"
	"github.com/PeterShin23/MyAssistant/backend/internal/screen"
)

const (
	triggerKeyRawcode = 50 // Rawcode for ` (backtick) on macOS
	holdThreshold     = 700 * time.Millisecond
	maxDuration       = 20 * time.Second
)

// listener encapsulates the key state and session lifecycle.
type listener struct {
    session *openai.Session
    noAudio bool
    pretty  bool

    mu           sync.Mutex
    running      bool        // Is a session currently running
    stopping     bool        // Has stop begun for the current session ✅ NEW
    keyHeld      bool
    holdTimer    *time.Timer
    sessionID    int64       // Unique ID for each session
    sessionCount int64       // Counter for generating session IDs

    screenshotPath string
    audioPath      string
}

// StartKeyListener launches the listener loop.
// It waits for backtick being held, and starts a session if held long enough.
func StartKeyListener(session *openai.Session, noAudio bool, pretty bool, wsURL, wsToken string) error {
	l := &listener{session: session, noAudio: noAudio, pretty: pretty}

	fmt.Printf("🎧 Listening: hold backtick ≥ %.0fms to trigger\n", holdThreshold.Seconds()*1000)

	eventChan := hook.Start()
	defer hook.End()

	for ev := range eventChan {
		if ev.Rawcode != triggerKeyRawcode {
			continue
		}

		switch ev.Kind {
		case hook.KeyDown:
			l.onKeyDown()
		case hook.KeyUp:
			l.onKeyUp()
		}
	}

	return nil
}

// onKeyDown schedules a delayed session start if the key remains held.
func (l *listener) onKeyDown() {
    l.mu.Lock()
    defer l.mu.Unlock()

    if l.keyHeld || l.running {
        return
    }
    l.keyHeld = true

    l.holdTimer = time.AfterFunc(holdThreshold, func() {
        l.mu.Lock()
        defer l.mu.Unlock()

        if l.keyHeld && !l.running {
            l.startSession() // call a locked variant for clarity
        }
    })
}

// onKeyUp either cancels the timer or stops the session if one is running.
func (l *listener) onKeyUp() {
    l.mu.Lock()
    if l.holdTimer != nil {
        l.holdTimer.Stop()
        l.holdTimer = nil
    }
    l.keyHeld = false
    l.mu.Unlock()

    l.stopSession("🔑 Key released")
}

// startSession begins audio recording and screenshot capture, and sets a timeout.
func (l *listener) startSession() {
    // caller already holds l.mu
    l.running = true
    l.stopping = false  // ✅ reset for the new session
    l.sessionID = atomic.AddInt64(&l.sessionCount, 1)

	fmt.Println("▶️  Starting capture session...")

    // Take screenshot and wait for it to complete
    go func() {
        screenshotPath, err := screen.CaptureScreenshot()
        l.mu.Lock()
        if err != nil {
            // Log error but continue without screenshot
            fmt.Println("❌ Screenshot failed:", err)
        } else {
            l.screenshotPath = screenshotPath
        }
        l.mu.Unlock()
    }()

    // Start audio recording in background
    if !l.noAudio {
        go func() {
            if err := audio.StartRecording(); err != nil {
				fmt.Println("❌ Failed to start audio recording:", err)
            }
        }()
    }

    // Auto-stop after max duration (can safely fire; stopSession is idempotent)
    go func() {
        time.Sleep(maxDuration)
        l.stopSession("⏱️ Max duration reached")
    }()
}


// stopSession finalizes recording, then calls OpenAI processor.
func (l *listener) stopSession(reason string) {
    l.mu.Lock()
    // If not running or already stopping, do nothing
    if !l.running || l.stopping {
        l.mu.Unlock()
        return
    }
    // Mark as stopping but keep running true until files are ready
    l.stopping = true
    l.mu.Unlock()

    // Wait for screenshot to be captured (with timeout)
    screenshotTimeout := time.After(10 * time.Second)
    screenshotReady := make(chan bool, 1)
    
    go func() {
        l.mu.Lock()
        hasScreenshot := l.screenshotPath != ""
        l.mu.Unlock()
        
        if hasScreenshot {
            screenshotReady <- true
            return
        }
        
        // Wait for screenshot goroutine to complete
        for {
            l.mu.Lock()
            hasScreenshot = l.screenshotPath != ""
            l.mu.Unlock()
            
            if hasScreenshot {
                screenshotReady <- true
                return
            }
            
            select {
            case <-screenshotTimeout:
                screenshotReady <- false
                return
            case <-time.After(100 * time.Millisecond):
                // Continue waiting
            }
        }
    }()

    // Wait for audio recording to complete if enabled
    if !l.noAudio {
        audioPath, err := audio.StopRecording()
        if err != nil {
			fmt.Println("❌ Failed to stop audio recording:", err)
        } else {
            l.mu.Lock()
            l.audioPath = audioPath
            l.mu.Unlock()
        }
		fmt.Println(reason)
    } else {
        l.mu.Lock()
        l.audioPath = ""
        l.mu.Unlock()
    }

    // Wait for screenshot to be ready
    screenshotSuccess := <-screenshotReady
    if !screenshotSuccess {
        l.mu.Lock()
        l.screenshotPath = ""
        l.mu.Unlock()
    }

    fmt.Println("✅ Sending to processor...")

    // Mark session as not running and allow new sessions
    l.mu.Lock()
    l.running = false
    l.mu.Unlock()

    go func() {
        if err := l.session.Process(l.screenshotPath, l.audioPath, l.pretty); err != nil {
			fmt.Println("❌ Error during OpenAI processing:", err)
        }
        
        // Ready for a new session
        l.mu.Lock()
        l.stopping = false
        l.mu.Unlock()
    }()
}

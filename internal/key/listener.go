package key

import (
	"fmt"
	"sync"
	"time"

	hook "github.com/robotn/gohook"

	"github.com/PeterShin23/MyAssistant/internal/audio"
	"github.com/PeterShin23/MyAssistant/internal/openai"
	"github.com/PeterShin23/MyAssistant/internal/screen"
)

const (
	triggerKeyRawcode = 50 // Rawcode for ` (backtick) on macOS
	holdThreshold     = 700 * time.Millisecond
	maxDuration       = 5 * time.Second
)

// listener encapsulates the key state and session lifecycle.
type listener struct {
	session *openai.Session
	noAudio bool

	mu        sync.Mutex
	running   bool        // Is a session currently running
	keyHeld   bool        // Is the key currently pressed
	holdTimer *time.Timer // Timer to trigger delayed start

	screenshotPath string
	audioPath      string
}

// StartKeyListener launches the listener loop.
// It waits for backtick being held, and starts a session if held long enough.
func StartKeyListener(session *openai.Session, noAudio bool) error {
	l := &listener{session: session, noAudio: noAudio}

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

	// Schedule session start only if key remains held after delay.
	l.holdTimer = time.AfterFunc(holdThreshold, func() {
		l.mu.Lock()
		defer l.mu.Unlock()

		if l.keyHeld && !l.running {
			l.startSession()
		}
	})
}

// onKeyUp either cancels the timer or stops the session if one is running.
func (l *listener) onKeyUp() {
	l.mu.Lock()

	// Cancel the pending hold trigger if it's still waiting.
	if l.holdTimer != nil {
		l.holdTimer.Stop()
	}

	l.keyHeld = false
	running := l.running
	l.mu.Unlock()

	// Stop session if it was running.
	if running {
		l.stopSession("🔑 Key released")
	}
}

// startSession begins audio recording and screenshot capture, and sets a timeout.
func (l *listener) startSession() {
	l.running = true // caller already holds mutex

	fmt.Println("▶️  Starting capture session...")

	// Take screenshot in background
	go func() {
		if path, err := screen.CaptureScreenshot(); err != nil {
			fmt.Println("❌ Screenshot failed:", err)
		} else {
			l.mu.Lock()
			l.screenshotPath = path
			l.mu.Unlock()
		}
	}()

	if !l.noAudio {
		// Start audio recording in background
		go func() {
			if err := audio.StartRecording(); err != nil {
				fmt.Println("❌ Failed to start audio recording:", err)
			}
		}()
	}

	// Auto-stop after max duration
	go func() {
		time.Sleep(maxDuration)
		l.stopSession("⏱️ Max duration reached")
	}()
}

// stopSession finalizes recording, then calls OpenAI processor.
func (l *listener) stopSession(reason string) {
	l.mu.Lock()
	if !l.running {
		l.mu.Unlock()
		return
	}
	l.running = false
	l.mu.Unlock()

	if !l.noAudio {
		audioPath, err := audio.StopRecording()
		if err != nil {
			fmt.Println("❌ Failed to stop audio recording:", err)
		}

		l.mu.Lock()
		l.audioPath = audioPath
		l.mu.Unlock()

		fmt.Println(reason)
	} else {
		l.mu.Lock()
		l.audioPath = ""
		l.mu.Unlock()
	}

	fmt.Println("✅ Sending to processor...")

	go func() {
		if err := l.session.Process(l.screenshotPath, l.audioPath); err != nil {
			fmt.Println("❌ Error during OpenAI processing:", err)
		}
	}()
}

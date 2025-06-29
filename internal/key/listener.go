package key

import (
	"fmt"
	"sync"
	"time"

	"github.com/PeterShin23/MyAssistant/internal/audio"
	"github.com/PeterShin23/MyAssistant/internal/openai"
	"github.com/PeterShin23/MyAssistant/internal/screen"
	hook "github.com/robotn/gohook"
)

var (
	maxDuration        = 5 * time.Second
	isRunning          = false
	awaitingKeyRelease = false
	mu                 sync.Mutex
	screenshotPath     string
	audioPath          string
)

func StartKeyListener() error {
	fmt.Println("🎧 Listening for key...")

	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		if ev.Rawcode != 50 {
			continue // only handle backtick
		}

		switch ev.Kind {
		case hook.KeyDown:
			mu.Lock()
			if !isRunning && !awaitingKeyRelease {
				isRunning = true
				mu.Unlock()
				go handleStart()
			} else {
				mu.Unlock()
			}

		case hook.KeyUp:
			mu.Lock()
			if isRunning {
				mu.Unlock()
				go handleStop("🔑 Key released")
			} else if awaitingKeyRelease {
				awaitingKeyRelease = false
				mu.Unlock()
			} else {
				mu.Unlock()
			}
		}
	}

	return nil
}

func handleStart() {
	go func() {
		if path, err := screen.CaptureScreenshot(); err != nil {
			fmt.Println("❌ Screenshot failed:", err)
		} else {
			mu.Lock()
			screenshotPath = path
			mu.Unlock()
		}
	}()

	go func() {
		if err := audio.StartRecording(); err != nil {
			fmt.Println("❌ Failed to start recording:", err)
		}
	}()

	// Auto-stop after max duration
	go func() {
		time.Sleep(maxDuration)
		handleStop("⏱️ Max duration reached")
	}()
}

func handleStop(reason string) {
	mu.Lock()
	if !isRunning {
		mu.Unlock()
		return
	}
	isRunning = false
	awaitingKeyRelease = true
	mu.Unlock()

	path, err := audio.StopRecording()
	if err != nil {
		fmt.Println("❌ Failed to stop recording:", err)
	}

	mu.Lock()
	audioPath = path
	mu.Unlock()

	fmt.Println(reason)
	fmt.Println("✅ Finished recording. Entering processing phase...")

	go openai.Process(screenshotPath, audioPath)
}

package key

import (
	"fmt"
	"os"
	"time"

	"github.com/PeterShin23/MyAssistant/internal/audio"
	"github.com/PeterShin23/MyAssistant/internal/screen"
	hook "github.com/robotn/gohook"
)

var (
	maxDuration   = 20 * time.Second
	hotKeyPressed = false
)

func StartKeyListener() error {
	fmt.Println("Listening...")

	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		if ev.Kind == hook.KeyDown && ev.Rawcode == 50 && !hotKeyPressed {
			hotKeyPressed = true

			go func() {
				if err := screen.CaptureScreenshot(); err != nil {
					fmt.Println("Screenshot failed", err)
				}
			}()

			go func() {
				if err := audio.StartRecording(); err != nil {
					fmt.Println("Failed to begin recording...", err)
				}
			}()

			go func() {
				time.Sleep(maxDuration)
				if hotKeyPressed {
					hotKeyPressed = false
					fmt.Println("⏱️ Max recording duration reached.")

					if err := audio.StopRecording(); err != nil {
						fmt.Println("Failed to stop recording...", err)
					}

					fmt.Println("✅ Finished recording. Entering processing phase...")
					os.Exit(0)
				}
			}()
		} else if ev.Kind == hook.KeyUp && ev.Rawcode == 50 && hotKeyPressed {
			hotKeyPressed = false

			if err := audio.StopRecording(); err != nil {
				fmt.Println("Failed to stop recording...", err)
			}

			fmt.Println("✅ Finished recording. Entering processing phase...")
			os.Exit(0)
		}
	}

	return nil
}

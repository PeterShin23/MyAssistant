package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/PeterShin23/MyAssistant/internal/screen"
)

// To Run: go run ./cmd/assistant
func main() {
	rootCmd := &cobra.Command{
		Use:   "assistant",
		Short: "MyAssistant CLI - your personal screen/audio capture tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("👋 Let's get to work!")

			err := screen.CaptureScreenshot()
			if err != nil {
				fmt.Println("Failed to capture screenshot", err)
				os.Exit(1)
			}
		},
	}

	if error := rootCmd.Execute(); error != nil {
		fmt.Println("Error", error)
		os.Exit(1)
	}
}

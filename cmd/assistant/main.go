package main

import (
	"fmt"
	"os"

	"github.com/PeterShin23/MyAssistant/internal/key"
	"github.com/spf13/cobra"
)

// To Run: go run ./cmd/assistant
func main() {
	fmt.Println("⌨️ Waiting for configured key hold")

	rootCmd := &cobra.Command{
		Use:   "assistant",
		Short: "MyAssistant CLI - your personal screen/audio capture tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("👋 Let's get to work!")

			if err := key.StartKeyListener(); err != nil {
				fmt.Println("Key Listener failed:", err)
				os.Exit(1)
			}
		},
	}

	if error := rootCmd.Execute(); error != nil {
		fmt.Println("Error", error)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// To Run: go run ./cmd/assistant
func main() {
	rootCmd := &cobra.Command{
		Use:   "assistant",
		Short: "MyAssistant CLI - your personal screen/audio capture tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("👋 Let's get to work!")
		},
	}

	if error := rootCmd.Execute(); error != nil {
		fmt.Println("Error")
		os.Exit(1)
	}
}

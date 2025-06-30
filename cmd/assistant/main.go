package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PeterShin23/MyAssistant/internal/key"
	"github.com/PeterShin23/MyAssistant/internal/openai"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// To Run: go run ./cmd/assistant
func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found. Exiting")
		os.Exit(1)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("No OPENAI_API_KEY configured. Exiting")
		os.Exit(1)
	}

	fmt.Println("⌨️ Waiting for configured key hold")

	var rootCmd = &cobra.Command{
		Use:   "assistant",
		Short: "MyAssistant CLI - your personal screen/audio capture tool",
	}

	var noAudio bool

	var listenCmd = &cobra.Command{
		Use:   "listen",
		Short: "Start listening for hotkey press",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("👋 Let's get to work!")

			session, err := openai.NewSession()
			if err != nil {
				fmt.Println("Failed to create OpenAI session:", err)
				os.Exit(1)
			}

			if err := key.StartKeyListener(session, noAudio); err != nil {
				fmt.Println("Key Listener failed:", err)
				os.Exit(1)
			}
		},
	}

	listenCmd.Flags().BoolVar(&noAudio, "no-audio", false, "Disable audio recording")

	var clearCmd = &cobra.Command{
		Use:   "clear",
		Short: "Delete all files in the .data folder",
		Run: func(cmd *cobra.Command, args []string) {
			err := clearDataFolder(".data")
			if err != nil {
				fmt.Println("Failed to clear .data:", err)
				os.Exit(1)
			}
			fmt.Println("🧹 Cleared all files in .data/")
		},
	}

	rootCmd.AddCommand(listenCmd)
	rootCmd.AddCommand(clearCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func clearDataFolder(folder string) error {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(folder, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

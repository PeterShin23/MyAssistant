package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PeterShin23/MyAssistant/backend/internal/capture"
	"github.com/PeterShin23/MyAssistant/backend/internal/key"
	"github.com/PeterShin23/MyAssistant/backend/internal/openai"
	"github.com/PeterShin23/MyAssistant/backend/internal/stream"

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
	var pretty bool
	var wsURL string
	var wsToken string
	var silent bool

	var listenCmd = &cobra.Command{
		Use:   "listen",
		Short: "Start listening for hotkey press",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("👋 Let's get to work!")

			// Check for environment variable fallbacks
			if wsURL == "" {
				wsURL = os.Getenv("MYASSISTANT_WS_URL")
			}
			if wsToken == "" {
				wsToken = os.Getenv("MYASSISTANT_WS_TOKEN")
			}

			// Validate silent mode requires WebSocket
			if silent && wsURL == "" {
				fmt.Println("❌ Error: --silent mode requires --ws-url to be set")
				fmt.Println("   Silent mode disables terminal output, so WebSocket is required for receiving responses")
				os.Exit(1)
			}

			// Create StreamWriter instances
			var writer stream.StreamWriter
			var wsWriter *stream.WSWriter

			// If WebSocket URL is provided, create a WSWriter
			if wsURL != "" {
				wsWriter = stream.NewWSWriter(wsURL, wsToken)

				if silent {
					// Silent mode: only use WebSocket, no stdout
					writer = wsWriter
					fmt.Println("🤫 Silent mode enabled - output only to WebSocket")
				} else {
					// Normal mode: use both stdout and WebSocket
					stdoutWriter := stream.NewStdoutWriter(pretty)
					writer = stream.NewTeeWriter(stdoutWriter, wsWriter)
				}
			} else {
				// No WebSocket: only stdout
				stdoutWriter := stream.NewStdoutWriter(pretty)
				writer = stdoutWriter
			}

			session, err := openai.NewSession(writer)
			if err != nil {
				fmt.Println("Failed to create OpenAI session:", err)
				os.Exit(1)
			}

			// Create capture manager for remote screenshot triggers
			captureManager := capture.NewManager(session)

			// If WebSocket is enabled, set up command handler for remote screenshot triggers
			if wsWriter != nil {
				wsWriter.SetCommandHandler(func(command string) {
					if command == "screenshot" {
						fmt.Println("📱 Remote screenshot command received")
						if err := captureManager.TriggerScreenshot(); err != nil {
							fmt.Printf("❌ Remote screenshot failed: %v\n", err)
						}
					}
				})
			}

			if err := key.StartKeyListener(session, noAudio, pretty, wsURL, wsToken); err != nil {
				fmt.Println("Key Listener failed:", err)
				os.Exit(1)
			}
		},
	}

	listenCmd.Flags().BoolVar(&noAudio, "no-audio", false, "Disable audio recording")
	listenCmd.Flags().BoolVar(&pretty, "pretty", false, "Outputs pretty markdown instead of streamed data")
	listenCmd.Flags().StringVar(&wsURL, "ws-url", "", "WebSocket URL for streaming output")
	listenCmd.Flags().StringVar(&wsToken, "ws-token", "", "Authorization token for WebSocket connection")
	listenCmd.Flags().BoolVar(&silent, "silent", false, "Disable terminal output (requires --ws-url)")

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

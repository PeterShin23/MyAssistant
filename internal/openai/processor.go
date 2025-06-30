package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // support PNG
	"io"
	"os"
	"path/filepath"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

var rulesJSON = filepath.Join(projectRoot(), "rules.json")

func projectRoot() string {
	dir, _ := os.Getwd()
	return dir
}

type Session struct {
	mu       sync.Mutex
	client   *openai.Client
	messages []openai.ChatCompletionMessage
}

type PromptConfig struct {
	TechnicalPrompt string `json:"whatDoYouNeedHelpWith"`
}

func NewSession() (*Session, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY not set")
	}
	client := openai.NewClient(apiKey)
	return &Session{
		client:   client,
		messages: []openai.ChatCompletionMessage{},
	}, nil
}

func (s *Session) Process(screenshotPath, audioPath string) error {
	ctx := context.Background()

	// 1. Transcribe audio using Whisper
	transcript, err := s.transcribeAudio(ctx, audioPath)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	// 2. Read and encode screenshot
	imgDataURI, err := compressAndEncodeImage(screenshotPath)
	if err != nil {
		return fmt.Errorf("Failed to prepare image: %w", err)
	}

	// Append system message once
	s.mu.Lock()
	if len(s.messages) == 0 {
		var systemPrompt = buildSystemPrompt()

		s.messages = append(s.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}

	var parts []openai.ChatMessagePart

	parts = append(parts, openai.ChatMessagePart{
		Type: openai.ChatMessagePartTypeImageURL,
		ImageURL: &openai.ChatMessageImageURL{
			URL:    imgDataURI,
			Detail: openai.ImageURLDetailAuto,
		},
	})

	if transcript != "" {
		fmt.Printf("transcript: %s\n", transcript)

		parts = append(parts, openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeText,
			Text: fmt.Sprintf("Transcript:\n\n%s", transcript),
		})
	}

	userMessage := openai.ChatCompletionMessage{
		Role:         openai.ChatMessageRoleUser,
		MultiContent: parts,
	}
	s.messages = append(s.messages, userMessage)
	s.mu.Unlock()

	stream, err := s.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    openai.GPT4o,
		Messages: s.messages,
		Stream:   true,
	})
	if err != nil {
		return fmt.Errorf("streaming chat completion failed: %w", err)
	}
	defer stream.Close()

	fmt.Print("🤖 GPT-4o Response:\n")

	var fullContent string
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}
		if delta := resp.Choices[0].Delta.Content; delta != "" {
			fmt.Print(delta)
			fullContent += delta
		}
	}

	fmt.Print("\n\n")

	// Maintain Session Context
	reply := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: fullContent,
	}

	s.mu.Lock()
	s.messages = append(s.messages, reply)
	s.mu.Unlock()

	return nil
}

func buildSystemPrompt() string {
	systemPrompt := `You are the user's personal helper. 
	Use the image and audio transcript provided as context. 
	Assume that the user needs help with the context that's provided to you.
	Make the best assumption about what the user needs help with.
	Always validate your own answer.
	void polite or generic statements like "let me know if you have other questions" or "feel free to ask".
	Only respond with the most relevant, concise, and helpful information.`

	// Load technical user prompt from JSON
	technicalPrompt := `I need general help with various tasks.`
	promptBytes, err := os.ReadFile(rulesJSON)
	if err == nil {
		var cfg PromptConfig
		if err := json.Unmarshal(promptBytes, &cfg); err == nil && cfg.TechnicalPrompt != "" {
			technicalPrompt = cfg.TechnicalPrompt
		}
	}

	return fmt.Sprintf(`%s

	What the user needs help with: %s`, systemPrompt, technicalPrompt)
}

func (s *Session) transcribeAudio(ctx context.Context, audioPath string) (string, error) {
	if audioPath == "" {
		return "", nil
	}

	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: audioPath,
	}
	resp, err := s.client.CreateTranscription(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

func compressAndEncodeImage(path string) (string, error) {
	// Read the file into memory
	imgBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Decode image format dynamically
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return "", err
	}

	// Compress to JPEG (low quality) in memory
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return "", err
	}

	// Convert to base64 data URI
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	dataURI := "data:image/jpeg;base64," + encoded
	return dataURI, nil
}

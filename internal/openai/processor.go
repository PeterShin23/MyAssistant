package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	imgData, err := os.ReadFile(screenshotPath)
	if err != nil {
		return fmt.Errorf("failed to read screenshot: %w", err)
	}
	imgB64 := base64.StdEncoding.EncodeToString(imgData)
	imgDataURI := "data:image/png;base64," + imgB64

	// 3. Load system prompt from JSON
	var cfg PromptConfig
	promptBytes, err := os.ReadFile(rulesJSON)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", rulesJSON, err)
	}
	if err := json.Unmarshal(promptBytes, &cfg); err != nil {
		return fmt.Errorf("failed to parse %s: %w", rulesJSON, err)
	}

	// 4. Append system message once
	s.mu.Lock()
	if len(s.messages) == 0 {
		var systemPrompt = buildSystemPrompt(cfg)

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

	// 5. Build user message with image and transcript
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

func buildSystemPrompt(r PromptConfig) string {
	systemPrompt := `You are the user's personal helper. 
	Use the image and audio transcript provided as context. 
	Assume that the user needs help with the context that's provided to you.
	Make the best assumption about what the user needs help with.
	Always validate your own answer.
	void polite or generic statements like "let me know if you have other questions" or "feel free to ask".
	Only respond with the most relevant, concise, and helpful information.`

	defaultTechincalPrompt := `
	I need general help with various tasks.
	`

	// Use technicalPrompt if it's not empty; otherwise use the default
	technicalPrompt := r.TechnicalPrompt
	if technicalPrompt == "" {
		technicalPrompt = defaultTechincalPrompt
	}

	return fmt.Sprintf(`%s

	What the user needs help with: %s`, systemPrompt, technicalPrompt)
}

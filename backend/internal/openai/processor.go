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
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/glamour"
	openai "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"

	"github.com/PeterShin23/MyAssistant/backend/internal/stream"
	// "github.com/openai/openai-go/v2/shared"
)

var rulesJSON = filepath.Join(projectRoot(), "rules.json")

func projectRoot() string {
	dir, _ := os.Getwd()
	return dir
}

type Session struct {
	mu       sync.Mutex
	client   openai.Client
	messages []openai.ChatCompletionMessageParamUnion
	writer   stream.StreamWriter
}

type PromptConfig struct {
	TechnicalPrompt string `json:"whatDoYouNeedHelpWith"`
}

func NewSession(writer stream.StreamWriter) (*Session, error) {
   apiKey := os.Getenv("OPENAI_API_KEY")

   if apiKey == "" {
	   return nil, errors.New("OPENAI_API_KEY not set")
   }

   client := openai.NewClient(option.WithAPIKey(apiKey))

   return &Session{
	   client:   client,
	   messages: []openai.ChatCompletionMessageParamUnion{},
	   writer:   writer,
   }, nil
}

func (s *Session) Process(screenshotPath, audioPath string, pretty bool) error {
   ctx := context.Background()

   // Wait for screenshot file with retry
   if err := waitForFileWithRetry(screenshotPath, 5, 2*time.Second); err != nil {
     return fmt.Errorf("screenshot file not available: %w", err)
   }

   // 1. Transcribe audio using Whisper (if available)
   var transcript string
   if audioPath != "" {
     if err := waitForFileWithRetry(audioPath, 5, 2*time.Second); err != nil {
       fmt.Printf("Audio file not available: %v (continuing without audio)\n", err)
     } else {
       transcript, err = s.transcribeAudio(ctx, audioPath)
       if err != nil {
         fmt.Printf("transcription failed: %v (continuing without transcript)\n", err)
       }
     }
   }

   // 2. Compress and encode screenshot as JPEG base64 data URI
   dataURI, err := compressAndEncodeImage(screenshotPath)
   if err != nil {
     return fmt.Errorf("failed to compress image: %w", err)
   }

   // Prepare image content part for OpenAI vision
   imagePart := openai.ChatCompletionContentPartUnionParam{
     OfImageURL: &openai.ChatCompletionContentPartImageParam{
       ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
         URL:   dataURI,
         Detail: "auto",
       },
     },
   }

   // Prepare transcript content part (if any)
   var contentParts []openai.ChatCompletionContentPartUnionParam
   contentParts = append(contentParts, imagePart)
   if transcript != "" {
     fmt.Printf("transcript: %s\n", transcript)
     contentParts = append(contentParts, openai.TextContentPart(fmt.Sprintf("Transcript:\n\n%s", transcript)))
   }

   // Prepare user message
   userMessage := openai.UserMessage(contentParts)

   // Append system message once and user message for this request
   s.mu.Lock()
   if len(s.messages) == 0 {
     var systemPrompt = buildSystemPrompt()
     s.messages = append(s.messages, openai.SystemMessage(systemPrompt))
   }
   s.messages = append(s.messages, userMessage)
   s.mu.Unlock()

   params := openai.ChatCompletionNewParams{
     Messages: s.messages,
     Model:    "gpt-4.1", // shared.ChatModelGPT5Mini,
    //  MaxCompletionTokens: openai.Int(4096),
   }

   stream := s.client.Chat.Completions.NewStreaming(ctx, params)
   defer stream.Close()

   fmt.Print("🤖 GPT Response:\n")

   var fullContent string
   chunkCount := 0
   for stream.Next() {
     chunk := stream.Current()
     if len(chunk.Choices) > 0 {
       delta := chunk.Choices[0].Delta.Content
       if delta != "" {
         chunkCount++
				//  fmt.Print(delta)
        //  fmt.Printf("[Processor] Processing chunk %d: %q\n", chunkCount, delta)
         if s.writer != nil {
           if err := s.writer.WriteChunk(delta); err != nil {
             // Log error but continue processing
             fmt.Printf("Warning: failed to write chunk %d to stream: %v\n", chunkCount, err)
           }
         }
         fullContent += delta
       }
     }
   }
   if err := stream.Err(); err != nil {
     return fmt.Errorf("stream error: %w", err)
   }

   fmt.Printf("[Processor] Stream completed. Total chunks received: %d, total content length: %d\n", chunkCount, len(fullContent))

   // Mark stream as complete but keep the connection open for next request
   if s.writer != nil {
     fmt.Printf("[Processor] Marking stream complete after %d chunks (keeping connection open)\n", chunkCount)
     
     if err := s.writer.MarkStreamComplete(); err != nil {
       fmt.Printf("Warning: failed to mark stream complete: %v\n", err)
     } else {
       fmt.Printf("[Processor] Stream marked as successfully\n")
     }
   }

   // Maintain Session Context - add assistant response to conversation
   s.mu.Lock()
   s.messages = append(s.messages, openai.AssistantMessage(fullContent))
   s.mu.Unlock()

   return nil
}

func buildSystemPrompt() string {
	systemPrompt := `You are the user's personal helper. 
	Use the image and audio transcript provided as context. 
	Assume that the user needs help with the context that's provided to you.
	Make the best assumption about what the user needs help with.
	Always validate your own answer.
	Avoid polite or generic statements like "let me know if you have other questions" or "feel free to ask".
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

   file, err := os.Open(audioPath)
   if err != nil {
	   return "", err
   }
   defer file.Close()

   params := openai.AudioTranscriptionNewParams{
	   File:  file,
	   Model: "whisper-1",
   }
   resp, err := s.client.Audio.Transcriptions.New(ctx, params)
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

// waitForFileWithRetry waits for a file to exist with retry logic
func waitForFileWithRetry(filePath string, maxRetries int, delay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		if _, err := os.Stat(filePath); err == nil {
			return nil // File exists
		}
		
		if i < maxRetries-1 {
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("file %s does not exist after %d retries", filePath, maxRetries)
}

func renderMarkdown(md string) (string, error) {
	out, err := glamour.Render(md, "dark")
	if err != nil {
		return "", err
	}
	return out, nil
}

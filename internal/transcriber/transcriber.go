package transcriber

import (
	"context"
	"fmt"

	"github.com/leonardotrapani/hyprvoice/internal/recording"
)

// Main transcriber interface
type Transcriber interface {
	Start(ctx context.Context, frameCh <-chan recording.AudioFrame) (<-chan error, error)
	Stop(ctx context.Context) error
	GetFinalTranscription() (string, error)
}

// Adapter interface for different transcription backends
type TranscriptionAdapter interface {
	Transcribe(ctx context.Context, audioData []byte) (string, error)
}

// Configuration for the transcriber
type Config struct {
	Provider  string
	APIKey    string
	Language  string
	Model     string
	ServerURL string // For local whisper.cpp server
}

// NewTranscriber creates a new simple transcriber
func NewTranscriber(config Config) (Transcriber, error) {
	// Create the appropriate adapter
	var adapter TranscriptionAdapter

	switch config.Provider {
	case "openai":
		if config.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key required")
		}
		adapter = NewOpenAIAdapter(config)

	case "groq-transcription":
		if config.APIKey == "" {
			return nil, fmt.Errorf("Groq API key required")
		}
		adapter = NewGroqTranscriptionAdapter(config)

	case "groq-translation":
		if config.APIKey == "" {
			return nil, fmt.Errorf("Groq API key required")
		}
		adapter = NewGroqTranslationAdapter(config)

	case "whisper-cpp":
		if config.ServerURL == "" {
			return nil, fmt.Errorf("whisper.cpp server URL required")
		}
		adapter = NewWhisperCppAdapter(config)

	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	// Create simple transcriber that collects all audio
	transcriber := NewSimpleTranscriber(config, adapter)

	return transcriber, nil
}

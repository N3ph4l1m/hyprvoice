package transcriber

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"
)

// WhisperCppAdapter implements TranscriptionAdapter for local whisper.cpp server
type WhisperCppAdapter struct {
	serverURL string
	config    Config
	client    *http.Client
}

// WhisperCppResponse represents the response from whisper.cpp server
type WhisperCppResponse struct {
	Text string `json:"text"`
}

func NewWhisperCppAdapter(config Config) *WhisperCppAdapter {
	return &WhisperCppAdapter{
		serverURL: config.ServerURL,
		config:    config,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (a *WhisperCppAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", nil
	}

	// Convert raw PCM to WAV format
	wavData, err := convertToWAV(audioData)
	if err != nil {
		return "", fmt.Errorf("convert to WAV: %w", err)
	}

	// Create multipart form request
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add audio file
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(wavData); err != nil {
		return "", fmt.Errorf("write audio data: %w", err)
	}

	// Add language if specified
	if a.config.Language != "" {
		if err := writer.WriteField("language", a.config.Language); err != nil {
			return "", fmt.Errorf("write language field: %w", err)
		}
	}

	// Add parameters for better accuracy (matching whisper.cpp server API)
	// temperature=0.0 makes output deterministic and more accurate
	if err := writer.WriteField("temperature", "0.0"); err != nil {
		return "", fmt.Errorf("write temperature field: %w", err)
	}

	// temperature_inc controls fallback temperature increases (0.2 is good default)
	if err := writer.WriteField("temperature_inc", "0.2"); err != nil {
		return "", fmt.Errorf("write temperature_inc field: %w", err)
	}

	// Request JSON response format
	if err := writer.WriteField("response_format", "json"); err != nil {
		return "", fmt.Errorf("write response_format field: %w", err)
	}

	// Close multipart writer
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", a.serverURL, &requestBody)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	start := time.Now()
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request to whisper.cpp server: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("whisper-cpp-adapter: server returned status %d: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("whisper.cpp server error: status %d", resp.StatusCode)
	}

	// Parse response
	var result WhisperCppResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	duration := time.Since(start)
	log.Printf("whisper-cpp-adapter: transcribed %d bytes in %v: %q", len(audioData), duration, result.Text)

	return result.Text, nil
}

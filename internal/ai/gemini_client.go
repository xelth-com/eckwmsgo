package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// GeminiClient interacts with Google Gemini API
type GeminiClient struct {
	APIKey string
	Model  string // e.g., "gemini-3-flash-preview"
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-3-flash-preview"
	}
	return &GeminiClient{
		APIKey: apiKey,
		Model:  model,
	}
}

// GeminiRequest represents a request to Gemini API
type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

// Content represents message content
type Content struct {
	Role  string `json:"role"`  // "user" or "model"
	Parts []Part `json:"parts"`
}

// Part represents a message part
type Part struct {
	Text string `json:"text"`
}

// GeminiResponse represents a response from Gemini API
type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

// Candidate represents a response candidate
type Candidate struct {
	Content       Content `json:"content"`
	FinishReason  string  `json:"finishReason"`
	SafetyRatings []struct {
		Category    string `json:"category"`
		Probability string `json:"probability"`
	} `json:"safetyRatings"`
}

// GenerateContent sends a prompt to Gemini and returns the response
func (c *GeminiClient) GenerateContent(prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		c.Model, c.APIKey)

	req := GeminiRequest{
		Contents: []Content{
			{
				Role: "user",
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return "", errors.New("no response candidates")
	}

	if len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("no response parts")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// ChatWithFunctionCalling sends a prompt with function calling capability
// This allows Gemini to decide which system functions to call
func (c *GeminiClient) ChatWithFunctionCalling(prompt string, availableFunctions []FunctionSpec) (string, []FunctionCall, error) {
	// TODO: Implement function calling with Gemini
	// Gemini supports function calling through the "tools" parameter
	// This would allow the AI to autonomously decide which functions to call
	return "", nil, errors.New("function calling not yet implemented")
}

// FunctionSpec represents a function specification for AI
type FunctionSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// FunctionCall represents an AI's decision to call a function
type FunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

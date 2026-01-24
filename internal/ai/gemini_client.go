package ai

import (
	"context"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiClient interacts with Google Gemini API using the official SDK
type GeminiClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(ctx context.Context, apiKey, modelName string) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is empty")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	if modelName == "" {
		modelName = "gemini-3-flash-preview"
	}

	model := client.GenerativeModel(modelName)

	// Optional: Configure safety settings if needed
	// model.SafetySettings = []*genai.SafetySetting{...}

	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

// Close closes the client connection
func (c *GeminiClient) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

// GenerateContent sends a prompt to Gemini and returns the response text
func (c *GeminiClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("gemini generation error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from gemini")
	}

	// Extract text from the first part
	// Note: In a real app, you might want to handle multiple parts/candidates
	var fullText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			fullText += string(txt)
		}
	}

	return fullText, nil
}

// ChatWithFunctionCalling sends a prompt with function calling capability
// This allows Gemini to decide which system functions to call
func (c *GeminiClient) ChatWithFunctionCalling(ctx context.Context, prompt string, availableFunctions []FunctionSpec) (string, []FunctionCall, error) {
	// TODO: Implement function calling with Gemini official SDK
	// The SDK supports function calling through model.Tools
	log.Println("⚠️ Function calling not yet implemented with official SDK")
	return "", nil, fmt.Errorf("function calling not yet implemented")
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

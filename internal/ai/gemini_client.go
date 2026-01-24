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
	client        *genai.Client
	primaryModel  string
	fallbackModel string
}

// NewGeminiClient creates a new Gemini API client with fallback support
func NewGeminiClient(ctx context.Context, apiKey, primaryModel, fallbackModel string) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is empty")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	if primaryModel == "" {
		primaryModel = "gemini-3-flash-preview"
	}
	if fallbackModel == "" {
		fallbackModel = "gemini-2.5-flash"
	}

	return &GeminiClient{
		client:        client,
		primaryModel:  primaryModel,
		fallbackModel: fallbackModel,
	}, nil
}

// Close closes the client connection
func (c *GeminiClient) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

// GenerateContent sends a prompt to Gemini (Primary -> Fallback)
func (c *GeminiClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	result, err := c.generateWithModel(ctx, c.primaryModel, prompt)
	if err == nil {
		return result, nil
	}

	log.Printf("⚠️ Primary model (%s) failed: %v. Switching to fallback (%s)...", c.primaryModel, err, c.fallbackModel)

	result, err = c.generateWithModel(ctx, c.fallbackModel, prompt)
	if err != nil {
		return "", fmt.Errorf("both primary and fallback models failed: %w", err)
	}

	return result, nil
}

// Helper to generate content with a specific model
func (c *GeminiClient) generateWithModel(ctx context.Context, modelName string, prompt string) (string, error) {
	model := c.client.GenerativeModel(modelName)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

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

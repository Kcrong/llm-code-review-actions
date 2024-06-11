package groq

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ModelID string

const (
	ModelIDLLAMA38B  ModelID = "llama3-8b-8192"
	ModelIDLLAMA370B ModelID = "llama3-70b-8192"
	ModelIDMIXTRAL   ModelID = "mixtral-8x7b-32768"
	ModelIDGEMMA     ModelID = "gemma-7b-it"
)

// ListModelsResponse represents the response from the list models API.
type ListModelsResponse struct {
	ObjectType string  `json:"object"` // Type of the object (e.g., "list")
	Data       []Model `json:"data"`   // List of models
}

// Model represents a single model returned by the list models or retrieve model API.
type Model struct {
	ID            ModelID `json:"id"`             // ID of the model
	ObjectType    string  `json:"object"`         // Type of the object (e.g., "model")
	Created       int64   `json:"created"`        // Timestamp of creation (seconds)
	OwnedBy       string  `json:"owned_by"`       // Owner of the model
	Active        bool    `json:"active"`         // Whether the model is active
	ContextWindow int     `json:"context_window"` // Context window size of the model
}

// ListModels sends a request to list all available models.
func (c *client) ListModels() (*ListModelsResponse, error) {
	url := fmt.Sprintf("%s/v1/models", c.baseURL)

	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	var modelsResp ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &modelsResp, nil
}

// RetrieveModel sends a request to retrieve a specific model by its ID.
func (c *client) RetrieveModel(t ModelID) (*Model, error) {
	url := fmt.Sprintf("%s/v1/models/%s", c.baseURL, t)

	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	var modelResp Model
	if err := json.NewDecoder(resp.Body).Decode(&modelResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &modelResp, nil
}

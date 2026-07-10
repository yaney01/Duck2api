package official

import (
	"encoding/json"
)

// ImageGenerationRequest is the OpenAI-compatible request for /v1/images/generations
type ImageGenerationRequest struct {
	Prompt          string `json:"prompt"`
	Model           string `json:"model,omitempty"`
	N               int    `json:"n,omitempty"`
	Size            string `json:"size,omitempty"`
	ResponseFormat  string `json:"response_format,omitempty"` // "url" or "b64_json"
	Quality         string `json:"quality,omitempty"`
	Style           string `json:"style,omitempty"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"` // "none", "low", "medium", "high"
}

// ImageEditRequest is the OpenAI-compatible request for /v1/images/edits
type ImageEditRequest struct {
	Image           string `json:"image"` // base64 encoded image
	Mask            string `json:"mask"`  // base64 encoded mask (optional)
	Prompt          string `json:"prompt"`
	Model           string `json:"model,omitempty"`
	N               int    `json:"n,omitempty"`
	Size            string `json:"size,omitempty"`
	ResponseFormat  string `json:"response_format,omitempty"`  // "url" or "b64_json"
	ReasoningEffort string `json:"reasoning_effort,omitempty"` // "none", "low", "medium", "high"
}

// ImageData represents a single generated image in the response
type ImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImageGenerationResponse is the OpenAI-compatible response for image endpoints.
// Duck.ai may return multiple intermediate candidates even when n=1. The custom
// JSON encoder keeps the last valid base64 candidate because the SSE stream emits
// legacy preview parts before the final GenerateImage result.
type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

func (r ImageGenerationResponse) MarshalJSON() ([]byte, error) {
	data := r.Data
	if len(data) > 1 {
		bestIndex := -1
		for i := len(data) - 1; i >= 0; i-- {
			if data[i].B64JSON != "" {
				bestIndex = i
				break
			}
		}

		if bestIndex >= 0 {
			data = []ImageData{data[bestIndex]}
		}
	}

	type responseAlias struct {
		Created int64       `json:"created"`
		Data    []ImageData `json:"data"`
	}

	return json.Marshal(responseAlias{
		Created: r.Created,
		Data:    data,
	})
}

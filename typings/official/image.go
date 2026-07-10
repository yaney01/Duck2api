package official

import (
	"encoding/base64"
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
	ResponseFormat  string `json:"response_format,omitempty"` // "url" or "b64_json"
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
// JSON encoder keeps only the largest valid base64 candidate so clients such as
// OpenWebUI receive the most complete image instead of an early preview.
type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

func (r ImageGenerationResponse) MarshalJSON() ([]byte, error) {
	data := r.Data
	if len(data) > 1 {
		bestIndex := -1
		bestSize := -1

		for i, image := range data {
			if image.B64JSON == "" {
				continue
			}

			size := len(image.B64JSON)
			if decoded, err := base64.StdEncoding.DecodeString(image.B64JSON); err == nil {
				size = len(decoded)
			}

			if size > bestSize {
				bestIndex = i
				bestSize = size
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

package official

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestImageGenerationResponseSelectsLastValidCandidate(t *testing.T) {
	preview := base64.StdEncoding.EncodeToString([]byte("this preview is intentionally larger"))
	final := base64.StdEncoding.EncodeToString([]byte("final"))

	payload, err := json.Marshal(ImageGenerationResponse{
		Created: 1,
		Data: []ImageData{
			{B64JSON: preview, RevisedPrompt: "legacy preview"},
			{B64JSON: final, RevisedPrompt: "final GenerateImage result"},
			{RevisedPrompt: "empty trailing event"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var decoded struct {
		Data []ImageData `json:"data"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}

	if len(decoded.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(decoded.Data))
	}
	if decoded.Data[0].B64JSON != final {
		t.Fatal("did not select the last valid candidate")
	}
}

func TestImageGenerationResponseKeepsSingleCandidate(t *testing.T) {
	image := base64.StdEncoding.EncodeToString([]byte("single"))
	payload, err := json.Marshal(ImageGenerationResponse{
		Created: 1,
		Data:    []ImageData{{B64JSON: image}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var decoded struct {
		Data []ImageData `json:"data"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Data) != 1 || decoded.Data[0].B64JSON != image {
		t.Fatal("single candidate changed unexpectedly")
	}
}

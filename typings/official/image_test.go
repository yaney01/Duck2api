package official

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestImageGenerationResponseSelectsLargestCandidate(t *testing.T) {
	small := base64.StdEncoding.EncodeToString([]byte("small"))
	large := base64.StdEncoding.EncodeToString([]byte("this candidate is larger"))

	payload, err := json.Marshal(ImageGenerationResponse{
		Created: 1,
		Data: []ImageData{
			{B64JSON: small, RevisedPrompt: "small"},
			{B64JSON: large, RevisedPrompt: "large"},
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
	if decoded.Data[0].B64JSON != large {
		t.Fatal("did not select the largest candidate")
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

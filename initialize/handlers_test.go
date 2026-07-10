package initialize

import (
	duckgoConvert "aurora/conversion/requests/duckgo"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildImageEditRequestIncludesReferenceImages(t *testing.T) {
	pngImage := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nimage"))
	webpImage := "data:image/webp;base64," + base64.StdEncoding.EncodeToString([]byte("webp"))

	request := buildImageEditRequest("edit only the marked area", "gpt-5.4-mini", []string{pngImage, webpImage})
	translated := duckgoConvert.ConvertAPIRequestWithOptions(request, "", false)

	payload, err := json.Marshal(translated)
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)

	if !strings.Contains(text, `"text":"edit only the marked area"`) {
		t.Fatalf("translated request does not contain the edit prompt: %s", text)
	}
	if !strings.Contains(text, `"image":"data:image/png;base64,`+pngImage+`"`) {
		t.Fatalf("translated request does not contain the normalized PNG reference: %s", text)
	}
	if !strings.Contains(text, `"image":"`+webpImage+`"`) {
		t.Fatalf("translated request does not contain the WebP reference: %s", text)
	}
}

func TestBuildImageEditRequestSkipsEmptyReferences(t *testing.T) {
	request := buildImageEditRequest("edit", "gpt-5.4-mini", []string{"", "  "})
	translated := duckgoConvert.ConvertAPIRequestWithOptions(request, "", false)

	payload, err := json.Marshal(translated)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"type":"image"`) {
		t.Fatalf("translated request unexpectedly contains an empty image: %s", payload)
	}
}

func TestImageEditReferenceLimit(t *testing.T) {
	if maxImageEditReferences != 4 {
		t.Fatalf("expected image edit reference limit 4, got %d", maxImageEditReferences)
	}
}

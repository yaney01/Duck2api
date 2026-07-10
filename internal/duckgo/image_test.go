package duckgo

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestReadImageResponsePrefersGenerateImageResult(t *testing.T) {
	body := strings.Join([]string{
		`data: {"action":"success","parts":[{"type":"generated-image","result":"larger legacy preview"}]}`,
		`data: {"action":"success","toolName":"GenerateImage","data":{"b64Image":"final","format":"jpeg","width":1024,"height":1365}}`,
		"data: [DONE]",
		"",
	}, "\n")
	response := &http.Response{Body: io.NopCloser(strings.NewReader(body))}

	result := ReadImageResponse(response)

	if len(result.Images) != 1 {
		t.Fatalf("expected 1 final image, got %d", len(result.Images))
	}
	if result.Images[0].Result != "final" {
		t.Fatalf("expected GenerateImage result, got %q", result.Images[0].Result)
	}
}

func TestReadImageResponseFallsBackToLegacyImage(t *testing.T) {
	body := strings.Join([]string{
		`data: {"action":"success","parts":[{"type":"generated-image","result":"legacy only"}]}`,
		"data: [DONE]",
		"",
	}, "\n")
	response := &http.Response{Body: io.NopCloser(strings.NewReader(body))}

	result := ReadImageResponse(response)

	if len(result.Images) != 1 || result.Images[0].Result != "legacy only" {
		t.Fatalf("expected legacy fallback, got %#v", result.Images)
	}
}

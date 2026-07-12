package duckgo

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"
)

func testBase64(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(value))
}

func TestReadImageResponsePrefersExplicitFinalCandidate(t *testing.T) {
	final := testBase64("final")
	preview := testBase64("this preview is intentionally much larger than final")
	body := strings.Join([]string{
		`data: {"action":"success","state":"completed","parts":[{"type":"generated-image","result":"` + final + `","data":{"status":"completed","type":"final"}}]}`,
		`data: {"action":"success","state":"processing","toolName":"GenerateImage","data":{"b64Image":"` + preview + `","status":"preview","type":"preview"}}`,
		"data: [DONE]", "",
	}, "\n")
	result := ReadImageResponse(&http.Response{Body: io.NopCloser(strings.NewReader(body))})
	if len(result.Images) != 1 || result.Images[0].Result != final {
		t.Fatalf("expected explicit final candidate, got %#v", result.Images)
	}
}

func TestReadImageResponseAcceptsNonSuccessFinalEvent(t *testing.T) {
	preview := testBase64("preview payload is intentionally larger")
	final := testBase64("final")
	body := strings.Join([]string{
		`data: {"action":"success","state":"data","toolName":"GenerateImage","data":{"b64Image":"` + preview + `","status":"partial","type":"image-partial"}}`,
		`data: {"action":"update","state":"completed","toolName":"GenerateImage","data":{"b64Image":"` + final + `","status":"completed","type":"image-final"}}`,
		"data: [DONE]", "",
	}, "\n")
	result := ReadImageResponse(&http.Response{Body: io.NopCloser(strings.NewReader(body))})
	if len(result.Images) != 1 || result.Images[0].Result != final {
		t.Fatalf("expected non-success final candidate, got %#v", result.Images)
	}
}

func TestReadImageResponseProcessesUnterminatedFinalLine(t *testing.T) {
	preview := testBase64("preview payload is larger")
	final := testBase64("final")
	body := `data: {"action":"success","toolName":"GenerateImage","data":{"b64Image":"` + preview + `","status":"partial","type":"image-partial"}}` + "\n" +
		`data: {"action":"success","state":"completed","toolName":"GenerateImage","data":{"b64Image":"` + final + `","status":"completed","type":"image-final"}}`
	result := ReadImageResponse(&http.Response{Body: io.NopCloser(strings.NewReader(body))})
	if len(result.Images) != 1 || result.Images[0].Result != final {
		t.Fatalf("expected final unterminated SSE line, got %#v", result.Images)
	}
}

func TestReadImageResponsePrefersLargerPayloadWithoutMetadata(t *testing.T) {
	larger := testBase64("higher quality image payload with more decoded bytes")
	smaller := testBase64("preview")
	body := strings.Join([]string{
		`data: {"action":"success","parts":[{"type":"generated-image","result":"` + larger + `"}]}`,
		`data: {"action":"success","toolName":"GenerateImage","data":{"b64Image":"` + smaller + `"}}`,
		"data: [DONE]", "",
	}, "\n")
	result := ReadImageResponse(&http.Response{Body: io.NopCloser(strings.NewReader(body))})
	if len(result.Images) != 1 || result.Images[0].Result != larger {
		t.Fatalf("expected larger decoded payload, got %#v", result.Images)
	}
}

func TestReadImageResponseRejectsPartialOnlyResult(t *testing.T) {
	preview := testBase64("blurred partial preview")
	body := strings.Join([]string{
		`data: {"action":"success","state":"data","toolName":"GenerateImage","data":{"b64Image":"` + preview + `","status":"partial","type":"image-partial","format":"jpeg","width":1024,"height":1536}}`,
		`data: {"action":"success","state":"result","result":"Image generation failed: The image could not be generated due to content policy."}`,
		`data: {"action":"success","state":"data","toolName":"GenerateImage","data":{"status":"error","error":"content_policy","message":"The image could not be generated due to content policy."}}`,
		"data: [DONE]", "",
	}, "\n")
	result := ReadImageResponse(&http.Response{Body: io.NopCloser(strings.NewReader(body))})
	if len(result.Images) != 0 {
		t.Fatalf("expected partial-only generation to be rejected, got %#v", result.Images)
	}
}

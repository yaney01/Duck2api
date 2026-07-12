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
	largerPreview := testBase64("this preview is intentionally much larger than final")
	body := strings.Join([]string{
		`data: {"action":"success","state":"completed","parts":[{"type":"generated-image","result":"` + final + `","width":1024,"height":1365,"data":{"status":"completed","type":"final"}}]}`,
		`data: {"action":"success","state":"processing","toolName":"GenerateImage","data":{"b64Image":"` + largerPreview + `","format":"jpeg","width":1024,"height":1365,"status":"preview","type":"preview"}}`,
		"data: [DONE]",
		"",
	}, "\n")
	response := &http.Response{Body: io.NopCloser(strings.NewReader(body))}

	result := ReadImageResponse(response)

	if len(result.Images) != 1 {
		t.Fatalf("expected 1 selected image, got %d", len(result.Images))
	}
	if result.Images[0].Result != final {
		t.Fatalf("expected explicit final candidate, got %q", result.Images[0].Result)
	}
}

func TestReadImageResponsePrefersLargerDecodedPayloadWhenMetadataMissing(t *testing.T) {
	larger := testBase64("higher quality image payload with more decoded bytes")
	smaller := testBase64("preview")
	body := strings.Join([]string{
		`data: {"action":"success","parts":[{"type":"generated-image","result":"` + larger + `","width":1024,"height":1365}]}`,
		`data: {"action":"success","toolName":"GenerateImage","data":{"b64Image":"` + smaller + `","format":"jpeg","width":1024,"height":1365}}`,
		"data: [DONE]",
		"",
	}, "\n")
	response := &http.Response{Body: io.NopCloser(strings.NewReader(body))}

	result := ReadImageResponse(response)

	if len(result.Images) != 1 || result.Images[0].Result != larger {
		t.Fatalf("expected larger decoded payload, got %#v", result.Images)
	}
}

func TestReadImageResponseUsesLaterCandidateForExactTie(t *testing.T) {
	first := testBase64("first")
	second := testBase64("later")
	body := strings.Join([]string{
		`data: {"action":"success","parts":[{"type":"generated-image","result":"` + first + `"}]}`,
		`data: {"action":"success","toolName":"GenerateImage","data":{"b64Image":"` + second + `","format":"jpeg"}}`,
		"data: [DONE]",
		"",
	}, "\n")
	response := &http.Response{Body: io.NopCloser(strings.NewReader(body))}

	result := ReadImageResponse(response)

	if len(result.Images) != 1 || result.Images[0].Result != second {
		t.Fatalf("expected later exact-tie candidate, got %#v", result.Images)
	}
}

func TestReadImageResponseFallsBackToLegacyImage(t *testing.T) {
	legacy := testBase64("legacy only")
	body := strings.Join([]string{
		`data: {"action":"success","parts":[{"type":"generated-image","result":"` + legacy + `"}]}`,
		"data: [DONE]",
		"",
	}, "\n")
	response := &http.Response{Body: io.NopCloser(strings.NewReader(body))}

	result := ReadImageResponse(response)

	if len(result.Images) != 1 || result.Images[0].Result != legacy {
		t.Fatalf("expected legacy fallback, got %#v", result.Images)
	}
}

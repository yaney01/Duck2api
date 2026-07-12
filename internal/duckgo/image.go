package duckgo

import (
	duckgotypes "aurora/typings/duckgo"
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

type ImageResult struct {
	Text   string
	Images []duckgotypes.ImagePart
}

type imageCandidate struct {
	Image      duckgotypes.ImagePart
	Source     string
	EventIndex int
	Action     string
	Status     string
	DataType   string
	Title      string
	State      string
	Name       string
	ToolName   string
	ToolCallID string
	ByteSize   int
	Hash       string
	Score      int
}

var finalStateTokens = []string{
	"completed", "complete", "final", "finished",
	"succeeded", "success", "ready", "done",
}

var previewStateTokens = []string{
	"preview", "thumbnail", "thumb", "draft", "intermediate",
	"partial", "progress", "processing", "pending", "generating",
	"started", "loading",
}

// ReadImageResponse processes every SSE image event, including non-success
// actions and a final line that is not terminated by a newline.
func ReadImageResponse(response *http.Response) ImageResult {
	reader := bufio.NewReader(response.Body)
	var textBuilder strings.Builder
	var candidates []imageCandidate
	eventIndex := 0

	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return ImageResult{}
		}

		payload, ok := ssePayload(line)
		if ok && payload != "" && !isControlPayload(payload) {
			eventIndex++
			var apiResp duckgotypes.ApiResponse
			if err := json.Unmarshal([]byte(payload), &apiResp); err == nil {
				if apiResp.Action == "success" && apiResp.Message != "" {
					textBuilder.WriteString(apiResp.Message)
				}
				candidates = append(candidates, imageCandidates(apiResp, eventIndex)...)
			}
		}

		if readErr == io.EOF {
			break
		}
	}

	best, ok := selectBestImageCandidate(candidates)
	if !ok {
		return ImageResult{Text: textBuilder.String()}
	}
	return ImageResult{
		Text:   textBuilder.String(),
		Images: []duckgotypes.ImagePart{best.Image},
	}
}

func ssePayload(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "data:") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(line, "data:")), true
}

func isControlPayload(payload string) bool {
	return strings.HasPrefix(payload, "[DONE]") ||
		strings.HasPrefix(payload, "[PING]") ||
		strings.HasPrefix(payload, "[CHAT_TITLE")
}

func imageCandidates(apiResp duckgotypes.ApiResponse, eventIndex int) []imageCandidate {
	base := imageCandidate{
		EventIndex: eventIndex,
		Action:     apiResp.Action,
		State:      apiResp.State,
		Name:       apiResp.Name,
		ToolName:   apiResp.ToolName,
		ToolCallID: apiResp.ToolCallId,
	}
	var candidates []imageCandidate

	if apiResp.ToolName == "GenerateImage" || len(apiResp.Parts) > 0 || apiResp.Result != "" {
		log.Printf(
			"[IMAGE_EVENT] event=%d action=%q tool=%q toolCallId=%s state=%q name=%q parts=%d dataBytes=%d resultBytes=%d",
			eventIndex, apiResp.Action, apiResp.ToolName, apiResp.ToolCallId,
			apiResp.State, apiResp.Name, len(apiResp.Parts), len(apiResp.Data), len(apiResp.Result),
		)
	}

	for _, part := range apiResp.Parts {
		if part.Type != "generated-image" && part.Type != "image" {
			continue
		}
		candidate := base
		candidate.Image = part
		candidate.Source = "parts"
		if part.Data != nil {
			candidate.Status = part.Data.Status
			candidate.DataType = part.Data.Type
			candidate.Title = part.Data.Title
			if candidate.Image.Result == "" {
				candidate.Image.Result = part.Data.B64Image
			}
			if candidate.Image.Format == "" {
				candidate.Image.Format = part.Data.Format
			}
			if candidate.Image.Width == 0 {
				candidate.Image.Width = part.Data.Width
			}
			if candidate.Image.Height == 0 {
				candidate.Image.Height = part.Data.Height
			}
		}
		if candidate.Image.Result != "" {
			candidates = append(candidates, finalizeCandidate(candidate))
		}
	}

	if apiResp.ToolName == "GenerateImage" && apiResp.Data != nil {
		if imgData := apiResp.GetImageData(); imgData != nil {
			candidate := base
			candidate.Source = "GenerateImage"
			candidate.Status = imgData.Status
			candidate.DataType = imgData.Type
			candidate.Title = imgData.Title
			candidate.Image = duckgotypes.ImagePart{
				Type:   "generated-image",
				Result: imgData.B64Image,
				Format: imgData.Format,
				Width:  imgData.Width,
				Height: imgData.Height,
			}
			candidates = append(candidates, finalizeCandidate(candidate))
		}
	}

	if apiResp.Result != "" && isEncodedImage(apiResp.Result) {
		candidate := base
		candidate.Source = "result"
		candidate.Status = apiResp.State
		candidate.DataType = apiResp.Name
		candidate.Image = duckgotypes.ImagePart{
			Type:   "generated-image",
			Result: apiResp.Result,
		}
		candidates = append(candidates, finalizeCandidate(candidate))
	}

	return candidates
}

func finalizeCandidate(candidate imageCandidate) imageCandidate {
	decoded, ok := decodeImageBase64(candidate.Image.Result)
	if ok {
		candidate.ByteSize = len(decoded)
		digest := sha256.Sum256(decoded)
		candidate.Hash = hex.EncodeToString(digest[:6])
	} else {
		candidate.ByteSize = len(candidate.Image.Result)
		digest := sha256.Sum256([]byte(candidate.Image.Result))
		candidate.Hash = hex.EncodeToString(digest[:6])
	}
	candidate.Score = candidateMetadataScore(candidate)
	return candidate
}

func selectBestImageCandidate(candidates []imageCandidate) (imageCandidate, bool) {
	if len(candidates) == 0 {
		return imageCandidate{}, false
	}
	bestIndex := -1
	for index, candidate := range candidates {
		log.Printf(
			"[IMAGE] candidate=%d event=%d action=%q source=%s toolCallId=%s state=%q status=%q type=%q title=%q format=%s size=%dx%d bytes=%d sha256=%s score=%d",
			index, candidate.EventIndex, candidate.Action, candidate.Source,
			candidate.ToolCallID, candidate.State, candidate.Status,
			candidate.DataType, candidate.Title, candidate.Image.Format,
			candidate.Image.Width, candidate.Image.Height, candidate.ByteSize,
			candidate.Hash, candidate.Score,
		)
		if bestIndex < 0 || candidateIsBetter(candidate, candidates[bestIndex]) {
			bestIndex = index
		}
	}
	best := candidates[bestIndex]
	log.Printf(
		"[IMAGE] selected=%d event=%d action=%q source=%s toolCallId=%s bytes=%d sha256=%s score=%d",
		bestIndex, best.EventIndex, best.Action, best.Source, best.ToolCallID,
		best.ByteSize, best.Hash, best.Score,
	)
	return best, true
}

func candidateIsBetter(candidate imageCandidate, current imageCandidate) bool {
	if candidate.Score != current.Score {
		return candidate.Score > current.Score
	}
	if candidate.ByteSize != current.ByteSize {
		return candidate.ByteSize > current.ByteSize
	}
	return candidate.EventIndex > current.EventIndex
}

func candidateMetadataScore(candidate imageCandidate) int {
	return metadataFieldScore(candidate.Status, 6000, -6000) +
		metadataFieldScore(candidate.DataType, 5000, -5000) +
		metadataFieldScore(candidate.State, 4000, -4000) +
		metadataFieldScore(candidate.Name, 2000, -2000) +
		metadataFieldScore(candidate.Title, 1000, -1000)
}

func metadataFieldScore(value string, finalScore int, previewScore int) int {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return 0
	}
	if containsAny(normalized, previewStateTokens) {
		return previewScore
	}
	if containsAny(normalized, finalStateTokens) {
		return finalScore
	}
	return 0
}

func containsAny(value string, tokens []string) bool {
	for _, token := range tokens {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func isEncodedImage(value string) bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "data:image/") {
		return true
	}
	decoded, ok := decodeImageBase64(value)
	if !ok || len(decoded) < 3 {
		return false
	}
	if decoded[0] == 0xff && decoded[1] == 0xd8 && decoded[2] == 0xff {
		return true
	}
	if len(decoded) >= 8 && string(decoded[:8]) == "\x89PNG\r\n\x1a\n" {
		return true
	}
	if len(decoded) >= 12 && string(decoded[:4]) == "RIFF" && string(decoded[8:12]) == "WEBP" {
		return true
	}
	return false
}

func decodeImageBase64(value string) ([]byte, bool) {
	value = strings.TrimSpace(value)
	if comma := strings.IndexByte(value, ','); comma >= 0 && strings.Contains(strings.ToLower(value[:comma]), "base64") {
		value = value[comma+1:]
	}
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil && len(decoded) > 0 {
			return decoded, true
		}
	}
	return nil, false
}

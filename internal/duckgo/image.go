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

// ImageResult holds the extracted image data from the SSE stream.
type ImageResult struct {
	Text   string
	Images []duckgotypes.ImagePart
}

type imageCandidate struct {
	Image      duckgotypes.ImagePart
	Source     string
	EventIndex int
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
	"completed",
	"complete",
	"final",
	"finished",
	"succeeded",
	"success",
	"ready",
	"done",
}

var previewStateTokens = []string{
	"preview",
	"thumbnail",
	"thumb",
	"draft",
	"intermediate",
	"partial",
	"progress",
	"processing",
	"pending",
	"generating",
	"started",
	"loading",
}

// ReadImageResponse reads the SSE response, keeps every image candidate and
// returns the candidate that is most likely to be the final-quality image.
func ReadImageResponse(response *http.Response) ImageResult {
	reader := bufio.NewReader(response.Body)
	var textBuilder strings.Builder
	var candidates []imageCandidate
	eventIndex := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return ImageResult{}
		}
		if len(line) < 6 {
			continue
		}
		line = line[6:]
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "[DONE]") || strings.HasPrefix(line, "[PING]") || strings.HasPrefix(line, "[CHAT_TITLE") {
			continue
		}

		eventIndex++
		var apiResp duckgotypes.ApiResponse
		if err = json.Unmarshal([]byte(line), &apiResp); err != nil || apiResp.Action != "success" {
			continue
		}

		if apiResp.Message != "" {
			textBuilder.WriteString(apiResp.Message)
		}

		// Legacy parts can contain either a preview or the final image. Keep all
		// of them instead of discarding this source when GenerateImage exists.
		for _, part := range apiResp.Parts {
			if part.Type != "generated-image" && part.Type != "image" {
				continue
			}

			candidate := imageCandidate{
				Image:      part,
				Source:     "parts",
				EventIndex: eventIndex,
				State:      apiResp.State,
				Name:       apiResp.Name,
				ToolName:   apiResp.ToolName,
				ToolCallID: apiResp.ToolCallId,
			}
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

		// New GenerateImage events may also include previews and final results.
		if apiResp.ToolName == "GenerateImage" && apiResp.Data != nil {
			if imgData := apiResp.GetImageData(); imgData != nil && imgData.B64Image != "" {
				candidate := imageCandidate{
					Image: duckgotypes.ImagePart{
						Type:   "generated-image",
						Result: imgData.B64Image,
						Format: imgData.Format,
						Width:  imgData.Width,
						Height: imgData.Height,
					},
					Source:     "GenerateImage",
					EventIndex: eventIndex,
					Status:     imgData.Status,
					DataType:   imgData.Type,
					Title:      imgData.Title,
					State:      apiResp.State,
					Name:       apiResp.Name,
					ToolName:   apiResp.ToolName,
					ToolCallID: apiResp.ToolCallId,
				}
				candidates = append(candidates, finalizeCandidate(candidate))
			}
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
			"[IMAGE] candidate=%d event=%d source=%s toolCallId=%s state=%q status=%q type=%q title=%q format=%s size=%dx%d bytes=%d sha256=%s score=%d",
			index,
			candidate.EventIndex,
			candidate.Source,
			candidate.ToolCallID,
			candidate.State,
			candidate.Status,
			candidate.DataType,
			candidate.Title,
			candidate.Image.Format,
			candidate.Image.Width,
			candidate.Image.Height,
			candidate.ByteSize,
			candidate.Hash,
			candidate.Score,
		)

		if bestIndex < 0 || candidateIsBetter(candidate, candidates[bestIndex]) {
			bestIndex = index
		}
	}

	best := candidates[bestIndex]
	log.Printf(
		"[IMAGE] selected=%d event=%d source=%s toolCallId=%s bytes=%d sha256=%s score=%d",
		bestIndex,
		best.EventIndex,
		best.Source,
		best.ToolCallID,
		best.ByteSize,
		best.Hash,
		best.Score,
	)
	return best, true
}

func candidateIsBetter(candidate imageCandidate, current imageCandidate) bool {
	// Explicit final/preview metadata is the strongest signal.
	if candidate.Score != current.Score {
		return candidate.Score > current.Score
	}

	// Duck.ai previews and final images can have identical dimensions. When
	// metadata is absent or tied, prefer the candidate carrying more decoded
	// image data rather than using width/height or Base64 string length.
	if candidate.ByteSize != current.ByteSize {
		return candidate.ByteSize > current.ByteSize
	}

	// Exact ties are resolved by SSE order because final events normally arrive
	// after intermediate events.
	return candidate.EventIndex > current.EventIndex
}

func candidateMetadataScore(candidate imageCandidate) int {
	score := 0

	// The inner image status/type is the most specific signal.
	score += metadataFieldScore(candidate.Status, 6000, -6000)
	score += metadataFieldScore(candidate.DataType, 5000, -5000)

	// Outer tool state/name and title are weaker but still useful.
	score += metadataFieldScore(candidate.State, 4000, -4000)
	score += metadataFieldScore(candidate.Name, 2000, -2000)
	score += metadataFieldScore(candidate.Title, 1000, -1000)

	return score
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

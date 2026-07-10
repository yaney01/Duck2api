package initialize

import (
	duckgoConvert "aurora/conversion/requests/duckgo"
	"aurora/httpclient/bogdanfinn"
	"aurora/internal/duckgo"
	"aurora/internal/proxys"
	duckgotypes "aurora/typings/duckgo"
	officialtypes "aurora/typings/official"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	proxy *proxys.IProxy
}

const maxImageEditReferences = 4

func NewHandle(proxy *proxys.IProxy) *Handler {
	// Wire up file store for file_id resolution in chat
	duckgoConvert.FileStore = func(fileID string) (string, string, []byte, bool) {
		f, ok := fileStorage[fileID]
		if !ok {
			return "", "", nil, false
		}
		return f.Filename, f.MimeType, f.Bytes, true
	}
	return &Handler{proxy: proxy}
}

func optionsHandler(c *gin.Context) {
	// Set headers for CORS
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "POST")
	c.Header("Access-Control-Allow-Headers", "*")
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func (h *Handler) duckduckgo(c *gin.Context) {
	var original_request officialtypes.APIRequest
	err := c.BindJSON(&original_request)
	if err != nil {
		c.JSON(400, gin.H{"error": gin.H{
			"message": "Request must be proper JSON",
			"type":    "invalid_request_error",
			"param":   nil,
			"code":    err.Error(),
		}})
		return
	}
	translated_request, response, err := h.startDuckDuckGoRequest(original_request)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer response.Body.Close()

	// Debug: log upstream response status
	if response.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(response.Body)
		log.Printf("[DEBUG] DuckDuckGo returned %d: %s", response.StatusCode, string(bodyBytes))
		// Reconstruct response for error handler
		c.JSON(response.StatusCode, gin.H{"error": gin.H{
			"message": string(bodyBytes),
			"type":    "upstream_error",
			"code":    response.Status,
			"model":   translated_request.Model,
		}})
		return
	}
	response_part := duckgo.Handler(c, response, translated_request, original_request.Stream)
	if c.Writer.Status() != 200 {
		return
	}
	if !original_request.Stream {
		c.JSON(200, officialtypes.NewChatCompletionWithModel(response_part, translated_request.Model))
	} else {
		c.String(200, "data: [DONE]\n\n")
	}
}

func (h *Handler) responses(c *gin.Context) {
	var responseRequest officialtypes.ResponseAPIRequest
	err := c.BindJSON(&responseRequest)
	if err != nil {
		c.JSON(400, gin.H{"error": gin.H{
			"message": "Request must be proper JSON",
			"type":    "invalid_request_error",
			"param":   nil,
			"code":    err.Error(),
		}})
		return
	}

	chatRequest := responseRequest.ToChatCompletionRequest()
	translatedRequest, response, err := h.startDuckDuckGoRequest(chatRequest)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		c.JSON(response.StatusCode, gin.H{
			"error": duckgo.ReadResponseError(response).Error(),
		})
		return
	}

	responseText := duckgo.ReadResponseText(response)

	if responseRequest.Stream {
		writeResponsesStream(c, responseText, translatedRequest.Model)
		return
	}

	c.JSON(http.StatusOK, officialtypes.NewResponseAPIWithModel(responseText, translatedRequest.Model))
}

func (h *Handler) startDuckDuckGoRequest(originalRequest officialtypes.APIRequest) (duckgotypes.ApiRequest, *http.Response, error) {
	proxyUrl := h.proxy.GetProxyIP()
	client := bogdanfinn.NewStdClient()
	token, err := duckgo.InitXVQD(client, proxyUrl)
	if err != nil {
		return duckgotypes.ApiRequest{}, nil, err
	}

	reasoningEffort := originalRequest.ReasoningEffort
	webSearch := originalRequest.WebSearch != nil && *originalRequest.WebSearch

	translatedRequest := duckgoConvert.ConvertAPIRequestWithOptions(originalRequest, reasoningEffort, webSearch)

	// Debug: log request
	reqJSON, _ := json.Marshal(translatedRequest)
	log.Printf("[DEBUG] DuckDuckGo request: %s", truncateStr(string(reqJSON), 2000))

	response, err := duckgo.POSTconversation(client, translatedRequest, token, proxyUrl)
	if err != nil {
		return duckgotypes.ApiRequest{}, nil, err
	}
	return translatedRequest, response, nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func writeResponsesStream(c *gin.Context, text string, model string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	response := officialtypes.NewResponseAPIWithModel("", model)
	response.Status = "in_progress"
	response.Output = []officialtypes.ResponseOutput{}
	output := officialtypes.NewResponseOutput("")
	output.Status = "in_progress"
	part := officialtypes.ResponseOutputContent{
		Type:        "output_text",
		Text:        "",
		Annotations: []interface{}{},
	}
	donePart := officialtypes.ResponseOutputContent{
		Type:        "output_text",
		Text:        text,
		Annotations: []interface{}{},
	}
	events := []officialtypes.ResponseStreamEvent{
		{Type: "response.created", Sequence: 1, Response: &response},
		{Type: "response.output_item.added", Sequence: 2, OutputIndex: 0, Item: &output},
		{Type: "response.content_part.added", Sequence: 3, ItemID: output.ID, OutputIndex: 0, ContentIndex: 0, Part: part},
		{Type: "response.output_text.delta", Sequence: 4, ItemID: output.ID, OutputIndex: 0, ContentIndex: 0, Delta: text},
		{Type: "response.output_text.done", Sequence: 5, ItemID: output.ID, OutputIndex: 0, ContentIndex: 0, Text: text},
		{Type: "response.content_part.done", Sequence: 6, ItemID: output.ID, OutputIndex: 0, ContentIndex: 0, Part: donePart},
	}

	completed := officialtypes.NewResponseAPIWithModel(text, model)
	events = append(events,
		officialtypes.ResponseStreamEvent{Type: "response.output_item.done", Sequence: 7, OutputIndex: 0, Item: &completed.Output[0]},
		officialtypes.ResponseStreamEvent{Type: "response.completed", Sequence: 8, Response: &completed},
	)

	for _, event := range events {
		c.Writer.WriteString("event: " + event.Type + "\n")
		c.Writer.WriteString("data: " + event.String() + "\n\n")
		c.Writer.Flush()
	}
}

func (h *Handler) imageGenerations(c *gin.Context) {
	var req officialtypes.ImageGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{
			"message": "Request must be proper JSON",
			"type":    "invalid_request_error",
			"param":   nil,
			"code":    err.Error(),
		}})
		return
	}

	if req.Prompt == "" {
		c.JSON(400, gin.H{"error": gin.H{
			"message": "prompt is required",
			"type":    "invalid_request_error",
			"param":   "prompt",
			"code":    "missing_prompt",
		}})
		return
	}

	if req.N == 0 {
		req.N = 1
	}

	// Build a chat request with image generation enabled
	model := req.Model
	if model == "" {
		model = "gpt-5.4-nano"
	}

	chatReq := officialtypes.APIRequest{
		Model: model,
		Messages: []officialtypes.ApiMessage{
			{Role: "user", Content: req.Prompt},
		},
		Stream: false,
	}

	proxyUrl := h.proxy.GetProxyIP()
	client := bogdanfinn.NewStdClient()
	token, err := duckgo.InitXVQD(client, proxyUrl)
	if err != nil {
		c.JSON(500, gin.H{"error": gin.H{
			"message": "Failed to initialize VQD token",
			"type":    "internal_server_error",
			"code":    err.Error(),
		}})
		return
	}

	translatedRequest := duckgoConvert.ConvertAPIRequestWithOptions(chatReq, req.ReasoningEffort, false)
	translatedRequest.Metadata.ToolChoice.GenerateImage = true

	response, err := duckgo.POSTconversation(client, translatedRequest, token, proxyUrl)
	if err != nil {
		c.JSON(500, gin.H{"error": gin.H{
			"message": "Failed to generate image",
			"type":    "internal_server_error",
			"code":    err.Error(),
		}})
		return
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		c.JSON(response.StatusCode, gin.H{"error": gin.H{
			"message": duckgo.ReadResponseError(response).Error(),
			"type":    "api_error",
			"code":    "upstream_error",
		}})
		return
	}

	result := duckgo.ReadImageResponse(response)

	if len(result.Images) == 0 {
		c.JSON(500, gin.H{"error": gin.H{
			"message": "No images were generated",
			"type":    "internal_server_error",
			"code":    "no_images",
		}})
		return
	}

	// Build OpenAI-compatible response
	imageData := make([]officialtypes.ImageData, 0, len(result.Images))
	for _, img := range result.Images {
		b64 := img.Result
		if b64 == "" && img.Data != nil {
			b64 = img.Data.B64Image
		}
		if b64 == "" {
			continue
		}
		imageData = append(imageData, officialtypes.ImageData{
			B64JSON:       b64,
			RevisedPrompt: result.Text,
		})
	}

	c.JSON(200, officialtypes.ImageGenerationResponse{
		Created: time.Now().Unix(),
		Data:    imageData,
	})
}

func (h *Handler) imageEdits(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		// Try JSON body for base64 input
		var req officialtypes.ImageEditRequest
		if jsonErr := c.ShouldBindJSON(&req); jsonErr != nil {
			c.JSON(400, gin.H{"error": gin.H{
				"message": "Request must be multipart form or proper JSON",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    err.Error(),
			}})
			return
		}
		h.handleImageEditJSON(c, req)
		return
	}

	// Multipart form handling
	prompt := c.Request.FormValue("prompt")
	if prompt == "" {
		c.JSON(400, gin.H{"error": gin.H{
			"message": "prompt is required",
			"type":    "invalid_request_error",
			"param":   "prompt",
			"code":    "missing_prompt",
		}})
		return
	}

	model := c.Request.FormValue("model")
	if model == "" {
		model = "gpt-5.4-nano"
	}

	imageHeaders := c.Request.MultipartForm.File["image"]
	if len(imageHeaders) == 0 {
		c.JSON(400, gin.H{"error": gin.H{
			"message": "image file is required",
			"type":    "invalid_request_error",
			"param":   "image",
			"code":    "missing_image",
		}})
		return
	}
	if len(imageHeaders) > maxImageEditReferences {
		imageHeaders = imageHeaders[:maxImageEditReferences]
	}

	imageDataURLs := make([]string, 0, len(imageHeaders))
	for _, imageHeader := range imageHeaders {
		file, err := imageHeader.Open()
		if err != nil {
			c.JSON(500, gin.H{"error": gin.H{
				"message": "Failed to open image file",
				"type":    "internal_server_error",
				"code":    err.Error(),
			}})
			return
		}

		imageBytes, readErr := io.ReadAll(file)
		file.Close()
		if readErr != nil {
			c.JSON(500, gin.H{"error": gin.H{
				"message": "Failed to read image file",
				"type":    "internal_server_error",
				"code":    readErr.Error(),
			}})
			return
		}

		mimeType := http.DetectContentType(imageBytes)
		imageDataURLs = append(imageDataURLs, "data:"+mimeType+";base64,"+base64.StdEncoding.EncodeToString(imageBytes))
	}

	h.doImageEdit(c, prompt, model, imageDataURLs, "")
}

func (h *Handler) handleImageEditJSON(c *gin.Context, req officialtypes.ImageEditRequest) {
	if req.Prompt == "" {
		c.JSON(400, gin.H{"error": gin.H{
			"message": "prompt is required",
			"type":    "invalid_request_error",
			"param":   "prompt",
			"code":    "missing_prompt",
		}})
		return
	}

	if req.Image == "" {
		c.JSON(400, gin.H{"error": gin.H{
			"message": "image is required",
			"type":    "invalid_request_error",
			"param":   "image",
			"code":    "missing_image",
		}})
		return
	}

	model := req.Model
	if model == "" {
		model = "gpt-5.4-nano"
	}

	h.doImageEdit(c, req.Prompt, model, []string{req.Image}, req.ReasoningEffort)
}

func normalizeImageDataURL(image string) string {
	image = strings.TrimSpace(image)
	if image == "" || strings.HasPrefix(image, "data:") {
		return image
	}

	mimeType := "image/webp"
	if imageBytes, err := base64.StdEncoding.DecodeString(image); err == nil {
		detectedType := http.DetectContentType(imageBytes)
		if strings.HasPrefix(detectedType, "image/") {
			mimeType = detectedType
		}
	}
	return "data:" + mimeType + ";base64," + image
}

func buildImageEditRequest(prompt string, model string, imageDataURLs []string) officialtypes.APIRequest {
	content := make([]interface{}, 0, 1+len(imageDataURLs))
	content = append(content, map[string]interface{}{
		"type": "text",
		"text": prompt,
	})

	for _, image := range imageDataURLs {
		image = normalizeImageDataURL(image)
		if image == "" {
			continue
		}
		content = append(content, map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]interface{}{
				"url": image,
			},
		})
	}

	return officialtypes.APIRequest{
		Model: model,
		Messages: []officialtypes.ApiMessage{
			{Role: "user", Content: content},
		},
		Stream: false,
	}
}

func (h *Handler) doImageEdit(c *gin.Context, prompt string, model string, imageDataURLs []string, reasoningEffort string) {
	chatReq := buildImageEditRequest(prompt, model, imageDataURLs)

	proxyUrl := h.proxy.GetProxyIP()
	client := bogdanfinn.NewStdClient()
	token, err := duckgo.InitXVQD(client, proxyUrl)
	if err != nil {
		c.JSON(500, gin.H{"error": gin.H{
			"message": "Failed to initialize VQD token",
			"type":    "internal_server_error",
			"code":    err.Error(),
		}})
		return
	}

	translatedRequest := duckgoConvert.ConvertAPIRequestWithOptions(chatReq, reasoningEffort, false)
	translatedRequest.Metadata.ToolChoice.GenerateImage = true

	response, err := duckgo.POSTconversation(client, translatedRequest, token, proxyUrl)
	if err != nil {
		c.JSON(500, gin.H{"error": gin.H{
			"message": "Failed to edit image",
			"type":    "internal_server_error",
			"code":    err.Error(),
		}})
		return
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		c.JSON(response.StatusCode, gin.H{"error": gin.H{
			"message": duckgo.ReadResponseError(response).Error(),
			"type":    "api_error",
			"code":    "upstream_error",
		}})
		return
	}

	result := duckgo.ReadImageResponse(response)

	if len(result.Images) == 0 {
		c.JSON(500, gin.H{"error": gin.H{
			"message": "No images were generated",
			"type":    "internal_server_error",
			"code":    "no_images",
		}})
		return
	}

	imageData := make([]officialtypes.ImageData, 0, len(result.Images))
	for _, img := range result.Images {
		b64 := img.Result
		if b64 == "" && img.Data != nil {
			b64 = img.Data.B64Image
		}
		if b64 == "" {
			continue
		}
		imageData = append(imageData, officialtypes.ImageData{
			B64JSON:       b64,
			RevisedPrompt: result.Text,
		})
	}

	c.JSON(200, officialtypes.ImageGenerationResponse{
		Created: time.Now().Unix(),
		Data:    imageData,
	})
}

func (h *Handler) engines(c *gin.Context) {
	type ResData struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int    `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	type JSONData struct {
		Object string    `json:"object"`
		Data   []ResData `json:"data"`
	}

	modelS := JSONData{
		Object: "list",
	}
	var resModelList []ResData

	// Supported models
	modelIDs := []string{
		"gpt-5.4-mini",
		"gpt-5.4-nano",
		"tinfoil/gpt-oss-120b",
		"claude-haiku-4-5",
		"mistral-small",
	}

	for _, modelID := range modelIDs {
		resModelList = append(resModelList, ResData{
			ID:      modelID,
			Object:  "model",
			Created: 1685474247,
			OwnedBy: "duckduckgo",
		})
	}

	modelS.Data = resModelList
	c.JSON(200, modelS)
}

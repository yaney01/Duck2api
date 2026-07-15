package duckgo

import (
	"aurora/httpclient"
	duckgotypes "aurora/typings/duckgo"
	officialtypes "aurora/typings/official"
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	duckDuckGoBaseURL      = "https://duckduckgo.com"
	duckDuckGoStatusURL    = duckDuckGoBaseURL + "/duckchat/v1/status"
	duckDuckGoChatURL      = duckDuckGoBaseURL + "/duckchat/v1/chat"
	duckDuckGoAuthTokenURL = duckDuckGoBaseURL + "/duckchat/v1/auth/token"
	duckDuckGoChatEntryURL = duckDuckGoBaseURL + "/?q=DuckDuckGo+AI+Chat&ia=chat&duckai=1"
)

var (
	Token     = &XqdgToken{}
	FEVersion = &XqdgToken{}
	UA        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
)

type XqdgToken struct {
	Token    string     `json:"token"`
	M        sync.Mutex `json:"-"`
	ExpireAt time.Time  `json:"expire"`
}

func InitXVQD(client httpclient.AuroraHttpClient, proxyUrl string) (string, error) {
	// VQD tokens are bound to the current anonymous browser session. Every handler
	// creates a fresh HTTP client and cookie jar, so reusing a process-global token
	// makes the token/cookie pair inconsistent and DuckDuckGo rejects it with 418.
	warmDuckDuckGoSession(client, proxyUrl)

	status, err := postStatus(client, proxyUrl)
	if err != nil {
		return "", err
	}
	defer status.Body.Close()
	if status.StatusCode != http.StatusOK {
		return "", ReadResponseError(status)
	}

	vqdHash := status.Header.Get("x-vqd-hash-1")
	if vqdHash == "" {
		return "", errors.New("no x-vqd-hash-1 token")
	}
	token, err := GenerateVQDHash(vqdHash)
	if err != nil {
		return "", err
	}
	Token.M.Lock()
	Token.Token = token
	Token.M.Unlock()
	return token, nil
}

func warmDuckDuckGoSession(client httpclient.AuroraHttpClient, proxyUrl string) {
	if proxyUrl != "" {
		_ = client.SetProxy(proxyUrl)
	}

	// These requests mirror the current web client bootstrap and, importantly,
	// populate the client's cookie jar before the status/challenge request.
	if response, err := client.Request(httpclient.GET, duckDuckGoChatEntryURL, createNavigationHeader(), nil, nil); err == nil {
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr == nil {
			if version, versionErr := extractFEVersion(body); versionErr == nil {
				cacheFEVersion(version)
			}
		}
	}

	header := createHeader()
	header.Set("accept", "*/*")
	if response, err := client.Request(httpclient.GET, duckDuckGoAuthTokenURL, header, nil, nil); err == nil {
		_, _ = io.Copy(io.Discard, response.Body)
		response.Body.Close()
	}
}

func postStatus(client httpclient.AuroraHttpClient, proxyUrl string) (*http.Response, error) {
	if proxyUrl != "" {
		client.SetProxy(proxyUrl)
	}
	header := createHeader()
	header.Set("accept", "*/*")
	header.Set("x-vqd-accept", "1")
	response, err := client.Request(httpclient.GET, duckDuckGoStatusURL, header, nil, nil)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func POSTconversation(client httpclient.AuroraHttpClient, request duckgotypes.ApiRequest, token string, proxyUrl string) (*http.Response, error) {
	if proxyUrl != "" {
		client.SetProxy(proxyUrl)
	}

	maxRetries := 3
	var response *http.Response
	var err error

	for i := 0; i <= maxRetries; i++ {
		response, err = postConversationOnce(client, request, token)
		if err != nil {
			return nil, err
		}

		if response.StatusCode != http.StatusTeapot && response.StatusCode != http.StatusTooManyRequests {
			return response, nil
		}

		response.Body.Close()
		ResetXVQD()
		token, err = InitXVQD(client, proxyUrl)
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}

func Handle_request_error(c *gin.Context, response *http.Response) bool {
	if response.StatusCode != 200 {
		// Try read response body as JSON
		var error_response map[string]interface{}
		err := json.NewDecoder(response.Body).Decode(&error_response)
		if err != nil {
			// Read response body
			body, _ := io.ReadAll(response.Body)
			c.JSON(response.StatusCode, gin.H{"error": gin.H{
				"message": "Unknown error",
				"type":    "internal_server_error",
				"param":   nil,
				"code":    "500",
				"details": string(body),
			}})
			return true
		}
		c.JSON(response.StatusCode, gin.H{"error": gin.H{
			"message": error_response["detail"],
			"type":    response.Status,
			"param":   nil,
			"code":    "error",
		}})
		return true
	}
	return false
}

func createHeader() httpclient.AuroraHeaders {
	header := make(httpclient.AuroraHeaders)
	header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	header.Set("cache-control", "no-cache")
	header.Set("content-type", "application/json")
	header.Set("origin", duckDuckGoBaseURL)
	header.Set("pragma", "no-cache")
	header.Set("referer", duckDuckGoBaseURL+"/")
	header.Set("sec-ch-ua", `"Google Chrome";v="149", "Chromium";v="149", "Not)A;Brand";v="24"`)
	header.Set("sec-ch-ua-mobile", "?0")
	header.Set("sec-ch-ua-platform", `"Windows"`)
	header.Set("sec-fetch-dest", "empty")
	header.Set("sec-fetch-mode", "cors")
	header.Set("sec-fetch-site", "same-origin")
	header.Set("user-agent", UA)
	return header
}

func createNavigationHeader() httpclient.AuroraHeaders {
	header := createHeader()
	header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	header.Set("sec-fetch-dest", "document")
	header.Set("sec-fetch-mode", "navigate")
	header.Set("sec-fetch-site", "none")
	header.Set("upgrade-insecure-requests", "1")
	return header
}

func postConversationOnce(client httpclient.AuroraHttpClient, request duckgotypes.ApiRequest, token string) (*http.Response, error) {
	bodyJSON, err := json.Marshal(request)
	if err != nil {
		return &http.Response{}, err
	}
	header := createHeader()
	header.Set("accept", "text/event-stream")
	header.Set("priority", "u=1, i")
	header.Set("x-ddg-journey-id", RandomHex(16))
	header.Set("x-fe-signals", CreateFESignals())
	if feVersion, err := InitFEVersion(client, ""); err == nil && feVersion != "" {
		header.Set("x-fe-version", feVersion)
	}
	header.Set("x-vqd-hash-1", token)
	return client.Request(httpclient.POST, duckDuckGoChatURL, header, nil, bytes.NewBuffer(bodyJSON))
}

func InitFEVersion(client httpclient.AuroraHttpClient, proxyUrl string) (string, error) {
	FEVersion.M.Lock()
	defer FEVersion.M.Unlock()
	if FEVersion.Token != "" && FEVersion.ExpireAt.After(time.Now()) {
		return FEVersion.Token, nil
	}

	if proxyUrl != "" {
		_ = client.SetProxy(proxyUrl)
	}
	response, err := client.Request(httpclient.GET, duckDuckGoChatEntryURL, createNavigationHeader(), nil, nil)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	version, err := extractFEVersion(body)
	if err != nil {
		return "", err
	}

	FEVersion.Token = version
	FEVersion.ExpireAt = time.Now().Add(30 * time.Minute)
	return FEVersion.Token, nil
}

func cacheFEVersion(version string) {
	if version == "" {
		return
	}
	FEVersion.M.Lock()
	defer FEVersion.M.Unlock()
	FEVersion.Token = version
	FEVersion.ExpireAt = time.Now().Add(30 * time.Minute)
}

func extractFEVersion(body []byte) (string, error) {
	// Current DuckDuckGo search/chat page markers.
	beVersion := regexp.MustCompile(`__DDG_BE_VERSION__=["']([^"']+)["']`).FindSubmatch(body)
	chatHash := regexp.MustCompile(`__DDG_FE_CHAT_HASH__=["']([^"']+)["']`).FindSubmatch(body)
	if len(beVersion) >= 2 && len(chatHash) >= 2 {
		return fmt.Sprintf("%s-%s", beVersion[1], chatHash[1]), nil
	}

	// Some page variants expose the complete version as a single token.
	if version := regexp.MustCompile(`serp_\d{8}_\d{6}_[A-Z]{2}-[0-9a-f]{20,40}`).Find(body); len(version) > 0 {
		return string(version), nil
	}

	// Backward compatibility with the older duck.ai page markup.
	versionTag := regexp.MustCompile(`data-version-tag="([^"]+)"`).FindSubmatch(body)
	versionSHA := regexp.MustCompile(`data-version-sha="([^"]+)"`).FindSubmatch(body)
	if len(versionTag) >= 2 && len(versionSHA) >= 2 {
		return fmt.Sprintf("%s-%s", versionTag[1], versionSHA[1]), nil
	}

	return "", errors.New("DuckDuckGo frontend version metadata not found")
}

func CreateFESignals() string {
	// Keep this event sequence aligned with the current DuckDuckGo web client.
	// The old onboarding/action/startNewChat_free payload is rejected as an
	// invalid anonymous-browser session and surfaces as HTTP 418.
	now := time.Now().UnixMilli()
	delta := int64(80) + randInt63n(101)
	events := []map[string]interface{}{
		{"name": "onboarding_impression_1", "delta": delta},
	}
	delta += 120 + randInt63n(141)
	events = append(events, map[string]interface{}{"name": "onboarding_impression_2", "delta": delta})
	delta += 200 + randInt63n(301)
	events = append(events, map[string]interface{}{"name": "startNewChat", "delta": delta})

	keyEvents := 6 + int(randInt63n(13))
	for i := 0; i < keyEvents; i++ {
		delta += 40 + randInt63n(141)
		events = append(events, map[string]interface{}{"name": "user_input", "delta": delta})
	}
	delta += 120 + randInt63n(231)
	events = append(events, map[string]interface{}{"name": "user_submit", "delta": delta})

	end := delta + 20 + randInt63n(71)
	if end < 3000 {
		end = 3000
	}
	payload := map[string]interface{}{
		"start":  now - 3000,
		"events": events,
		"end":    end,
	}
	body, _ := json.Marshal(payload)
	return base64.StdEncoding.EncodeToString(body)
}

// randInt63n returns a uniform random non-negative int64 in [0, n).
func randInt63n(n int64) int64 {
	if n <= 0 {
		return 0
	}
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return 0
	}
	var v int64
	for _, b := range buf {
		v = v<<8 | int64(b)
	}
	if v < 0 {
		v = -v
	}
	return v % n
}

func RandomHex(byteLength int) string {
	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}

func ResetXVQD() {
	if Token == nil {
		return
	}
	Token.M.Lock()
	defer Token.M.Unlock()
	Token.Token = ""
}

func ReadResponseError(response *http.Response) error {
	var errorResponse map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&errorResponse); err == nil {
		if detail, ok := errorResponse["detail"]; ok {
			return fmt.Errorf("%s: %v", response.Status, detail)
		}
		return fmt.Errorf("%s: %v", response.Status, errorResponse)
	}

	body, _ := io.ReadAll(response.Body)
	if len(body) == 0 {
		return fmt.Errorf("%s", response.Status)
	}
	return fmt.Errorf("%s: %s", response.Status, string(body))
}

func ReadResponseText(response *http.Response) string {
	reader := bufio.NewReader(response.Body)
	var previousText strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return ""
		}
		if len(line) < 6 {
			continue
		}
		line = line[6:]
		if strings.HasPrefix(line, "[DONE]") {
			continue
		}

		var originalResponse duckgotypes.ApiResponse
		err = json.Unmarshal([]byte(line), &originalResponse)
		if err != nil || originalResponse.Action != "success" {
			continue
		}
		previousText.WriteString(originalResponse.Message)
	}
	return previousText.String()
}

func Handler(c *gin.Context, response *http.Response, oldRequest duckgotypes.ApiRequest, stream bool) string {
	reader := bufio.NewReader(response.Body)
	if stream {
		// Response content type is text/event-stream
		c.Header("Content-Type", "text/event-stream")
	} else {
		// Response content type is application/json
		c.Header("Content-Type", "application/json")
	}

	var previousText strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return ""
		}
		if len(line) < 6 {
			continue
		}
		line = line[6:]
		if !strings.HasPrefix(line, "[DONE]") {
			var originalResponse duckgotypes.ApiResponse
			err = json.Unmarshal([]byte(line), &originalResponse)
			if err != nil {
				continue
			}
			if originalResponse.Action != "success" {
				c.JSON(500, gin.H{"error": "Error"})
				return ""
			}
			responseString := ""
			if originalResponse.Message != "" {
				previousText.WriteString(originalResponse.Message)
				translatedResponse := officialtypes.NewChatCompletionChunkWithModel(originalResponse.Message, originalResponse.Model)
				responseString = "data: " + translatedResponse.String() + "\n\n"
			}

			if responseString == "" {
				continue
			}

			if stream {
				_, err = c.Writer.WriteString(responseString)
				if err != nil {
					return ""
				}
				c.Writer.Flush()
			}
		} else {
			if stream {
				final_line := officialtypes.StopChunkWithModel("stop", oldRequest.Model)
				c.Writer.WriteString("data: " + final_line.String() + "\n\n")
			}
		}
	}
	return previousText.String()
}

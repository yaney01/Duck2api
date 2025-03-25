package duckgo

import (
	"aurora/httpclient"
	duckgotypes "aurora/typings/duckgo"
	officialtypes "aurora/typings/official"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	Token *XqdgToken
	UA    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

type XqdgToken struct {
	Token    string     `json:"token"`
	M        sync.Mutex `json:"-"`
	ExpireAt time.Time  `json:"expire"`
}

func InitXVQD(client httpclient.AuroraHttpClient, proxyUrl string) (string, error) {
	if Token == nil {
		Token = &XqdgToken{
			Token: "",
			M:     sync.Mutex{},
		}
	}
	Token.M.Lock()
	defer Token.M.Unlock()
	if Token.Token == "" || Token.ExpireAt.Before(time.Now()) {
		status, err := postStatus(client, proxyUrl)
		if err != nil {
			return "", err
		}
		defer status.Body.Close()
		token := status.Header.Get("x-vqd-4")
		if token == "" {
			return "", errors.New("no x-vqd-4 token")
		}
		Token.Token = token
		Token.ExpireAt = time.Now().Add(time.Minute * 10)
	}

	return Token.Token, nil
}

func postStatus(client httpclient.AuroraHttpClient, proxyUrl string) (*http.Response, error) {
	if proxyUrl != "" {
		client.SetProxy(proxyUrl)
	}
	header := createHeader()
	header.Set("accept", "*/*")
	header.Set("x-vqd-accept", "1")
	response, err := client.Request(httpclient.GET, "https://duckduckgo.com/duckchat/v1/status", header, nil, nil)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func POSTconversation(client httpclient.AuroraHttpClient, request duckgotypes.ApiRequest, token string, proxyUrl string) (*http.Response, error) {
	if proxyUrl != "" {
		client.SetProxy(proxyUrl)
	}
	body_json, err := json.Marshal(request)
	if err != nil {
		return &http.Response{}, err
	}
	header := createHeader()
	header.Set("accept", "text/event-stream")
	header.Set("x-vqd-4", token)
	response, err := client.Request(httpclient.POST, "https://duckduckgo.com/duckchat/v1/chat", header, nil, bytes.NewBuffer(body_json))
	if err != nil {
		return nil, err
	}
	return response, nil
}

func Handle_request_error(c *gin.Context, response *http.Response) bool {
	if response.StatusCode != 200 {
		// 特殊处理418错误
		if response.StatusCode == 418 {
			// 读取响应体
			body, _ := io.ReadAll(response.Body)
			c.JSON(response.StatusCode, gin.H{"error": gin.H{
				"message": "DuckDuckGo API拒绝了请求，可能是由于请求过于频繁或IP被限制",
				"type":    "418 I'm a teapot",
				"param":   nil,
				"code":    "error",
				"details": string(body),
			}})
			return true
		}
		
		// 尝试读取响应体为JSON
		var error_response map[string]interface{}
		// 先读取响应体
		body, _ := io.ReadAll(response.Body)
		// 尝试解析JSON
		err := json.Unmarshal(body, &error_response)
		if err != nil {
			// 非JSON格式响应
			c.JSON(response.StatusCode, gin.H{"error": gin.H{
				"message": "API请求失败",
				"type":    response.Status,
				"param":   nil,
				"code":    "error",
				"details": string(body),
			}})
			return true
		}
		
		// 处理JSON格式响应
		message := "未知错误"
		if detail, ok := error_response["detail"]; ok && detail != nil {
			message = detail.(string)
		}
		c.JSON(response.StatusCode, gin.H{"error": gin.H{
			"message": message,
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
	header.Set("accept-language", "zh-CN,zh;q=0.9")
	header.Set("content-type", "application/json")
	header.Set("origin", "https://duckduckgo.com")
	header.Set("referer", "https://duckduckgo.com/")
	header.Set("sec-ch-ua", `"Chromium";v="120", "Google Chrome";v="120", "Not-A.Brand";v="99"`)
	header.Set("sec-ch-ua-mobile", "?0")
	header.Set("sec-ch-ua-platform", `"Windows"`)
	header.Set("sec-fetch-dest", "empty")
	header.Set("sec-fetch-mode", "cors")
	header.Set("sec-fetch-site", "same-origin")
	header.Set("user-agent", UA)
	return header
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

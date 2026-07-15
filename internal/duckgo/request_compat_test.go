package duckgo

import (
	"aurora/httpclient"
	duckgotypes "aurora/typings/duckgo"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type recordedHTTPRequest struct {
	method  httpclient.HttpMethod
	url     string
	headers httpclient.AuroraHeaders
	body    []byte
}

type recordingHTTPClient struct {
	requests []recordedHTTPRequest
	proxy    string
}

func (c *recordingHTTPClient) Request(method httpclient.HttpMethod, url string, headers httpclient.AuroraHeaders, _ []*http.Cookie, body io.Reader) (*http.Response, error) {
	copiedHeaders := make(httpclient.AuroraHeaders, len(headers))
	for key, value := range headers {
		copiedHeaders[key] = value
	}
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = io.ReadAll(body)
	}
	c.requests = append(c.requests, recordedHTTPRequest{
		method:  method,
		url:     url,
		headers: copiedHeaders,
		body:    bodyBytes,
	})

	response := &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}
	switch url {
	case duckDuckGoChatEntryURL:
		response.Body = io.NopCloser(strings.NewReader(`<html><script>window.__DDG_BE_VERSION__="serp_20260715_120000_ET";window.__DDG_FE_CHAT_HASH__="0123456789abcdef0123456789abcdef01234567";</script></html>`))
	case duckDuckGoStatusURL:
		challenge := base64.StdEncoding.EncodeToString([]byte(`({client_hashes:["test"],meta:{}})`))
		response.Header.Set("x-vqd-hash-1", challenge)
	}
	return response, nil
}

func (c *recordingHTTPClient) SetProxy(url string) error {
	c.proxy = url
	return nil
}

func resetRequestCompatibilityState() {
	Token.M.Lock()
	Token.Token = ""
	Token.M.Unlock()
	FEVersion.M.Lock()
	FEVersion.Token = ""
	FEVersion.ExpireAt = time.Time{}
	FEVersion.M.Unlock()
}

func TestCreateHeaderUsesDuckDuckGoSameOrigin(t *testing.T) {
	header := createHeader()
	if got := header["origin"]; got != duckDuckGoBaseURL {
		t.Fatalf("origin = %q, want %q", got, duckDuckGoBaseURL)
	}
	if got := header["referer"]; got != duckDuckGoBaseURL+"/" {
		t.Fatalf("referer = %q, want %q", got, duckDuckGoBaseURL+"/")
	}
	if got := header["sec-fetch-site"]; got != "same-origin" {
		t.Fatalf("sec-fetch-site = %q, want same-origin", got)
	}
}

func TestPostConversationUsesDuckDuckGoEndpoint(t *testing.T) {
	resetRequestCompatibilityState()
	cacheFEVersion("serp_20260715_120000_ET-0123456789abcdef0123456789abcdef01234567")
	client := &recordingHTTPClient{}

	_, err := postConversationOnce(client, duckgotypes.ApiRequest{}, "test-vqd")
	if err != nil {
		t.Fatalf("postConversationOnce failed: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(client.requests))
	}
	request := client.requests[0]
	if request.method != httpclient.POST {
		t.Fatalf("method = %q, want POST", request.method)
	}
	if request.url != duckDuckGoChatURL {
		t.Fatalf("url = %q, want %q", request.url, duckDuckGoChatURL)
	}
	if request.headers["x-vqd-hash-1"] != "test-vqd" {
		t.Fatalf("x-vqd-hash-1 was not forwarded")
	}
	if request.headers["x-fe-version"] == "" {
		t.Fatalf("x-fe-version was not forwarded")
	}
	if !json.Valid(request.body) {
		t.Fatalf("request body is not valid JSON: %s", request.body)
	}
}

func TestInitXVQDAcquiresFreshSessionToken(t *testing.T) {
	resetRequestCompatibilityState()
	client := &recordingHTTPClient{}

	first, err := InitXVQD(client, "")
	if err != nil {
		t.Fatalf("first InitXVQD failed: %v", err)
	}
	second, err := InitXVQD(client, "")
	if err != nil {
		t.Fatalf("second InitXVQD failed: %v", err)
	}
	if first == "" || second == "" {
		t.Fatalf("expected non-empty VQD tokens")
	}

	var entryRequests, authRequests, statusRequests int
	for _, request := range client.requests {
		switch request.url {
		case duckDuckGoChatEntryURL:
			entryRequests++
		case duckDuckGoAuthTokenURL:
			authRequests++
		case duckDuckGoStatusURL:
			statusRequests++
		}
	}
	if entryRequests != 2 || authRequests != 2 || statusRequests != 2 {
		t.Fatalf("bootstrap requests = entry:%d auth:%d status:%d, want 2 each", entryRequests, authRequests, statusRequests)
	}
}

func TestExtractFEVersion(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "current page markers",
			body: `__DDG_BE_VERSION__="serp_20260715_120000_ET";__DDG_FE_CHAT_HASH__="0123456789abcdef0123456789abcdef01234567"`,
			want: "serp_20260715_120000_ET-0123456789abcdef0123456789abcdef01234567",
		},
		{
			name: "complete token",
			body: `data-version="serp_20260715_120000_ET-0123456789abcdef0123456789abcdef01234567"`,
			want: "serp_20260715_120000_ET-0123456789abcdef0123456789abcdef01234567",
		},
		{
			name: "legacy markers",
			body: `data-version-tag="duckai_20260715" data-version-sha="0123456789abcdef"`,
			want: "duckai_20260715-0123456789abcdef",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := extractFEVersion([]byte(test.body))
			if err != nil {
				t.Fatalf("extractFEVersion failed: %v", err)
			}
			if got != test.want {
				t.Fatalf("version = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCreateFESignalsUsesCurrentInputSequence(t *testing.T) {
	decoded, err := base64.StdEncoding.DecodeString(CreateFESignals())
	if err != nil {
		t.Fatalf("decode signals: %v", err)
	}
	var payload struct {
		Start  int64 `json:"start"`
		Events []struct {
			Name  string `json:"name"`
			Delta int64  `json:"delta"`
		} `json:"events"`
		End int64 `json:"end"`
	}
	if err := json.NewDecoder(bytes.NewReader(decoded)).Decode(&payload); err != nil {
		t.Fatalf("decode signal payload: %v", err)
	}
	if len(payload.Events) < 10 || len(payload.Events) > 22 {
		t.Fatalf("event count = %d, want 10..22", len(payload.Events))
	}
	wantPrefix := []string{"onboarding_impression_1", "onboarding_impression_2", "startNewChat"}
	for index, want := range wantPrefix {
		if got := payload.Events[index].Name; got != want {
			t.Fatalf("event %d = %q, want %q", index, got, want)
		}
	}
	for index := 3; index < len(payload.Events)-1; index++ {
		if got := payload.Events[index].Name; got != "user_input" {
			t.Fatalf("event %d = %q, want user_input", index, got)
		}
	}
	if got := payload.Events[len(payload.Events)-1].Name; got != "user_submit" {
		t.Fatalf("last event = %q, want user_submit", got)
	}
	for index := 1; index < len(payload.Events); index++ {
		if payload.Events[index].Delta <= payload.Events[index-1].Delta {
			t.Fatalf("event deltas are not strictly increasing at index %d", index)
		}
	}
	if payload.End < 3000 || payload.End <= payload.Events[len(payload.Events)-1].Delta {
		t.Fatalf("end = %d is inconsistent with final delta %d", payload.End, payload.Events[len(payload.Events)-1].Delta)
	}
}

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWebHandler(t *testing.T) {
	assets := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><title>Calculator</title>")},
		"assets/app.js": {Data: []byte("console.log('calculator');")},
	}
	handler := NewWebHandler(assets)
	cases := []struct {
		name, method, path, body string
		status                   int
		contentType, content     string
	}{
		{"home", "GET", "/", "", 200, "text/html", "Calculator"},
		{"head", "HEAD", "/", "", 200, "text/html", ""},
		{"asset", "GET", "/assets/app.js", "", 200, "javascript", "console.log"},
		{"asset query", "GET", "/assets/app.js?v=1", "", 200, "javascript", "console.log"},
		{"missing asset", "GET", "/assets/missing.js", "", 404, "text/plain", "404"},
		{"directory listing", "GET", "/assets/", "", 404, "text/plain", "404"},
		{"unknown page", "GET", "/unknown", "", 404, "text/plain", "404"},
		{"static method", "POST", "/", "", 405, "text/plain", "Method not allowed"},
		{"calculation", "POST", "/api/calculations", `{"operation":"add","a":2,"b":3}`, 200, "application/json", `{"result":5}`},
		{"domain error", "POST", "/api/calculations", `{"operation":"divide","a":2,"b":0}`, 400, "application/json", "DIVISION_BY_ZERO"},
		{"api method", "GET", "/api/calculations", "", 405, "application/json", "METHOD_NOT_ALLOWED"},
		{"unknown api", "GET", "/api/unknown", "", 404, "application/json", "NOT_FOUND"},
		{"api root", "GET", "/api", "", 404, "application/json", "NOT_FOUND"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status || !strings.Contains(response.Header().Get("Content-Type"), tc.contentType) {
				t.Fatalf("status=%d content-type=%q body=%s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
			}
			if !strings.Contains(response.Body.String(), tc.content) {
				t.Fatalf("body=%q, want substring %q", response.Body.String(), tc.content)
			}
			if tc.method == http.MethodHead && response.Body.Len() != 0 {
				t.Fatal("HEAD returned a body")
			}
			if tc.name == "static method" && response.Header().Get("Allow") != "GET, HEAD" {
				t.Fatalf("Allow=%q", response.Header().Get("Allow"))
			}
		})
	}
}

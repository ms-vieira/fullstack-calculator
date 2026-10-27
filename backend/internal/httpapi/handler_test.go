package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ms-vieira/fullstack-calculator/backend/internal/calculator"
)

type response struct {
	Result float64           `json:"result"`
	Error  *calculator.Error `json:"error"`
}

func call(handler http.Handler, method, path, contentType, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestHTTPContract(t *testing.T) {
	tests := []struct {
		name, body, code, field string
		want                    float64
	}{
		{name: "add", body: `{"operation":"add","a":10,"b":5}`, want: 15},
		{name: "subtract", body: `{"operation":"subtract","a":5,"b":10}`, want: -5},
		{name: "multiply", body: `{"operation":"multiply","a":-2.5,"b":4}`, want: -10},
		{name: "divide", body: `{"operation":"divide","a":10,"b":2}`, want: 5},
		{name: "power", body: `{"operation":"power","a":2,"b":10}`, want: 1024},
		{name: "unary square root", body: `{"operation":"sqrt","a":81}`, want: 9},
		{name: "square root zero", body: `{"operation":"sqrt","a":-0}`, want: 0},
		{name: "percentage", body: `{"operation":"percent","a":10,"b":200}`, want: 20},
		{name: "percentage intermediate overflow", body: `{"operation":"percent","a":1e308,"b":2}`, want: 2e306},
		{name: "percentage subnormal", body: `{"operation":"percent","a":5e-324,"b":100}`, want: 5e-324},
		{name: "percentage rounding at underflow boundary", body: `{"operation":"percent","a":1e-150,"b":2.4703282292062328e-172}`, want: 5e-324},
		{name: "percentage rounding at overflow boundary", body: `{"operation":"percent","a":4.172657595118774e272,"b":4.308268996155542e37}`, want: 1.7976931348623157e308},
		{name: "sqrt rejects extra operand", body: `{"operation":"sqrt","a":4,"b":2}`, code: "INVALID_REQUEST", field: "b"},
		{name: "sqrt rejects null extra operand", body: `{"operation":"sqrt","a":4,"b":null}`, code: "INVALID_REQUEST", field: "b"},
		{name: "sqrt missing operand", body: `{"operation":"sqrt"}`, code: "REQUIRED_FIELD", field: "a"},
		{name: "sqrt null operand", body: `{"operation":"sqrt","a":null}`, code: "INVALID_NUMBER", field: "a"},
		{name: "sqrt duplicate operand", body: `{"operation":"sqrt","a":4,"a":9}`, code: "INVALID_REQUEST"},
		{name: "sqrt negative", body: `{"operation":"sqrt","a":-1}`, code: "INVALID_DOMAIN", field: "a"},
		{name: "zero to zero", body: `{"operation":"power","a":0,"b":0}`, code: "INVALID_DOMAIN", field: "b"},
		{name: "zero to negative", body: `{"operation":"power","a":0,"b":-1}`, code: "INVALID_DOMAIN", field: "b"},
		{name: "power non-real", body: `{"operation":"power","a":-4,"b":0.5}`, code: "INVALID_DOMAIN", field: "b"},
		{name: "power overflow", body: `{"operation":"power","a":1e308,"b":2}`, code: "NUMERIC_OVERFLOW"},
		{name: "power underflow", body: `{"operation":"power","a":2,"b":-1075}`, code: "NUMERIC_UNDERFLOW"},
		{name: "percentage overflow", body: `{"operation":"percent","a":1e308,"b":1e308}`, code: "NUMERIC_OVERFLOW"},
		{name: "percentage underflow", body: `{"operation":"percent","a":5e-324,"b":1}`, code: "NUMERIC_UNDERFLOW"},
		{name: "power missing exponent", body: `{"operation":"power","a":2}`, code: "REQUIRED_FIELD", field: "b"},
		{name: "percentage missing base", body: `{"operation":"percent","a":10}`, code: "REQUIRED_FIELD", field: "b"},
		{name: "zero is present", body: `{"operation":"add","a":0,"b":0}`, want: 0},
		{name: "scientific", body: `{"operation":"add","a":1e3,"b":1E-3}`, want: 1000.001},
		{name: "subnormal", body: `{"operation":"divide","a":5e-324,"b":1}`, want: 5e-324},
		{name: "negative zero normalized", body: `{"operation":"multiply","a":-0,"b":5}`, want: 0},
		{name: "outer whitespace", body: " \n{\"operation\":\"add\",\"a\":1,\"b\":2}\t ", want: 3},
		{name: "zero negative exponent", body: `{"operation":"add","a":0e-400,"b":0}`, want: 0},
		{name: "missing operation", body: `{"a":1,"b":2}`, code: "REQUIRED_FIELD", field: "operation"},
		{name: "missing first", body: `{"operation":"add","b":2}`, code: "REQUIRED_FIELD", field: "a"},
		{name: "missing second", body: `{"operation":"add","a":1}`, code: "REQUIRED_FIELD", field: "b"},
		{name: "null first", body: `{"operation":"add","a":null,"b":2}`, code: "INVALID_NUMBER", field: "a"},
		{name: "string first", body: `{"operation":"add","a":"1","b":2}`, code: "INVALID_NUMBER", field: "a"},
		{name: "boolean second", body: `{"operation":"add","a":1,"b":true}`, code: "INVALID_NUMBER", field: "b"},
		{name: "array second", body: `{"operation":"add","a":1,"b":[]}`, code: "INVALID_NUMBER", field: "b"},
		{name: "object second", body: `{"operation":"add","a":1,"b":{}}`, code: "INVALID_NUMBER", field: "b"},
		{name: "unknown operation", body: `{"operation":"log","a":1,"b":2}`, code: "INVALID_OPERATION", field: "operation"},
		{name: "null operation", body: `{"operation":null,"a":1,"b":2}`, code: "INVALID_OPERATION", field: "operation"},
		{name: "number operation", body: `{"operation":1,"a":1,"b":2}`, code: "INVALID_OPERATION", field: "operation"},
		{name: "unknown field", body: `{"operation":"add","a":1,"b":2,"extra":3}`, code: "INVALID_REQUEST"},
		{name: "duplicate field", body: `{"operation":"add","a":1,"a":2,"b":3}`, code: "INVALID_REQUEST"},
		{name: "escaped duplicate", body: `{"operation":"add","a":1,"\u0061":2,"b":3}`, code: "INVALID_REQUEST"},
		{name: "case sensitive keys", body: `{"operation":"add","A":1,"b":2}`, code: "INVALID_REQUEST"},
		{name: "empty", body: ``, code: "INVALID_JSON"},
		{name: "malformed", body: `{"operation":"add","a":`, code: "INVALID_JSON"},
		{name: "trailing comma", body: `{"operation":"add","a":1,"b":2,}`, code: "INVALID_JSON"},
		{name: "array root", body: `[]`, code: "INVALID_REQUEST"},
		{name: "null root", body: `null`, code: "INVALID_REQUEST"},
		{name: "number root", body: `5`, code: "INVALID_REQUEST"},
		{name: "two objects", body: `{"operation":"add","a":1,"b":2}{}`, code: "INVALID_REQUEST"},
		{name: "trailing text", body: `{"operation":"add","a":1,"b":2}junk`, code: "INVALID_REQUEST"},
		{name: "divide zero", body: `{"operation":"divide","a":5,"b":0}`, code: "DIVISION_BY_ZERO", field: "b"},
		{name: "divide negative zero", body: `{"operation":"divide","a":5,"b":-0}`, code: "DIVISION_BY_ZERO", field: "b"},
		{name: "input overflow", body: `{"operation":"add","a":1e309,"b":1}`, code: "NUMBER_OUT_OF_RANGE", field: "a"},
		{name: "input underflow", body: `{"operation":"add","a":1e-400,"b":1}`, code: "NUMERIC_UNDERFLOW", field: "a"},
		{name: "result overflow", body: `{"operation":"multiply","a":1e308,"b":10}`, code: "NUMERIC_OVERFLOW"},
		{name: "result underflow", body: `{"operation":"multiply","a":1e-300,"b":1e-300}`, code: "NUMERIC_UNDERFLOW"},
		{name: "token maximum", body: `{"operation":"add","a":` + strings.Repeat("1", 128) + `,"b":0}`, want: 1.1111111111111111e127},
		{name: "token above maximum", body: `{"operation":"add","a":` + strings.Repeat("1", 129) + `,"b":0}`, code: "INPUT_TOO_LONG", field: "a"},
	}
	handler := NewHandler()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := call(handler, http.MethodPost, "/api/calculations", "application/json; charset=utf-8", test.body)
			if recorder.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatal("response must declare JSON")
			}
			var got response
			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if test.code == "" {
				if recorder.Code != http.StatusOK || got.Error != nil || got.Result != test.want {
					t.Fatalf("got status=%d body=%s; want result=%g", recorder.Code, recorder.Body.String(), test.want)
				}
			} else if recorder.Code != http.StatusBadRequest || got.Error == nil || got.Error.Code != test.code || got.Error.Field != test.field {
				t.Fatalf("got status=%d body=%s; want %s field=%s", recorder.Code, recorder.Body.String(), test.code, test.field)
			}
		})
	}
}

func TestTransportErrors(t *testing.T) {
	tests := []struct {
		name, method, path, mediaType string
		status                        int
		code                          string
	}{
		{"method", "GET", "/api/calculations", "application/json", 405, "METHOD_NOT_ALLOWED"},
		{"preflight not configured", "OPTIONS", "/api/calculations", "", 405, "METHOD_NOT_ALLOWED"},
		{"route", "POST", "/missing", "application/json", 404, "NOT_FOUND"},
		{"content type missing", "POST", "/api/calculations", "", 415, "UNSUPPORTED_MEDIA_TYPE"},
		{"content type text", "POST", "/api/calculations", "text/plain", 415, "UNSUPPORTED_MEDIA_TYPE"},
		{"content type malformed", "POST", "/api/calculations", "application/json; broken", 415, "UNSUPPORTED_MEDIA_TYPE"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := call(NewHandler(), test.method, test.path, test.mediaType, `{}`)
			var got response
			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != test.status || got.Error == nil || got.Error.Code != test.code {
				t.Fatalf("got %d %s", recorder.Code, recorder.Body.String())
			}
			if test.status == 405 && recorder.Header().Get("Allow") != "POST" {
				t.Fatal("method errors must advertise Allow: POST")
			}
		})
	}
}

func TestBodyLimits(t *testing.T) {
	base := `{"operation":"add","a":1,"b":2}`
	for _, length := range []int{int(MaxBodyBytes), int(MaxBodyBytes) + 1} {
		for _, contentLength := range []int64{-1, int64(length)} {
			t.Run(fmt.Sprintf("bytes=%d/contentLength=%d", length, contentLength), func(t *testing.T) {
				body := base + strings.Repeat(" ", length-len(base))
				request := httptest.NewRequest("POST", "/api/calculations", strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				request.ContentLength = contentLength
				recorder := httptest.NewRecorder()
				NewHandler().ServeHTTP(recorder, request)
				if length == int(MaxBodyBytes) {
					if recorder.Code != http.StatusOK {
						t.Fatalf("exact body limit rejected: %s", recorder.Body.String())
					}
				} else {
					var got response
					if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
						t.Fatal(err)
					}
					if recorder.Code != 413 || got.Error == nil || got.Error.Code != "PAYLOAD_TOO_LARGE" {
						t.Fatalf("oversized body accepted: %d %s", recorder.Code, recorder.Body.String())
					}
				}
			})
		}
	}
}

func TestRepeatedRequests(t *testing.T) {
	handler := NewHandler()
	for _, body := range []string{`{"operation":"add","a":10,"b":5}`, `{"operation":"divide","a":1,"b":0}`, `{"operation":"power","a":2,"b":10}`, `{"operation":"sqrt","a":9}`, `{"operation":"percent","a":10,"b":200}`, `{"operation":"sqrt","a":-1}`} {
		first := call(handler, "POST", "/api/calculations", "application/json", body)
		for attempt := 0; attempt < 5; attempt++ {
			next := call(handler, "POST", "/api/calculations", "application/json", body)
			if next.Code != first.Code || next.Body.String() != first.Body.String() {
				t.Fatal("repeated request changed response")
			}
		}
	}
}

func TestConcurrentRequestsAreIsolated(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	start := make(chan struct{})
	failures := make(chan error, 50)
	var workers sync.WaitGroup
	for index := 0; index < 50; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			operations := []struct {
				name  string
				a, b  int
				want  float64
				unary bool
			}{
				{"add", index, 2, float64(index + 2), false},
				{"subtract", index, 2, float64(index - 2), false},
				{"multiply", index, 2, float64(index * 2), false},
				{"divide", index, 2, float64(index) / 2, false},
				{"power", index, 2, float64(index * index), false},
				{"sqrt", index * index, 0, float64(index), true},
				{"percent", index, 200, float64(index * 2), false},
			}
			selected := operations[index%len(operations)]
			operation, b := selected.name, selected.b
			if index%10 == 0 {
				operation, b = "divide", 0
			}
			body := fmt.Sprintf(`{"operation":%q,"a":%d,"b":%d}`, operation, selected.a, b)
			if selected.unary && operation == selected.name {
				body = fmt.Sprintf(`{"operation":%q,"a":%d}`, operation, selected.a)
			}
			res, err := client.Post(server.URL+"/api/calculations", "application/json", strings.NewReader(body))
			if err != nil {
				failures <- err
				return
			}
			defer res.Body.Close()
			var got response
			if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
				failures <- err
				return
			}
			if operation == "divide" && b == 0 {
				if res.StatusCode != 400 || got.Error == nil || got.Error.Code != "DIVISION_BY_ZERO" {
					failures <- fmt.Errorf("request %d: wrong error: %#v", index, got)
				}
			} else if res.StatusCode != 200 || got.Error != nil || got.Result != selected.want {
				failures <- fmt.Errorf("request %d: wrong result: %#v", index, got)
			}
		}(index)
	}
	close(start)
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestBodyReadFailure(t *testing.T) {
	request := httptest.NewRequest("POST", "/api/calculations", failingReader{})
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	NewHandler().ServeHTTP(recorder, request)
	if recorder.Code != 400 || !strings.Contains(recorder.Body.String(), `"INVALID_JSON"`) {
		t.Fatalf("got %d %s", recorder.Code, recorder.Body.String())
	}
}

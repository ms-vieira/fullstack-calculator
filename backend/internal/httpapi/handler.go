package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/ms-vieira/fullstack-calculator/backend/internal/calculator"
)

const MaxBodyBytes int64 = 4 * 1024

type handler struct {
	calculator *calculator.Calculator
}

func NewHandler() http.Handler {
	return &handler{calculator: calculator.New()}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/calculations" {
		writeError(w, http.StatusNotFound, &calculator.Error{Code: "NOT_FOUND", Message: "Route not found."})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, &calculator.Error{Code: "METHOD_NOT_ALLOWED", Message: "Use POST for calculations."})
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, &calculator.Error{Code: "UNSUPPORTED_MEDIA_TYPE", Message: "Content-Type must be application/json."})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, &calculator.Error{Code: "PAYLOAD_TOO_LARGE", Message: "The request body must not exceed 4 KiB."})
		} else {
			writeError(w, http.StatusBadRequest, &calculator.Error{Code: "INVALID_JSON", Message: "Could not read a valid JSON request."})
		}
		return
	}
	request, appErr := h.decodeRequest(body)
	if appErr != nil {
		writeError(w, http.StatusBadRequest, appErr)
		return
	}
	result, err := h.calculator.Calculate(request.operation, request.operands...)
	if err != nil {
		var domainErr *calculator.Error
		if errors.As(err, &domainErr) {
			writeError(w, http.StatusBadRequest, domainErr)
		} else {
			writeError(w, http.StatusInternalServerError, &calculator.Error{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred."})
		}
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Result float64 `json:"result"`
	}{result})
}

type calculationRequest struct {
	operation calculator.Operation
	operands  []float64
}

func invalidJSON() *calculator.Error {
	return &calculator.Error{Code: "INVALID_JSON", Message: "Provide a valid JSON object."}
}

func (h *handler) decodeRequest(body []byte) (calculationRequest, *calculator.Error) {
	var request calculationRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil {
		return request, invalidJSON()
	}
	if opening != json.Delim('{') {
		return request, &calculator.Error{Code: "INVALID_REQUEST", Message: "The request must be a JSON object."}
	}
	fields := make(map[string]json.RawMessage, 3)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return request, invalidJSON()
		}
		key, ok := token.(string)
		if !ok {
			return request, invalidJSON()
		}
		if key != "operation" && key != "a" && key != "b" {
			return request, &calculator.Error{Code: "INVALID_REQUEST", Message: "The request contains an unknown field."}
		}
		if _, exists := fields[key]; exists {
			return request, &calculator.Error{Code: "INVALID_REQUEST", Message: "The request contains a duplicate field."}
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return request, invalidJSON()
		}
		fields[key] = value
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return request, invalidJSON()
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return request, &calculator.Error{Code: "INVALID_REQUEST", Message: "Provide exactly one JSON object."}
	}

	operation, ok := fields["operation"]
	if !ok {
		return request, required("operation")
	}
	if err := json.Unmarshal(operation, &request.operation); err != nil || !h.calculator.ValidOperation(request.operation) {
		return request, &calculator.Error{Code: "INVALID_OPERATION", Message: "Choose a supported operation.", Field: "operation"}
	}
	keys := []string{"a", "b"}
	if h.calculator.OperandCount(request.operation) == 1 {
		if _, exists := fields["b"]; exists {
			return request, &calculator.Error{Code: "INVALID_REQUEST", Message: "This operation accepts only operand a.", Field: "b"}
		}
		keys = keys[:1]
	}
	for _, key := range keys {
		raw, ok := fields[key]
		if !ok {
			return request, required(key)
		}
		if len(raw) == 0 || (raw[0] != '-' && (raw[0] < '0' || raw[0] > '9')) {
			return request, &calculator.Error{Code: "INVALID_NUMBER", Message: "Operands must be JSON numbers.", Field: key}
		}
		number, err := calculator.ParseNumber(string(raw), key)
		if err != nil {
			var domainErr *calculator.Error
			if errors.As(err, &domainErr) {
				return request, domainErr
			}
			return request, &calculator.Error{Code: "INVALID_NUMBER", Message: "Enter a valid number.", Field: key}
		}
		request.operands = append(request.operands, number)
	}
	return request, nil
}

func required(field string) *calculator.Error {
	return &calculator.Error{Code: "REQUIRED_FIELD", Message: "This field is required.", Field: field}
}

func writeError(w http.ResponseWriter, status int, appErr *calculator.Error) {
	writeJSON(w, status, struct {
		Error *calculator.Error `json:"error"`
	}{appErr})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":{"code":"INTERNAL_ERROR","message":"An unexpected error occurred."}}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

package server

import "encoding/json"

// Request represents an HTTP request.
type Request struct {
	Method     string
	Path       string
	Headers    map[string]string
	Body       string
	RemoteAddr string

	// Context values set by middleware
	TenantID  string
	AccessKey string
	AdminID   string
	IsAdmin   bool
	QuotaData interface{}
}

// Response represents an HTTP response.
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

// Handler is a request handler function.
type Handler func(req *Request) Response

// Middleware wraps a Handler.
type Middleware func(next Handler) Handler

// JSON writes a JSON response.
func JSON(statusCode int, data interface{}) Response {
	body, _ := json.Marshal(data)
	return Response{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}

// Text writes a text response.
func Text(statusCode int, text string) Response {
	return Response{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       []byte(text),
	}
}

// Error is a standard error response.
func Error(statusCode int, code, message string, details map[string]interface{}) Response {
	return JSON(statusCode, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}

// Binary writes a binary response.
func Binary(statusCode int, contentType string, body []byte) Response {
	return Response{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		Body: body,
	}
}

// NoContent returns a 204 response.
func NoContent() Response {
	return Response{
		StatusCode: 204,
		Headers:    map[string]string{},
		Body:       nil,
	}
}

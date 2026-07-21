package types

import "encoding/json"

// Request represents an HTTP request.
type Request struct {
	Method     string
	Path       string
	Headers    map[string]string
	Body       string
	RemoteAddr string

	// Context values set by middleware
	RequestID string
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

// InternalError returns a sanitized 500 error that doesn't leak details.
// The actual error is logged via the msg parameter which callers should log separately.
func InternalError(msg string) Response {
	return Error(500, "internal_error", msg, nil)
}

// BadRequest returns a 400 error.
func BadRequest(msg string) Response {
	return Error(400, "bad_request", msg, nil)
}

// NotFound returns a 404 error.
func NotFound(msg string) Response {
	return Error(404, "not_found", msg, nil)
}

// Unauthorized returns a 401 error.
func Unauthorized(msg string) Response {
	return Error(401, "unauthorized", msg, nil)
}

// Forbidden returns a 403 error.
func Forbidden(msg string) Response {
	return Error(403, "forbidden", msg, nil)
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

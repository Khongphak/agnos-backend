package response

// ErrorDetail is the standard error payload for all endpoints.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse wraps ErrorDetail for a consistent error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// HealthResponse is the payload for GET /health.
type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// NewError builds a standard error response.
func NewError(code, message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}

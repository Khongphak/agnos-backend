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

// StaffCreatedResponse is the payload for a successful POST /staff/create.
type StaffCreatedResponse struct {
	ID           int64  `json:"id"            example:"1"`
	Username     string `json:"username"      example:"nurse01"`
	HospitalCode string `json:"hospital_code" example:"BKK001"`
	Role         string `json:"role"          example:"staff"`
}

// TokenResponse is the payload for a successful POST /staff/login.
type TokenResponse struct {
	AccessToken  string `json:"access_token"  example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"a1b2c3d4e5f6..."`
	ExpiresIn    int64  `json:"expires_in"    example:"900"`
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

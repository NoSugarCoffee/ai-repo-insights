package errors

import (
	"fmt"
)

// ErrorType represents different categories of errors in the system
type ErrorType string

const (
	// Critical errors that should abort the pipeline
	ErrorTypeConfig       ErrorType = "CONFIG"
	ErrorTypeDataFetch    ErrorType = "DATA_FETCH"
	ErrorTypeFilesystem   ErrorType = "FILESYSTEM"
	ErrorTypeReportGen    ErrorType = "REPORT_GEN"
	
	// Non-critical errors that can be logged and continued
	ErrorTypeRepoFetch    ErrorType = "REPO_FETCH"
	ErrorTypeClassify     ErrorType = "CLASSIFY"
	ErrorTypeCache        ErrorType = "CACHE"
	
	// Recoverable errors that should be retried
	ErrorTypeRateLimit    ErrorType = "RATE_LIMIT"
	ErrorTypeNetwork      ErrorType = "NETWORK"
	ErrorTypeAPI          ErrorType = "API"
	ErrorTypeLLM          ErrorType = "LLM"
)

// AppError represents an application-specific error with context
type AppError struct {
	Type       ErrorType
	Message    string
	Context    map[string]string
	Underlying error
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Underlying)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Underlying
}

// IsCritical returns true if the error should abort the pipeline
func (e *AppError) IsCritical() bool {
	switch e.Type {
	case ErrorTypeConfig, ErrorTypeDataFetch, ErrorTypeFilesystem, ErrorTypeReportGen:
		return true
	default:
		return false
	}
}

// IsRetryable returns true if the error should be retried
func (e *AppError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeRateLimit, ErrorTypeNetwork, ErrorTypeAPI, ErrorTypeLLM:
		return true
	default:
		return false
	}
}

// NewConfigError creates a new configuration error
func NewConfigError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeConfig,
		Message:    message,
		Underlying: err,
		Context:    make(map[string]string),
	}
}

// NewDataFetchError creates a new data fetch error
func NewDataFetchError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeDataFetch,
		Message:    message,
		Underlying: err,
		Context:    make(map[string]string),
	}
}

// NewFilesystemError creates a new filesystem error
func NewFilesystemError(message string, path string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeFilesystem,
		Message:    message,
		Underlying: err,
		Context:    map[string]string{"path": path},
	}
}

// NewRepoFetchError creates a new repository fetch error
func NewRepoFetchError(owner string, repo string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeRepoFetch,
		Message:    fmt.Sprintf("failed to fetch repository %s/%s", owner, repo),
		Underlying: err,
		Context:    map[string]string{"owner": owner, "repo": repo},
	}
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(resetTime string) *AppError {
	return &AppError{
		Type:    ErrorTypeRateLimit,
		Message: "GitHub API rate limit exceeded",
		Context: map[string]string{"reset_time": resetTime},
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeNetwork,
		Message:    message,
		Underlying: err,
		Context:    make(map[string]string),
	}
}

// NewLLMError creates a new LLM error
func NewLLMError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeLLM,
		Message:    message,
		Underlying: err,
		Context:    make(map[string]string),
	}
}

// WithContext adds context to an error
func (e *AppError) WithContext(key string, value string) *AppError {
	e.Context[key] = value
	return e
}

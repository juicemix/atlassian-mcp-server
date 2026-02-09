package domain

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// DefaultResponseMapper is the default implementation of ResponseMapper.
// It converts Atlassian API responses to MCP-compliant tool responses.
type DefaultResponseMapper struct{}

// NewResponseMapper creates a new instance of DefaultResponseMapper.
func NewResponseMapper() ResponseMapper {
	return &DefaultResponseMapper{}
}

// MapToToolResponse converts an API response to MCP format.
// It handles responses from all four Atlassian tools (Jira, Confluence, Bitbucket, Bamboo).
// The apiResponse parameter should be the deserialized JSON response from an Atlassian API.
func (m *DefaultResponseMapper) MapToToolResponse(apiResponse interface{}) (*ToolResponse, error) {
	if apiResponse == nil {
		return &ToolResponse{
			Content: []ContentBlock{
				{
					Type: "text",
					Text: "{}",
				},
			},
		}, nil
	}

	// Convert the response to JSON
	jsonBytes, err := json.MarshalIndent(apiResponse, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal API response: %w", err)
	}

	// Create a text content block with the JSON response
	contentBlock := ContentBlock{
		Type: "text",
		Text: string(jsonBytes),
	}

	// Check if the response has pagination metadata
	// This handles SearchResults and other paginated responses
	paginationInfo := extractPaginationInfo(apiResponse)
	if paginationInfo != "" {
		// Add pagination info as a separate content block
		return &ToolResponse{
			Content: []ContentBlock{
				contentBlock,
				{
					Type: "text",
					Text: paginationInfo,
				},
			},
		}, nil
	}

	return &ToolResponse{
		Content: []ContentBlock{contentBlock},
	}, nil
}

// extractPaginationInfo extracts pagination metadata from responses that support it.
// Returns a formatted string with pagination information, or empty string if not applicable.
func extractPaginationInfo(apiResponse interface{}) string {
	// Check if it's a Jira SearchResults
	if searchResults, ok := apiResponse.(*SearchResults); ok {
		return fmt.Sprintf("\nPagination: Showing %d-%d of %d total results",
			searchResults.StartAt+1,
			searchResults.StartAt+len(searchResults.Issues),
			searchResults.Total)
	}

	// Check if it's a SearchResults value (not pointer)
	if searchResults, ok := apiResponse.(SearchResults); ok {
		return fmt.Sprintf("\nPagination: Showing %d-%d of %d total results",
			searchResults.StartAt+1,
			searchResults.StartAt+len(searchResults.Issues),
			searchResults.Total)
	}

	// Add more pagination checks for other response types as needed
	// For now, we handle the most common case (Jira search results)

	return ""
}

// MapError converts an API error to MCP error format.
// This method maps HTTP status codes and error responses from Atlassian APIs
// to appropriate JSON-RPC error codes and messages.
func (m *DefaultResponseMapper) MapError(err error) *Error {
	if err == nil {
		return nil
	}

	// Check if it's an HTTP error with status code
	if httpErr, ok := err.(HTTPError); ok {
		return mapHTTPError(httpErr)
	}

	// Check if it's already a domain Error
	if domainErr, ok := err.(*Error); ok {
		return domainErr
	}

	// Default to internal error for unknown error types
	return &Error{
		Code:    InternalError,
		Message: err.Error(),
	}
}

// HTTPError represents an HTTP error with status code and message.
// This is used to wrap HTTP errors from Atlassian API calls.
type HTTPError struct {
	StatusCode int
	Message    string
	Body       string
}

// Error implements the error interface for HTTPError.
func (e HTTPError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("HTTP %d: %s - %s", e.StatusCode, e.Message, e.Body)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// NewHTTPError creates a new HTTPError with the given status code and message.
func NewHTTPError(statusCode int, message string, body string) HTTPError {
	return HTTPError{
		StatusCode: statusCode,
		Message:    message,
		Body:       body,
	}
}

// mapHTTPError maps HTTP status codes to JSON-RPC error codes.
func mapHTTPError(httpErr HTTPError) *Error {
	var code int
	var message string

	switch httpErr.StatusCode {
	case http.StatusUnauthorized:
		code = AuthenticationError
		message = "Authentication failed"
	case http.StatusForbidden:
		code = AuthenticationError
		message = "Access forbidden - insufficient permissions"
	case http.StatusNotFound:
		code = APIError
		message = "Resource not found"
	case http.StatusBadRequest:
		code = InvalidParams
		message = "Bad request - invalid parameters"
	case http.StatusConflict:
		code = APIError
		message = "Conflict - resource already exists or version mismatch"
	case http.StatusTooManyRequests:
		code = RateLimitError
		message = "Rate limit exceeded"
	case http.StatusInternalServerError:
		code = APIError
		message = "Internal server error"
	case http.StatusServiceUnavailable:
		code = NetworkError
		message = "Service unavailable"
	case http.StatusGatewayTimeout:
		code = NetworkError
		message = "Gateway timeout"
	default:
		if httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
			code = APIError
			message = fmt.Sprintf("Client error: %s", httpErr.Message)
		} else if httpErr.StatusCode >= 500 {
			code = APIError
			message = fmt.Sprintf("Server error: %s", httpErr.Message)
		} else {
			code = InternalError
			message = httpErr.Message
		}
	}

	// Include the original error details in the data field
	errorData := map[string]interface{}{
		"statusCode": httpErr.StatusCode,
		"message":    httpErr.Message,
	}
	if httpErr.Body != "" {
		errorData["body"] = httpErr.Body
	}

	return &Error{
		Code:    code,
		Message: message,
		Data:    errorData,
	}
}

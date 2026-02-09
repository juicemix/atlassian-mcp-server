package application

import (
	"fmt"

	"atlassian-mcp-server/internal/domain"
)

// getStringParam extracts a string parameter from the arguments map.
// Returns an error if the parameter is required but missing or not a string.
func getStringParam(args map[string]interface{}, name string, required bool) (string, error) {
	value, exists := args[name]
	if !exists {
		if required {
			return "", &domain.Error{
				Code:    domain.InvalidParams,
				Message: fmt.Sprintf("missing required parameter: %s", name),
			}
		}
		return "", nil
	}

	strValue, ok := value.(string)
	if !ok {
		return "", &domain.Error{
			Code:    domain.InvalidParams,
			Message: fmt.Sprintf("parameter %s must be a string", name),
		}
	}

	return strValue, nil
}

// getIntParam extracts an integer parameter from the arguments map.
// Returns an error if the parameter is required but missing or not a number.
// Also returns an error if the parameter exists but is not a valid number type.
func getIntParam(args map[string]interface{}, name string, required bool) (int, error) {
	value, exists := args[name]
	if !exists {
		if required {
			return 0, &domain.Error{
				Code:    domain.InvalidParams,
				Message: fmt.Sprintf("missing required parameter: %s", name),
			}
		}
		return 0, nil
	}

	// Handle both float64 (from JSON) and int
	switch v := value.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		// If the parameter exists but is not a valid type, return an error
		// even if it's not required
		return 0, &domain.Error{
			Code:    domain.InvalidParams,
			Message: fmt.Sprintf("parameter %s must be an integer", name),
		}
	}
}

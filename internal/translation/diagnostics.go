package translation

// ResponseDiagnostic contains only fixed categories. It never retains provider
// responses, prompts, document text, endpoint URLs, model names or API keys.
type ResponseDiagnostic struct {
	Category     string `json:"category"`
	FinishReason string `json:"finish_reason"`
	Truncation   string `json:"truncation"`
}

type AIResponseError struct {
	Diagnostic ResponseDiagnostic
	message    string
}

func (e *AIResponseError) Error() string { return e.message }

func responseDiagnostic(category, finishReason string) ResponseDiagnostic {
	truncation := "not_reported"
	switch finishReason {
	case "length":
		truncation = "reported"
	case "stop", "content_filter", "tool_calls", "function_call":
	default:
		finishReason = "unknown"
		truncation = "unknown"
	}
	return ResponseDiagnostic{Category: category, FinishReason: finishReason, Truncation: truncation}
}

func (d ResponseDiagnostic) Valid() bool {
	switch d.Category {
	case "no_choices", "invalid_format", "incomplete_translations":
		return d == responseDiagnostic(d.Category, d.FinishReason)
	default:
		return false
	}
}

func aiResponseError(category, finishReason, message string) error {
	return &AIResponseError{Diagnostic: responseDiagnostic(category, finishReason), message: message}
}

package models

// SearchRequest represents a search request
type SearchRequest struct {
	Query          string   `json:"query"`
	DocTypes       []string `json:"doc_types,omitempty"`
	TopK           int      `json:"top_k,omitempty"`
	MinScore       float32  `json:"min_score,omitempty"`
	IncludeContent bool     `json:"include_content,omitempty"`
}

// Validate validates the search request
func (r *SearchRequest) Validate() error {
	if r.Query == "" {
		return &ValidationError{Field: "query", Message: "query is required"}
	}
	return nil
}

// ApplyDefaults applies default values
func (r *SearchRequest) ApplyDefaults() {
	if r.TopK <= 0 {
		r.TopK = 10
	}
	if r.TopK > 50 {
		r.TopK = 50
	}
	if r.MinScore <= 0 {
		r.MinScore = 0.3
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

package translation

type Progress struct {
	RequestID      string `json:"requestId"`
	Completed      int    `json:"completed"`
	Total          int    `json:"total"`
	Provider       string `json:"provider"`
	TargetLanguage string `json:"targetLanguage"`
}

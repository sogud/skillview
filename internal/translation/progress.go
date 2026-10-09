package translation

import "sync"

type Progress struct {
	RequestID      string `json:"requestId"`
	Completed      int    `json:"completed"`
	Total          int    `json:"total"`
	Provider       string `json:"provider"`
	TargetLanguage string `json:"targetLanguage"`
}

func translationReporter(settings Settings, total int, callbacks []func(Progress)) func(int) {
	var mu sync.Mutex
	completed := 0
	return func(count int) {
		mu.Lock()
		defer mu.Unlock()
		completed += count
		if len(callbacks) > 0 && callbacks[0] != nil {
			callbacks[0](Progress{Completed: completed, Total: total, Provider: settings.Provider, TargetLanguage: settings.TargetLanguage})
		}
	}
}

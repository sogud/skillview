package library

type ScanProgress struct {
	Phase        string `json:"phase"`
	Directories  int    `json:"directories"`
	Skills       int    `json:"skills"`
	SourcesDone  int    `json:"sourcesDone"`
	SourcesTotal int    `json:"sourcesTotal"`
	Root         string `json:"root"`
}

func reportProgress(callbacks []func(ScanProgress), progress ScanProgress) {
	if len(callbacks) > 0 && callbacks[0] != nil {
		callbacks[0](progress)
	}
}

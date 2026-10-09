package library

type Source struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	ResolvedPath string   `json:"resolvedPath"`
	Tools        []string `json:"tools"`
	Writable     bool     `json:"writable"`
	Approved     bool     `json:"approved"`
	Identity     string   `json:"identity"`
	Status       string   `json:"status"`
	Message      string   `json:"message"`
	Automatic    bool     `json:"automatic"`
	Scope        string   `json:"scope"`
}

type Occurrence struct {
	SourceID     string `json:"sourceId"`
	Path         string `json:"path"`
	Linked       bool   `json:"linked"`
	Removable    bool   `json:"removable"`
	LinkRevision string `json:"linkRevision"`
}

type Skill struct {
	ID          string       `json:"id"`
	ContentKey  string       `json:"contentKey"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Path        string       `json:"path"`
	Revision    string       `json:"revision"`
	Warnings    []string     `json:"warnings"`
	Occurrences []Occurrence `json:"occurrences"`
	Tools       []string     `json:"tools"`
	Editable    bool         `json:"editable"`
}

type Issue struct {
	SourceID string `json:"sourceId"`
	Path     string `json:"path"`
	Target   string `json:"target"`
	Message  string `json:"message"`
}

type Catalog struct {
	Skills    []Skill         `json:"skills"`
	Sources   []Source        `json:"sources"`
	Issues    []Issue         `json:"issues"`
	ScannedAt string          `json:"scannedAt"`
	NeedsScan bool            `json:"needsScan"`
	Discovery DiscoveryReport `json:"discovery"`
	Agents    []AgentInfo     `json:"agents"`
}

type DiscoveryReport struct {
	Roots       []string `json:"roots"`
	Directories int      `json:"directories"`
	Denied      int      `json:"denied"`
	Limited     bool     `json:"limited"`
	Completed   bool     `json:"completed"`
}

type Resource struct {
	Revision string `json:"revision"`
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Size     int64  `json:"size"`
}

type Document struct {
	SkillID        string `json:"skillId"`
	Path           string `json:"path"`
	Text           string `json:"text"`
	Body           string `json:"body"`
	Revision       string `json:"revision"`
	Kind           string `json:"kind"`
	Size           int64  `json:"size"`
	DataURL        string `json:"dataUrl"`
	Editable       bool   `json:"editable"`
	ReadOnlyReason string `json:"readOnlyReason"`
	Newline        string `json:"newline"`
}

type SaveResult struct {
	Status       string   `json:"status"`
	Document     Document `json:"document"`
	RecoveryPath string   `json:"recoveryPath"`
}

type TrashReceipt struct {
	ID             string `json:"id"`
	OriginalPath   string `json:"originalPath"`
	TrashPath      string `json:"trashPath"`
	Identity       string `json:"identity"`
	ParentIdentity string `json:"parentIdentity"`
	Warning        string `json:"warning"`
}

type Preferences struct {
	RecentTrash     []TrashReceipt  `json:"recentTrash"`
	Sources         []Source        `json:"sources"`
	AccentColor     string          `json:"accentColor"`
	Theme           string          `json:"theme"`
	Language        string          `json:"language"`
	ExcludedSources map[string]bool `json:"excludedSources"`
}

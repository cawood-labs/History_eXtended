package search

// Filter scopes history results (atuin-style Ctrl-R filter cycle).
type Filter int

const (
	FilterGlobal Filter = iota
	FilterHost
	FilterDir
	FilterSession
)

// Mode selects how the query string is matched.
type Mode int

const (
	ModeFuzzy Mode = iota
	ModePrefix
	ModeFTS
	ModeSemantic
)

// Request is input to Search.
type Request struct {
	Query     string
	Filter    Filter
	Mode      Mode
	Host      string // for FilterHost
	Cwd       string // for FilterDir
	SessionID string // for FilterSession
	Dedup     bool
	Limit     int
	NoImport  bool
}

// Row is one search result for display or machine export.
type Row struct {
	EventID    int64
	Cmd        string
	Cwd        string
	Host       string
	SessionID  string
	Origin     string
	ExitCode   *int
	DurationMs *int64
	StartedAt  float64
	GitBranch  string
	GitCommit  string
	DupCount   int
}

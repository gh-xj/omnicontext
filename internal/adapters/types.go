package adapters

type Turn struct {
	UserMessage      string
	AssistantSummary string
	Timestamp        string
}

type Session struct {
	SessionID      string
	SessionType    string
	SessionPath    string
	WorkspacePath  string
	StartedAt      string
	LastActivityAt string
	SessionTitle   string
	SessionSummary string
	Metadata       string
	Turns          []Turn
}

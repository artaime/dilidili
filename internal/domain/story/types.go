package story

import "time"

const (
	ActionGenerate   = "generate"
	ActionReplay     = "replay"
	ActionResume     = "resume"
	ActionListRecent = "list_recent"

	StoryRefLast      = "last"
	StoryRefLastNight = "last_night"
	StoryRefFavorite  = "favorite"

	PlayStatusCompleted   = "completed"
	PlayStatusInterrupted = "interrupted"
	PlayStatusAbandoned   = "abandoned"
	PlayStatusPlaying     = "playing"

	StatusNeedParams = "need_params"
	StatusNotFound   = "not_found"
	StatusCandidates = "candidates"
	StatusReady      = "ready"
	StatusReplay     = "replay"
	StatusResume     = "resume"
	StatusStreaming  = "streaming"
)

type MemoryHint struct {
	Field      string  `json:"field"`
	Value      string  `json:"value"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence"`
}

type PlayPosition struct {
	SegmentIndex      int    `json:"segment_index"`
	CharOffset        int    `json:"char_offset"`
	LastSentenceIndex int    `json:"last_sentence_index,omitempty"`
	LastSentence      string `json:"last_sentence,omitempty"`
}

type StoryParams struct {
	RequestType    string       `json:"request_type,omitempty"`
	NarrationMode  string       `json:"narration_mode,omitempty"` // canonical|creative
	Theme          string       `json:"theme,omitempty"`
	Style          string       `json:"style,omitempty"`
	AgeBand        string       `json:"age_band,omitempty"`
	AgeYears       *int         `json:"age_years,omitempty"`
	IsBedtime      *bool        `json:"is_bedtime,omitempty"`
	DurationHint   string       `json:"duration_hint,omitempty"`
	Interests      []string     `json:"interests,omitempty"`
	MemoryHints    []MemoryHint `json:"memory_hints,omitempty"`
	UserSaidCasual bool         `json:"user_said_casual,omitempty"`
}

type StoryRecord struct {
	StoryID        string         `json:"story_id"`
	DeviceID       string         `json:"device_id"`
	AgentID        string         `json:"agent_id"`
	Title          string         `json:"title"`
	FullText       string         `json:"full_text"`
	Segments       []string       `json:"segments"`
	Mode           string         `json:"mode,omitempty"`
	AgeBand        string         `json:"age_band,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	LastPlayedAt   time.Time      `json:"last_played_at"`
	PlayCount      int            `json:"play_count"`
	CompleteCount  int            `json:"complete_count"`
	LastPlayStatus string         `json:"last_play_status,omitempty"`
	LastPosition   PlayPosition   `json:"last_position"`
	GenerationComplete bool       `json:"generation_complete,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	ParamsSnapshot map[string]any `json:"params_snapshot,omitempty"`
	SeriesID       string         `json:"series_id,omitempty"`
	SeriesComplete bool           `json:"series_complete,omitempty"`
	ReplaySuggestCount int        `json:"replay_suggest_count,omitempty"`
}

type StoryCandidate struct {
	StoryID      string    `json:"story_id"`
	Title        string    `json:"title"`
	LastPlayedAt time.Time `json:"last_played_at"`
	PlayCount    int       `json:"play_count"`
}

type ToolRequest struct {
	Action        string
	StoryRef      string
	FromBeginning *bool
	StoryParams
	DeviceID       string
	AgentID        string
	MemoryContext  string
	Now            time.Time
}

type ToolResult struct {
	Status           string           `json:"status"`
	Message          string           `json:"message,omitempty"`
	StoryID          string           `json:"story_id,omitempty"`
	Title            string           `json:"title,omitempty"`
	TextToSpeak      string           `json:"text_to_speak,omitempty"`
	Segments         []string         `json:"segments,omitempty"`
	StartSegment     int              `json:"start_segment,omitempty"`
	Missing          []string         `json:"missing,omitempty"`
	Candidates       []StoryCandidate `json:"candidates,omitempty"`
	SuggestNewStory  bool             `json:"suggest_new_story,omitempty"`
	Meta             map[string]any   `json:"meta,omitempty"`
}

type ResolvedParams struct {
	Params      StoryParams
	Missing     []string
	AssumedFields map[string]string
}

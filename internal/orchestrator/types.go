package orchestrator

import "time"

type ChunkType int

const (
	ChunkStageEvent ChunkType = iota
	ChunkToken
	ChunkToolCall
	ChunkDone
	ChunkError
)

type ResponseChunk struct {
	Type     ChunkType
	Text     string
	Stage    string
	ToolName string
	Error    error
}

type AttachmentType string

const (
	AttachmentTypeImage    AttachmentType = "image"
	AttachmentTypeDocument AttachmentType = "document"
	AttachmentTypeVoice    AttachmentType = "voice"
	AttachmentTypeVideo    AttachmentType = "video"
)

type Attachment struct {
	Type        AttachmentType
	Data        []byte
	MimeType    string
	FileName    string
	Caption     string
	Transcribed string
	ExtractedText string
}

type NormalizedInboundMessage struct {
	UserID      string
	PlatformID  string
	SessionID   string
	MessageID   string
	Text        string
	Attachments []Attachment
	Metadata    map[string]any
	Timestamp   time.Time
}

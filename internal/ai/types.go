package ai

type AttachmentFile []byte

type Attachment struct {
	URI string
	Mimetype string
}

type Message struct {
	Attachments []*Attachment
	Msg         string
	SenderInfo *MessageSenderInfo
}

type MessageSenderInfo struct {
	User *MessageUser
	Household *MessageHousehold
}

type MessageUser struct {
	ID int32
	Name string
	DiscordID *string
}

type MessageHousehold struct {
	ID int32
	Name string
}

type UsageStats struct {
	TotalTokens int
	InputTokens int
	OutputTokens int
}

type AgentUpdate struct {
	Text string `json:"text,omitempty"`
	ToolCall *ToolCallUpdate `json:"toolCall,omitempty"`
	ToolResponse *ToolResponseUpdate `json:"toolResponse,omitempty"`
	Err error
}

type ToolCallUpdate struct {
	Name string
	Args any
}

type ToolResponseUpdate struct {
	Name string
	Response any
}

type StreamingMode string
const (
	StreamingModeComplete StreamingMode = "complete"
	StreamingModeChunks StreamingMode = "chunks"
)

func (s StreamingMode) Valid() bool {
	switch s {
	case StreamingModeComplete, StreamingModeChunks:
		return true
	default:
		return false
	}
}
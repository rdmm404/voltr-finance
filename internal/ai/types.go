package ai

import (
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

type GenerationConfig struct {
	Model       string
	Temperature float32
}


type AttachmentFile []byte

// func (a AttachmentFile) MarshalJSON() ([]byte, error) {
// 	return []byte(fmt.Sprintf("File of size %d kB", len(a) / 1024)), nil
// }

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

// this is just to override the json marshalling of files
type CustomContent struct {
	Parts []*CustomPart `json:"parts,omitempty"`
	genai.Content
}

type CustomPart struct {
	InlineData *CustomBlob `json:"inlineData,omitempty"`
	genai.Part
}

type CustomBlob genai.Blob

func (b *CustomBlob) MarshalJson() ([]byte, error) {
	type Alias CustomBlob
	return json.Marshal(&struct {
		Data string `json:"data,omitempty"`
		*Alias
	}{
		Data: fmt.Sprintf("File of size %d kB", len(b.Data) / 1024),
		Alias:    (*Alias)(b),
	})
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
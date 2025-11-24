package agent

import (
	"errors"
	"fmt"

	gai "github.com/firebase/genkit/go/ai"
)

type AttachmentFile []byte

type Attachment struct {
	URI      string
	Mimetype string
}

type Message struct {
	Attachments []*Attachment `json:"attachments,omitempty"`
	Msg         string
	SenderInfo  *MessageSenderInfo
}

func (m *Message) ToGenkit() (*gai.Message, error) {
	var parts []*gai.Part

	if m.Msg != "" {
		parts = append(parts, gai.NewTextPart(m.Msg))
	}

	for _, att := range m.Attachments {
		if att.Mimetype == "" {
			return nil, fmt.Errorf("attachment missing mimetype %+v", att)
		}
		parts = append(parts, gai.NewMediaPart(att.Mimetype, att.URI))
	}

	return gai.NewUserMessage(parts...), nil
}

type MessageSenderInfo struct {
	User      *MessageUser
	Household *MessageHousehold
	ChannelID string
}

type MessageUser struct {
	ID        int64
	Name      string
	DiscordID string
}

type MessageHousehold struct {
	ID   int64
	Name string
}

type UsageStats struct {
	TotalTokens  int
	InputTokens  int
	OutputTokens int
}

type AgentUpdate struct {
	Text         string              `json:"text,omitempty"`
	ToolCall     *ToolCallUpdate     `json:"toolCall,omitempty"`
	ToolResponse *ToolResponseUpdate `json:"toolResponse,omitempty"`
	Err          error
}

type ToolCallUpdate struct {
	Name string
	Args any
}

type ToolResponseUpdate struct {
	Name     string
	Response any
}

type StreamingMode string

const (
	StreamingModeMessages StreamingMode = "messages"
	StreamingModeChunks   StreamingMode = "chunks"
)

func (s StreamingMode) Valid() bool {
	switch s {
	case StreamingModeMessages, StreamingModeChunks:
		return true
	default:
		return false
	}
}

var ErrMessagePersistance = errors.New("error persisting message")

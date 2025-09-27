package ai

import (
	"encoding/json"
	"fmt"
	database "rdmm404/voltr-finance/internal/database/repository"
	"rdmm404/voltr-finance/internal/transaction"

	"google.golang.org/genai"
)

type AgentResponse struct {
	GenerateReponse *genai.GenerateContentResponse
	Err error
}

type GenerationConfig struct {
	Model       string
	Temperature float32
}

type Transaction struct {
	Name            string                      `json:"name"`
	Description     string                      `json:"description"`
	Amount          float32                     `json:"amount"`
	TransactionType transaction.TransactionType `json:"transactionType"`
}

type AttachmentFile []byte

func (a AttachmentFile) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("File of size %d kB", len(a) / 1024)), nil
}

type Attachment struct {
	File     AttachmentFile
	URI string
	Mimetype string
}

type Message struct {
	Attachments []*Attachment
	Msg         string
	SenderInfo *MessageSenderInfo
}

type MessageSenderInfo struct {
	User *database.User
	Household *database.Household
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
	TotalTokens int32
	InputTokens int32
	OutputTokens int32
}

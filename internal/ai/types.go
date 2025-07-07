package ai

import (
	"rdmm404/voltr-finance/internal/transaction"

	"google.golang.org/genai"
)

type AgentResponse genai.GenerateContentResponse

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

type Attachment struct {
	File     []byte
	Mimetype string
}

type Message struct {
	Attachments []Attachment
	Msg         string
}

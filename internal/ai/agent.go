package ai

import (
	"context"
	"errors"
	"fmt"
	"rdmm404/voltr-finance/internal/ai/tools"

	"google.golang.org/genai"
)

type Agent interface {
	GetClient() *genai.Client
	SendMessage(ctx context.Context, msg *Message) (*LlmResponse, error)
}

type agent struct {
	client *genai.Client
}

func NewAgent(ctx context.Context) (Agent, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{})
	if err != nil {
		return &agent{}, fmt.Errorf("error while creating the LLM client - %w", err)
	}
	return &agent{
		client: client,
	}, nil
}

func (lc *agent) GetClient() *genai.Client {
	return lc.client
}

func (lc *agent) SendMessage(ctx context.Context, msg *Message) (*LlmResponse, error) {
	if msg == nil {
		return nil, errors.New("arguments are required")
	}

	if (len(msg.Attachments) == 0) && msg.Msg == "" {
		return nil, errors.New("at least one of (img, msg) must be set")
	}

	agentTools := tools.GetGenaiTools()
	config := genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		Tools:            agentTools,
	}

	content := &genai.Content{
		Role:  "user",
		Parts: make([]*genai.Part, 0),
	}

	if msg.Msg != "" {
		content.Parts = append(content.Parts, &genai.Part{Text: msg.Msg})
	}

	for _, attachment := range msg.Attachments {
		content.Parts = append(content.Parts, &genai.Part{InlineData: &genai.Blob{Data: attachment.File, MIMEType: attachment.Mimetype}})
	}

	response, err := lc.client.Models.GenerateContent(ctx, "gemini-2.5-flash-lite-preview-06-17", []*genai.Content{content}, &config)

	if err == nil {
		return &LlmResponse{}, err
	}

	fmt.Printf("response from llm %+v", response)

	return (*LlmResponse)(response), nil
}

package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"rdmm404/voltr-finance/internal/ai/tools"

	"google.golang.org/genai"
)

type Agent struct {
	Client *genai.Client
	config *AgentConfig
}

type AgentConfig struct {
	Model string
}

func NewAgent(ctx context.Context, cfg *AgentConfig) (*Agent, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{})
	if err != nil {
		return &Agent{}, fmt.Errorf("error while creating the LLM client - %w", err)
	}

	if (cfg == nil) {
		cfg = &AgentConfig{}
	}

	if (cfg.Model == "") {
		cfg.Model = "gemini-2.5-flash-lite-preview-06-17"
	}

	return &Agent{
		Client: client,
		config: cfg,
	}, nil
}

func (a *Agent) SendMessage(ctx context.Context, msg *Message) (*LlmResponse, error) {
	if msg == nil {
		return nil, errors.New("arguments are required")
	}

	if (len(msg.Attachments) == 0) && msg.Msg == "" {
		return nil, errors.New("at least one of (img, msg) must be set")
	}

	agentTools := tools.GetGenaiTools()
	config := genai.GenerateContentConfig{
		ResponseMIMEType: "text/plain",
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

	contentJson, errContent := json.MarshalIndent(content, "", "  ")
	configJson, errConfig := json.MarshalIndent(config, "", "  ")

	contentStr := string(contentJson)
	configStr := string(configJson)
	if errContent != nil {
		fmt.Printf("Something happened while marshaling LLM content, falling back to struct %v", errContent)
		contentStr = fmt.Sprintf("%+v", content)
	}

	if errConfig != nil {
		fmt.Printf("Something happened while marshaling LLM config, falling back to struct %v", errConfig)
		configStr = fmt.Sprintf("%+v", config)
	}

	fmt.Printf("Sending request to LLM\n CONTENT: %v \n CONFIG: %v\n", contentStr, configStr)


	response, err := a.Client.Models.GenerateContent(ctx, a.config.Model, []*genai.Content{content}, &config)

	if err != nil {
		return &LlmResponse{}, err
	}

	fmt.Printf("response from llm %+v\n", response.Text())

	return (*LlmResponse)(response), nil
}

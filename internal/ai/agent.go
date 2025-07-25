package ai

import (
	"context"
	"errors"
	"fmt"
	"rdmm404/voltr-finance/internal/ai/tool"

	"google.golang.org/genai"
)

type Agent struct {
	Client   *genai.Client
	config   *AgentConfig
	messages []*genai.Content
	tp *tool.ToolProvider
}

type AgentConfig struct {
	Model            string
	MaxTokens        int32
	generationConfig *genai.GenerateContentConfig
}

func NewAgent(ctx context.Context, cfg *AgentConfig, tp *tool.ToolProvider) (*Agent, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{})
	if err != nil {
		return &Agent{}, fmt.Errorf("error while creating the LLM client - %w", err)
	}

	if cfg == nil {
		cfg = &AgentConfig{}
	}

	if cfg.Model == "" {
		cfg.Model = "gemini-2.5-flash-lite-preview-06-17"
	}

	systemInstruction, err := formatSystemPrompt(43)

	if err != nil {
		return &Agent{}, fmt.Errorf("error while creating system prompt - %w", err)
	}

	cfg.generationConfig = &genai.GenerateContentConfig{
		ResponseMIMEType: "text/plain",
		Tools:            tp.GetGenaiTools(),
		SystemInstruction: &genai.Content{
			Role: "system",
			Parts: []*genai.Part{
				{Text: systemInstruction},
			},
		},
		MaxOutputTokens: cfg.MaxTokens,
	}

	return &Agent{
		Client: client,
		config: cfg,
		tp: tp,
	}, nil
}

func (a *Agent) SendMessage(ctx context.Context, msg *Message) (*AgentResponse, error) {
	if msg == nil {
		return nil, errors.New("arguments are required")
	}

	if (len(msg.Attachments) == 0) && msg.Msg == "" {
		return nil, errors.New("at least one of (img, msg) must be set")
	}

	content := &genai.Content{
		Role:  "user",
		Parts: make([]*genai.Part, 0),
	}

	if msg.Msg != "" {
		content.Parts = append(content.Parts, genai.NewPartFromText(msg.Msg))
	}

	for _, attachment := range msg.Attachments {
		if len(attachment.File) > 0 {
			content.Parts = append(content.Parts, genai.NewPartFromBytes(attachment.File, attachment.Mimetype))
			continue
		}

		if attachment.URI != "" {
			content.Parts = append(content.Parts, genai.NewPartFromURI(attachment.URI, attachment.Mimetype))
			continue
		}

		return nil, fmt.Errorf("invalid attachment provided - %+v",  attachment)

	}

	a.messages = append(a.messages, content)

	contentStr, configStr := LLMRequestToString(a.messages, a.config.generationConfig)

	fmt.Printf("Sending request to LLM\n CONTENT: %v \n CONFIG: %v\n", contentStr, configStr)

	response, err := a.Client.Models.GenerateContent(ctx, a.config.Model, a.messages, a.config.generationConfig)

	if err != nil {
		return &AgentResponse{}, err
	}

	a.messages = append(a.messages, response.Candidates[0].Content)

	fmt.Printf("response from llm %+v\n", LLMResponseToString(response))

	toolCalls := response.FunctionCalls()

	for _, call := range response.FunctionCalls() {
		a.messages = append(a.messages, genai.NewContentFromFunctionCall(call.Name, call.Args, "model"))
		result := a.tp.ExecuteToolCall(call)
		a.messages = append(a.messages, &genai.Content{
			Role:  "user",
			Parts: []*genai.Part{{FunctionResponse: result}},
		})
	}

	if len(toolCalls) > 0 {
		contentStr, configStr := LLMRequestToString(a.messages, a.config.generationConfig)

		fmt.Printf("Tool calls detected. sending request to LLM\n CONTENT: %v \n CONFIG: %v\n", contentStr, configStr)
		response, err = a.Client.Models.GenerateContent(ctx, a.config.Model, a.messages, a.config.generationConfig)
		if err != nil {
			return &AgentResponse{}, err
		}
		a.messages = append(a.messages, response.Candidates[0].Content)
		fmt.Printf("response from llm %+v\n", LLMResponseToString(response))
	}

	return (*AgentResponse)(response), nil
}

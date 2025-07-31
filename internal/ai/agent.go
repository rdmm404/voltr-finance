package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"rdmm404/voltr-finance/internal/ai/tool"
	"time"

	"google.golang.org/genai"
)

type Agent struct {
	Client   *genai.Client
	config   *AgentConfig
	messages []*genai.Content
	tp *tool.ToolProvider
	usageStats UsageStats
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

	configJson, err := json.MarshalIndent(cfg.generationConfig, "", "  ")
	configStr := string(configJson)
	if err != nil {
		configStr = fmt.Sprintf("%+v", cfg.generationConfig)
	}
	fmt.Println("Initializing agent with config" + configStr)

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

	fmt.Printf("Sending request to LLM\n CONTENT: %s\n", LLMRequestToString(a.messages))

	response, err := a.generateContentRetry(ctx, a.config.Model, a.messages, a.config.generationConfig)

	if err != nil {
		return &AgentResponse{}, err
	}

	a.countTokens(response)

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
		fmt.Printf("Tool calls detected. sending request to LLM\n CONTENT: %v \n", LLMRequestToString(a.messages))
		response, err = a.generateContentRetry(ctx, a.config.Model, a.messages, a.config.generationConfig)
		if err != nil {
			return &AgentResponse{}, err
		}
		a.countTokens(response)
		a.messages = append(a.messages, response.Candidates[0].Content)
		fmt.Printf("response from llm %+v\n", LLMResponseToString(response))
	}

	return (*AgentResponse)(response), nil
}

func (a *Agent) countTokens(resp *genai.GenerateContentResponse) {
	if resp == nil || resp.UsageMetadata == nil {
		return
	}

	a.usageStats.InputTokens += resp.UsageMetadata.PromptTokenCount + resp.UsageMetadata.ToolUsePromptTokenCount
	a.usageStats.OutputTokens += resp.UsageMetadata.CandidatesTokenCount
	a.usageStats.TotalTokens += resp.UsageMetadata.TotalTokenCount

	fmt.Printf("\nCurrent usage stats: %+v\n", a.usageStats)
}

func (a *Agent) generateContentRetry(
	ctx context.Context,
	model string,
	contents []*genai.Content,
	config *genai.GenerateContentConfig,
) (*genai.GenerateContentResponse, error) {
	// TODO make this configurable
	maxRetries := 5
	delay := 2 * time.Second // TODO exponential backoff?

	var lastErr error

	for retry := range maxRetries {
		time.Sleep(delay * time.Duration(retry))
		response, err := a.Client.Models.GenerateContent(ctx, model, contents, config)
		lastErr = err
		if err == nil {
			return response, nil
		}

		apiErr, ok := err.(genai.APIError)
		if !ok {
			return nil, err
		}

		if apiErr.Code != 500 {
			return nil, err
		}

		fmt.Printf("Error response received, retrying - %v\n", err)
	}

	return nil, lastErr
}
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"rdmm404/voltr-finance/internal/ai/tool"

	gai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

type chatFlow = *core.Flow[*gai.Message, string, string]

type flows struct {
	chat chatFlow
}

type Agent struct {
	g   *genkit.Genkit
	messages []*gai.Message
	tp *tool.ToolProvider
	usageStats UsageStats
	flows *flows
}

func NewAgent(ctx context.Context, tp *tool.ToolProvider) (*Agent, error) {
	g := genkit.Init(
		ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}),
		genkit.WithDefaultModel("googleai/gemini-2.0-flash"),
	)

	tp.Init(g)

	// systemInstruction, err := systemPrompt(43)

	// if err != nil {
	// 	return &Agent{}, fmt.Errorf("error while creating system prompt - %w", err)
	// }

	// cfg.generationConfig = &genai.GenerateContentConfig{
	// 	Tools:            tp.GetGenaiTools(),
	// 	SystemInstruction: &genai.Content{
	// 		Role: "system",
	// 		Parts: []*genai.Part{
	// 			{Text: systemInstruction},
	// 		},
	// 	},
	// }

	a := &Agent{
		g: g,
		tp: tp,
	}

	// TODO: Find a better way to do this
	a.flows = &flows{
		chat: a.chatFlow(),
	}

	return a, nil
}

func (a *Agent) chatFlow() chatFlow {
	return genkit.DefineStreamingFlow(a.g, "chat",
		func(ctx context.Context, message *gai.Message, callback core.StreamCallback[string]) (string, error) {
			a.messages = append(a.messages, message)

			resp, err := genkit.Generate(
				ctx,
				a.g,
				gai.WithTools(a.tp.GetAvailableTools()...),
				gai.WithMessages(
					a.messages...
				),
				// ai.WithStreaming(func(ctx context.Context, chunk *ai.ModelResponseChunk))
			)

			a.messages = append(a.messages, resp.Message)

			if err != nil {
				return "", fmt.Errorf("error while calling LLM %w", err)
			}
			return resp.Text(), nil
		},
	)
}

func (a *Agent) SendMessage(ctx context.Context, message *gai.Message) (string, error) {
	resp, err := a.flows.chat.Run(ctx, message)

	for _, msg := range a.messages {
		msgJson, _ := json.Marshal(msg)
		fmt.Println(string(msgJson))
	}
	if err != nil {
		return "", err
	}

	return resp, nil
	// resp, err := genkit.Generate(ctx, a.g,
	// 	ai.WithMessages(
	// 		ai.NewUserMessage(
	// 			ai.NewMediaPart("image/jpeg", "https://example.com/photo.jpg"),
	// 			ai.NewTextPart("Compose a poem about this image."),
	// 		),
	// 	),
	// )
}

// func (a *Agent) sendMessage(ctx context.Context, msg *Message, ch chan<- *AgentResponse){
// 	if msg == nil {
// 		ch <- &AgentResponse{Err: errors.New("arguments are required")}
// 	}

// 	if (len(msg.Attachments) == 0) && msg.Msg == "" {
// 		ch <- &AgentResponse{Err: errors.New("at least one of (img, msg) must be set")};
// 	}

// 	content := &genai.Content{
// 		Role:  "user",
// 		Parts: make([]*genai.Part, 0),
// 	}

// 	userInfoMsg, err := userInfoPrompt(msg.SenderInfo)

// 	if err != nil {
// 		ch <- &AgentResponse{Err: fmt.Errorf("error while formatting user info msg - %w", err)}
// 	}

// 	content.Parts = append(content.Parts, genai.NewPartFromText(userInfoMsg))

// 	if msg.Msg != "" {
// 		content.Parts = append(content.Parts, genai.NewPartFromText(msg.Msg))
// 	}

// 	for _, attachment := range msg.Attachments {
// 		if len(attachment.File) > 0 {
// 			content.Parts = append(content.Parts, genai.NewPartFromBytes(attachment.File, attachment.Mimetype))
// 			continue
// 		}

// 		if attachment.URI != "" {
// 			content.Parts = append(content.Parts, genai.NewPartFromURI(attachment.URI, attachment.Mimetype))
// 			continue
// 		}

// 		ch <- &AgentResponse{Err: fmt.Errorf("invalid attachment provided - %+v",  attachment)}

// 	}

// 	a.messages = append(a.messages, content)

// 	fmt.Printf("Sending request to LLM\n CONTENT: %s\n", LLMRequestToString(a.messages))

// 	response, err := a.generateContentRetry(ctx, a.config.Model, a.messages, a.config.generationConfig)
// 	ch <- &AgentResponse{GenerateReponse: response};

// 	if err != nil {
// 		ch <- &AgentResponse{Err: err};
// 	}

// 	a.countTokens(response)

// 	a.messages = append(a.messages, response.Candidates[0].Content)

// 	fmt.Printf("response from llm %+v\n", LLMResponseToString(response))

// 	toolCalls := response.FunctionCalls()

// 	for _, call := range response.FunctionCalls() {
// 		result := a.tp.ExecuteToolCall(call)
// 		a.messages = append(a.messages, &genai.Content{
// 			Role:  "user",
// 			Parts: []*genai.Part{{FunctionResponse: result}},
// 		})
// 	}

// 	if len(toolCalls) > 0 {
// 		fmt.Printf("Tool calls detected. sending request to LLM\n CONTENT: %v \n", LLMRequestToString(a.messages))
// 		response, err = a.generateContentRetry(ctx, a.config.Model, a.messages, a.config.generationConfig)
// 		if err != nil {
// 			ch <- &AgentResponse{Err: err}
// 		}
// 		ch <- &AgentResponse{GenerateReponse: response};
// 		a.countTokens(response)
// 		a.messages = append(a.messages, response.Candidates[0].Content)
// 		fmt.Printf("response from llm %+v\n", LLMResponseToString(response))
// 	}

// 	close(ch)
// }

// func (a *Agent) countTokens(resp *genai.GenerateContentResponse) {
// 	if resp == nil || resp.UsageMetadata == nil {
// 		return
// 	}

// 	a.usageStats.InputTokens += resp.UsageMetadata.PromptTokenCount + resp.UsageMetadata.ToolUsePromptTokenCount
// 	a.usageStats.OutputTokens += resp.UsageMetadata.CandidatesTokenCount
// 	a.usageStats.TotalTokens += resp.UsageMetadata.TotalTokenCount

// 	fmt.Printf("\nCurrent usage stats: %+v\n", a.usageStats)
// }

// func (a *Agent) generateContentRetry(
// 	ctx context.Context,
// 	model string,
// 	contents []*genai.Content,
// 	config *genai.GenerateContentConfig,
// ) (*genai.GenerateContentResponse, error) {
	// TODO make this configurable
// 	maxRetries := 5
// 	delay := 2 * time.Second // TODO exponential backoff?

// 	var lastErr error

// 	for retry := range maxRetries {
// 		time.Sleep(delay * time.Duration(retry))
// 		response, err := a.Client.Models.GenerateContent(ctx, model, contents, config)
// 		lastErr = err
// 		if err == nil {
// 			return response, nil
// 		}

// 		apiErr, ok := err.(genai.APIError)
// 		if !ok {
// 			return nil, err
// 		}

// 		if apiErr.Code != 500 {
// 			return nil, err
// 		}

// 		fmt.Printf("Error response received, retrying - %v\n", err)
// 	}

// 	return nil, lastErr
// }
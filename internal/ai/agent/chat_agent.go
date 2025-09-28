package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"rdmm404/voltr-finance/internal/ai/tool"
	"strings"

	gai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

type chatFlow = *core.Flow[*Message, string, *AgentUpdate]

type flows struct {
	chat chatFlow
}

type ChatAgent struct {
	g          *genkit.Genkit
	messages   []*gai.Message
	tp         *tool.ToolProvider
	usageStats UsageStats
	flows      *flows
}

func NewChatAgent(ctx context.Context, tp *tool.ToolProvider) (Agent[Message, AgentUpdate], error) {
	g := genkit.Init(
		ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}),
		genkit.WithDefaultModel("googleai/gemini-2.0-flash"),
	)

	tp.Init(g)

	a := &ChatAgent{
		g:  g,
		tp: tp,
	}

	// TODO: Find a better way to do this
	a.flows = &flows{
		chat: a.chatFlow(),
	}

	return a, nil
}

func (a *ChatAgent) chatFlow() chatFlow {
	return genkit.DefineStreamingFlow(a.g, "chat",
		func(ctx context.Context, msg *Message, callback core.StreamCallback[*AgentUpdate]) (string, error) {
			if msg == nil {
				return "", errors.New("message is required")
			}

			if (len(msg.Attachments) == 0) && msg.Msg == "" {
				return "", errors.New("at least one of (img, msg) must be set")
			}

			message := gai.NewUserMessage()

			userInfoMsg, err := userInfoPrompt(msg.SenderInfo)

			if err != nil {
				return "", fmt.Errorf("error while formatting user info msg - %w", err)
			}

			message.Content = append(message.Content, gai.NewTextPart(userInfoMsg))

			if msg.Msg != "" {
				message.Content = append(message.Content, gai.NewTextPart(msg.Msg))
			}

			for _, attachment := range msg.Attachments {
				if attachment.Mimetype == "" {
					return "", fmt.Errorf("attachment missing mimetype %+v", attachment)
				}

				// if len(attachment.File) > 0 {
				// 	message.Content = append(
				// 		message.Content,
				// 		gai.NewMediaPart(
				// 			attachment.Mimetype,
				// 			"data:" + attachment.Mimetype + ";base64," + base64.StdEncoding.EncodeToString(attachment.File),
				// 		),
				// 	)
				// }

				if attachment.URI != "" {
					message.Content = append(message.Content, gai.NewMediaPart(attachment.Mimetype, attachment.URI))
				} else {
					return "", fmt.Errorf("invalid attachment provided - %+v", attachment)
				}
			}

			a.messages = append(a.messages, message)

			resp, err := genkit.Generate(
				ctx,
				a.g,
				gai.WithTools(a.tp.GetAvailableTools()...),
				gai.WithSystem(systemPrompt(43)),
				gai.WithMessages(
					a.messages...,
				),
				gai.WithStreaming(func(ctx context.Context, chunk *gai.ModelResponseChunk) error {
					update := AgentUpdate{Text: chunk.Text()}

					for _, content := range chunk.Content {
						if content.ToolRequest != nil {
							update.ToolCall = &ToolCallUpdate{
								Args: content.ToolRequest.Input,
								Name: content.ToolRequest.Name,
							}
						}

						if content.ToolResponse != nil {
							update.ToolResponse = &ToolResponseUpdate{
								Response: content.ToolResponse.Output,
								Name:     content.ToolResponse.Name,
							}
						}
					}

					err := callback(ctx, &update)
					if err != nil {
						return fmt.Errorf("error in streaming callback - %w", err)
					}
					return nil
				}),
			)

			a.trackUsage(resp)

			if err != nil {
				return "", fmt.Errorf("error while calling LLM %w", err)
			}

			a.messages = append(a.messages, resp.Message)

			return resp.Text(), nil
		},
	)
}

func (a *ChatAgent) Run(ctx context.Context, input *Message, mode StreamingMode) (<-chan *AgentUpdate, error) {
	if !mode.Valid() {
		return nil, fmt.Errorf("invalid streaming mode received %v", mode)
	}

	ch := make(chan *AgentUpdate)

	go func() {
		defer close(ch)
		var mb strings.Builder

		a.flows.chat.Stream(ctx, input)(
			func(resp *core.StreamingFlowValue[string, *AgentUpdate], err error) bool {
				if err != nil {
					log.Printf("Error while streaming response %v\n", err)
					ch <- &AgentUpdate{Err: err}
					return false
				}

				// jsonContent, _ := json.MarshalIndent(resp.Stream, "", "  ")
				// log.Printf("*** BEGIN CHUNK ***\n%v\n*** END CHUNK ***\n", string(jsonContent))

				switch mode {
				case StreamingModeComplete:
					if resp.Done {
						ch <- &AgentUpdate{Text: resp.Output}
					} else if resp.Stream.ToolCall == nil {
						mb.WriteString(resp.Stream.Text)
					} else {
						resp.Stream.Text = mb.String()
						ch <- resp.Stream
					}
				case StreamingModeChunks:
					if !resp.Done {
						ch <- resp.Stream
					}
				default:
					ch <- &AgentUpdate{Err: fmt.Errorf("invalid streaming mode received %v", mode)}
					return false
				}
				return true
			},
		)
	}()

	return ch, nil
}

func (a *ChatAgent) trackUsage(resp *gai.ModelResponse) {
	if resp == nil || resp.Usage == nil {
		return
	}

	a.usageStats.InputTokens += resp.Usage.InputTokens
	a.usageStats.OutputTokens += resp.Usage.OutputTokens
	a.usageStats.TotalTokens += resp.Usage.TotalTokens

	fmt.Printf("\nCurrent usage stats: %+v\n", a.usageStats)
}

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

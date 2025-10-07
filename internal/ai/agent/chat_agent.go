package agent

import (
	"context"
	"encoding/json"
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

type ChatAgent = Agent[Message, AgentUpdate]

type chatAgent struct {
	g     *genkit.Genkit
	tp    *tool.ToolProvider
	sm    *SessionManager
	flows *flows

	usageStats UsageStats
}

func NewChatAgent(ctx context.Context, tp *tool.ToolProvider, sm *SessionManager) (ChatAgent, error) {
	g := genkit.Init(
		ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}),
		genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
	)

	tp.Init(g)

	a := &chatAgent{
		g:  g,
		tp: tp,
		sm: sm,
	}

	// TODO: Find a better way to do this
	a.flows = &flows{
		chat: a.chatFlow(),
	}

	return a, nil
}

func (a *chatAgent) chatFlow() chatFlow {
	return genkit.DefineStreamingFlow(a.g, "chat",
		func(ctx context.Context, msg *Message, callback core.StreamCallback[*AgentUpdate]) (string, error) {
			if msg == nil {
				return "", errors.New("message is required")
			}

			if (len(msg.Attachments) == 0) && msg.Msg == "" {
				return "", errors.New("at least one of (img, msg) must be set")
			}

			userID := msg.SenderInfo.User.ID

			session, err := a.sm.GetOrCreateSession(ctx, msg.SenderInfo.ChannelID, userID)

			if err != nil {
				return "", errors.Join(ErrMessagePersistance, err)
			}

			genkitMsg, err := msg.ToGenkit()

			if err != nil {
				return "", fmt.Errorf("invalid message provided %w", err)
			}

			if err = session.StoreMessage(ctx, genkitMsg, userID); err != nil {
				return "", errors.Join(ErrMessagePersistance, err)
			}

			msgHistory, err := session.GetMessageHistory(ctx)

			if err != nil {
				return "", errors.Join(ErrMessagePersistance, err)
			}

			resp, err := genkit.Generate(
				ctx,
				a.g,
				gai.WithTools(a.tp.GetAvailableTools()...),
				gai.WithReturnToolRequests(true),
				gai.WithSystem(systemPrompt(43)),
				gai.WithMessages(
					msgHistory...,
				),
				gai.WithStreaming(func(ctx context.Context, chunk *gai.ModelResponseChunk) error {
					if chunk.Text() == "" {
						return nil
					}
					if err := callback(ctx, &AgentUpdate{Text: chunk.Text()}); err != nil {
						return fmt.Errorf("error in streaming callback - %w", err)
					}
					return nil
				}),
			)

			if err != nil {
				return "", fmt.Errorf("error while calling LLM %w", err)
			}

			// TODO store text as single part with resp.Text() + add tool calls iterating
			if err = session.StoreMessage(ctx, gai.NewModelMessage(resp.Message.Content...), userID); err != nil {
				return "", errors.Join(ErrMessagePersistance, err)
			}

			if len(resp.ToolRequests()) == 0 {
				return resp.Text(), nil
			}

			var parts []*gai.Part
			for _, req := range resp.ToolRequests() {

				if err := callback(ctx, &AgentUpdate{ToolCall: &ToolCallUpdate{Name: req.Name, Args: req.Input}}); err != nil {
					return "", fmt.Errorf("error in streaming callback - %w", err)
				}

				tool := genkit.LookupTool(a.g, req.Name)
				if tool == nil {
					log.Fatalf("tool %q not found", req.Name)
				}

				output, err := tool.RunRaw(ctx, req.Input)
				if err != nil {
					log.Fatalf("tool %q execution failed: %v", tool.Name(), err)
				}

				// TODO figure out this update when streaming tool calls
				if err := callback(ctx, &AgentUpdate{ToolResponse: &ToolResponseUpdate{Name: req.Name, Response: output}}); err != nil {
					return "", fmt.Errorf("error in streaming callback - %w", err)
				}

				parts = append(parts,
					gai.NewToolResponsePart(&gai.ToolResponse{
						Name:   req.Name,
						Ref:    req.Ref,
						Output: output,
					}))
			}

			toolRespMsg := gai.NewMessage(gai.RoleTool, nil, parts...)

			if err = session.StoreMessage(ctx, toolRespMsg, userID); err != nil {
				return "", errors.Join(ErrMessagePersistance, err)
			}

			// TODO: track in db
			a.trackUsage(resp)

			resp, err = genkit.Generate(ctx, a.g,
				gai.WithMessages(append(resp.History(), toolRespMsg)...),
				gai.WithStreaming(func(ctx context.Context, chunk *gai.ModelResponseChunk) error {
					if chunk.Text() == "" {
						return nil
					}
					if err := callback(ctx, &AgentUpdate{Text: chunk.Text()}); err != nil {
						return fmt.Errorf("error in streaming callback - %w", err)
					}
					return nil
				}),
			)

			if err != nil {
				return "", fmt.Errorf("error while calling LLM %w", err)
			}

			if err := session.StoreMessage(ctx, gai.NewModelTextMessage(resp.Text()), userID); err != nil {
				return "", errors.Join(ErrMessagePersistance, err)
			}

			return resp.Text(), nil
		},
	)
}

func (a *chatAgent) Run(ctx context.Context, input *Message, mode StreamingMode) (<-chan *AgentUpdate, error) {
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

				if !resp.Done {
					jsonStream, _ := json.Marshal(resp.Stream)
					fmt.Println(string(jsonStream))
				} else {
					fmt.Println(resp.Output)
				}

				switch mode {
				case StreamingModeMessages:
					if resp.Done {
						ch <- &AgentUpdate{Text: resp.Output}
					} else if resp.Stream.ToolCall != nil {
						resp.Stream.Text = mb.String()
						ch <- resp.Stream
						mb.Reset()
					} else if resp.Stream.ToolResponse != nil {
						ch <- resp.Stream
					} else {
						mb.WriteString(resp.Stream.Text)
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

func (a *chatAgent) trackUsage(resp *gai.ModelResponse) {
	if resp == nil || resp.Usage == nil {
		return
	}

	a.usageStats.InputTokens += resp.Usage.InputTokens
	a.usageStats.OutputTokens += resp.Usage.OutputTokens
	a.usageStats.TotalTokens += resp.Usage.TotalTokens

	fmt.Printf("\nCurrent usage stats: %+v\n", a.usageStats)
}

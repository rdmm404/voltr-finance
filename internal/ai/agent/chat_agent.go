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

type ChatAgent = Agent[Message, AgentUpdate]

type chatAgent struct {
	g     *genkit.Genkit
	tp    *tool.ToolProvider
	sm    *SessionManager
	flows *flows
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

			if _, err = session.StoreMessage(ctx, genkitMsg, userID, nil); err != nil {
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

			a.trackUsage(resp)

			modelMsg := gai.NewModelTextMessage(resp.Message.Text())
			for _, part := range resp.Message.Content {
				if part.Kind == gai.PartToolRequest {
					modelMsg.Content = append(modelMsg.Content, part)
				}
			}

			msgID, err := session.StoreMessage(ctx, modelMsg, userID, nil)
			if err != nil {
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
					return "", fmt.Errorf("tool %q not found", req.Name)
				}

				output, err := tool.RunRaw(ctx, req.Input)
				// TODO instead of returnig error, pass to agent
				if err != nil {
					return "", fmt.Errorf("tool %q execution failed: %v", tool.Name(), err)
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

			if _, err = session.StoreMessage(ctx, toolRespMsg, userID, &msgID); err != nil {
				return "", errors.Join(ErrMessagePersistance, err)
			}

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

			if _, err := session.StoreMessage(ctx, gai.NewModelTextMessage(resp.Text()), userID, nil); err != nil {
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

// TODO: track in db
func (a *chatAgent) trackUsage(resp *gai.ModelResponse) {
	if resp == nil || resp.Usage == nil {
		fmt.Println("nil")
		return
	}

	fmt.Printf("\nCurrent usage stats: %+v\n", *resp.Usage)
}

package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"rdmm404/voltr-finance/internal/ai/tool"
	"rdmm404/voltr-finance/internal/config"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/utils"
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
	db *sqlc.Queries
}

func NewChatAgent(ctx context.Context, tp *tool.ToolProvider, sm *SessionManager, db *sqlc.Queries) (ChatAgent, error) {
	g := genkit.Init(
		ctx,
		genkit.WithPlugins(&googlegenai.VertexAI{Location: "global"}),
		genkit.WithDefaultModel("vertexai/gemini-2.5-flash"),
	)

	tp.Init(g)

	a := &chatAgent{
		g:  g,
		tp: tp,
		sm: sm,
		db: db,
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

			householdInfoMsg := noHouseholdPromptTemplate
			if household := msg.SenderInfo.Household; household != nil {
				users, err := a.db.GetHouseholdUsers(ctx, household.ID)
				if err != nil {
					slog.Error("ChatAgent: error while getting users for household", "error", err)
				}

				usersForPrompt := make([]userDataForPrompt, 0, len(users))
				for _, user := range users {
					usersForPrompt = append(usersForPrompt, userDataForPrompt{userId: int(user.ID), userName: user.Name})
				}
				householdInfoMsg, err = householdInfoPrompt(household.ID, household.Name, usersForPrompt)
			}

			resp, genErr := genkit.Generate(
				ctx,
				a.g,
				gai.WithTools(a.tp.GetAvailableTools()...),
				gai.WithSystem(systemPrompt(43)),
				gai.WithMessages(
					append(
						[]*gai.Message{gai.NewSystemTextMessage(householdInfoMsg)},
						msgHistory...,
					)...,
				),
				gai.WithStreaming(func(ctx context.Context, chunk *gai.ModelResponseChunk) error {
					slog.Debug("chunk received", "chunk", utils.JsonMarshalIgnore(chunk))

					if chunk == nil {
						slog.Warn("ChatAgent: nil chunk received from LLM")
						return nil
					}

					update := AgentUpdate{}
					for _, part := range chunk.Content {
						if part == nil {
							slog.Warn("ChatAgent: nil part received in chunk")
							continue
						}

						switch part.Kind {
						case gai.PartText:
							update.Text = part.Text
						case gai.PartToolRequest:
							update.ToolCall = &ToolCallUpdate{Name: part.ToolRequest.Name, Args: part.ToolRequest.Input}
						case gai.PartToolResponse:
							update.ToolResponse = &ToolResponseUpdate{Name: part.ToolResponse.Name, Response: part.ToolResponse.Output}
						}
					}

					if err := callback(ctx, &update); err != nil {
						return fmt.Errorf("error in streaming callback - %w", err)
					}
					return nil
				}),
				gai.WithMaxTurns(config.AGENT_MAX_TURNS),
			)

			a.trackUsage(resp)

			// TODO find a better way to store intermediate messages
			// right now on any error, resp will be nil so nothing will be saved
			if resp != nil {
				for i := len(msgHistory) + 1; i < len(resp.Request.Messages); i++ {
					msg := resp.Request.Messages[i]
					if msg.Role == "" {
						msg.Role = gai.RoleModel
					}
					switch msg.Role {
					case gai.RoleModel, gai.RoleTool:
						_, err = session.StoreMessage(ctx, msg, userID, nil)
						if err != nil {
							return "", errors.Join(ErrMessagePersistance, err)
						}
					default:
						slog.Warn("unexpected role received", "role", msg.Role)
					}
				}
			}

			if genErr != nil {
				var genkitErr *core.GenkitError
				if !errors.As(genErr, &genkitErr) {
					return "", fmt.Errorf("error while calling LLM: %w", genErr)
				}

				slog.Error("error while calling llm", "error", genkitErr)

				if genkitErr.Status == core.ABORTED {
					return "", fmt.Errorf("max iterations exceeded: %w", genErr)
				}

				return "", fmt.Errorf("error while calling LLM: %w", genErr)
			}

			// the model's message has an empty Role
			_, err = session.StoreMessage(ctx, gai.NewModelTextMessage(resp.Message.Text()), userID, nil)
			if err != nil {
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
		defer func() {
			if r := recover(); r != nil {
				slog.Error("ChatAgent: recovered from panic", "error", r)
				ch <- &AgentUpdate{Err: fmt.Errorf("panic in chat agent: %v", r)}
			}
		}()

		var mb strings.Builder

		a.flows.chat.Stream(ctx, input)(
			func(resp *core.StreamingFlowValue[string, *AgentUpdate], err error) bool {
				if err != nil {
					slog.Error("Error while streaming response", "error", err)
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
		slog.Debug("Usage tracking: nil response or usage data")
		return
	}

	slog.Info("Current usage stats", "usage", *resp.Usage)
}

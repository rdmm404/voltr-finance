package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	database "rdmm404/voltr-finance/internal/database/repository"

	gai "github.com/firebase/genkit/go/ai"
	"github.com/jackc/pgx/v5"
)

type SessionManager struct {
	db         *pgx.Conn
	repository *database.Queries
}

func NewSessionManager(db *pgx.Conn, repository *database.Queries) (*SessionManager, error) {
	if db == nil || repository == nil {
		return nil, errors.New("db and repository must be set")
	}

	return &SessionManager{
		db:         db,
		repository: repository,
	}, nil
}

func (ms *SessionManager) GetOrCreateSession(ctx context.Context, sourceId string, userId int32) (*Session, error) {
	if sourceId == "" {
		return nil, fmt.Errorf("source id was not provided")
	}

	if userId == 0 {
		return nil, fmt.Errorf("user id was not provided")
	}

	tx, err := ms.db.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("error creating db transaction %w", err)
	}

	rtx := ms.repository.WithTx(tx)
	defer tx.Rollback(ctx)

	session, err := rtx.GetCurrentSessionBySourceId(ctx, sourceId)

	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("error committing transaction: %w", err)
		}
		return &Session{db: ms.db, repository: ms.repository, SessionData: &session}, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("error while getting current session %w", err)
	}

	session, err = rtx.CreateLlmSession(
		ctx,
		database.CreateLlmSessionParams{UserID: userId, SourceID: sourceId},
	)

	if err != nil {
		return nil, fmt.Errorf("error creating session %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return &Session{db: ms.db, repository: ms.repository, SessionData: &session}, nil
}

type Session struct {
	db          *pgx.Conn
	repository  *database.Queries
	SessionData *database.LlmSession
}

func (s *Session) StoreMessage(ctx context.Context, msg *gai.Message, userId int32) error {
	if userId == 0 {
		return errors.New("userId is required")
	}

	jsonContent, err := json.Marshal(msg.Content)
	if err != nil {
		return fmt.Errorf("message contents are not valid json %w", err)
	}

	err = s.repository.CreateLlmMessage(ctx, database.CreateLlmMessageParams{
		SessionID: s.SessionData.ID,
		Role:      string(msg.Role),
		Contents:  jsonContent,
		UserID:    userId,
	})

	if err != nil {
		return fmt.Errorf("error creating message %w", err)
	}

	return nil
}

func (s *Session) GetMessageHistory(ctx context.Context) ([]*gai.Message, error) {
	dbMsgs, err := s.repository.ListLlmMessagesBySessionId(ctx, s.SessionData.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting messages %w", err)
	}

	var messages []*gai.Message
	var errs []error
	for _, msg := range dbMsgs {
		aiMsg, err := dbMessageToGenkit(&msg.LlmMessage, &msg.User, &msg.Household)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		messages = append(messages, aiMsg)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return messages, nil
}

func dbMessageToGenkit(msg *database.LlmMessage, user *database.User, household *database.Household) (*gai.Message, error) {
	content := []*gai.Part{}

	if err := json.Unmarshal(msg.Contents, &content); err != nil {
		return nil, fmt.Errorf("invalid contents %q: %w", msg.ID, err)
	}

	aiMsg := gai.NewMessage(gai.Role(msg.Role), map[string]any{}, content...)

	switch aiMsg.Role {
	case gai.RoleUser:
		var msgParts []*gai.Part

		for _, part := range aiMsg.Content {
			if part.Kind != gai.PartMedia {
				continue
			}
			msgParts = append(msgParts, part)
		}

		msgText, err := userMsgPrompt(int(msg.UserID), user.Name, int(household.ID), aiMsg.Text(), len(msgParts))
		if err != nil {
			return nil, fmt.Errorf("invalid database message %q: %w", msg.ID, err)
		}

		msgParts = append(msgParts, gai.NewTextPart(msgText))

		aiMsg = gai.NewUserMessage(msgParts...)
	case gai.RoleModel, gai.RoleTool:
	default:
		return nil, fmt.Errorf("invalid message role %s", aiMsg.Role)
	}

	return aiMsg, nil
}

package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	database "rdmm404/voltr-finance/internal/database/repository"

	gai "github.com/firebase/genkit/go/ai"
)

type MessageStorage struct {
	repository *database.Queries
}

func (ms *MessageStorage) BeginSession(ctx context.Context, userId int32) (database.LlmSession, error) {
	if userId == 0 {
		return database.LlmSession{}, fmt.Errorf("user id was not provided")
	}

	session, err := ms.repository.CreateLlmSession(ctx, userId)
	if err != nil {
		return database.LlmSession{}, fmt.Errorf("error creating session %w", err)
	}

	return session, nil
}

func (ms *MessageStorage) StoreMessage(ctx context.Context, message gai.Message, session database.LlmSession) error {
	jsonContent, err := json.Marshal(message.Content)
	if err != nil {
		return fmt.Errorf("message contents are not valid json %w", err)
	}

	if (session == database.LlmSession{}) {
		return errors.New("no session has been created")
	}

	err = ms.repository.CreateLlmMessage(ctx, database.CreateLlmMessageParams{
		SessionID: session.ID,
		Role:      string(message.Role),
		Contents:  jsonContent,
	})

	if err == nil {
		return fmt.Errorf("error creating message %w", err)
	}

	return nil
}

func (ms *MessageStorage) GetMessages(ctx context.Context, userId int32) ([]*gai.Message, error) {
	dbMsgs, err := ms.repository.ListLlmMessagesByUserId(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("error getting messages %w", err)
	}

	var messages []*gai.Message
	var errs []error
	for _, msg := range dbMsgs {
		content := []*gai.Part{}
		if err := json.Unmarshal(msg.Contents, &content); err != nil {
			errs = append(errs, fmt.Errorf("invalid contents %q: %w", msg.ID, err))
			continue
		}

		aiMsg := gai.NewMessage(gai.Role(msg.Role), map[string]any{}, content...)
		messages = append(messages, aiMsg)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return messages, nil
}

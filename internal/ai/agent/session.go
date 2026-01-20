package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"rdmm404/voltr-finance/internal/config"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	gai "github.com/firebase/genkit/go/ai"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionManager struct {
	db         *pgxpool.Pool
	repository *sqlc.Queries
	bucket     *storage.BucketHandle
}

func NewSessionManager(db *pgxpool.Pool, repository *sqlc.Queries, storageClient *storage.Client) (*SessionManager, error) {
	if db == nil || repository == nil {
		return nil, errors.New("db and repository must be set")
	}

	if config.AGENT_MEDIA_BUCKET_NAME == "" {
		return nil, errors.New("invalid env: AGENT_MEDIA_BUCKET_NAME must be set")
	}

	return &SessionManager{
		db:         db,
		repository: repository,
		bucket:     storageClient.Bucket(config.AGENT_MEDIA_BUCKET_NAME),
	}, nil
}

func (sm *SessionManager) GetOrCreateSession(ctx context.Context, sourceId string, userId int64) (*Session, error) {
	if sourceId == "" {
		return nil, fmt.Errorf("source id was not provided")
	}

	if userId == 0 {
		return nil, fmt.Errorf("user id was not provided")
	}

	tx, err := sm.db.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("error creating db transaction %w", err)
	}

	rtx := sm.repository.WithTx(tx)
	defer tx.Rollback(ctx)

	session, err := rtx.GetActiveSessionBySourceId(ctx, sourceId)

	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("error committing transaction: %w", err)
		}
		return &Session{db: sm.db, repository: sm.repository, SessionData: &session, bucket: sm.bucket}, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("error while getting current session %w", err)
	}

	session, err = rtx.CreateLlmSession(
		ctx,
		sqlc.CreateLlmSessionParams{UserID: userId, SourceID: sourceId},
	)

	if err != nil {
		return nil, fmt.Errorf("error creating session %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	fmt.Printf("bucket %+v\n", sm.bucket)

	return &Session{db: sm.db, repository: sm.repository, SessionData: &session, bucket: sm.bucket}, nil
}

type Session struct {
	db          *pgxpool.Pool
	repository  *sqlc.Queries
	SessionData *sqlc.LlmSession
	bucket      *storage.BucketHandle
}

func (s *Session) StoreMessage(ctx context.Context, msg *gai.Message, userId int64, parentId *int64) (int64, error) {
	if userId == 0 {
		return 0, errors.New("userId is required")
	}

	if msg == nil {
		return 0, errors.New("nil msg received")
	}

	jsonContent, err := json.Marshal(msg.Content)
	if err != nil {
		return 0, fmt.Errorf("message contents are not valid json %w", err)
	}

	createdMsgId, err := s.repository.CreateLlmMessage(ctx, sqlc.CreateLlmMessageParams{
		SessionID: s.SessionData.ID,
		Role:      string(msg.Role),
		Contents:  jsonContent,
		UserID:    userId,
		ParentID:  parentId,
	})

	if err != nil {
		return createdMsgId, fmt.Errorf("error creating message: %w", err)
	}

	go s.postProcessMessageContent(ctx, createdMsgId, userId, msg.Content)

	return createdMsgId, nil
}

func (s *Session) postProcessMessageContent(ctx context.Context, msgId int64, userId int64, content []*gai.Part) error {
	for _, part := range content {
		if part == nil {
			slog.Warn("StoreMessage: nil part received", "content", content)
			continue
		}

		// TODO add retries
		var err error
		switch part.Kind {
		case gai.PartMedia:
			err = s.postProcessFilePart(ctx, userId, part)
		}

		if err != nil {
			slog.Error("error received while post-processing part", "part", part, "messageId", msgId, "error", err)
			return err
		}
	}

	jsonContent, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("message contents are not valid json %w", err)
	}

	s.repository.UpdateMessageContents(ctx, sqlc.UpdateMessageContentsParams{ID: msgId, Contents: jsonContent})
	return nil
}

// TODO: right now, this would download an image twice. once here and once in the main path
// It doesn't really hurt performance as this happens in the background, but it's kinda weird
func (s *Session) postProcessFilePart(ctx context.Context, userId int64, part *gai.Part) error {
	splitContentType := strings.Split(part.ContentType, "/")
	fileExtension := ""
	if len(splitContentType) > 1 {
		fileExtension = "." + splitContentType[len(splitContentType)-1]
	}

	slog.Info("session", "session", s.bucket)
	objectName := fmt.Sprintf("%s/%v/%v-%v%s", config.ENVIRONMENT, s.SessionData.ID, userId, time.Now().Unix(), fileExtension)
	obj := s.bucket.Object(objectName)

	w := obj.NewWriter(ctx)
	var r io.Reader

	if strings.HasPrefix(part.Text, "data:") {
		// TODO: Handle raw base64
	} else {
		resp, err := http.Get(part.Text)
		if err != nil {
			return fmt.Errorf("failed to get file from %q: %w", part.Text, err)
		}
		defer resp.Body.Close()
		r = resp.Body
	}

	bytes, err := io.Copy(w, r)

	if err != nil {
		return fmt.Errorf("failed to upload file to %q: %w", objectName, err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer to %q: %w", objectName, err)
	}

	slog.Info("uploaded file to bucket", "bytes", bytes, "object", objectName)

	part.Text = fmt.Sprintf("gs://%s/%s", s.bucket.BucketName(), objectName)
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
		aiMsg, err := dbMessageToGenkit(&msg.LlmMessage, &msg.User)
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

func dbMessageToGenkit(msg *sqlc.LlmMessage, user *sqlc.User) (*gai.Message, error) {
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

		msgText, err := userMsgPrompt(userDataForPrompt{userId: int(msg.UserID), userName: user.Name}, aiMsg.Text(), len(msgParts))
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

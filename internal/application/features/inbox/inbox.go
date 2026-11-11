package inbox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("invalid cursor")

type Cursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func EncodeCursor(c Cursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func DecodeCursor(value string) (*Cursor, error) {
	if value == "" {
		return nil, nil
	}
	if len(value) > 512 {
		return nil, ErrInvalidCursor
	}
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var c Cursor
	if json.Unmarshal(b, &c) != nil || c.CreatedAt.IsZero() {
		return nil, ErrInvalidCursor
	}
	if _, err := uuid.Parse(c.ID); err != nil {
		return nil, ErrInvalidCursor
	}
	return &c, nil
}

type Page struct {
	Items      []entities.Notification `json:"items"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

type Repository interface {
	List(context.Context, string, int, *Cursor) (Page, error)
	UnreadCount(context.Context, string) (int64, error)
	MarkRead(context.Context, string, string) (bool, error)
	Create(context.Context, *entities.Notification) error
	CreateBatch(context.Context, []entities.Notification) error
}

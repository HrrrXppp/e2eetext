package news

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Cursor is the keyset position of the last item on a page.
type Cursor struct {
	PublishedAt time.Time
	ID          int64
}

var ErrInvalidCursor = errors.New("invalid cursor")

// Encode returns an opaque URL-safe token.
func (c Cursor) Encode() string {
	raw := fmt.Sprintf("%d:%d", c.PublishedAt.UnixMicro(), c.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses a token produced by Encode.
func DecodeCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	micro, id, ok := strings.Cut(string(raw), ":")
	if !ok {
		return Cursor{}, ErrInvalidCursor
	}
	m, err := strconv.ParseInt(micro, 10, 64)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	i, err := strconv.ParseInt(id, 10, 64)
	if err != nil || i < 0 {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{PublishedAt: time.UnixMicro(m).UTC(), ID: i}, nil
}

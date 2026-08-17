package invitations

import (
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

// Same scheme as events.encodeCursor / events.decodeCursor: opaque
// base64.RawURLEncoding of the uuidv7 key. Do not invent a third style.
func encodeCursor(id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(id))
}

func decodeCursor(raw string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(b) == 0 {
		return "", fmt.Errorf("invalid cursor")
	}
	s := string(b)
	if _, err := uuid.Parse(s); err != nil {
		return "", fmt.Errorf("invalid cursor")
	}
	return s, nil
}

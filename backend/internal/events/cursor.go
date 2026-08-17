package events

import (
	"encoding/base64"
	"fmt"
)

func encodeCursor(id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(id))
}

func decodeCursor(raw string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(b) == 0 {
		return "", fmt.Errorf("invalid cursor")
	}
	return string(b), nil
}

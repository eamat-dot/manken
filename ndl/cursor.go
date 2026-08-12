package ndl

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

type cursorPayload struct {
	Version     int    `json:"v"`
	StartRecord int    `json:"start_record"`
	Limit       int    `json:"limit"`
	SearchHash  string `json:"search_sha256"`
}

// encodeCursor は、次ページ位置と検索条件を不透明Cursorへ符号化する
func encodeCursor(startRecord int, query string, limit int) (string, error) {
	if startRecord < 1 || startRecord > 500 {
		return "", errors.New("cursor start record is outside NDL Search range")
	}
	data, err := json.Marshal(cursorPayload{1, startRecord, limit, hashQuery(query)})
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// decodeCursor は、不透明Cursorを検証して次の開始位置を返す
func decodeCursor(encoded, query string, limit int) (int, error) {
	if encoded == "" {
		return 1, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return 0, errors.New("cursor is not valid base64url")
	}
	var payload cursorPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0, errors.New("cursor does not contain valid JSON")
	}
	if payload.Version != 1 || payload.StartRecord < 1 || payload.StartRecord > 500 {
		return 0, errors.New("cursor contains an invalid start record")
	}
	if payload.Limit != limit {
		return 0, errors.New("cursor does not match the requested limit")
	}
	if payload.SearchHash != hashQuery(query) {
		return 0, errors.New("cursor does not match the requested search conditions")
	}
	return payload.StartRecord, nil
}

// hashQuery は、Cursor照合用のCQLハッシュを返す
func hashQuery(query string) string {
	sum := sha256.Sum256([]byte(query))
	return fmt.Sprintf("%x", sum)
}

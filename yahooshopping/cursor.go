package yahooshopping

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const cursorVersion = 1

type cursorPayload struct {
	Version    int    `json:"v"`
	Start      int    `json:"start"`
	Limit      int    `json:"limit"`
	SearchHash string `json:"search_sha256"`
}

// encodeCursor は、検索条件に結び付いた次ページカーソルを符号化する
func encodeCursor(start int, key string, limit int) (string, error) {
	if !requestStartIsValid(start, limit) {
		return "", errors.New("cursor start and limit exceed API pagination limit")
	}
	data, err := json.Marshal(cursorPayload{cursorVersion, start, limit, hashSearchKey(key)})
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// decodeCursor は、検索条件と一致する次ページ開始位置を復号する
func decodeCursor(encoded, key string, limit int) (int, error) {
	if encoded == "" {
		return 1, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return 0, errors.New("cursor is not valid base64url")
	}
	var payload cursorPayload
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return 0, errors.New("cursor does not contain valid JSON")
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return 0, errors.New("cursor contains trailing data")
	}
	if payload.Version != cursorVersion || payload.Start < 2 || !requestStartIsValid(payload.Start, payload.Limit) {
		return 0, errors.New("cursor contains an invalid start")
	}
	if payload.Limit != limit {
		return 0, errors.New("cursor does not match the requested limit")
	}
	if payload.SearchHash != hashSearchKey(key) {
		return 0, errors.New("cursor does not match the requested search conditions")
	}
	return payload.Start, nil
}

// requestStartIsValid は、Yahoo!ショッピングのページング上限内かを判定する
func requestStartIsValid(start, limit int) bool {
	return start >= 1 && limit >= 1 && limit <= 100 && start+limit <= 1000
}

// hashSearchKey は、検索条件をカーソル用のSHA-256値へ変換する
func hashSearchKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// ensureJSONEnd は、JSON値の後ろに別の値がないことを確認する
func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("another JSON value follows")
	}
	return err
}

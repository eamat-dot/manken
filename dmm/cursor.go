package dmm

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

const (
	maxOffset     = 50000
	cursorVersion = 1
)

// cursorPayload は、次ページの取得に必要な非公開情報を保持する
type cursorPayload struct {
	Version    int    `json:"v"`
	API        string `json:"api"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
	SearchHash string `json:"search_sha256"`
}

// encodeCursor は、次ページの情報を不透明なカーソルへ変換する
func encodeCursor(offset int, api, key string, limit int) (string, error) {
	if offset < 2 || offset > maxOffset {
		return "", errors.New("cursor offset exceeds API pagination limit")
	}
	body, err := json.Marshal(cursorPayload{Version: cursorVersion, API: api, Offset: offset, Limit: limit, SearchHash: hashKey(key)})
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

// decodeCursor は、カーソルを検証して次のDMM offsetへ戻す
func decodeCursor(encoded, api, key string, limit int) (int, error) {
	if encoded == "" {
		return 1, nil
	}
	body, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return 0, errors.New("cursor is not valid base64url")
	}
	var payload cursorPayload
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return 0, errors.New("cursor does not contain valid JSON")
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return 0, errors.New("cursor contains trailing data")
	}
	if payload.Version != cursorVersion || payload.API != api || payload.Offset < 2 || payload.Offset > maxOffset || payload.Limit != limit || payload.SearchHash != hashKey(key) {
		return 0, errors.New("cursor does not match the requested search conditions")
	}
	return payload.Offset, nil
}

// ensureJSONEnd は、JSON値の後に別の値が存在しないことを確認する
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

// hashKey は、検索条件をCursorに保存するハッシュ値へ変換する
func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

package madb

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const cursorVersion = 1

// cursorPayload は、次ページの取得に必要な非公開情報を保持する
type cursorPayload struct {
	Version   int    `json:"v"`
	After     string `json:"after"`
	Limit     int    `json:"limit"`
	TitleHash string `json:"title_sha256"`
}

// encodeCursor は、次ページの情報を不透明なカーソルへ変換する
func encodeCursor(after, title string, limit int) (string, error) {
	payload := cursorPayload{
		Version:   cursorVersion,
		After:     after,
		Limit:     limit,
		TitleHash: hashTitle(title),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// decodeCursor は、カーソルを検証して次ページの情報へ戻す
func decodeCursor(encoded, title string, limit int) (cursorPayload, error) {
	if encoded == "" {
		return cursorPayload{}, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return cursorPayload{}, errors.New("cursor is not valid base64url")
	}

	var payload cursorPayload
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return cursorPayload{}, errors.New("cursor does not contain valid JSON")
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return cursorPayload{}, errors.New("cursor contains trailing data")
	}
	if payload.Version != cursorVersion {
		return cursorPayload{}, errors.New("cursor version is not supported")
	}
	if !isValidResourceURI(payload.After) {
		return cursorPayload{}, errors.New("cursor contains an invalid resource URI")
	}
	if payload.Limit != limit {
		return cursorPayload{}, errors.New("cursor does not match the requested limit")
	}
	if payload.TitleHash != hashTitle(title) {
		return cursorPayload{}, errors.New("cursor does not match the requested title")
	}
	return payload, nil
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

// hashTitle は、整形後の検索タイトルをSHA-256の16進文字列へ変換する
func hashTitle(title string) string {
	sum := sha256.Sum256([]byte(title))
	return hex.EncodeToString(sum[:])
}

// isValidResourceURI は、MADBのマンガ単行本リソースURIか検証する
func isValidResourceURI(resourceURI string) bool {
	const prefix = "https://mediaarts-db.artmuseums.go.jp/id/M"
	if !strings.HasPrefix(resourceURI, prefix) {
		return false
	}
	id := strings.TrimPrefix(resourceURI, prefix)
	if id == "" {
		return false
	}
	for _, character := range id {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

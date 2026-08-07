package googlebooks

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

// cursorPayload は、次ページの取得に必要な非公開情報を保持する
type cursorPayload struct {
	Version    int    `json:"v"`
	StartIndex int    `json:"start_index"`
	Limit      int    `json:"limit"`
	SearchHash string `json:"search_sha256"`
}

// encodeCursor は、次ページの情報を不透明なカーソルへ変換する
func encodeCursor(startIndex int, query string, limit int) (string, error) {
	searchHash := hashSearchQuery(query)
	payload := cursorPayload{Version: cursorVersion, StartIndex: startIndex, Limit: limit, SearchHash: searchHash}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// decodeCursor は、カーソルを検証して次ページの開始位置へ戻す
func decodeCursor(encoded string, query string, limit int) (int, error) {
	if encoded == "" {
		return 0, nil
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
	if payload.Version != cursorVersion {
		return 0, errors.New("cursor version is not supported")
	}
	if payload.StartIndex < 1 {
		return 0, errors.New("cursor contains an invalid start index")
	}
	if payload.Limit != limit {
		return 0, errors.New("cursor does not match the requested limit")
	}
	if payload.SearchHash != hashSearchQuery(query) {
		return 0, errors.New("cursor does not match the requested search conditions")
	}
	return payload.StartIndex, nil
}

// hashSearchQuery は、正規化済み検索式をSHA-256の16進文字列へ変換する
func hashSearchQuery(query string) string {
	sum := sha256.Sum256([]byte(query))
	return hex.EncodeToString(sum[:])
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

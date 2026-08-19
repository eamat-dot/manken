package dmm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/eamat-dot/manken/internal/titlemeta"
)

type response struct {
	Result result `json:"result"`
}
type result struct {
	Status        int    `json:"status"`
	ResultCount   int    `json:"result_count"`
	TotalCount    int    `json:"total_count"`
	FirstPosition int    `json:"first_position"`
	Items         []item `json:"items"`
}
type item struct {
	ContentID    string    `json:"content_id"`
	URL          string    `json:"URL"`
	AffiliateURL string    `json:"affiliateURL"`
	Title        string    `json:"title"`
	ImageURL     imageURLs `json:"imageURL"`
	ItemInfo     struct {
		Series      []named `json:"series"`
		Author      []named `json:"author"`
		Manufacture []named `json:"manufacture"`
		Genre       []named `json:"genre"`
	} `json:"iteminfo"`
}

// seriesSearchItemFromItem は、DMM keyword応答の代表商品をシリーズ候補と補助情報へ変換する
func seriesSearchItemFromItem(v item) (SeriesSearchItem, error) {
	series, err := bookSeriesFromItem(v)
	if err != nil {
		return SeriesSearchItem{}, err
	}
	return SeriesSearchItem{
		BookSeries: series,
		Title:      v.Title,
		Authors:    authorsFromItem(v),
		Publishers: publishersFromItem(v),
		Subjects:   subjectsFromItem(v),
		Images:     imagesFromItem(v),
		Sources:    sourcesFromItem(v),
	}, nil
}

type named struct {
	ID   json.Number `json:"id"`
	Name string      `json:"name"`
}

type imageURLs struct {
	List  string `json:"list"`
	Small string `json:"small"`
	Large string `json:"large"`
}

// decodeResponse はDMM ItemListレスポンスを復号して整合性を検証する
func decodeResponse(b []byte) (response, error) {
	var r response
	d := json.NewDecoder(bytes.NewReader(b))
	if err := d.Decode(&r); err != nil {
		return r, fmt.Errorf("decode dmm response: %w", err)
	}
	if err := ensureJSONEnd(d); err != nil {
		return r, errors.New("response contains trailing data")
	}
	if r.Result.Status < 0 || r.Result.ResultCount < 0 || r.Result.TotalCount < 0 || r.Result.FirstPosition < 0 {
		return r, errors.New("response pagination values must not be negative")
	}
	if r.Result.ResultCount != len(r.Result.Items) {
		return r, errors.New("response result_count does not match items")
	}
	return r, nil
}

// sanitizeRaw はRaw responseからClientに設定した認証情報を除去する
func sanitizeRaw(b []byte, apiID, affiliateID string) ([]byte, error) {
	var v map[string]any
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	if err := decoder.Decode(&v); err != nil {
		return nil, fmt.Errorf("decode raw response: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, errors.New("raw response contains trailing data")
	}
	sanitizeRawValue(v, []string{apiID, affiliateID})
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, fmt.Errorf("encode sanitized raw response: %w", err)
	}
	if buffer.Len() > 0 && buffer.Bytes()[buffer.Len()-1] == '\n' {
		buffer.Truncate(buffer.Len() - 1)
	}
	return buffer.Bytes(), nil
}

// sanitizeRawValue はJSON値を再帰的に走査して認証情報を秘匿する
func sanitizeRawValue(v any, secrets []string) {
	switch value := v.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "api_id" || key == "affiliate_id" {
				value[key] = "[REDACTED]"
				continue
			}
			if text, ok := child.(string); ok {
				value[key] = sanitizeRawString(text, secrets)
				continue
			}
			sanitizeRawValue(child, secrets)
		}
	case []any:
		for index, child := range value {
			if text, ok := child.(string); ok {
				value[index] = sanitizeRawString(text, secrets)
				continue
			}
			sanitizeRawValue(child, secrets)
		}
	}
}

// sanitizeRawString は認証情報と認証情報を含むURL queryの値を秘匿する
func sanitizeRawString(value string, secrets []string) string {
	if parsed, err := url.Parse(value); err == nil && isAbsoluteHTTPURL(parsed) && parsed.RawQuery != "" {
		query := parsed.Query()
		for key, values := range query {
			for index, value := range values {
				if isSecret(value, secrets) {
					query[key][index] = "[REDACTED]"
				}
			}
		}
		parsed.RawQuery = query.Encode()
		value = parsed.String()
	}
	for _, secret := range secrets {
		value = redactSecretTokens(value, secret)
		value = redactSecretTokens(value, url.QueryEscape(secret))
	}
	return value
}

// isAbsoluteHTTPURL は、HTTP(S)の絶対URLか判定する
func isAbsoluteHTTPURL(value *url.URL) bool {
	return value.Host != "" && (value.Scheme == "http" || value.Scheme == "https")
}

// isSecret は値が設定済みの認証情報と一致するかを返す
func isSecret(value string, secrets []string) bool {
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		if value == secret {
			return true
		}
	}
	return false
}

// redactSecretTokens は文字列内の区切られた認証情報だけを秘匿する
func redactSecretTokens(value, secret string) string {
	if secret == "" {
		return value
	}
	var out strings.Builder
	for start := 0; ; {
		index := strings.Index(value[start:], secret)
		if index < 0 {
			out.WriteString(value[start:])
			return out.String()
		}
		index += start
		end := index + len(secret)
		if isSecretBoundary(value, index-1) && isSecretBoundary(value, end) {
			out.WriteString(value[start:index])
			out.WriteString("[REDACTED]")
			start = end
			continue
		}
		out.WriteString(value[start : index+1])
		start = index + 1
	}
}

// isSecretBoundary は認証情報トークンの端か非英数字の境界かを返す
func isSecretBoundary(value string, index int) bool {
	if index < 0 || index >= len(value) {
		return true
	}
	if (value[index] >= 'a' && value[index] <= 'z') || (value[index] >= 'A' && value[index] <= 'Z') || (value[index] >= '0' && value[index] <= '9') || value[index] == '_' || value[index] == '-' {
		return false
	}
	return true
}

// bookSeriesFromItem は、DMM itemが明示する唯一のシリーズを共通モデルへ変換する
func bookSeriesFromItem(v item) (BookSeries, error) {
	if len(v.ItemInfo.Series) != 1 {
		return BookSeries{}, errors.New("item must contain exactly one DMM series")
	}
	series := v.ItemInfo.Series[0]
	id, name := strings.TrimSpace(series.ID.String()), strings.TrimSpace(series.Name)
	if id == "" || name == "" {
		return BookSeries{}, errors.New("dmm series ID and name must not be empty")
	}
	return BookSeries{Name: name, ID: id, Source: SourceDMM}, nil
}

// convertItem はDMMの個別商品情報を共通書籍モデルへ限定的に変換する
func convertItem(v item, series BookSeries) Book {
	b := Book{Title: v.Title, Medium: PublicationMediumDigital, Sources: sourcesFromItem(v)}
	b.Authors = authorsFromItem(v)
	b.Publishers = publishersFromItem(v)
	for _, author := range b.Authors {
		b.Contributors = append(b.Contributors, Contributor{Name: author})
	}
	b.BookSeries = []BookSeries{series}
	b.Subjects = subjectsFromItem(v)
	b.CoverURL = coverURLFromItem(v)
	applyTitleMetadata(&b)
	return b
}

// applyTitleMetadata は、タイトルから安全に抽出できた付加情報だけを未設定のBook項目へ補う
func applyTitleMetadata(book *Book) {
	titlemeta.Apply(book)
}

// authorsFromItem は、DMM itemの著者名を返却順で変換する
func authorsFromItem(v item) []string {
	authors := make([]string, 0, len(v.ItemInfo.Author))
	for _, author := range v.ItemInfo.Author {
		if name := strings.TrimSpace(author.Name); name != "" {
			authors = append(authors, name)
		}
	}
	return authors
}

// publishersFromItem は、DMM電子コミックのmanufacturer名を出版社名として返却順で変換する
func publishersFromItem(v item) []string {
	publishers := make([]string, 0, len(v.ItemInfo.Manufacture))
	for _, manufacture := range v.ItemInfo.Manufacture {
		if name := strings.TrimSpace(manufacture.Name); name != "" {
			publishers = append(publishers, name)
		}
	}
	return publishers
}

// subjectsFromItem は、DMM itemのgenreをDMM固有の主題として変換する
func subjectsFromItem(v item) []Subject {
	subjects := make([]Subject, 0, len(v.ItemInfo.Genre))
	for _, g := range v.ItemInfo.Genre {
		name := strings.TrimSpace(g.Name)
		if name != "" {
			subjects = append(subjects, Subject{Scheme: "dmm", Code: strings.TrimSpace(g.ID.String()), Name: name})
		}
	}
	return subjects
}

// imagesFromItem は、DMM itemの画像をlarge、list、smallの順で1件だけ変換する
func imagesFromItem(v item) []Image {
	if coverURL := coverURLFromItem(v); coverURL != "" {
		return []Image{{URL: coverURL}}
	}
	return []Image{}
}

// coverURLFromItem は、DMM itemの最大サイズの有効な画像URLを返す
func coverURLFromItem(v item) string {
	for _, imageURL := range []string{v.ImageURL.Large, v.ImageURL.List, v.ImageURL.Small} {
		if u := validURL(imageURL); u != "" {
			return u
		}
	}
	return ""
}

// sourcesFromItem は、DMM itemの商品参照先を取得元情報として変換する
func sourcesFromItem(v item) []BookSource {
	return []BookSource{{Source: SourceDMM, ID: v.ContentID, URL: validURL(v.URL), AffiliateURL: validURL(v.AffiliateURL)}}
}

// validURL は有効なHTTP(S)の絶対URLだけを返す
func validURL(s string) string {
	p, e := url.Parse(s)
	if e != nil || (p.Scheme != "http" && p.Scheme != "https") || p.Host == "" {
		return ""
	}
	return s
}

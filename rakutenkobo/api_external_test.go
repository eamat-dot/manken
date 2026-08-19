package rakutenkobo_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/eamat-dot/manken/rakutenkobo"
)

// requireRawSearchSignature は、SearchBooksWithRawResponseの公開シグネチャをコンパイル時に確認する
func requireRawSearchSignature(
	func(context.Context, rakutenkobo.SearchRequest) (rakutenkobo.SearchBooksResult, []byte, error),
) {
}

// TestPublicAPI は、外部パッケージから楽天Koboの公開APIを利用できることを確認する
func TestPublicAPI(t *testing.T) {
	client, err := rakutenkobo.NewClient(
		&http.Client{},
		rakutenkobo.WithApplicationID("application-id"),
		rakutenkobo.WithAccessKey("access-key"),
		rakutenkobo.WithComicGenre(rakutenkobo.ComicGenreBL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() client = nil")
	}
	if rakutenkobo.SourceRakutenKobo != "rakutenkobo" {
		t.Fatalf("SourceRakutenKobo = %q", rakutenkobo.SourceRakutenKobo)
	}
	if rakutenkobo.PublicationMediumDigital != "digital" {
		t.Fatalf("public constant = %q", rakutenkobo.PublicationMediumDigital)
	}
	_ = rakutenkobo.Book{Title: "Title", CurrentPrice: &rakutenkobo.Price{Amount: 0, Source: rakutenkobo.SourceRakutenKobo}}
	requireRawSearchSignature(client.SearchBooksWithRawResponse)
}

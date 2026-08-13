package yahooshopping

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

type searchResponse struct {
	TotalResultsAvailable int    `json:"totalResultsAvailable"`
	TotalResultsReturned  int    `json:"totalResultsReturned"`
	FirstResultsPosition  int    `json:"firstResultsPosition"`
	Hits                  []item `json:"hits"`
}
type item struct {
	Code                  string          `json:"code"`
	Name                  string          `json:"name"`
	Description           string          `json:"description"`
	JanCode               string          `json:"janCode"`
	URL                   string          `json:"url"`
	Price                 *int64          `json:"price"`
	PriceLabel            *priceLabel     `json:"priceLabel"`
	Image                 itemImage       `json:"image"`
	ExImage               exImage         `json:"exImage"`
	Seller                seller          `json:"seller"`
	GenreCategory         genre           `json:"genreCategory"`
	ParentGenreCategories []genre         `json:"parentGenreCategories"`
	ReleaseDate           json.RawMessage `json:"releaseDate"`
}
type itemImage struct {
	Small  string `json:"small"`
	Medium string `json:"medium"`
}
type exImage struct {
	URL    string `json:"url"`
	Width  *int   `json:"width"`
	Height *int   `json:"height"`
}
type priceLabel struct {
	Taxable *bool `json:"taxable"`
}
type seller struct {
	SellerID string `json:"sellerId"`
}
type genre struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// decodeResponse は、Yahoo!ショッピングの応答本文を検証しながら復号する
func decodeResponse(body []byte) (searchResponse, error) {
	var response searchResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&response); err != nil {
		return searchResponse{}, fmt.Errorf("decode Yahoo Shopping response: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return searchResponse{}, err
	}
	if response.TotalResultsAvailable < 0 || response.TotalResultsReturned < 0 || response.FirstResultsPosition < 0 {
		return searchResponse{}, errors.New("response pagination values must not be negative")
	}
	if response.TotalResultsReturned != len(response.Hits) {
		return searchResponse{}, errors.New("response totalResultsReturned does not match hits")
	}
	if len(response.Hits) > 0 && response.FirstResultsPosition < 1 {
		return searchResponse{}, errors.New("response firstResultsPosition is missing")
	}
	return response, nil
}

package madb

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const fullTextEndpoint = "https://vpc-mediaarts-db-qaymrmtqbprlhmqq33a2ncf4ke.ap-northeast-1.es.amazonaws.com"

// buildSearchQuery は、検索条件からMADB向けSPARQLクエリを生成する
func buildSearchQuery(title string, limit int, after string) string {
	fullTextQuery := `"` + escapeFullText(title) + `"`
	cursorFilter := ""
	if after != "" {
		cursorFilter = fmt.Sprintf(
			"\n      FILTER (STR(?resource) > \"%s\")",
			escapeSPARQLString(after),
		)
	}

	return fmt.Sprintf(`PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX schema: <https://schema.org/>
PREFIX dcterms: <http://purl.org/dc/terms/>
PREFIX ma: <https://mediaarts-db.artmuseums.go.jp/data/property#>
PREFIX class: <https://mediaarts-db.artmuseums.go.jp/data/class#>
PREFIX neptune-fts: <http://aws.amazon.com/neptune/vocab/v01/services/fts#>

SELECT ?resource ?id ?title ?titleKana ?subtitle ?seriesName
       ?seriesResource ?relatedSeriesName ?seriesID
       ?volumeNumber ?version ?creator ?agentName ?publisher ?brand
       ?isbn ?publishedDate ?pageCount ?size
WHERE {
  {
    SELECT DISTINCT ?resource
    WHERE {
      SERVICE neptune-fts:search {
        neptune-fts:config neptune-fts:endpoint "%s" .
        neptune-fts:config neptune-fts:field schema:name .
        neptune-fts:config neptune-fts:queryType "simple_query_string" .
        neptune-fts:config neptune-fts:query "%s" .
        neptune-fts:config neptune-fts:return ?resource .
      }
      ?resource rdf:type class:MangaBook .%s
    }
    ORDER BY ?resource
    LIMIT %d
  }
  OPTIONAL { ?resource schema:identifier ?id . }
  OPTIONAL { ?resource schema:name ?title . FILTER (LANG(?title) = "") }
  OPTIONAL { ?resource schema:name ?titleKana . FILTER (LANG(?titleKana) = "ja-hrkt") }
  OPTIONAL { ?resource schema:alternativeHeadline ?subtitle . FILTER (LANG(?subtitle) = "") }
  OPTIONAL { ?resource ma:seriesName ?seriesName . FILTER (LANG(?seriesName) = "") }
  OPTIONAL {
    ?resource schema:isPartOf ?seriesResource .
    ?seriesResource rdf:type class:MangaBookSeries .
    OPTIONAL {
      ?seriesResource schema:name ?relatedSeriesName .
      FILTER (LANG(?relatedSeriesName) = "")
    }
    OPTIONAL { ?seriesResource schema:identifier ?seriesID . }
  }
  OPTIONAL { ?resource schema:volumeNumber ?volumeNumber . }
  OPTIONAL { ?resource schema:version ?version . FILTER (LANG(?version) = "") }
  OPTIONAL { ?resource schema:creator ?creator . FILTER (LANG(?creator) = "") }
  OPTIONAL {
    ?resource dcterms:creator ?agent .
    ?agent rdfs:label ?agentName .
  }
  OPTIONAL { ?resource schema:publisher ?publisher . }
  OPTIONAL { ?resource schema:brand ?brand . FILTER (LANG(?brand) = "") }
  OPTIONAL { ?resource schema:isbn ?isbn . }
  OPTIONAL { ?resource schema:datePublished ?publishedDate . }
  OPTIONAL { ?resource schema:numberOfPages ?pageCount . }
  OPTIONAL { ?resource schema:size ?size . }
}
ORDER BY ?resource`,
		fullTextEndpoint,
		escapeSPARQLString(fullTextQuery),
		cursorFilter,
		limit+1,
	)
}

// escapeFullText は、全文検索の引用句内で特別な文字をエスケープする
func escapeFullText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

// escapeSPARQLString は、値をSPARQLの二重引用符文字列へ安全に埋め込める形にする
func escapeSPARQLString(value string) string {
	var builder strings.Builder
	for len(value) > 0 {
		character, size := utf8.DecodeRuneInString(value)
		value = value[size:]
		switch character {
		case '\\':
			builder.WriteString(`\\`)
		case '"':
			builder.WriteString(`\"`)
		case '\t':
			builder.WriteString(`\t`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\b':
			builder.WriteString(`\b`)
		case '\f':
			builder.WriteString(`\f`)
		default:
			if character < 0x20 || character == 0x7f {
				builder.WriteString(`\u`)
				hexValue := strings.ToUpper(strconv.FormatInt(int64(character), 16))
				builder.WriteString(strings.Repeat("0", 4-len(hexValue)))
				builder.WriteString(hexValue)
				continue
			}
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

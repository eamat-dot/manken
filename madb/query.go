package madb

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const fullTextEndpoint = "https://vpc-mediaarts-db-qaymrmtqbprlhmqq33a2ncf4ke.ap-northeast-1.es.amazonaws.com"

// searchConditions は、検証と正規化を終えたMADB検索条件を保持する
type searchConditions struct {
	Title        string `json:"title"`
	Author       string `json:"author"`
	FreeText     string `json:"free_text"`
	ExcludedText string `json:"excluded_text"`
}

// buildISBNLookupQuery は、複数のISBN候補を1回で参照するSPARQLクエリを生成する
func buildISBNLookupQuery(candidates []string) string {
	values := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		values = append(values, `"`+escapeSPARQLString(candidate)+`"`)
	}

	return fmt.Sprintf(`PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX schema: <https://schema.org/>
PREFIX dcterms: <http://purl.org/dc/terms/>
PREFIX ma: <https://mediaarts-db.artmuseums.go.jp/data/property#>
PREFIX class: <https://mediaarts-db.artmuseums.go.jp/data/class#>

SELECT ?resource ?matchedISBN ?id ?title ?titleKana ?subtitle ?seriesName
       ?seriesResource ?relatedSeriesName ?seriesID
       ?volumeNumber ?version ?creator ?agentName ?publisher ?brand
       ?isbn ?publishedDate ?pageCount ?size
WHERE {
  {
    SELECT DISTINCT ?resource ?matchedISBN
    WHERE {
      VALUES ?matchedISBN { %s }
      ?resource schema:isbn ?matchedISBN .
      ?resource rdf:type class:MangaBook .
    }
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
ORDER BY ?resource ?matchedISBN`, strings.Join(values, " "))
}

// buildSearchQuery は、検索条件からMADB向けSPARQLクエリを生成する
func buildSearchQuery(conditions searchConditions, limit int, after string) string {
	conditionPatterns := buildSearchConditionPatterns(conditions)
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
%s
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
		conditionPatterns,
		cursorFilter,
		limit+1,
	)
}

// buildSearchConditionPatterns は、指定された正条件をAND結合するSPARQLパターンを生成する
func buildSearchConditionPatterns(conditions searchConditions) string {
	patterns := make([]string, 0, 4)
	if conditions.Title != "" {
		patterns = append(patterns, fmt.Sprintf(`      SERVICE neptune-fts:search {
		neptune-fts:config neptune-fts:endpoint "%s" .
		neptune-fts:config neptune-fts:field schema:name .
		neptune-fts:config neptune-fts:queryType "query_string" .
        neptune-fts:config neptune-fts:query "%s" .
        neptune-fts:config neptune-fts:return ?resource .
      }`, fullTextEndpoint, escapeSPARQLString(buildFullTextQuery(conditions.Title))))
	}
	if conditions.Author != "" {
		fullTextQuery := escapeSPARQLString(buildFullTextQuery(conditions.Author))
		patterns = append(patterns, fmt.Sprintf(`      {
        SERVICE neptune-fts:search {
          neptune-fts:config neptune-fts:endpoint "%s" .
          neptune-fts:config neptune-fts:field schema:creator .
          neptune-fts:config neptune-fts:queryType "query_string" .
          neptune-fts:config neptune-fts:query "%s" .
          neptune-fts:config neptune-fts:return ?resource .
        }
      }
      UNION
      {
        SERVICE neptune-fts:search {
          neptune-fts:config neptune-fts:endpoint "%s" .
          neptune-fts:config neptune-fts:field rdfs:label .
          neptune-fts:config neptune-fts:queryType "query_string" .
          neptune-fts:config neptune-fts:query "%s" .
          neptune-fts:config neptune-fts:return ?searchAgent .
        }
        ?resource dcterms:creator ?searchAgent .
      }`, fullTextEndpoint, fullTextQuery, fullTextEndpoint, fullTextQuery))
	}
	if conditions.FreeText != "" {
		patterns = append(patterns, fmt.Sprintf(`      SERVICE neptune-fts:search {
        neptune-fts:config neptune-fts:endpoint "%s" .
%s
        neptune-fts:config neptune-fts:queryType "query_string" .
        neptune-fts:config neptune-fts:query "%s" .
        neptune-fts:config neptune-fts:return ?resource .
      }`,
			fullTextEndpoint,
			buildFreeTextFieldPatterns("        "),
			escapeSPARQLString(buildFullTextQueryWithExclusions(
				conditions.FreeText,
				conditions.ExcludedText,
			)),
		))
	}
	if conditions.ExcludedText != "" && conditions.FreeText == "" {
		patterns = append(patterns, fmt.Sprintf(`      MINUS {
        SERVICE neptune-fts:search {
          neptune-fts:config neptune-fts:endpoint "%s" .
%s
          neptune-fts:config neptune-fts:queryType "query_string" .
          neptune-fts:config neptune-fts:query "%s" .
          neptune-fts:config neptune-fts:return ?resource .
        }
      }`,
			fullTextEndpoint,
			buildFreeTextFieldPatterns("          "),
			escapeSPARQLString(buildExcludedFullTextQuery(conditions.ExcludedText)),
		))
	}
	return strings.Join(patterns, "\n")
}

// buildFreeTextFieldPatterns は、フリーワード対象フィールドの設定行を生成する
func buildFreeTextFieldPatterns(indent string) string {
	fields := freeTextSearchFields()
	patterns := make([]string, 0, len(fields))
	for _, field := range fields {
		patterns = append(
			patterns,
			indent+"neptune-fts:config neptune-fts:field "+field+" .",
		)
	}
	return strings.Join(patterns, "\n")
}

// freeTextSearchFields は、フリーワード検索の対象フィールドを返す
func freeTextSearchFields() []string {
	return []string{
		"schema:name",
		"schema:alternativeHeadline",
		"ma:seriesName",
		"schema:volumeNumber",
		"schema:creator",
		"schema:publisher",
		"schema:brand",
		"schema:version",
		"schema:isbn",
		"schema:description",
		"schema:keywords",
		"schema:size",
	}
}

// buildFullTextQuery は、各検索語を引用してAND結合する
func buildFullTextQuery(value string) string {
	return strings.Join(quoteFullTextTerms(value), " AND ")
}

// buildFullTextQueryWithExclusions は、正の検索語へ引用済み除外語をAND NOTで追加する
func buildFullTextQueryWithExclusions(value string, excluded string) string {
	query := buildFullTextQuery(value)
	for _, term := range quoteFullTextTerms(excluded) {
		query += " AND NOT " + term
	}
	return query
}

// buildExcludedFullTextQuery は、どれか1語に一致する除外用のOR式を生成する
func buildExcludedFullTextQuery(value string) string {
	return strings.Join(quoteFullTextTerms(value), " OR ")
}

// quoteFullTextTerms は、Unicode空白で分けた各語を安全な引用句へ変換する
func quoteFullTextTerms(value string) []string {
	terms := strings.Fields(value)
	quotedTerms := make([]string, 0, len(terms))
	for _, term := range terms {
		quotedTerms = append(quotedTerms, `"`+escapeFullText(term)+`"`)
	}
	return quotedTerms
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

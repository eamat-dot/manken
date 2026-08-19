package ndl

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/eamat-dot/manken/internal/authorrole"
	internalisbn "github.com/eamat-dot/manken/internal/isbn"
	"github.com/eamat-dot/manken/internal/titlemeta"
)

const (
	nsDC      = "http://purl.org/dc/elements/1.1/"
	nsDCTerms = "http://purl.org/dc/terms/"
	nsDCNDL   = "http://ndl.go.jp/dcndl/terms/"
	nsRDF     = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	nsFOAF    = "http://xmlns.com/foaf/0.1/"
	nsRDFS    = "http://www.w3.org/2000/01/rdf-schema#"

	ndlbibIDDatatype = "http://ndl.go.jp/dcndl/terms/NDLBibID"
	isbnDatatype     = "http://ndl.go.jp/dcndl/terms/ISBN"
)

var (
	pageCountPattern = regexp.MustCompile(`^([0-9]+)\s*p(?:\s|$)`)
	sizePattern      = regexp.MustCompile(`(?:^|;)\s*([0-9]+\s*cm)\s*(?:\+|$)`)
	pricePattern     = regexp.MustCompile(`^([0-9]+)\s*円\z`)
)

// xmlNode は、名前空間を保持してXML要素を扱う内部表現である
type xmlNode struct {
	Name     xml.Name
	Attr     []xml.Attr
	Text     string
	Children []*xmlNode
}

// sruResponse は、SRU本文から取り出した書誌とページング状態を保持する
type sruResponse struct {
	records  []*xmlNode
	next     int
	notFound bool
}

// diagnosticError は、成功HTTPで返ったNDLサーチDiagnosticsを表す
type diagnosticError struct{ message string }

// Error は、Diagnosticsの要約を返す
func (err diagnosticError) Error() string { return "SRU diagnostic: " + err.message }

// decodeResponse は、SRU XMLからDC-NDL書誌とDiagnosticsを解析する
func decodeResponse(body []byte) (sruResponse, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	root, err := decodeNode(decoder)
	if err != nil {
		return sruResponse{}, fmt.Errorf("decode SRU response: %w", err)
	}
	if root == nil {
		return sruResponse{}, errors.New("response is empty")
	}
	diagnostics := descendantsLocal(root, "diagnostic")
	if len(diagnostics) > 0 {
		messages := make([]string, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			if message := diagnosticMessage(diagnostic); message != "" {
				messages = append(messages, message)
			}
		}
		if len(messages) == 1 && strings.EqualFold(strings.TrimSpace(messages[0]), "record does not exist") {
			return sruResponse{notFound: true}, nil
		}
		if len(messages) == 0 {
			return sruResponse{}, diagnosticError{message: "unknown diagnostic"}
		}
		return sruResponse{}, diagnosticError{message: strings.Join(messages, "; ")}
	}
	result := sruResponse{}
	for _, recordData := range descendantsLocal(root, "recordData") {
		result.records = append(result.records, descendants(recordData, nsDCNDL, "BibResource")...)
	}
	nexts := descendantsLocal(root, "nextRecordPosition")
	if len(nexts) > 0 {
		next := nexts[0]
		value, err := strconv.Atoi(strings.TrimSpace(nodeText(next)))
		if err != nil || value < 1 {
			return sruResponse{}, errors.New("invalid nextRecordPosition")
		}
		result.next = value
	}
	return result, nil
}

// diagnosticMessage は、SRU Diagnosticのmessage要素だけを返す
func diagnosticMessage(diagnostic *xmlNode) string {
	for _, child := range diagnostic.Children {
		if child.Name.Local == "message" {
			return strings.TrimSpace(nodeText(child))
		}
	}
	return ""
}

// decodeNode は、XML文書の最初の要素を内部ツリーへ読み込む
func decodeNode(decoder *xml.Decoder) (*xmlNode, error) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		if start, ok := token.(xml.StartElement); ok {
			return readNode(decoder, start)
		}
	}
}

// readNode は、開始要素以下を再帰的に内部ツリーへ読み込む
func readNode(decoder *xml.Decoder, start xml.StartElement) (*xmlNode, error) {
	node := &xmlNode{Name: start.Name, Attr: start.Attr}
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			child, err := readNode(decoder, value)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, child)
		case xml.CharData:
			node.Text += string(value)
		case xml.EndElement:
			if value.Name == start.Name {
				return node, nil
			}
		}
	}
}

// descendants は、指定名前空間とローカル名に一致する子孫要素を返す
func descendants(node *xmlNode, space, local string) []*xmlNode {
	result := []*xmlNode{}
	var walk func(*xmlNode)
	walk = func(current *xmlNode) {
		if current.Name.Space == space && current.Name.Local == local {
			result = append(result, current)
		}
		for _, child := range current.Children {
			walk(child)
		}
	}
	walk(node)
	return result
}

// descendantsLocal は、SRUラッパーのローカル名に一致する子孫要素を返す
func descendantsLocal(node *xmlNode, local string) []*xmlNode {
	result := []*xmlNode{}
	var walk func(*xmlNode)
	walk = func(current *xmlNode) {
		if current.Name.Local == local {
			result = append(result, current)
		}
		for _, child := range current.Children {
			walk(child)
		}
	}
	walk(node)
	return result
}

// children は、指定名前空間とローカル名に一致する直接の子要素を返す
func children(node *xmlNode, space, local string) []*xmlNode {
	result := []*xmlNode{}
	for _, child := range node.Children {
		if child.Name.Space == space && child.Name.Local == local {
			result = append(result, child)
		}
	}
	return result
}

// firstChild は、指定名前空間とローカル名に一致する最初の直接の子要素を返す
func firstChild(node *xmlNode, space, local string) *xmlNode {
	values := children(node, space, local)
	if len(values) == 0 {
		return nil
	}
	return values[0]
}

// nodeText は、要素とその子孫が持つテキストを連結して返す
func nodeText(node *xmlNode) string {
	values := []string{}
	var walk func(*xmlNode)
	walk = func(current *xmlNode) {
		values = append(values, current.Text)
		for _, child := range current.Children {
			walk(child)
		}
	}
	walk(node)
	return strings.TrimSpace(strings.Join(values, ""))
}

// attribute は、指定名前空間とローカル名に一致する属性値を返す
func attribute(node *xmlNode, space, local string) string {
	for _, attr := range node.Attr {
		if attr.Name.Space == space && attr.Name.Local == local {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

// descriptionValue は、rdf:Description内のrdf:valueを返す
func descriptionValue(node *xmlNode) string {
	for _, description := range children(node, nsRDF, "Description") {
		if value := firstChild(description, nsRDF, "value"); value != nil {
			return nodeText(value)
		}
	}
	return ""
}

// convertRecord は、DC-NDL v3のBibResourceを共通Bookへ変換する
func convertRecord(record *xmlNode) (Book, error) {
	id := ndlBibID(record)
	if id == "" {
		return Book{}, errors.New("BibResource is missing NDLBibID")
	}
	book := Book{Sources: []BookSource{{Source: SourceNDL, ID: id, URL: "https://ndlsearch.ndl.go.jp/books/R100000002-I" + id}}}
	if title := structuredTitle(record); title != "" {
		book.Title = title
	} else if title := firstChild(record, nsDCTerms, "title"); title != nil {
		book.Title = nodeText(title)
	}
	book.TitleReading = structuredTitleReading(record)
	if volume := structuredValue(record, nsDCNDL, "volume"); volume != "" {
		book.Volume.Label = volume
		if isASCIIInteger(volume) {
			if number, err := strconv.Atoi(volume); err == nil {
				book.Volume.Number = &number
			}
		}
	}
	book.PublicationSeries = publicationSeries(record)
	book.Editions = directTexts(record, nsDCNDL, "edition")
	book.Authors, book.Contributors = creators(record)
	book.Publishers = agents(record, nsDCTerms, "publisher", true)
	book.ISBN10, book.ISBN13 = identifiers(record)
	date := firstDirectText(record, nsDCTerms, "date")
	if date == "" {
		date = firstDirectText(record, nsDCTerms, "issued")
	}
	if date != "" {
		book.PublishedDate = date
	}
	book.Languages = directTexts(record, nsDCTerms, "language")
	book.Subjects = subjects(record)
	if extent := firstDirectText(record, nsDCTerms, "extent"); extent != "" {
		book.PageCount = pageCount(extent)
		book.Size = size(extent)
	}
	book.ListPrice = listPrice(record)
	if materialTypeIsBook(record) {
		book.Medium = PublicationMediumPrint
	}
	applyTitleMetadata(&book)
	return book, nil
}

// applyTitleMetadata は、タイトルから安全に抽出できた付加情報だけを未設定のBook項目へ補う
func applyTitleMetadata(book *Book) {
	titlemeta.Apply(book)
}

// creators は、表示用dc:creatorを優先して著者と寄与者へ変換し、利用できない場合は典拠形へfallbackする
func creators(record *xmlNode) ([]string, []Contributor) {
	creators := make([]Contributor, 0)
	indexes := map[string]int{}
	authors := make([]string, 0)
	authorNames := map[string]struct{}{}
	for _, node := range children(record, nsDC, "creator") {
		name, roles, ok := parseCreatorLiteral(nodeText(node))
		if !ok {
			continue
		}
		creators = mergeCreator(creators, indexes, name, roles)
		if authorrole.Contains(roles) {
			if _, exists := authorNames[name]; !exists {
				authorNames[name] = struct{}{}
				authors = append(authors, name)
			}
		}
	}
	if len(creators) != 0 {
		return authors, creators
	}
	return agents(record, nsDCTerms, "creator", false), nil
}

// parseCreatorLiteral は、末尾の既知roleを持つdc:creatorリテラルから表示名と役割を取り出す
func parseCreatorLiteral(value string) (string, []string, bool) {
	value = strings.TrimSpace(value)
	fields := strings.Fields(value)
	if len(fields) < 2 {
		return "", nil, false
	}
	trailingExpression := fields[len(fields)-1]
	roleExpression := trailingExpression
	if strings.HasPrefix(roleExpression, "[") || strings.HasSuffix(roleExpression, "]") {
		if !strings.HasPrefix(roleExpression, "[") || !strings.HasSuffix(roleExpression, "]") {
			return "", nil, false
		}
		roleExpression = strings.TrimSuffix(strings.TrimPrefix(roleExpression, "["), "]")
		if roleExpression == "" || strings.ContainsAny(roleExpression, "[]") {
			return "", nil, false
		}
	}
	roles, ok := mapCreatorRoles(roleExpression)
	if !ok {
		return "", nil, false
	}
	name := strings.TrimSpace(strings.TrimSuffix(value, trailingExpression))
	if name == "" {
		return "", nil, false
	}
	return name, roles, true
}

// mapCreatorRoles は、中黒で連結された既知roleだけを共通役割へ変換する
func mapCreatorRoles(value string) ([]string, bool) {
	parts := strings.Split(value, "・")
	roles := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		role, ok := mapCreatorRole(part)
		if !ok {
			return nil, false
		}
		if _, exists := seen[role]; !exists {
			seen[role] = struct{}{}
			roles = append(roles, role)
		}
	}
	return roles, len(roles) != 0
}

// mapCreatorRole は、NDLで確認した表示用の役割語を共通役割へ変換する
func mapCreatorRole(value string) (string, bool) {
	switch value {
	case "著", "著者", "作", "共著", "編著":
		return "著者", true
	case "原作", "原案":
		return "原作", true
	case "脚本", "シナリオ", "構成", "文":
		return "脚本", true
	case "作画", "画", "絵", "漫画":
		return "作画", true
	case "キャラクター原作", "キャラクター原案":
		return "キャラクター原案", true
	case "キャラクターデザイン":
		return "キャラクターデザイン", true
	case "監修", "キャラクター監修":
		return "監修", true
	case "編", "編集":
		return "編集", true
	case "訳":
		return "翻訳", true
	case "解説":
		return "解説", true
	case "装丁", "装幀", "デザイン":
		return "デザイン", true
	default:
		return "", false
	}
}

// mergeCreator は、同じ表示名の寄与者を応答順のまままとめ、未追加の役割を加える
func mergeCreator(creators []Contributor, indexes map[string]int, name string, roles []string) []Contributor {
	index, exists := indexes[name]
	if !exists {
		indexes[name] = len(creators)
		return append(creators, Contributor{Name: name, Roles: roles})
	}
	seen := make(map[string]struct{}, len(creators[index].Roles))
	for _, role := range creators[index].Roles {
		seen[role] = struct{}{}
	}
	for _, role := range roles {
		if _, exists := seen[role]; !exists {
			seen[role] = struct{}{}
			creators[index].Roles = append(creators[index].Roles, role)
		}
	}
	return creators
}

// ndlBibID は、datatypeで区別されたdcterms:identifierからNDLBibIDを返す
func ndlBibID(record *xmlNode) string {
	for _, identifier := range children(record, nsDCTerms, "identifier") {
		if attribute(identifier, nsRDF, "datatype") == ndlbibIDDatatype {
			return nodeText(identifier)
		}
	}
	return ""
}

// structuredTitle は、dc:titleの構造化された主タイトルを返す
func structuredTitle(record *xmlNode) string {
	for _, title := range children(record, nsDC, "title") {
		if value := descriptionValue(title); value != "" {
			return value
		}
	}
	return ""
}

// structuredTitleReading は、dc:title内のdcndl:transcriptionを返す
func structuredTitleReading(record *xmlNode) string {
	for _, title := range children(record, nsDC, "title") {
		for _, description := range children(title, nsRDF, "Description") {
			if transcription := firstChild(description, nsDCNDL, "transcription"); transcription != nil {
				return nodeText(transcription)
			}
		}
	}
	return ""
}

// structuredValue は、rdf:Description内のrdf:valueを持つ指定要素の値を返す
func structuredValue(record *xmlNode, space, local string) string {
	for _, node := range children(record, space, local) {
		if value := descriptionValue(node); value != "" {
			return value
		}
	}
	return ""
}

// directTexts は、指定要素の空でない直接値を応答順に返す
func directTexts(record *xmlNode, space, local string) []string {
	values := []string{}
	for _, node := range children(record, space, local) {
		if value := nodeText(node); value != "" {
			values = append(values, value)
		}
	}
	return unique(values)
}

// firstDirectText は、指定要素の最初の空でない直接値を返す
func firstDirectText(record *xmlNode, space, local string) string {
	for _, node := range children(record, space, local) {
		if value := nodeText(node); value != "" {
			return value
		}
	}
	return ""
}

// isASCIIInteger は、値がASCII数字だけで構成されるか判定する
func isASCIIInteger(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

// publicationSeries は、DC-NDLの構造化シリーズ名を共通モデルへ変換する
func publicationSeries(record *xmlNode) []string {
	result := []string{}
	for _, node := range children(record, nsDCNDL, "seriesTitle") {
		for _, description := range children(node, nsRDF, "Description") {
			nameNode := firstChild(description, nsRDF, "value")
			if nameNode == nil || nodeText(nameNode) == "" {
				continue
			}
			result = append(result, nodeText(nameNode))
		}
	}
	return unique(result)
}

// agents は、構造化Agentの名前を返し、出版者では明示的な非出版関係を除外する
func agents(record *xmlNode, space, local string, publisherOnly bool) []string {
	result := []string{}
	for _, relation := range children(record, space, local) {
		for _, agent := range relation.Children {
			if (agent.Name.Space != nsFOAF || agent.Name.Local != "Agent") && (agent.Name.Space != nsRDF || agent.Name.Local != "Description") {
				continue
			}
			if publisherOnly && !isPublisherRelation(agent) {
				continue
			}
			if name := firstChild(agent, nsFOAF, "name"); name != nil && nodeText(name) != "" {
				result = append(result, nodeText(name))
			}
		}
	}
	return unique(result)
}

// isPublisherRelation は、明示された関係が出版社を表すか、関係なしであるか判定する
func isPublisherRelation(agent *xmlNode) bool {
	values := append(children(agent, nsDCTerms, "description"), children(agent, nsDCNDL, "role")...)
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		role := strings.TrimSpace(nodeText(value))
		if role == "出版" || role == "出版社" || role == "発行" {
			return true
		}
	}
	return false
}

// identifiers は、typed dcterms:identifierから取得元にあるISBNを種類別に変換する
func identifiers(record *xmlNode) ([]string, []string) {
	isbn10 := []string{}
	isbn13 := []string{}
	for _, identifier := range children(record, nsDCTerms, "identifier") {
		if attribute(identifier, nsRDF, "datatype") != isbnDatatype {
			continue
		}
		value := strings.ReplaceAll(strings.TrimSpace(nodeText(identifier)), "-", "")
		switch {
		case internalisbn.IsValidISBN10(value):
			isbn10 = append(isbn10, value)
		case internalisbn.IsValidISBN13(value):
			isbn13 = append(isbn13, value)
		}
	}
	return unique(isbn10), unique(isbn13)
}

// subjects は、DC-NDLの分類・典拠URIを安全に共通Subjectへ変換する
func subjects(record *xmlNode) []Subject {
	result := []Subject{}
	for _, node := range append(children(record, nsDCTerms, "subject"), children(record, nsDCNDL, "genre")...) {
		uri := attribute(node, nsRDF, "resource")
		name := ""
		if description := firstChild(node, nsRDF, "Description"); description != nil {
			uri = firstNonEmpty(uri, attribute(description, nsRDF, "about"))
			if value := firstChild(description, nsRDF, "value"); value != nil {
				name = nodeText(value)
			}
			if name == "" {
				if label := firstChild(description, nsRDFS, "label"); label != nil {
					name = nodeText(label)
				}
			}
		}
		scheme, code := subjectSchemeAndCode(uri)
		if scheme == "" {
			continue
		}
		result = append(result, Subject{Scheme: scheme, Code: code, Name: name})
	}
	return uniqueSubjects(result)
}

// subjectSchemeAndCode は、NDL典拠または分類URIからSchemeとCodeを抽出する
func subjectSchemeAndCode(uri string) (string, string) {
	value := strings.TrimSpace(uri)
	lower := strings.ToLower(value)
	for _, candidate := range []struct{ marker, scheme string }{{"/ndc8/", "NDC"}, {"/ndc9/", "NDC"}, {"/ndc10/", "NDC"}, {"/ndc/", "NDC"}, {"/ndlc/", "NDLC"}, {"/ndlsh/", "NDLSH"}, {"/ndlgft/", "NDLGFT"}} {
		if position := strings.LastIndex(lower, candidate.marker); position >= 0 {
			code := strings.Trim(value[position+len(candidate.marker):], "/#")
			if code != "" {
				return candidate.scheme, code
			}
		}
	}
	return "", ""
}

// materialTypeIsBook は、resource URIまたはラベルから図書媒体か判定する
func materialTypeIsBook(record *xmlNode) bool {
	for _, materialType := range children(record, nsDCNDL, "materialType") {
		if strings.EqualFold(attribute(materialType, nsRDF, "resource"), "http://ndl.go.jp/ndltype/Book") {
			return true
		}
		if strings.TrimSpace(nodeText(materialType)) == "図書" {
			return true
		}
		for _, description := range children(materialType, nsRDF, "Description") {
			if strings.EqualFold(attribute(description, nsRDF, "about"), "http://ndl.go.jp/ndltype/Book") {
				return true
			}
			if label := firstChild(description, nsRDFS, "label"); label != nil && strings.TrimSpace(nodeText(label)) == "図書" {
				return true
			}
		}
	}
	return false
}

// pageCount は、先頭が単純なNNNp形式のextentからページ数を返す
func pageCount(value string) *int {
	matches := pageCountPattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return nil
	}
	count, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil
	}
	return &count
}

// size は、extentのセミコロン以降にある単純なcm表記を取得元表記のまま返す
func size(value string) string {
	matches := sizePattern.FindStringSubmatch(value)
	if matches == nil {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

// listPrice は、同じ単純なdcndl:priceだけを書誌に記録された販売価格として返す
func listPrice(record *xmlNode) *Price {
	var amount *int64
	for _, node := range children(record, nsDCNDL, "price") {
		matches := pricePattern.FindStringSubmatch(strings.TrimSpace(nodeText(node)))
		if matches == nil {
			continue
		}
		parsed, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			continue
		}
		if amount == nil {
			amount = &parsed
			continue
		}
		if *amount != parsed {
			return nil
		}
	}
	if amount == nil {
		return nil
	}
	return &Price{Amount: *amount, Currency: "JPY", Source: SourceNDL}
}

// unique は、空文字列を除き最初の出現順で値を重複排除する
func unique(values []string) []string {
	result := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		if _, ok := seen[value]; value != "" && !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}

// uniqueSubjects は、Scheme、Code、Nameが同じ主題を最初の出現順で重複排除する
func uniqueSubjects(values []Subject) []Subject {
	result := []Subject{}
	seen := map[Subject]struct{}{}
	for _, value := range values {
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}

// firstNonEmpty は、最初の空でない値を返す
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

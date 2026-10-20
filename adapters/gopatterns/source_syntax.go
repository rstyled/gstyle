package gopatterns

import (
	"encoding/binary"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

// The syntax projection stores node tags, source-relative positions and child
// indexes. It owns no compiler objects. Names and literals reference source
// bytes when possible. Each rewrite gets a fresh, disposable rendering tree.
const sourceSyntaxProfile = "gstyle.go.syntax/1"

var errSourceSyntax = errors.New("invalid source syntax evidence")

var syntaxTypes = []reflect.Type{
	reflect.TypeOf(ast.File{}), reflect.TypeOf(ast.Comment{}), reflect.TypeOf(ast.CommentGroup{}),
	reflect.TypeOf(ast.Field{}), reflect.TypeOf(ast.FieldList{}), reflect.TypeOf(ast.BadExpr{}),
	reflect.TypeOf(ast.Ident{}), reflect.TypeOf(ast.Ellipsis{}), reflect.TypeOf(ast.BasicLit{}),
	reflect.TypeOf(ast.FuncLit{}), reflect.TypeOf(ast.CompositeLit{}), reflect.TypeOf(ast.ParenExpr{}),
	reflect.TypeOf(ast.SelectorExpr{}), reflect.TypeOf(ast.IndexExpr{}), reflect.TypeOf(ast.IndexListExpr{}),
	reflect.TypeOf(ast.SliceExpr{}), reflect.TypeOf(ast.TypeAssertExpr{}), reflect.TypeOf(ast.CallExpr{}),
	reflect.TypeOf(ast.StarExpr{}), reflect.TypeOf(ast.UnaryExpr{}), reflect.TypeOf(ast.BinaryExpr{}),
	reflect.TypeOf(ast.KeyValueExpr{}), reflect.TypeOf(ast.ArrayType{}), reflect.TypeOf(ast.StructType{}),
	reflect.TypeOf(ast.FuncType{}), reflect.TypeOf(ast.InterfaceType{}), reflect.TypeOf(ast.MapType{}),
	reflect.TypeOf(ast.ChanType{}), reflect.TypeOf(ast.BadStmt{}), reflect.TypeOf(ast.DeclStmt{}),
	reflect.TypeOf(ast.EmptyStmt{}), reflect.TypeOf(ast.LabeledStmt{}), reflect.TypeOf(ast.ExprStmt{}),
	reflect.TypeOf(ast.SendStmt{}), reflect.TypeOf(ast.IncDecStmt{}), reflect.TypeOf(ast.AssignStmt{}),
	reflect.TypeOf(ast.GoStmt{}), reflect.TypeOf(ast.DeferStmt{}), reflect.TypeOf(ast.ReturnStmt{}),
	reflect.TypeOf(ast.BranchStmt{}), reflect.TypeOf(ast.BlockStmt{}), reflect.TypeOf(ast.IfStmt{}),
	reflect.TypeOf(ast.CaseClause{}), reflect.TypeOf(ast.SwitchStmt{}), reflect.TypeOf(ast.TypeSwitchStmt{}),
	reflect.TypeOf(ast.CommClause{}), reflect.TypeOf(ast.SelectStmt{}), reflect.TypeOf(ast.ForStmt{}),
	reflect.TypeOf(ast.RangeStmt{}), reflect.TypeOf(ast.ImportSpec{}), reflect.TypeOf(ast.ValueSpec{}),
	reflect.TypeOf(ast.TypeSpec{}), reflect.TypeOf(ast.BadDecl{}), reflect.TypeOf(ast.GenDecl{}),
	reflect.TypeOf(ast.FuncDecl{}),
}

var syntaxTags = makeSyntaxTags()

type syntaxEncoder struct {
	source  string
	base    int
	ids     map[ast.Node]uint64
	records [][]byte
}

func (e *syntaxEncoder) node(value reflect.Value) (uint64, error) {
	if value.IsNil() {
		return 0, nil
	}
	node := value.Interface().(ast.Node)
	if id := e.ids[node]; id != 0 {
		return id, nil
	}
	typ := value.Elem().Type()
	tag := syntaxTags[typ]
	if tag == 0 {
		return 0, fmt.Errorf("unsupported source syntax: %s", typ)
	}
	id := uint64(len(e.records) + 1)
	e.ids[node] = id
	e.records = append(e.records, nil)
	record, err := e.fields(binary.AppendUvarint(nil, tag), value.Elem(), node)
	e.records[id-1] = record
	return id, err
}

func (e *syntaxEncoder) fields(data []byte, value reflect.Value, node ast.Node) ([]byte, error) {
	for i := range value.NumField() {
		if !syntaxField(value.Type().Field(i).Name) {
			continue
		}
		var err error
		data, err = e.value(data, value.Field(i), node)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (e *syntaxEncoder) value(data []byte, value reflect.Value, node ast.Node) ([]byte, error) {
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		return e.reference(data, value)
	case reflect.Slice:
		return e.slice(data, value, node)
	case reflect.String:
		return e.text(data, value.String(), node), nil
	case reflect.Bool, reflect.Int:
		return e.scalar(data, value), nil
	default:
		return nil, fmt.Errorf("unsupported syntax field: %s", value.Type())
	}
}

func (e *syntaxEncoder) reference(data []byte, value reflect.Value) ([]byte, error) {
	if value.IsNil() {
		return binary.AppendUvarint(data, 0), nil
	}
	if value.Kind() == reflect.Interface {
		value = value.Elem()
	}
	id, err := e.node(value)
	return binary.AppendUvarint(data, id), err
}

func (e *syntaxEncoder) scalar(data []byte, value reflect.Value) []byte {
	if value.Kind() == reflect.Int {
		return e.integer(data, value)
	}
	number := uint64(0)
	if value.Bool() {
		number = 1
	}
	return binary.AppendUvarint(data, number)
}

func (e *syntaxEncoder) integer(data []byte, value reflect.Value) []byte {
	number := value.Int()
	if value.Type() == reflect.TypeFor[token.Pos]() && number != 0 {
		number -= int64(e.base - 1)
	}
	return binary.AppendVarint(data, number)
}

func (e *syntaxEncoder) slice(data []byte, value reflect.Value, node ast.Node) ([]byte, error) {
	if value.IsNil() {
		return binary.AppendUvarint(data, 0), nil
	}
	data = binary.AppendUvarint(data, uint64(value.Len()+1))
	for i := range value.Len() {
		var err error
		data, err = e.value(data, value.Index(i), node)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (e *syntaxEncoder) text(data []byte, text string, node ast.Node) []byte {
	start := int(node.Pos()) - e.base
	if start >= 0 && start+len(text) <= len(e.source) && e.source[start:start+len(text)] == text {
		data = binary.AppendUvarint(data, uint64(start+1))
		return binary.AppendUvarint(data, uint64(len(text)))
	}
	data = binary.AppendUvarint(data, 0)
	data = binary.AppendUvarint(data, uint64(len(text)))
	return append(data, text...)
}

type syntaxDecoder struct {
	base   int
	data   string
	source string
	nodes  []reflect.Value
	cursor int
}

func (d *syntaxDecoder) file() (*ast.File, error) {
	count, err := d.number()
	if err != nil {
		return nil, err
	}
	if !d.validNodeCount(count) {
		return nil, fmt.Errorf("invalid syntax node count")
	}
	if err := d.allocate(int(count)); err != nil {
		return nil, err
	}
	d.cursor = 0
	_, _ = d.number()
	if err := d.fill(); err != nil {
		return nil, err
	}
	return d.root()
}

func (d *syntaxDecoder) root() (*ast.File, error) {
	file, ok := d.nodes[0].Interface().(*ast.File)
	if !ok {
		return nil, fmt.Errorf("invalid syntax root")
	}
	if d.cursor != len(d.data) {
		return nil, fmt.Errorf("invalid syntax root or trailing data")
	}
	return file, nil
}

func (d *syntaxDecoder) validNodeCount(count uint64) bool {
	return count != 0 && count <= uint64(len(d.data))
}

// Allocate nodes before resolving references; shared comment groups and
// import nodes retain their identity in the disposable rendering tree.
func (d *syntaxDecoder) allocate(count int) error {
	d.nodes = make([]reflect.Value, count)
	for i := range count {
		tag, err := d.number()
		if err != nil {
			return err
		}
		if tag == 0 || tag > uint64(len(syntaxTypes)) {
			return fmt.Errorf("invalid syntax tag")
		}
		d.nodes[i] = reflect.New(syntaxTypes[tag-1])
		if err := d.fields(d.nodes[i].Elem(), false); err != nil {
			return err
		}
	}
	return nil
}

func (d *syntaxDecoder) fill() error {
	for _, value := range d.nodes {
		if _, err := d.number(); err != nil {
			return err
		}
		if err := d.fields(value.Elem(), true); err != nil {
			return err
		}
	}
	return nil
}

func (d *syntaxDecoder) fields(value reflect.Value, assign bool) error {
	for i := range value.NumField() {
		if !syntaxField(value.Type().Field(i).Name) {
			continue
		}
		if err := d.value(value.Field(i), assign); err != nil {
			return err
		}
	}
	return nil
}

func (d *syntaxDecoder) value(value reflect.Value, assign bool) error {
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		return d.reference(value, assign)
	case reflect.Slice:
		return d.slice(value, assign)
	case reflect.String:
		return d.text(value, assign)
	case reflect.Bool, reflect.Int:
		return d.scalar(value, assign)
	default:
		return fmt.Errorf("unsupported syntax field: %s", value.Type())
	}
}

func (d *syntaxDecoder) scalar(value reflect.Value, assign bool) error {
	if value.Kind() == reflect.Bool {
		return d.boolean(value, assign)
	}
	number, size := binary.Varint([]byte(d.data[d.cursor:]))
	if size <= 0 {
		return fmt.Errorf("invalid syntax integer")
	}
	d.cursor += size
	if assign {
		if value.Type() == reflect.TypeFor[token.Pos]() && number != 0 {
			number += int64(d.base - 1)
		}
		value.SetInt(number)
	}
	return nil
}

func (d *syntaxDecoder) reference(value reflect.Value, assign bool) error {
	id, err := d.number()
	if err != nil {
		return err
	}
	if id > uint64(len(d.nodes)) {
		return fmt.Errorf("invalid syntax reference")
	}
	if !assign || id == 0 {
		return nil
	}
	node := d.nodes[id-1]
	if !node.Type().AssignableTo(value.Type()) {
		return fmt.Errorf("invalid syntax reference type")
	}
	value.Set(node)
	return nil
}

func (d *syntaxDecoder) slice(value reflect.Value, assign bool) error {
	count, err := d.number()
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	count--
	if count > uint64(len(d.data)-d.cursor) {
		return fmt.Errorf("invalid syntax slice length")
	}
	return d.sliceItems(value, assign, int(count))
}

func (d *syntaxDecoder) sliceItems(value reflect.Value, assign bool, count int) error {
	target := reflect.MakeSlice(value.Type(), count, count)
	for i := range count {
		if err := d.value(target.Index(i), assign); err != nil {
			return err
		}
	}
	if assign {
		value.Set(target)
	}
	return nil
}

func (d *syntaxDecoder) boolean(value reflect.Value, assign bool) error {
	number, err := d.number()
	if err != nil {
		return err
	}
	if number > 1 {
		return fmt.Errorf("invalid syntax boolean")
	}
	if assign {
		value.SetBool(number != 0)
	}
	return nil
}

func (d *syntaxDecoder) text(value reflect.Value, assign bool) error {
	start, err := d.number()
	if err != nil {
		return err
	}
	length, err := d.number()
	if err != nil {
		return err
	}
	text, err := d.textBytes(start, length)
	if err != nil {
		return err
	}
	if assign {
		value.SetString(text)
	}
	return nil
}

func (d *syntaxDecoder) textBytes(start, length uint64) (string, error) {
	if start != 0 {
		start--
		if start > uint64(len(d.source)) || length > uint64(len(d.source))-start {
			return "", fmt.Errorf("invalid syntax source range")
		}
		return d.source[start : start+length], nil
	}
	if length > uint64(len(d.data)-d.cursor) {
		return "", fmt.Errorf("invalid syntax text length")
	}
	text := strings.Clone(d.data[d.cursor : d.cursor+int(length)])
	d.cursor += int(length)
	return text, nil
}

func (d *syntaxDecoder) number() (uint64, error) {
	number, size := binary.Uvarint([]byte(d.data[d.cursor:]))
	if size <= 0 {
		return 0, fmt.Errorf("invalid syntax number")
	}
	d.cursor += size
	return number, nil
}

func makeSyntaxTags() map[reflect.Type]uint64 {
	tags := make(map[reflect.Type]uint64, len(syntaxTypes))
	for i, typ := range syntaxTypes {
		tags[typ] = uint64(i + 1)
	}
	return tags
}

func syntaxField(name string) bool {
	return name != "Obj" && name != "Scope" && name != "Unresolved"
}

func captureSourceSyntax(fset *token.FileSet, file *ast.File, source []byte) (string, error) {
	encoder := &syntaxEncoder{source: string(source), base: fset.File(file.Pos()).Base(), ids: make(map[ast.Node]uint64)}
	if _, err := encoder.node(reflect.ValueOf(file)); err != nil {
		return "", err
	}
	data := binary.AppendUvarint(nil, uint64(len(encoder.records)))
	for _, record := range encoder.records {
		data = append(data, record...)
	}
	return compactSourceSyntax(data)
}

func sourceSyntaxTree(path string, sources *pdg.SourceContext) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	file, err := sourceSyntaxTreeIn(path, sources, fset)
	return fset, file, err
}

func sourceSyntaxTreeIn(path string, sources *pdg.SourceContext, fset *token.FileSet) (*ast.File, error) {
	snapshot, ok := sources.Snapshot(sources.Reference(path))
	if !ok {
		return nil, fmt.Errorf("%w: source evidence is missing: %s", errSourceSyntax, path)
	}
	if snapshot.SyntaxProfile != sourceSyntaxProfile || snapshot.Syntax == "" {
		return nil, fmt.Errorf("%w: source syntax inventory is missing: %s", errSourceSyntax, path)
	}
	data, err := inflateSourceSyntax(snapshot.Syntax, len(snapshot.Content))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", errSourceSyntax, path, err)
	}
	tokenFile := fset.AddFile(path, -1, len(snapshot.Content))
	tokenFile.SetLinesForContent([]byte(snapshot.Content))
	decoder := &syntaxDecoder{data: data, source: snapshot.Content, base: tokenFile.Base()}
	file, err := decoder.file()
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", errSourceSyntax, path, err)
	}
	return file, nil
}

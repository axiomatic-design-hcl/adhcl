package schema

import (
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

const codeInvalidStructure = "invalid_structure"

// labelPattern is the label grammar for named blocks.
var labelPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

// metadata lists the common metadata that one block admits. The zero value admits none.
type metadata struct {
	cites   bool
	origins []model.Origin // empty means the origin field is not admitted
	alias   bool
}

var (
	allOrigins       = []model.Origin{model.OriginUser, model.OriginSource, model.OriginProposed, model.OriginInferred}
	criterionOrigins = []model.Origin{model.OriginUser, model.OriginProposed, model.OriginSource}

	// standardMeta admits cites and origin, as most core blocks do.
	standardMeta = metadata{cites: true, origins: allOrigins}
	// aliasedMeta admits cites, origin, and alias. It is used by fr, dp, pv, and module.
	aliasedMeta = metadata{cites: true, origins: allOrigins, alias: true}
	// criterionMeta admits cites and the origin subset that criteria allow.
	criterionMeta = metadata{cites: true, origins: criterionOrigins}
)

// blockReader reads one block body against its schema. It marks each attribute
// and child block that it reads, and finish reports whatever was left unread.
type blockReader struct {
	file         string
	block        *hclsyntax.Block
	findings     *[]result.Finding
	subject      string
	readAttrs    map[string]bool
	readChildren map[*hclsyntax.Block]bool
}

// newBlockReader starts reading one block. Findings are appended to findings.
func newBlockReader(file string, block *hclsyntax.Block, findings *[]result.Finding) *blockReader {
	return &blockReader{
		file:         file,
		block:        block,
		findings:     findings,
		subject:      blockSubject(block),
		readAttrs:    map[string]bool{},
		readChildren: map[*hclsyntax.Block]bool{},
	}
}

// blockSubject names a block by its type and labels, such as fr.build_db.
func blockSubject(block *hclsyntax.Block) string {
	return strings.Join(append([]string{block.Type}, block.Labels...), ".")
}

// blockHeaderRange covers the block type, its labels, and the opening brace.
func blockHeaderRange(block *hclsyntax.Block) hcl.Range {
	return hcl.RangeBetween(block.TypeRange, block.OpenBraceRange)
}

// addFinding appends one invalid_structure finding.
func addFinding(findings *[]result.Finding, subject string, rng hcl.Range, message string) {
	*findings = append(*findings, result.Finding{
		Code:     codeInvalidStructure,
		Severity: result.Error,
		Subject:  subject,
		Message:  message,
		Range:    &rng,
	})
}

// fail reports one invalid_structure finding for this block.
func (r *blockReader) fail(rng hcl.Range, message string) {
	addFinding(r.findings, r.subject, rng, message)
}

// namedHeader reads the single label of a named block and its common metadata.
// The identity is prefix, a dot, and the label. Nested blocks pass their full
// parent path as the prefix.
func (r *blockReader) namedHeader(prefix string, admit metadata) model.Header {
	label := ""
	switch n := len(r.block.Labels); {
	case n == 1:
		label = r.block.Labels[0]
		if !labelPattern.MatchString(label) {
			r.fail(blockHeaderRange(r.block), "the label must use lower case letters, digits, and single underscores, and start with a letter")
		}
	case n == 0:
		r.fail(blockHeaderRange(r.block), fmt.Sprintf("a %s block needs exactly one quoted label", r.block.Type))
	default:
		label = r.block.Labels[0]
		r.fail(blockHeaderRange(r.block), fmt.Sprintf("a %s block needs exactly one label, found %d", r.block.Type, n))
	}
	cites, origin, alias := r.readMetadata(admit)
	return model.Header{
		ID:     model.ID(prefix + "." + label),
		File:   r.file,
		Range:  hcl.RangeBetween(r.block.TypeRange, r.block.CloseBraceRange),
		Cites:  cites,
		Origin: origin,
		Alias:  alias,
	}
}

// unlabeledHeader reads an unlabeled block, which takes no label, and its common metadata.
func (r *blockReader) unlabeledHeader(admit metadata) model.Unlabeled {
	if len(r.block.Labels) != 0 {
		r.fail(blockHeaderRange(r.block), fmt.Sprintf("a %s block takes no label", r.block.Type))
	}
	cites, origin, _ := r.readMetadata(admit)
	return model.Unlabeled{
		File:   r.file,
		Range:  hcl.RangeBetween(r.block.TypeRange, r.block.CloseBraceRange),
		Cites:  cites,
		Origin: origin,
	}
}

// readMetadata reads the metadata that admit allows. Metadata that is not
// admitted stays unread, so finish reports it as an unknown field.
func (r *blockReader) readMetadata(admit metadata) ([]model.Ref, *model.Origin, *string) {
	var cites []model.Ref
	if admit.cites {
		cites = r.oneOrMoreRefs("cites")
	}
	var origin *model.Origin
	if len(admit.origins) > 0 {
		origin = optionalField(r, "origin", originOf(admit.origins))
	}
	var alias *string
	if admit.alias {
		alias = r.optionalString("alias")
	}
	return cites, origin, alias
}

// childBlocks returns the child blocks of one type and marks them read.
// Blocks of other types stay unread, so finish reports them.
func (r *blockReader) childBlocks(blockType string) []*hclsyntax.Block {
	var found []*hclsyntax.Block
	for _, child := range r.block.Body.Blocks {
		if child.Type == blockType {
			r.readChildren[child] = true
			found = append(found, child)
		}
	}
	return found
}

// finish reports every attribute and child block that no reader consumed.
func (r *blockReader) finish() {
	for _, attr := range sortedAttributes(r.block.Body.Attributes) {
		if !r.readAttrs[attr.Name] {
			r.fail(attr.SrcRange, fmt.Sprintf("%q is not a field of a %s block", attr.Name, r.block.Type))
		}
	}
	for _, child := range r.block.Body.Blocks {
		if !r.readChildren[child] {
			r.fail(blockHeaderRange(child), fmt.Sprintf("a %s block is not allowed inside a %s block", child.Type, r.block.Type))
		}
	}
}

// take marks the named attribute read and returns it. A missing required
// attribute is reported at the block header, and the result is nil.
func (r *blockReader) take(name string, required bool) *hclsyntax.Attribute {
	r.readAttrs[name] = true
	attr, ok := r.block.Body.Attributes[name]
	if !ok {
		if required {
			r.fail(blockHeaderRange(r.block), name+" is required")
		}
		return nil
	}
	return attr
}

// field reads one attribute with parse. A present attribute with a defect is
// reported once at its line. The caller gets the zero value and false.
func field[T any](r *blockReader, name string, required bool, parse func(hclsyntax.Expression) (T, string)) (T, bool) {
	var zero T
	attr := r.take(name, required)
	if attr == nil {
		return zero, false
	}
	value, problem := parse(attr.Expr)
	if problem != "" {
		r.fail(attr.SrcRange, name+" "+problem)
		return zero, false
	}
	return value, true
}

// requiredField reads a required attribute and returns the zero value on any defect.
func requiredField[T any](r *blockReader, name string, parse func(hclsyntax.Expression) (T, string)) T {
	value, _ := field(r, name, true, parse)
	return value
}

// optionalField reads an optional attribute and returns nil when it is absent or invalid.
func optionalField[T any](r *blockReader, name string, parse func(hclsyntax.Expression) (T, string)) *T {
	value, ok := field(r, name, false, parse)
	if !ok {
		return nil
	}
	return &value
}

func (r *blockReader) requiredString(name string) string {
	return requiredField(r, name, stringOf)
}

func (r *blockReader) optionalString(name string) *string {
	return optionalField(r, name, stringOf)
}

func (r *blockReader) requiredEnum(name string, allowed ...string) string {
	return requiredField(r, name, enumOf(allowed...))
}

func (r *blockReader) optionalEnum(name string, allowed ...string) *string {
	return optionalField(r, name, enumOf(allowed...))
}

func (r *blockReader) optionalBool(name string) *bool {
	return optionalField(r, name, boolOf)
}

func (r *blockReader) requiredNumber(name string) *big.Float {
	return requiredField(r, name, numberOf)
}

// requiredInteger reads a required whole number. When positive is true, zero and negative values are rejected.
func (r *blockReader) requiredInteger(name string, positive bool) *big.Float {
	return requiredField(r, name, integerOf(positive))
}

// optionalInteger reads an optional whole number. When positive is true, zero and negative values are rejected.
func (r *blockReader) optionalInteger(name string, positive bool) *big.Float {
	return flatten(optionalField(r, name, integerOf(positive)))
}

// flatten turns an absent optional number into nil and a present one into its value.
func flatten(value **big.Float) *big.Float {
	if value == nil {
		return nil
	}
	return *value
}

func (r *blockReader) requiredRef(name string) model.Ref {
	return requiredField(r, name, refOf)
}

func (r *blockReader) optionalRef(name string) *model.Ref {
	return optionalField(r, name, refOf)
}

// requiredRefs reads a required list of references with at least one item.
func (r *blockReader) requiredRefs(name string) []model.Ref {
	return requiredField(r, name, nonEmptyRefsOf)
}

// optionalRefs reads an optional list of references. An empty list is allowed.
func (r *blockReader) optionalRefs(name string) []model.Ref {
	refs, _ := field(r, name, false, refsOf)
	return refs
}

// oneOrMoreRefs reads an optional single reference or a nonempty list of references.
// A single reference and a one-item list give the same result.
func (r *blockReader) oneOrMoreRefs(name string) []model.Ref {
	refs, _ := field(r, name, false, oneOrMoreRefsOf)
	return refs
}

// sortedAttributes returns the attributes in authored order.
func sortedAttributes(attrs hclsyntax.Attributes) []*hclsyntax.Attribute {
	sorted := make([]*hclsyntax.Attribute, 0, len(attrs))
	for _, attr := range attrs {
		sorted = append(sorted, attr)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].SrcRange.Start.Byte < sorted[j].SrcRange.Start.Byte
	})
	return sorted
}

// literal evaluates an expression that is a literal value: a constant, a
// string template with literal parts, or a sign applied to a literal number.
// It reports false for references, lists, objects, and other shapes.
func literal(expr hclsyntax.Expression) (cty.Value, bool) {
	switch expr.(type) {
	case *hclsyntax.LiteralValueExpr, *hclsyntax.TemplateExpr, *hclsyntax.UnaryOpExpr:
	default:
		return cty.NilVal, false
	}
	val, diags := expr.Value(nil)
	if diags.HasErrors() || !val.IsWhollyKnown() {
		return cty.NilVal, false
	}
	return val, true
}

// isNull reports whether the expression is the literal null.
func isNull(expr hclsyntax.Expression) bool {
	val, ok := literal(expr)
	return ok && val.IsNull()
}

// listItems returns the items of a literal list.
func listItems(expr hclsyntax.Expression) ([]hclsyntax.Expression, bool) {
	tuple, ok := expr.(*hclsyntax.TupleConsExpr)
	if !ok {
		return nil, false
	}
	return tuple.Exprs, true
}

// stringOf returns the text of a nonblank string literal. Otherwise it returns a plain-English problem.
func stringOf(expr hclsyntax.Expression) (string, string) {
	val, ok := literal(expr)
	if !ok {
		if _, isRef := expr.(*hclsyntax.ScopeTraversalExpr); isRef {
			return "", "must be a string, not a reference"
		}
		return "", "must be a string"
	}
	if val.IsNull() {
		return "", "must not be null"
	}
	if !val.Type().Equals(cty.String) {
		return "", "must be a string"
	}
	text := val.AsString()
	if strings.TrimSpace(text) == "" {
		return "", "must not be blank"
	}
	return text, ""
}

// stringsOf returns a literal list of nonblank strings. An empty list is allowed.
func stringsOf(expr hclsyntax.Expression) ([]string, string) {
	items, ok := listItems(expr)
	if !ok {
		return nil, "must be a list of strings"
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		text, problem := stringOf(item)
		if problem != "" {
			return nil, "each item " + problem
		}
		values = append(values, text)
	}
	return values, ""
}

// enumOf accepts one string that is exactly one of the allowed values.
func enumOf(allowed ...string) func(hclsyntax.Expression) (string, string) {
	return func(expr hclsyntax.Expression) (string, string) {
		text, problem := stringOf(expr)
		if problem != "" {
			return "", problem
		}
		for _, value := range allowed {
			if value == text {
				return text, ""
			}
		}
		return "", "must be one of " + quoteList(allowed)
	}
}

// originOf accepts one origin from the admitted set.
func originOf(admitted []model.Origin) func(hclsyntax.Expression) (model.Origin, string) {
	names := make([]string, 0, len(admitted))
	for _, origin := range admitted {
		names = append(names, string(origin))
	}
	parse := enumOf(names...)
	return func(expr hclsyntax.Expression) (model.Origin, string) {
		text, problem := parse(expr)
		return model.Origin(text), problem
	}
}

// quoteList joins values as quoted words, such as "a", "b", "c".
func quoteList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}

// boolOf accepts one Boolean literal.
func boolOf(expr hclsyntax.Expression) (bool, string) {
	val, ok := literal(expr)
	if !ok {
		return false, "must be true or false"
	}
	if val.IsNull() {
		return false, "must not be null"
	}
	if !val.Type().Equals(cty.Bool) {
		return false, "must be true or false"
	}
	return val.True(), ""
}

// numberOf accepts one number literal. The value is kept exact.
func numberOf(expr hclsyntax.Expression) (*big.Float, string) {
	val, ok := literal(expr)
	if !ok {
		return nil, "must be a number"
	}
	if val.IsNull() {
		return nil, "must not be null"
	}
	if !val.Type().Equals(cty.Number) {
		return nil, "must be a number"
	}
	return val.AsBigFloat(), ""
}

// integerOf accepts one whole number. When positive is true, the number must be greater than zero.
func integerOf(positive bool) func(hclsyntax.Expression) (*big.Float, string) {
	return func(expr hclsyntax.Expression) (*big.Float, string) {
		number, problem := numberOf(expr)
		if problem != "" {
			return nil, problem
		}
		if !number.IsInt() {
			return nil, "must be a whole number"
		}
		if positive && number.Sign() <= 0 {
			return nil, "must be a positive whole number"
		}
		return number, ""
	}
}

// refOf accepts one static traversal such as kind.label. Target kinds are checked during resolution.
func refOf(expr hclsyntax.Expression) (model.Ref, string) {
	trav, ok := expr.(*hclsyntax.ScopeTraversalExpr)
	if !ok {
		if isNull(expr) {
			return model.Ref{}, "must not be null"
		}
		return model.Ref{}, "must be a reference such as kind.label"
	}
	path, ok := traversalPath(trav.Traversal)
	if !ok {
		return model.Ref{}, "must be a static reference such as kind.label"
	}
	return model.Ref{Target: model.ID(path), Range: trav.Range()}, ""
}

// traversalPath joins the names of a traversal with dots.
func traversalPath(trav hcl.Traversal) (string, bool) {
	parts := make([]string, 0, len(trav))
	for _, step := range trav {
		switch s := step.(type) {
		case hcl.TraverseRoot:
			parts = append(parts, s.Name)
		case hcl.TraverseAttr:
			parts = append(parts, s.Name)
		default:
			return "", false
		}
	}
	return strings.Join(parts, "."), true
}

// refsOf accepts a literal list of references. An empty list is allowed.
// Repeated targets are rejected, and order is kept.
func refsOf(expr hclsyntax.Expression) ([]model.Ref, string) {
	items, ok := listItems(expr)
	if !ok {
		return nil, "must be a list of references"
	}
	refs := make([]model.Ref, 0, len(items))
	seen := make(map[model.ID]bool, len(items))
	for _, item := range items {
		ref, problem := refOf(item)
		if problem != "" {
			return nil, "each item " + problem
		}
		if seen[ref.Target] {
			return nil, fmt.Sprintf("must not repeat %s", ref.Target)
		}
		seen[ref.Target] = true
		refs = append(refs, ref)
	}
	return refs, ""
}

// nonEmptyRefsOf accepts a literal list of references with at least one item.
func nonEmptyRefsOf(expr hclsyntax.Expression) ([]model.Ref, string) {
	refs, problem := refsOf(expr)
	if problem != "" {
		return nil, problem
	}
	if len(refs) == 0 {
		return nil, "must not be an empty list"
	}
	return refs, ""
}

// oneOrMoreRefsOf accepts one reference or a nonempty list of references.
func oneOrMoreRefsOf(expr hclsyntax.Expression) ([]model.Ref, string) {
	if _, isList := expr.(*hclsyntax.TupleConsExpr); isList {
		return nonEmptyRefsOf(expr)
	}
	ref, problem := refOf(expr)
	if problem != "" {
		return nil, problem
	}
	return []model.Ref{ref}, ""
}

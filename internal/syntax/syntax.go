package syntax

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

type Unit struct {
	Source   model.Source
	Body     *hclsyntax.Body
	Findings []result.Finding
}

func Parse(source model.Source) Unit {
	unit := Unit{Source: source}
	file, diagnostics := hclsyntax.ParseConfig(source.Bytes, source.Path, hcl.Pos{Line: 1, Column: 1})
	unit.Findings = append(unit.Findings, syntaxFindings(source.Path, diagnostics)...)
	if file != nil {
		unit.Body, _ = file.Body.(*hclsyntax.Body)
	}
	if unit.Body != nil {
		unit.Findings = append(unit.Findings, inspectBody(source.Path, unit.Body)...)
	}
	sortFindings(unit.Findings)
	return unit
}

func syntaxFindings(path string, diagnostics hcl.Diagnostics) []result.Finding {
	findings := make([]result.Finding, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		severity := result.Error
		if diagnostic.Severity == hcl.DiagWarning {
			severity = result.Warning
		}
		code := "invalid_syntax"
		if diagnostic.Summary == "Attribute redefined" {
			code = "duplicate_identity"
		}
		var sourceRange *hcl.Range
		if diagnostic.Subject != nil {
			rangeCopy := *diagnostic.Subject
			sourceRange = &rangeCopy
		}
		message := diagnostic.Summary
		if diagnostic.Detail != "" {
			message += ": " + diagnostic.Detail
		}
		findings = append(findings, result.Finding{
			Code:     code,
			Severity: severity,
			Subject:  path,
			Message:  message,
			Range:    sourceRange,
		})
	}
	return findings
}

func inspectBody(path string, body *hclsyntax.Body) []result.Finding {
	var findings []result.Finding
	attributeNames := make([]string, 0, len(body.Attributes))
	for name := range body.Attributes {
		attributeNames = append(attributeNames, name)
	}
	sort.Strings(attributeNames)
	for _, name := range attributeNames {
		findings = append(findings, inspectExpression(path, body.Attributes[name].Expr)...)
	}
	for _, block := range body.Blocks {
		findings = append(findings, inspectBody(path, block.Body)...)
	}
	return findings
}

func inspectExpression(path string, expression hclsyntax.Expression) []result.Finding {
	switch value := expression.(type) {
	case *hclsyntax.LiteralValueExpr:
		return nil
	case *hclsyntax.ScopeTraversalExpr:
		if isStaticTraversal(value.Traversal) {
			return nil
		}
		return []result.Finding{syntaxFinding(path, value.Range(), "Only attribute names are allowed after the root of a reference.")}
	case *hclsyntax.TemplateExpr:
		if isLiteralTemplate(value) {
			return nil
		}
		return []result.Finding{syntaxFinding(path, value.Range(), "String templates are not allowed.")}
	case *hclsyntax.TupleConsExpr:
		var findings []result.Finding
		for _, item := range value.Exprs {
			findings = append(findings, inspectExpression(path, item)...)
		}
		return findings
	case *hclsyntax.ObjectConsExpr:
		findings := inspectObjectKeys(path, value)
		for _, item := range value.Items {
			findings = append(findings, inspectExpression(path, item.ValueExpr)...)
		}
		return findings
	case *hclsyntax.UnaryOpExpr:
		if value.Op == hclsyntax.OpNegate && isNumericLiteral(value.Val) {
			return nil
		}
		return []result.Finding{syntaxFinding(path, value.Range(), "Only a minus sign directly on a number is allowed.")}
	default:
		return []result.Finding{syntaxFinding(path, expression.Range(), "This expression form is not allowed.")}
	}
}

func inspectObjectKeys(path string, object *hclsyntax.ObjectConsExpr) []result.Finding {
	var findings []result.Finding
	keys := make(map[string]hcl.Range, len(object.Items))
	for _, item := range object.Items {
		key, ok := literalObjectKey(item.KeyExpr)
		if !ok {
			findings = append(findings, syntaxFinding(path, item.KeyExpr.Range(), "Object keys must be literal identifiers or strings."))
			continue
		}
		if _, exists := keys[key]; exists {
			findings = append(findings, result.Finding{
				Code:     "duplicate_identity",
				Severity: result.Error,
				Subject:  path,
				Message:  "Object key is repeated.",
				Range:    rangePointer(item.KeyExpr.Range()),
				Related:  []hcl.Range{keys[key]},
			})
			continue
		}
		keys[key] = item.KeyExpr.Range()
	}
	return findings
}

func literalObjectKey(expression hclsyntax.Expression) (string, bool) {
	key, ok := expression.(*hclsyntax.ObjectConsKeyExpr)
	if !ok || key.ForceNonLiteral {
		return "", false
	}
	if name := hcl.ExprAsKeyword(key.Wrapped); name != "" {
		return name, true
	}
	if value, ok := literalStringExpression(key.Wrapped); ok {
		return value, true
	}
	return "", false
}

func literalStringExpression(expression hclsyntax.Expression) (string, bool) {
	switch value := expression.(type) {
	case *hclsyntax.LiteralValueExpr:
		if value.Val.IsKnown() && !value.Val.IsNull() && value.Val.Type().Equals(cty.String) {
			return value.Val.AsString(), true
		}
	case *hclsyntax.TemplateExpr:
		var text strings.Builder
		for _, part := range value.Parts {
			literal, ok := part.(*hclsyntax.LiteralValueExpr)
			if !ok || literal.Val.IsNull() || !literal.Val.Type().Equals(cty.String) {
				return "", false
			}
			text.WriteString(literal.Val.AsString())
		}
		return text.String(), true
	}
	return "", false
}

func isLiteralTemplate(expression *hclsyntax.TemplateExpr) bool {
	for _, part := range expression.Parts {
		literal, ok := part.(*hclsyntax.LiteralValueExpr)
		if !ok || literal.Val.IsNull() || !literal.Val.Type().Equals(cty.String) {
			return false
		}
	}
	return true
}

func isNumericLiteral(expression hclsyntax.Expression) bool {
	literal, ok := expression.(*hclsyntax.LiteralValueExpr)
	return ok && !literal.Val.IsNull() && literal.Val.Type().Equals(cty.Number)
}

func isStaticTraversal(traversal hcl.Traversal) bool {
	if len(traversal) == 0 {
		return false
	}
	for _, step := range traversal[1:] {
		if _, isAttribute := step.(hcl.TraverseAttr); !isAttribute {
			return false
		}
	}
	return true
}

func syntaxFinding(path string, sourceRange hcl.Range, message string) result.Finding {
	return result.Finding{
		Code:     "invalid_syntax",
		Severity: result.Error,
		Subject:  path,
		Message:  message,
		Range:    rangePointer(sourceRange),
	}
}

func rangePointer(sourceRange hcl.Range) *hcl.Range {
	rangeCopy := sourceRange
	return &rangeCopy
}

func sortFindings(findings []result.Finding) {
	sort.SliceStable(findings, func(left, right int) bool {
		leftRange, rightRange := findings[left].Range, findings[right].Range
		if leftRange == nil || rightRange == nil {
			return leftRange != nil && rightRange == nil
		}
		if leftRange.Start.Line != rightRange.Start.Line {
			return leftRange.Start.Line < rightRange.Start.Line
		}
		if leftRange.Start.Column != rightRange.Start.Column {
			return leftRange.Start.Column < rightRange.Start.Column
		}
		if leftRange.End.Line != rightRange.End.Line {
			return leftRange.End.Line < rightRange.End.Line
		}
		if leftRange.End.Column != rightRange.End.Column {
			return leftRange.End.Column < rightRange.End.Column
		}
		if findings[left].Code != findings[right].Code {
			return findings[left].Code < findings[right].Code
		}
		return findings[left].Subject < findings[right].Subject
	})
}

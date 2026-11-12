package schema

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

// DecodeManifest decodes one project manifest. It admits one project block,
// any number of extension blocks, and no other top-level content.
func DecodeManifest(unit syntax.Unit) (model.Manifest, []result.Finding) {
	var manifest model.Manifest
	var findings []result.Finding
	if unit.Body == nil {
		return manifest, findings
	}
	path := unit.Source.Path
	for _, attr := range sortedAttributes(unit.Body.Attributes) {
		addFinding(&findings, path, attr.SrcRange, fmt.Sprintf("%q is not allowed at the top level of a manifest", attr.Name))
	}
	for _, block := range unit.Body.Blocks {
		switch block.Type {
		case "project":
			if manifest.Project != nil {
				addFinding(&findings, path, blockHeaderRange(block), "a manifest may contain only one project block")
				continue
			}
			project := decodeProject(path, block, &findings)
			manifest.Project = &project
		case "extension":
			manifest.Extensions = append(manifest.Extensions, decodeExtension(path, block, &findings))
		default:
			addFinding(&findings, path, blockHeaderRange(block), fmt.Sprintf("a manifest cannot contain a %q block", block.Type))
		}
	}
	if manifest.Project == nil {
		addFinding(&findings, path, hcl.Range{Filename: path}, "a manifest needs one project block")
	}
	return manifest, findings
}

// decodeProject decodes the project block of a manifest.
func decodeProject(file string, block *hclsyntax.Block, findings *[]result.Finding) model.Project {
	r := newBlockReader(file, block, findings)
	label := manifestLabel(r)
	language := model.Language(r.requiredEnum("language", string(model.LanguageCore1)))
	coreFiles, _ := field(r, "core_files", true, nonEmptyListedPathsOf)
	attachmentFiles, _ := field(r, "attachment_files", true, listedPathsOf)
	r.finish()

	return model.Project{
		Label:           label,
		File:            file,
		Range:           blockRange(block),
		Language:        language,
		CoreFiles:       coreFiles,
		AttachmentFiles: attachmentFiles,
	}
}

// decodeExtension decodes one extension block of a manifest.
func decodeExtension(file string, block *hclsyntax.Block, findings *[]result.Finding) model.Extension {
	r := newBlockReader(file, block, findings)
	label := manifestLabel(r)
	role := model.ExtensionRole(r.requiredEnum("role",
		string(model.ExtensionRoleBinding),
		string(model.ExtensionRoleBehaviorLink),
		string(model.ExtensionRoleView),
	))
	contract := r.requiredString("contract")
	r.finish()

	return model.Extension{
		ID:       model.ID(string(model.KindExtension) + "." + label),
		File:     file,
		Range:    blockRange(block),
		Role:     role,
		Contract: contract,
	}
}

// manifestLabel reads the single label of a manifest block.
func manifestLabel(r *blockReader) string {
	if len(r.block.Labels) != 1 {
		r.fail(blockHeaderRange(r.block), fmt.Sprintf("a %s block needs exactly one label", r.block.Type))
		return ""
	}
	label := r.block.Labels[0]
	if !labelPattern.MatchString(label) {
		r.fail(blockHeaderRange(r.block), "label must match the label grammar: lower case letters, digits, and single underscores")
	}
	return label
}

// listedPathsOf parses a list of file paths. An empty list is allowed.
func listedPathsOf(expr hclsyntax.Expression) ([]model.ListedPath, string) {
	items, ok := listItems(expr)
	if !ok {
		return nil, "must be a list of file paths"
	}
	paths := make([]model.ListedPath, 0, len(items))
	for _, item := range items {
		text, problem := stringOf(item)
		if problem != "" {
			return nil, problem
		}
		paths = append(paths, model.ListedPath{Path: text, Range: item.Range()})
	}
	return paths, ""
}

// nonEmptyListedPathsOf parses a list of file paths that must not be empty.
func nonEmptyListedPathsOf(expr hclsyntax.Expression) ([]model.ListedPath, string) {
	paths, problem := listedPathsOf(expr)
	if problem != "" {
		return nil, problem
	}
	if len(paths) == 0 {
		return nil, "must list at least one file"
	}
	return paths, ""
}

// DecodeAttachments decodes the attachment envelopes of the listed attachment
// files. An attachment that fails is dropped; the others still decode.
func DecodeAttachments(units []syntax.Unit) (model.Attachments, []result.Finding) {
	var attachments model.Attachments
	var findings []result.Finding
	for _, unit := range units {
		if unit.Body == nil {
			continue
		}
		path := unit.Source.Path
		for _, attr := range sortedAttributes(unit.Body.Attributes) {
			addFinding(&findings, path, attr.SrcRange, fmt.Sprintf("%q is not allowed at the top level of an attachment file", attr.Name))
		}
		for _, block := range unit.Body.Blocks {
			if block.Type != "attachment" {
				addFinding(&findings, path, blockHeaderRange(block), fmt.Sprintf("an attachment file cannot contain a %q block", block.Type))
				continue
			}
			attachment, ok := decodeAttachment(path, block, &findings)
			if !ok {
				continue
			}
			attachments = append(attachments, attachment)
		}
	}
	return attachments, findings
}

// decodeAttachment decodes one attachment envelope made of owner, subject,
// and data. Envelope defects are invalid attachments. Data keys are not
// judged here. It reports whether the envelope was valid.
func decodeAttachment(file string, block *hclsyntax.Block, findings *[]result.Finding) (model.Attachment, bool) {
	before := len(*findings)
	subject := blockSubject(block)
	header := blockHeaderRange(block)
	report := func(rng hcl.Range, message string) {
		addCodedFinding(findings, codeInvalidAttachment, subject, rng, message)
	}

	label := ""
	if len(block.Labels) != 1 {
		report(header, "an attachment needs exactly one label")
	} else {
		label = block.Labels[0]
		if !labelPattern.MatchString(label) {
			report(header, "attachment label must match the label grammar: lower case letters, digits, and single underscores")
		}
	}

	for _, child := range block.Body.Blocks {
		report(blockHeaderRange(child), fmt.Sprintf("an attachment cannot contain a %q block", child.Type))
	}
	for _, attr := range sortedAttributes(block.Body.Attributes) {
		switch attr.Name {
		case "owner", "subject", "data":
		default:
			report(attr.SrcRange, fmt.Sprintf("%q is not a field of an attachment", attr.Name))
		}
	}

	owner := attachmentRef(block, "owner", report)
	subjectRef := attachmentRef(block, "subject", report)
	data := attachmentData(block, findings, report)

	if len(*findings) != before {
		return model.Attachment{}, false
	}
	return model.Attachment{
		Label:   label,
		File:    file,
		Range:   blockRange(block),
		Owner:   owner,
		Subject: subjectRef,
		Data:    data,
	}, true
}

// attachmentRef reads the owner or subject reference of an attachment.
func attachmentRef(block *hclsyntax.Block, name string, report func(hcl.Range, string)) model.Ref {
	attr, ok := block.Body.Attributes[name]
	if !ok {
		report(blockHeaderRange(block), fmt.Sprintf("an attachment needs %s", name))
		return model.Ref{}
	}
	ref, problem := refOf(attr.Expr)
	if problem != "" {
		report(attr.SrcRange, name+" "+problem)
		return model.Ref{}
	}
	return ref
}

// attachmentData reads the data object of an attachment. Keys must be unique
// and values must be non-null literals. Traversals and expressions are rejected.
func attachmentData(block *hclsyntax.Block, findings *[]result.Finding, report func(hcl.Range, string)) []model.DataField {
	attr, ok := block.Body.Attributes["data"]
	if !ok {
		report(blockHeaderRange(block), "an attachment needs data")
		return nil
	}
	object, isObject := attr.Expr.(*hclsyntax.ObjectConsExpr)
	if !isObject {
		report(attr.SrcRange, "data must be an object of literal values")
		return nil
	}

	seen := make(map[string]bool, len(object.Items))
	fields := make([]model.DataField, 0, len(object.Items))
	for _, item := range object.Items {
		key, problem := objectKeyOf(item.KeyExpr)
		if problem != "" {
			report(attr.SrcRange, "data "+problem)
			continue
		}
		if seen[key] {
			addCodedFinding(findings, codeDuplicateIdentity, blockSubject(block), item.KeyExpr.Range(), fmt.Sprintf("data key %q appears more than once", key))
			continue
		}
		seen[key] = true
		value, isLiteral := literal(item.ValueExpr)
		if !isLiteral {
			report(attr.SrcRange, fmt.Sprintf("data value for %q must be a literal", key))
			continue
		}
		if value.IsNull() {
			report(attr.SrcRange, fmt.Sprintf("data value for %q must not be null", key))
			continue
		}
		fields = append(fields, model.DataField{
			Key:        key,
			KeyRange:   item.KeyExpr.Range(),
			Value:      value,
			ValueRange: item.ValueExpr.Range(),
		})
	}
	return fields
}

// blockRange spans a block from its type keyword to its closing brace.
func blockRange(block *hclsyntax.Block) hcl.Range {
	return hcl.RangeBetween(block.TypeRange, block.CloseBraceRange)
}

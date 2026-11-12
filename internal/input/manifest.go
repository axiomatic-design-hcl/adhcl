package input

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

const (
	projectBlockType     = "project"
	extensionBlockType   = "extension"
	languageField        = "language"
	coreFilesField       = "core_files"
	attachmentFilesField = "attachment_files"
	coreLanguage         = "requirements_hcl/core/1"

	messageManifestUnreadable = "The project manifest cannot be read."
	messageManifestUnresolved = "The project manifest cannot be resolved to a regular file."
	messageManifestDirectory  = "The manifest directory cannot be resolved."
)

var (
	errPathNotRelative = errors.New("the project path must be relative")
	errPathHasGlob     = errors.New("the project path cannot contain a glob pattern")
	errPathWrongSuffix = errors.New("the project path has the wrong file suffix")
	errPathLeaves      = errors.New("the project path leaves the manifest directory")
	errPathUnresolved  = errors.New("the project path cannot be resolved to a file")
	errPathNotRegular  = errors.New("the project path is not a regular file")
)

// pathList is one manifest file field. Valid is false when the field is malformed.
type pathList struct {
	paths []string
	valid bool
}

// manifestEnvelope holds the manifest facts that member acquisition needs.
type manifestEnvelope struct {
	path            string
	info            os.FileInfo
	coreFiles       pathList
	attachmentFiles pathList
}

func selectManifest(selection *Selection, path string, readFile fileReader) {
	selection.AttachmentsComplete = false
	physicalPath, info, ok := regularPhysicalFile(path)
	if !ok {
		addPathFinding(selection, path, messageManifestUnresolved)
		return
	}
	contents, err := readFile(physicalPath)
	if err != nil {
		addPathFinding(selection, path, messageManifestUnreadable)
		return
	}
	manifest := syntax.Parse(model.Source{Path: path, Bytes: contents})
	selection.Manifest = &manifest
	selection.Findings = append(selection.Findings, manifest.Findings...)
	if hasErrors(manifest.Findings) || manifest.Body == nil {
		return
	}
	envelope, admitted := readManifestEnvelope(selection, manifest.Body, path, info)
	if admitted {
		acquireManifestMembers(selection, envelope, readFile)
	}
}

// regularPhysicalFile resolves symbolic links and reports whether the target is a regular file.
func regularPhysicalFile(path string) (string, os.FileInfo, bool) {
	physicalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", nil, false
	}
	info, ok := regularFileInfo(physicalPath)
	return physicalPath, info, ok
}

// readManifestEnvelope checks the manifest structure, language, and file fields. It
// records every problem it finds. The bool is false when the manifest must not
// admit any member.
func readManifestEnvelope(selection *Selection, body *hclsyntax.Body, path string, info os.FileInfo) (manifestEnvelope, bool) {
	envelope := manifestEnvelope{path: path, info: info}
	project, structureValid := checkManifestStructure(selection, body, path)
	if project == nil {
		return envelope, false
	}
	languageValid := checkLanguage(selection, project, path)
	envelope.coreFiles = readPathList(selection, project, path, coreFilesField, true)
	envelope.attachmentFiles = readPathList(selection, project, path, attachmentFilesField, false)
	return envelope, structureValid && languageValid
}

// checkManifestStructure checks top-level blocks and the project block body. It returns
// the first project block, or nil when there is none.
func checkManifestStructure(selection *Selection, body *hclsyntax.Body, path string) (*hclsyntax.Block, bool) {
	attributesValid := rejectTopLevelAttributes(selection, body, path)
	project, blocksValid := checkTopLevelBlocks(selection, body, path)
	if project == nil {
		return nil, false
	}
	projectValid := checkProjectBody(selection, project, path)
	return project, attributesValid && blocksValid && projectValid
}

func rejectTopLevelAttributes(selection *Selection, body *hclsyntax.Body, path string) bool {
	for _, attribute := range sortedAttributes(body.Attributes) {
		addLocatedFinding(selection, path, attribute.NameRange, "Top-level manifest attributes are not allowed.")
	}
	return len(body.Attributes) == 0
}

func checkTopLevelBlocks(selection *Selection, body *hclsyntax.Body, path string) (*hclsyntax.Block, bool) {
	var project *hclsyntax.Block
	valid := true
	for _, block := range body.Blocks {
		switch {
		case block.Type == projectBlockType && project == nil:
			project = block
		case block.Type == projectBlockType:
			addLocatedFinding(selection, path, block.DefRange(), "A manifest must contain exactly one project block.")
			valid = false
		case block.Type == extensionBlockType:
			if len(block.Labels) != 1 {
				addLocatedFinding(selection, path, block.DefRange(), "An extension block requires one label.")
				valid = false
			}
		default:
			addLocatedFinding(selection, path, block.DefRange(), "The manifest contains an unknown block.")
			valid = false
		}
	}
	if project == nil {
		addLocatedFinding(selection, path, body.SrcRange, "A manifest requires one project block.")
		return nil, false
	}
	return project, valid
}

func checkProjectBody(selection *Selection, project *hclsyntax.Block, path string) bool {
	valid := true
	if len(project.Labels) != 1 {
		addLocatedFinding(selection, path, project.DefRange(), "A project block requires one label.")
		valid = false
	}
	for _, attribute := range sortedAttributes(project.Body.Attributes) {
		if !isProjectField(attribute.Name) {
			addLocatedFinding(selection, path, attribute.NameRange, "The project block contains an unknown field.")
			valid = false
		}
	}
	for _, block := range project.Body.Blocks {
		addLocatedFinding(selection, path, block.DefRange(), "The project block cannot contain nested blocks.")
		valid = false
	}
	return valid
}

func isProjectField(name string) bool {
	switch name {
	case languageField, coreFilesField, attachmentFilesField:
		return true
	}
	return false
}

func checkLanguage(selection *Selection, project *hclsyntax.Block, path string) bool {
	attribute, exists := project.Body.Attributes[languageField]
	if !exists {
		addLocatedFinding(selection, path, project.DefRange(), "The project block is missing language.")
		return false
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if diagnostics.HasErrors() || !isKnownString(value) || value.AsString() != coreLanguage {
		addLocatedFinding(selection, path, attribute.Expr.Range(), "The project language must be requirements_hcl/core/1.")
		return false
	}
	return true
}

func isKnownString(value cty.Value) bool {
	return value.IsKnown() && !value.IsNull() && value.Type().Equals(cty.String)
}

// readPathList reads one literal list of path strings from the project block.
func readPathList(selection *Selection, project *hclsyntax.Block, path, name string, requireItems bool) pathList {
	attribute, exists := project.Body.Attributes[name]
	if !exists {
		addLocatedFinding(selection, path, project.DefRange(), "The project block is missing "+name+".")
		return pathList{}
	}
	tuple, isTuple := attribute.Expr.(*hclsyntax.TupleConsExpr)
	if !isTuple {
		addLocatedFinding(selection, path, attribute.Expr.Range(), name+" must be a literal list of strings.")
		return pathList{}
	}
	if requireItems && len(tuple.Exprs) == 0 {
		addLocatedFinding(selection, path, attribute.Expr.Range(), name+" must contain at least one path.")
		return pathList{}
	}
	return readLiteralPaths(selection, path, tuple.Exprs)
}

// readLiteralPaths keeps the list valid only when every item is a known literal string.
// It reports each item that is not.
func readLiteralPaths(selection *Selection, path string, expressions []hclsyntax.Expression) pathList {
	paths := make([]string, 0, len(expressions))
	valid := true
	for _, expression := range expressions {
		value, diagnostics := expression.Value(nil)
		if diagnostics.HasErrors() || !isKnownString(value) {
			addLocatedFinding(selection, path, expression.Range(), "A project file path must be a literal string.")
			valid = false
			continue
		}
		paths = append(paths, value.AsString())
	}
	return pathList{paths: paths, valid: valid}
}

// acquireManifestMembers resolves the listed core and attachment paths, then reads each
// physical file once. Core files are read before attachments, so a file listed in both
// roles belongs to the core role.
func acquireManifestMembers(selection *Selection, envelope manifestEnvelope, readFile fileReader) {
	root, err := manifestRoot(envelope.path)
	if err != nil {
		addPathFinding(selection, envelope.path, messageManifestDirectory)
		return
	}
	coreMembers, coreResolved := resolveMembers(selection, root, envelope.path, envelope.coreFiles, coreSuffix)
	attachmentMembers, attachmentsResolved := resolveMembers(selection, root, envelope.path, envelope.attachmentFiles, attachmentSuffix)

	seen := []os.FileInfo{envelope.info}
	coreSources, coreAcquired := acquireSources(selection, coreMembers, &seen, readFile)
	attachmentSources, attachmentsAcquired := acquireSources(selection, attachmentMembers, &seen, readFile)

	selection.Core = coreSources
	selection.Attachments = attachmentSources
	selection.CoreComplete = coreResolved && coreAcquired
	selection.AttachmentsComplete = attachmentsResolved && attachmentsAcquired
}

// manifestRoot returns the canonical directory that holds the manifest. Member paths
// must stay inside it after symbolic links are resolved.
func manifestRoot(manifestPath string) (string, error) {
	directory, err := filepath.Abs(filepath.Dir(manifestPath))
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(directory)
}

// resolveMembers checks each listed path. It returns the members that passed, sorted
// for display. The bool is false when the list is invalid or any path failed.
func resolveMembers(selection *Selection, root, manifestPath string, list pathList, suffix string) ([]fileMember, bool) {
	if !list.valid {
		return nil, false
	}
	members := make([]fileMember, 0, len(list.paths))
	resolved := true
	for _, listed := range list.paths {
		member, err := resolveManifestMember(root, manifestPath, listed, suffix)
		if err != nil {
			addPathFinding(selection, memberDisplayPath(manifestPath, listed), sentence(err))
			resolved = false
			continue
		}
		members = append(members, member)
	}
	sort.Slice(members, func(first, second int) bool {
		return members[first].displayPath < members[second].displayPath
	})
	return members, resolved
}

// sentence turns a lowercase error text into a capitalized sentence for a finding message.
func sentence(err error) string {
	text := err.Error()
	return strings.ToUpper(text[:1]) + text[1:] + "."
}

func resolveManifestMember(root, manifestPath, listed, suffix string) (fileMember, error) {
	if err := checkListedPath(listed, suffix); err != nil {
		return fileMember{}, err
	}
	candidate := filepath.Join(root, listed)
	if !isWithin(root, candidate) {
		return fileMember{}, errPathLeaves
	}
	physicalPath, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return fileMember{}, errPathUnresolved
	}
	if !isWithin(root, physicalPath) {
		return fileMember{}, errPathLeaves
	}
	info, ok := regularFileInfo(physicalPath)
	if !ok {
		return fileMember{}, errPathNotRegular
	}
	return fileMember{
		displayPath: memberDisplayPath(manifestPath, listed),
		readPath:    physicalPath,
		info:        info,
	}, nil
}

// checkListedPath checks the lexical rules for one listed path. It runs before any
// file system access.
func checkListedPath(listed, suffix string) error {
	switch {
	case listed == "" || filepath.IsAbs(listed):
		return errPathNotRelative
	case hasGlobPattern(listed):
		return errPathHasGlob
	case !strings.HasSuffix(listed, suffix):
		return errPathWrongSuffix
	}
	return nil
}

func memberDisplayPath(manifestPath, listed string) string {
	if filepath.IsAbs(listed) {
		return listed
	}
	return filepath.Join(filepath.Dir(manifestPath), listed)
}

func hasGlobPattern(listed string) bool {
	return strings.ContainsAny(listed, "*?[")
}

// isWithin reports whether candidate is root or a path below root.
func isWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// sortedAttributes returns attributes in source order so that findings are stable.
func sortedAttributes(attributes hclsyntax.Attributes) []*hclsyntax.Attribute {
	sorted := make([]*hclsyntax.Attribute, 0, len(attributes))
	for _, attribute := range attributes {
		sorted = append(sorted, attribute)
	}
	sort.Slice(sorted, func(first, second int) bool {
		return sorted[first].NameRange.Start.Byte < sorted[second].NameRange.Start.Byte
	})
	return sorted
}

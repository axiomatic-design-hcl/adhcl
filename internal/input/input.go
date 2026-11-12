package input

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

const (
	coreSuffix       = ".ad.hcl"
	attachmentSuffix = ".ad.ext.hcl"
	manifestSuffix   = ".ad.project.hcl"

	codeInputFailure      = "input_failure"
	codeDuplicateIdentity = "duplicate_identity"
	codeInvalidStructure  = "invalid_structure"

	messageInputUnreadable     = "The input path cannot be read."
	messageInputNotRegular     = "The input is not a regular file or directory."
	messageInputWrongSuffix    = "The input file must be a core file or project manifest."
	messageDirectoryUnreadable = "The core directory cannot be read."
	messageDirectoryEmpty      = "The directory contains no immediate core files."
	messageMemberNotRegular    = "The selected path is not a readable regular file."
	messageFileUnreadable      = "The selected file cannot be read."
	messageDuplicatePhysical   = "The selected path repeats a physical file."
)

// Selection holds the sources chosen from one input path. Core and Attachments keep
// the sources that were read. CoreComplete and AttachmentsComplete report whether every
// selected file in that role was read.
type Selection struct {
	Input               string
	Core                []model.Source
	Attachments         []model.Source
	Manifest            *syntax.Unit
	Findings            []result.Finding
	CoreComplete        bool
	AttachmentsComplete bool
}

// fileReader reads one physical file. Tests inject it to observe reads.
type fileReader func(path string) ([]byte, error)

// fileMember is one selected file. displayPath is the path the user selected and
// readPath is the physical path that is read.
type fileMember struct {
	displayPath string
	readPath    string
	info        os.FileInfo
}

// Select selects one core file, the immediate core files of a directory, or the
// files listed by an explicit project manifest.
func Select(path string) Selection {
	return selectWithReader(path, os.ReadFile)
}

func selectWithReader(path string, readFile fileReader) Selection {
	selection := Selection{
		Input:               path,
		AttachmentsComplete: true,
	}
	info, err := os.Stat(path)
	if err != nil {
		addPathFinding(&selection, path, messageInputUnreadable)
		return selection
	}

	switch {
	case info.IsDir():
		selectDirectory(&selection, path, readFile)
	case !info.Mode().IsRegular():
		addPathFinding(&selection, path, messageInputNotRegular)
	case strings.HasSuffix(path, manifestSuffix):
		selectManifest(&selection, path, readFile)
	case strings.HasSuffix(path, coreSuffix):
		selectSingleCore(&selection, path, info, readFile)
	default:
		addPathFinding(&selection, path, messageInputWrongSuffix)
	}
	return selection
}

func selectSingleCore(selection *Selection, path string, info os.FileInfo, readFile fileReader) {
	member := fileMember{displayPath: path, readPath: path, info: info}
	acquireCore(selection, []fileMember{member}, true, readFile)
}

func selectDirectory(selection *Selection, directory string, readFile fileReader) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		addPathFinding(selection, directory, messageDirectoryUnreadable)
		return
	}
	names := coreEntryNames(entries)
	if len(names) == 0 {
		addPathFinding(selection, directory, messageDirectoryEmpty)
		return
	}
	members, resolved := directoryMembers(selection, directory, names)
	acquireCore(selection, members, resolved, readFile)
}

// coreEntryNames returns the immediate entries that carry the core suffix. os.ReadDir
// returns entries sorted by name.
func coreEntryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), coreSuffix) {
			names = append(names, entry.Name())
		}
	}
	return names
}

// directoryMembers stats each core entry. The bool is false when any entry is not a
// readable regular file.
func directoryMembers(selection *Selection, directory string, names []string) ([]fileMember, bool) {
	members := make([]fileMember, 0, len(names))
	resolved := true
	for _, name := range names {
		memberPath := filepath.Join(directory, name)
		info, ok := regularFileInfo(memberPath)
		if !ok {
			addPathFinding(selection, memberPath, messageMemberNotRegular)
			resolved = false
			continue
		}
		members = append(members, fileMember{displayPath: memberPath, readPath: memberPath, info: info})
	}
	return members, resolved
}

// acquireCore reads core members and records the core result on the selection.
func acquireCore(selection *Selection, members []fileMember, resolved bool, readFile fileReader) {
	var seen []os.FileInfo
	sources, acquired := acquireSources(selection, members, &seen, readFile)
	selection.Core = sources
	selection.CoreComplete = resolved && acquired
}

// acquireSources reads each member once. It skips a member whose physical file an
// earlier member already owns. The bool is false when any member was skipped or failed.
func acquireSources(selection *Selection, members []fileMember, seen *[]os.FileInfo, readFile fileReader) ([]model.Source, bool) {
	sources := make([]model.Source, 0, len(members))
	complete := true
	for _, member := range members {
		source, ok := acquireSource(selection, member, seen, readFile)
		if !ok {
			complete = false
			continue
		}
		sources = append(sources, source)
	}
	return sources, complete
}

func acquireSource(selection *Selection, member fileMember, seen *[]os.FileInfo, readFile fileReader) (model.Source, bool) {
	if containsSameFile(*seen, member.info) {
		addDuplicateFinding(selection, member.displayPath)
		return model.Source{}, false
	}
	*seen = append(*seen, member.info)
	contents, err := readFile(member.readPath)
	if err != nil {
		addPathFinding(selection, member.displayPath, messageFileUnreadable)
		return model.Source{}, false
	}
	return model.Source{Path: member.displayPath, Bytes: contents}, true
}

// regularFileInfo follows symbolic links and reports whether the target is a regular file.
func regularFileInfo(path string) (os.FileInfo, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, false
	}
	return info, true
}

func containsSameFile(files []os.FileInfo, candidate os.FileInfo) bool {
	for _, file := range files {
		if os.SameFile(file, candidate) {
			return true
		}
	}
	return false
}

func hasErrors(findings []result.Finding) bool {
	for _, finding := range findings {
		if finding.Severity == result.Error {
			return true
		}
	}
	return false
}

func addPathFinding(selection *Selection, subject, message string) {
	selection.Findings = append(selection.Findings, result.Finding{
		Code:     codeInputFailure,
		Severity: result.Error,
		Subject:  subject,
		Message:  message,
	})
}

func addDuplicateFinding(selection *Selection, subject string) {
	selection.Findings = append(selection.Findings, result.Finding{
		Code:     codeDuplicateIdentity,
		Severity: result.Error,
		Subject:  subject,
		Message:  messageDuplicatePhysical,
	})
}

func addLocatedFinding(selection *Selection, subject string, sourceRange hcl.Range, message string) {
	selection.Findings = append(selection.Findings, result.Finding{
		Code:     codeInvalidStructure,
		Severity: result.Error,
		Subject:  subject,
		Message:  message,
		Range:    &sourceRange,
	})
}

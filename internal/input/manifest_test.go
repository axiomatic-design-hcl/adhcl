package input

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

func TestSelectExplicitManifestUsesOnlyListedFiles(t *testing.T) {
	// Given
	directory := t.TempDir()
	corePath := writeInputFile(t, directory, "core.ad.hcl", "selected core")
	attachmentPath := writeInputFile(t, directory, "payload.ad.ext.hcl", "selected attachment")
	writeInputFile(t, directory, "neighbor.ad.hcl", "excluded core")
	writeInputFile(t, directory, "neighbor.ad.ext.hcl", "excluded attachment")
	manifestPath := writeManifest(t, directory, `"core.ad.hcl"`, `"payload.ad.ext.hcl"`)

	// When
	selection := Select(manifestPath)

	// Then
	if !selection.CoreComplete || !selection.AttachmentsComplete {
		t.Errorf("completion is Core=%t Attachments=%t, want both complete", selection.CoreComplete, selection.AttachmentsComplete)
	}
	if selection.Manifest == nil {
		t.Fatal("Manifest is nil for an explicit manifest input")
	}
	if got, want := corePaths(selection), []string{corePath}; !reflect.DeepEqual(got, want) {
		t.Errorf("Core paths are %q, want %q", got, want)
	}
	if got, want := sourcePaths(selection.Attachments), []string{attachmentPath}; !reflect.DeepEqual(got, want) {
		t.Errorf("Attachment paths are %q, want %q", got, want)
	}
	if string(selection.Core[0].Bytes) != "selected core" {
		t.Errorf("Core bytes are %q", selection.Core[0].Bytes)
	}
	if string(selection.Attachments[0].Bytes) != "selected attachment" {
		t.Errorf("Attachment bytes are %q", selection.Attachments[0].Bytes)
	}
	if hasFinding(selection, "input_failure") {
		t.Errorf("valid manifest reported input_failure: %#v", selection.Findings)
	}
}
func TestSelectManifestKeepsCoreCompleteWhenAnAttachmentIsMissing(t *testing.T) {
	// Given
	directory := t.TempDir()
	corePath := writeInputFile(t, directory, "core.ad.hcl", "readable core")
	manifestPath := writeManifest(t, directory, `"core.ad.hcl"`, `"missing.ad.ext.hcl"`)

	// When
	selection := Select(manifestPath)

	// Then
	if got, want := corePaths(selection), []string{corePath}; !reflect.DeepEqual(got, want) {
		t.Errorf("Core paths are %q, want %q", got, want)
	}
	if !selection.CoreComplete {
		t.Error("CoreComplete is false although every selected core file was read")
	}
	if selection.AttachmentsComplete {
		t.Error("AttachmentsComplete is true although a selected attachment is missing")
	}
	if !hasFinding(selection, "input_failure") {
		t.Errorf("findings %#v do not include input_failure", selection.Findings)
	}
}

func TestSelectManifestKeepsReadableAttachmentsWhenACoreMemberIsMissing(t *testing.T) {
	// Given
	directory := t.TempDir()
	attachmentPath := writeInputFile(t, directory, "available.ad.ext.hcl", "readable attachment")
	manifestPath := writeManifest(t, directory, `"missing.ad.hcl"`, `"available.ad.ext.hcl"`)

	// When
	selection := Select(manifestPath)

	// Then
	if len(selection.Core) != 0 {
		t.Errorf("Core has %d sources, want none", len(selection.Core))
	}
	if got, want := sourcePaths(selection.Attachments), []string{attachmentPath}; !reflect.DeepEqual(got, want) {
		t.Errorf("Attachment paths are %q, want %q", got, want)
	}
	if selection.CoreComplete {
		t.Error("CoreComplete is true although a selected core member is missing")
	}
	if !selection.AttachmentsComplete {
		t.Error("AttachmentsComplete is false although every selected attachment was read")
	}
	if !hasFinding(selection, "input_failure") {
		t.Errorf("findings %#v do not include input_failure", selection.Findings)
	}
}

func TestSelectManifestRetainsReadableSourcesWhenMembersAreMissing(t *testing.T) {
	// Given
	directory := t.TempDir()
	corePath := writeInputFile(t, directory, "available.ad.hcl", "available core")
	attachmentPath := writeInputFile(t, directory, "available.ad.ext.hcl", "available attachment")
	manifestPath := writeManifest(t, directory, `"available.ad.hcl", "missing.ad.hcl"`, `"available.ad.ext.hcl", "missing.ad.ext.hcl"`)

	// When
	selection := Select(manifestPath)

	// Then
	if got, want := corePaths(selection), []string{corePath}; !reflect.DeepEqual(got, want) {
		t.Errorf("Core paths are %q, want %q", got, want)
	}
	if got, want := sourcePaths(selection.Attachments), []string{attachmentPath}; !reflect.DeepEqual(got, want) {
		t.Errorf("Attachment paths are %q, want %q", got, want)
	}
	finding := findingByCode(selection, "input_failure")
	if finding == nil {
		t.Fatalf("findings %#v do not include input_failure", selection.Findings)
	}
	if finding.Range != nil {
		t.Errorf("missing member finding has range %#v, want no range", finding.Range)
	}
	if selection.CoreComplete || selection.AttachmentsComplete {
		t.Errorf("completion is Core=%t Attachments=%t, want both incomplete", selection.CoreComplete, selection.AttachmentsComplete)
	}
}

func TestSelectManifestRejectsDirectoriesInEitherFileList(t *testing.T) {
	tests := []struct {
		name                    string
		coreFiles               string
		attachmentFiles         string
		directoryName           string
		readableName            string
		readableIsCore          bool
		wantCoreComplete        bool
		wantAttachmentsComplete bool
	}{
		{
			name:                    "core list",
			coreFiles:               `"folder.ad.hcl"`,
			attachmentFiles:         `"available.ad.ext.hcl"`,
			directoryName:           "folder.ad.hcl",
			readableName:            "available.ad.ext.hcl",
			wantAttachmentsComplete: true,
		},
		{
			name:             "attachment list",
			coreFiles:        `"available.ad.hcl"`,
			attachmentFiles:  `"folder.ad.ext.hcl"`,
			directoryName:    "folder.ad.ext.hcl",
			readableName:     "available.ad.hcl",
			readableIsCore:   true,
			wantCoreComplete: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			directory := t.TempDir()
			if err := os.Mkdir(filepath.Join(directory, test.directoryName), 0o755); err != nil {
				t.Fatal(err)
			}
			readablePath := writeInputFile(t, directory, test.readableName, "readable")
			manifestPath := writeManifest(t, directory, test.coreFiles, test.attachmentFiles)

			// When
			selection := Select(manifestPath)

			// Then
			finding := findingByCode(selection, "input_failure")
			if finding == nil {
				t.Fatalf("findings %#v do not include input_failure", selection.Findings)
			}
			if finding.Range != nil {
				t.Errorf("directory path finding has range %#v, want no range", finding.Range)
			}
			if selection.CoreComplete != test.wantCoreComplete || selection.AttachmentsComplete != test.wantAttachmentsComplete {
				t.Errorf("completion is Core=%t Attachments=%t, want Core=%t Attachments=%t", selection.CoreComplete, selection.AttachmentsComplete, test.wantCoreComplete, test.wantAttachmentsComplete)
			}

			if test.readableIsCore {
				if got, want := corePaths(selection), []string{readablePath}; !reflect.DeepEqual(got, want) {
					t.Errorf("Core paths are %q, want %q", got, want)
				}
			} else if got, want := sourcePaths(selection.Attachments), []string{readablePath}; !reflect.DeepEqual(got, want) {
				t.Errorf("Attachment paths are %q, want %q", got, want)
			}
		})
	}
}

func TestSelectManifestRejectsUnsafeAndUnsupportedPaths(t *testing.T) {
	tests := []struct {
		name string
		path func(*testing.T, string) string
	}{
		{
			name: "absolute path",
			path: func(t *testing.T, directory string) string {
				return writeInputFile(t, directory, "absolute.ad.hcl", "absolute core")
			},
		},
		{
			name: "escaping path",
			path: func(t *testing.T, directory string) string {
				writeInputFile(t, filepath.Dir(directory), "outside.ad.hcl", "outside core")
				return filepath.Join("..", "outside.ad.hcl")
			},
		},
		{
			name: "glob path",
			path: func(_ *testing.T, _ string) string { return `*.ad.hcl` },
		},
		{
			name: "nested manifest",
			path: func(t *testing.T, directory string) string {
				contents := `project "nested" {
  language = "requirements_hcl/core/1"
  core_files = ["good.ad.hcl"]
  attachment_files = []
}`
				writeInputFile(t, directory, "nested.ad.project.hcl", contents)
				return "nested.ad.project.hcl"
			},
		},
		{
			name: "wrong suffix",
			path: func(t *testing.T, directory string) string {
				writeInputFile(t, directory, "wrong.hcl", "wrong suffix")
				return "wrong.hcl"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			root := t.TempDir()
			directory := filepath.Join(root, "project")
			if err := os.Mkdir(directory, 0o755); err != nil {
				t.Fatal(err)
			}
			goodPath := writeInputFile(t, directory, "good.ad.hcl", "good source")
			invalidPath := test.path(t, directory)
			manifestPath := writeManifest(t, directory, `"good.ad.hcl", "`+invalidPath+`"`, "")

			// When
			selection := Select(manifestPath)

			// Then
			if got, want := corePaths(selection), []string{goodPath}; !reflect.DeepEqual(got, want) {
				t.Errorf("Core paths are %q, want retained source %q", got, want)
			}
			finding := findingByCode(selection, "input_failure")
			if finding == nil {
				t.Fatalf("findings %#v do not include input_failure", selection.Findings)
			}
			if finding.Range != nil {
				t.Errorf("path finding has range %#v, want no range", finding.Range)
			}
			if selection.CoreComplete {
				t.Error("CoreComplete is true after a manifest path violates its selection contract")
			}
		})
	}
}

func TestSelectManifestRejectsComputedPathExpressionsWithSourceRanges(t *testing.T) {
	tests := []struct {
		name       string
		expression string
	}{
		{name: "interpolation", expression: `"${var.path}"`},
		{name: "function call", expression: `file("path.txt")`},
		{name: "arithmetic", expression: `"one.ad.hcl" + "two.ad.hcl"`},
		{name: "conditional", expression: `true ? ["one.ad.hcl"] : []`},
		{name: "comprehension", expression: `[for path in [] : path]`},
		{name: "indexing", expression: `var.paths[0]`},
		{name: "splat", expression: `var.paths[*].name`},
		{name: "comparison", expression: `1 == 1`},
		{name: "template directive", expression: `"%{ if true }path.ad.hcl%{ endif }"`},
		{name: "heredoc template", expression: "<<EOT\n${var.path}\nEOT\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			directory := t.TempDir()
			manifestPath := writeManifest(t, directory, test.expression, "")

			// When
			selection := Select(manifestPath)

			// Then
			if selection.CoreComplete {
				t.Error("CoreComplete is true for an unparseable manifest expression")
			}
			finding := findingByCode(selection, "invalid_syntax")
			if finding == nil {
				t.Fatalf("findings %#v do not include invalid_syntax", selection.Findings)
			}
			if finding.Range == nil {
				t.Fatal("manifest expression finding has no source range")
			}
			if finding.Range.Filename != manifestPath || finding.Range.Start.Line < 1 || finding.Range.End.Line < finding.Range.Start.Line {
				t.Errorf("manifest expression range %#v is not positioned in %q", finding.Range, manifestPath)
			}
		})
	}
}

func TestSelectManifestRejectsSymlinkEscapeBeforeReadingEscapedSource(t *testing.T) {
	// Given
	directory := t.TempDir()
	externalDirectory := t.TempDir()
	goodPath := writeInputFile(t, directory, "good.ad.hcl", "inside")
	externalPath := writeInputFile(t, externalDirectory, "outside.ad.hcl", "outside")
	linkPath := filepath.Join(directory, "linked.ad.hcl")
	if err := os.Symlink(externalPath, linkPath); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	manifestPath := writeManifest(t, directory, `"good.ad.hcl", "linked.ad.hcl"`, "")
	externalInfo, err := os.Stat(externalPath)
	if err != nil {
		t.Fatal(err)
	}
	var escapedSourceRead bool

	// When

	selection := selectWithReader(manifestPath, func(path string) ([]byte, error) {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if os.SameFile(info, externalInfo) {
			escapedSourceRead = true
		}
		return os.ReadFile(path)
	})
	// Then
	if got, want := corePaths(selection), []string{goodPath}; !reflect.DeepEqual(got, want) {
		t.Errorf("Core paths are %q, want %q", got, want)
	}
	finding := findingByCode(selection, "input_failure")
	if finding == nil {
		t.Fatalf("findings %#v do not include input_failure", selection.Findings)
	}
	if finding.Range != nil {
		t.Errorf("escape finding has range %#v, want no range", finding.Range)
	}
	if escapedSourceRead {
		t.Error("reader accessed the physical file outside the manifest directory")
	}
}

func TestSelectManifestRejectsRepeatedPhysicalFilesAcrossRoles(t *testing.T) {
	tests := []struct {
		name      string
		makeAlias func(string, string) error
	}{
		{
			name: "symbolic link",
			makeAlias: func(source, alias string) error {
				return os.Symlink(source, alias)
			},
		},
		{
			name: "hard link",
			makeAlias: func(source, alias string) error {
				return os.Link(source, alias)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			directory := t.TempDir()
			corePath := writeInputFile(t, directory, "core.ad.hcl", "one physical source")
			aliasPath := filepath.Join(directory, "alias.ad.ext.hcl")
			if err := test.makeAlias(corePath, aliasPath); err != nil {
				if test.name == "hard link" {
					t.Skipf("hard links are unavailable: %v", err)
				}
				t.Fatal(err)
			}
			manifestPath := writeManifest(t, directory, `"core.ad.hcl"`, `"alias.ad.ext.hcl"`)
			var readInfos []os.FileInfo
			var readPaths []string

			// When
			selection := selectWithReader(manifestPath, func(path string) ([]byte, error) {
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				readInfos = append(readInfos, info)
				readPaths = append(readPaths, path)
				return os.ReadFile(path)
			})

			// Then
			if got, want := corePaths(selection), []string{corePath}; !reflect.DeepEqual(got, want) {
				t.Errorf("Core paths are %q, want %q", got, want)
			}
			if len(selection.Attachments) != 0 {
				t.Errorf("Attachments has %d sources, want no duplicate physical source", len(selection.Attachments))
			}
			if !hasFinding(selection, "duplicate_identity") {
				t.Errorf("findings %#v do not include duplicate_identity", selection.Findings)
			}
			if !samePathSet(readPaths, []string{manifestPath, corePath}) {
				t.Errorf("reader paths are %q, want one read of the manifest and physical source", readPaths)
			}
			for first := range readInfos {
				for second := first + 1; second < len(readInfos); second++ {
					if os.SameFile(readInfos[first], readInfos[second]) {
						t.Errorf("reader accessed one physical file more than once: %q and %q", readPaths[first], readPaths[second])
					}
				}
			}
		})
	}
}

func TestSelectManifestReadsEachPhysicalSourceOnceAndKeepsReadBytes(t *testing.T) {
	// Given
	directory := t.TempDir()
	corePath := writeInputFile(t, directory, "core.ad.hcl", "core before change")
	attachmentPath := writeInputFile(t, directory, "payload.ad.ext.hcl", "attachment before change")
	manifestPath := writeManifest(t, directory, `"core.ad.hcl"`, `"payload.ad.ext.hcl"`)
	var readInfos []os.FileInfo
	var readPaths []string
	reader := func(path string) ([]byte, error) {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		readInfos = append(readInfos, info)
		readPaths = append(readPaths, path)
		return os.ReadFile(path)
	}

	// When
	selection := selectWithReader(manifestPath, reader)
	if err := os.WriteFile(corePath, []byte("core after change"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(attachmentPath, []byte("attachment after change"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Then
	wantPaths := []string{manifestPath, corePath, attachmentPath}
	if !samePathSet(readPaths, wantPaths) {
		t.Errorf("reader paths are %q, want one read each of %q", readPaths, wantPaths)
	}
	for first := range readInfos {
		for second := first + 1; second < len(readInfos); second++ {
			if os.SameFile(readInfos[first], readInfos[second]) {
				t.Errorf("reader accessed one physical file more than once: %q and %q", readPaths[first], readPaths[second])
			}
		}
	}
	if len(selection.Core) != 1 || string(selection.Core[0].Bytes) != "core before change" {
		t.Errorf("Core does not retain bytes read before file change: %#v", selection.Core)
	}
	if len(selection.Attachments) != 1 || string(selection.Attachments[0].Bytes) != "attachment before change" {
		t.Errorf("Attachments do not retain bytes read before file change: %#v", selection.Attachments)
	}
}

func writeManifest(t *testing.T, directory, coreFiles, attachmentFiles string) string {
	t.Helper()
	contents := fmt.Sprintf(`project "sample" {
  language = "requirements_hcl/core/1"
  core_files = [%s]
  attachment_files = [%s]
}
`, coreFiles, attachmentFiles)
	return writeInputFile(t, directory, "project.ad.project.hcl", contents)
}

func sourcePaths(sources []model.Source) []string {
	paths := make([]string, len(sources))
	for index, source := range sources {
		paths[index] = source.Path
	}
	return paths
}

func samePathSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	gotSorted := append([]string(nil), got...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)
	return reflect.DeepEqual(gotSorted, wantSorted)
}

package input

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

func TestSelectCoreFileRetainsItsBytes(t *testing.T) {
	// Given
	directory := t.TempDir()
	path := writeInputFile(t, directory, "model.ad.hcl", "design \"sample\" {}\n")

	// When
	selection := Select(path)
	if !selection.CoreComplete || !selection.AttachmentsComplete {
		t.Errorf("completion is Core=%t Attachments=%t, want both complete", selection.CoreComplete, selection.AttachmentsComplete)
	}

	// Then
	if len(selection.Core) != 1 {
		t.Fatalf("Core has %d sources, want 1", len(selection.Core))
	}
	if selection.Core[0].Path != path {
		t.Errorf("Core path is %q, want %q", selection.Core[0].Path, path)
	}
	if string(selection.Core[0].Bytes) != "design \"sample\" {}\n" {
		t.Errorf("Core bytes are %q", selection.Core[0].Bytes)
	}
	if len(selection.Attachments) != 0 {
		t.Errorf("Attachments has %d sources, want none", len(selection.Attachments))
	}
	if selection.Manifest != nil {
		t.Error("Manifest is set for a core file input")
	}
}

func TestSelectDirectoryReadsImmediateCoreFilesInSortedOrder(t *testing.T) {
	// Given
	directory := t.TempDir()
	writeInputFile(t, directory, "zeta.ad.hcl", "zeta")
	writeInputFile(t, directory, "alpha.ad.hcl", "alpha")
	writeInputFile(t, directory, "neighbor.ad.ext.hcl", "attachment")
	writeInputFile(t, directory, "neighbor.ad.project.hcl", "manifest")
	nestedDirectory := filepath.Join(directory, "nested")
	if err := os.Mkdir(nestedDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeInputFile(t, nestedDirectory, "nested.ad.hcl", "nested")

	// When
	selection := Select(directory)
	if !selection.CoreComplete || !selection.AttachmentsComplete {
		t.Errorf("completion is Core=%t Attachments=%t, want both complete", selection.CoreComplete, selection.AttachmentsComplete)
	}

	// Then
	wantPaths := []string{
		filepath.Join(directory, "alpha.ad.hcl"),
		filepath.Join(directory, "zeta.ad.hcl"),
	}
	if gotPaths := corePaths(selection); !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Errorf("Core paths are %q, want %q", gotPaths, wantPaths)
	}
	if got := []string{string(selection.Core[0].Bytes), string(selection.Core[1].Bytes)}; !reflect.DeepEqual(got, []string{"alpha", "zeta"}) {
		t.Errorf("Core bytes are %q, want alpha and zeta", got)
	}
	if len(selection.Attachments) != 0 {
		t.Errorf("Attachments has %d sources, want none", len(selection.Attachments))
	}
	if selection.Manifest != nil {
		t.Error("directory input loaded a manifest")
	}
	if hasFinding(selection, "input_failure") {
		t.Errorf("directory selection reported input_failure: %#v", selection.Findings)
	}
}
func TestSelectRejectsEmptyDirectory(t *testing.T) {
	// Given
	directory := t.TempDir()

	// When
	selection := Select(directory)

	// Then
	if len(selection.Core) != 0 {
		t.Errorf("Core has %d sources, want none", len(selection.Core))
	}
	if selection.CoreComplete {
		t.Error("CoreComplete is true for an empty directory")
	}
	if !hasFinding(selection, "input_failure") {
		t.Errorf("findings %#v do not include input_failure", selection.Findings)
	}
}

func TestSelectRejectsDirectoryWithoutImmediateCoreFiles(t *testing.T) {
	// Given
	directory := t.TempDir()
	writeInputFile(t, directory, "only.ad.ext.hcl", "attachment")
	writeInputFile(t, directory, "only.ad.project.hcl", "manifest")
	nestedDirectory := filepath.Join(directory, "nested")
	if err := os.Mkdir(nestedDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeInputFile(t, nestedDirectory, "hidden.ad.hcl", "nested core")

	// When
	selection := Select(directory)

	// Then
	if len(selection.Core) != 0 {
		t.Errorf("Core has %d sources, want none", len(selection.Core))
	}
	if !hasFinding(selection, "input_failure") {
		t.Errorf("findings %#v do not include input_failure", selection.Findings)
	}
	if selection.CoreComplete {
		t.Error("CoreComplete is true when the directory contains no immediate core files")
	}

}

func TestSelectRejectsWrongSuffixAndMissingPath(t *testing.T) {
	tests := []struct {
		name string
		path func(*testing.T, string) string
	}{
		{
			name: "wrong suffix",
			path: func(t *testing.T, directory string) string {
				return writeInputFile(t, directory, "model.hcl", "design {}")
			},
		},
		{
			name: "nonexistent path",
			path: func(_ *testing.T, directory string) string {
				return filepath.Join(directory, "missing.ad.hcl")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			path := test.path(t, t.TempDir())

			// When
			selection := Select(path)

			// Then
			if len(selection.Core) != 0 {
				t.Errorf("Core has %d sources, want none", len(selection.Core))
			}
			finding := findingByCode(selection, "input_failure")
			if finding == nil {
				t.Fatalf("findings %#v do not include input_failure", selection.Findings)
			}
			if finding.Range != nil {
				t.Errorf("path failure has range %#v, want no range", finding.Range)
			}
			if selection.CoreComplete {
				t.Error("CoreComplete is true for an invalid core input path")
			}
		})
	}
}

func writeInputFile(t *testing.T, directory, name, contents string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func corePaths(selection Selection) []string {
	paths := make([]string, len(selection.Core))
	for index, source := range selection.Core {
		paths[index] = source.Path
	}
	return paths
}

func hasFinding(selection Selection, code string) bool {
	return findingByCode(selection, code) != nil
}

func findingByCode(selection Selection, code string) *result.Finding {
	for index := range selection.Findings {
		if selection.Findings[index].Code == code {
			return &selection.Findings[index]
		}
	}
	return nil
}

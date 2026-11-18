package validate_test

import (
	"path/filepath"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

func TestResultContainsSnapshotOnlyWhenCoreAssessmentCompletes(t *testing.T) {
	valid := validate.Run(validate.Request{Path: filepath.Join("..", "..", "examples", "minimal.ad.hcl"), Profile: "source"})
	if valid.Snapshot == nil {
		t.Fatal("completed core assessment has no snapshot")
	}
	blocked := validate.Run(validate.Request{Path: filepath.Join(t.TempDir(), "missing.ad.hcl"), Profile: "source"})
	if blocked.Snapshot != nil {
		t.Fatal("blocked core assessment has a snapshot")
	}
}

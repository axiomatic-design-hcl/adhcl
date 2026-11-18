package diagram

import (
	"path/filepath"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

func TestBuildRejectsBlockedAssessment(t *testing.T) {
	if _, err := Build(validate.Result{}); err == nil {
		t.Fatal("blocked assessment accepted")
	}
}

func TestBuildIncludesDesignAndFindings(t *testing.T) {
	result := validate.Run(validate.Request{Path: filepath.Join("..", "..", "examples", "minimal.ad.hcl"), Profile: "source"})
	page, err := Build(result)
	if err != nil {
		t.Fatal(err)
	}
	if page.ID != "design.thermostat" || page.Boundary == "" {
		t.Fatalf("design: %+v", page)
	}
	found := false
	for _, finding := range page.Findings {
		if finding.Code == "missing_leaf_scenario" && finding.Severity == "warning" && finding.Location != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing finding and source location")
	}
}

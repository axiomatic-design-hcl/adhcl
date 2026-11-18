package diagram

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

func TestImpactFollowsAssessedCellsAndAcceptanceChecks(t *testing.T) {
	result := validate.Run(validate.Request{Path: filepath.Join("..", "..", "examples", "architecture.ad.hcl"), Profile: "source"})
	impacts := assessment.Impacts(result.Snapshot, result.Scopes)
	for _, impact := range impacts {
		if impact.DP != "dp.scan_admission" {
			continue
		}
		if !reflect.DeepEqual(impact.Affected, []string{"fr.accept_scan", "fr.count_pallet"}) {
			t.Fatalf("affected: %v", impact.Affected)
		}
		if !reflect.DeepEqual(impact.Needs, []string{"need.valid_scans_only", "need.exact_count"}) {
			t.Fatalf("needs: %v", impact.Needs)
		}
		if !reflect.DeepEqual(impact.Scenarios, []string{"scenario.scan_digits", "scenario.batch_counts"}) {
			t.Fatalf("scenarios: %v", impact.Scenarios)
		}
		if !reflect.DeepEqual(impact.Modules, []string{"module.scan_check", "module.batch_count"}) {
			t.Fatalf("modules: %v", impact.Modules)
		}
		return
	}
	t.Fatal("scan admission impact missing")
}

package diagram

import (
	"path/filepath"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

func TestMatrixShowsDeclaredInfluence(t *testing.T) {
	result := validate.Run(validate.Request{Path: filepath.Join("..", "..", "examples", "architecture.ad.hcl"), Profile: "source"})
	if result.Snapshot == nil {
		t.Fatalf("assessment blocked: %+v", result.Findings)
	}
	for _, matrix := range newBuilder(result).matrices() {
		if matrix.Scope != "master" {
			continue
		}
		if matrix.Classification != "decoupled" {
			t.Fatalf("classification: %s", matrix.Classification)
		}
		for _, row := range matrix.Rows {
			for _, cell := range row.Cells {
				if row.FR.ID == "fr.accept_scan" && cell.DP == "dp.batch_counter" && cell.Symbol == "0" && cell.Explanation != "" {
					return
				}
			}
		}
		t.Fatal("missing justified zero in master matrix")
	}
	t.Fatal("master matrix missing")
}

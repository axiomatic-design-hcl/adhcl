package diagram

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

func TestUnknownOverridesDefaultAndStaysPossible(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join("..", "..", "testdata", "spec", "conformance", "core-1")
	for _, name := range []string{"base.ad.hcl", "exact-unknown.ad.hcl"} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	r := validate.Run(validate.Request{Path: dir, Profile: "source"})
	p, err := Build(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range p.Matrices {
		if m.Scope != "master" {
			continue
		}
		for _, row := range m.Rows {
			for _, c := range row.Cells {
				if row.FR.ID == "fr.left" && c.DP == "dp.right_output" && c.Symbol != "?" {
					t.Errorf("unknown cell is %q", c.Symbol)
				}
				if row.FR.ID == "fr.right" && c.DP == "dp.left_output" && (c.Symbol != "0" || !strings.Contains(c.Explanation, "immutable-input")) {
					t.Errorf("default zero: %+v", c)
				}
			}
		}
	}
	for _, i := range p.Impacts {
		if i.DP == "dp.right_output" {
			if !slices.Contains(i.Possible, "fr.left") {
				t.Fatalf("unknown is not possible: %+v", i)
			}
			return
		}
	}
	t.Fatal("right output impact missing")
}

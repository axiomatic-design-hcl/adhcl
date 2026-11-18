package diagram

import (
	"bytes"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

func buildFixture(t *testing.T, request validate.Request) Page {
	t.Helper()
	page, err := Build(validate.Run(request))
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func inheritPage(t *testing.T) Page {
	return buildFixture(t, validate.Request{Path: filepath.Join("testdata", "inherit.ad.hcl"), Profile: "source"})
}

func TestChildRequirementInheritsNeedInListsAndImpact(t *testing.T) {
	page := inheritPage(t)
	var first *FRRow
	for i := range page.Lists.FRs {
		if page.Lists.FRs[i].ID == "fr.first" {
			first = &page.Lists.FRs[i]
		}
	}
	if first == nil || !first.NeedsDerived || !slices.Equal(first.Needs, []Ref{{ID: "need.outcome", Anchor: true}}) || !slices.Equal(first.NeedFrom, []Ref{{ID: "fr.root", Anchor: true}}) {
		t.Fatalf("fr.first: %+v", first)
	}
	for _, impact := range page.Impacts {
		if impact.DP == "dp.first" {
			if !slices.Equal(impact.Needs, []string{"need.outcome"}) {
				t.Fatalf("impact needs: %v", impact.Needs)
			}
			return
		}
	}
	t.Fatal("dp.first impact missing")
}

func TestHierarchyNumbersAndInactiveDP(t *testing.T) {
	page := inheritPage(t)
	var frs, dps []string
	for _, row := range page.Lists.FRs {
		frs = append(frs, row.Number+" "+row.ID)
	}
	for _, row := range page.Lists.DPs {
		dps = append(dps, row.Number+" "+row.ID)
	}
	if want := []string{"1 fr.root", "1.1 fr.first", "1.2 fr.second"}; !slices.Equal(frs, want) {
		t.Errorf("FR rows = %v, want %v", frs, want)
	}
	if want := []string{"1 dp.root", "1.1 dp.first", "1.2 dp.second", " dp.unused"}; !slices.Equal(dps, want) {
		t.Errorf("DP rows = %v, want %v", dps, want)
	}
	for _, node := range page.Map.Nodes {
		if node.ID == "dp.unused" {
			t.Error("inactive DP appears in the map")
		}
	}
	var scopes []string
	for _, m := range page.Matrices {
		scopes = append(scopes, m.Scope+" "+m.Label)
	}
	if want := []string{"design.inherit Level 1", "dp.root Level 2, under dp.root", "master All leaf FRs and leaf DPs"}; !slices.Equal(scopes, want) {
		t.Errorf("matrices = %v, want %v", scopes, want)
	}
}

func TestMatrixCellsShowDiagonalAndBasis(t *testing.T) {
	page := inheritPage(t)
	for _, m := range page.Matrices {
		if m.Scope != "dp.root" {
			continue
		}
		cells := map[string]MatrixCell{}
		for _, row := range m.Rows {
			for _, c := range row.Cells {
				cells[row.FR.ID+" "+c.DP] = c
			}
		}
		if c := cells["fr.second dp.first"]; c.Symbol != "X" || c.Diagonal {
			t.Errorf("off-diagonal cell: %+v", c)
		}
		if c := cells["fr.first dp.first"]; c.Symbol != "X" || !c.Diagonal {
			t.Errorf("diagonal cell: %+v", c)
		}
		basis := map[string]string{}
		for _, r := range m.Reasons {
			basis[r.FR.ID+" "+r.DP.ID] = r.Label
		}
		if basis["fr.first dp.first"] != labelInferred || basis["fr.second dp.second"] != labelSource {
			t.Errorf("reason basis: %v", basis)
		}
		if len(m.Order) != 2 || m.Order[0].ID != "dp.first" {
			t.Errorf("design order: %+v", m.Order)
		}
		return
	}
	t.Fatal("dp.root matrix missing")
}

func TestEveryFindingCodeHasGuideText(t *testing.T) {
	for _, code := range result.Codes() {
		if guide := codeGuides[code]; guide.Meaning == "" || guide.Action == "" {
			t.Errorf("code %s has no guide text", code)
		}
	}
}

var (
	idAttribute   = regexp.MustCompile(` id="([^"]+)"`)
	hrefAttribute = regexp.MustCompile(` href="#([^"]+)"`)
)

// TestPageLinksResolve renders each example and requires every in-page link to name
// exactly one element, so that each record has one home row.
func TestPageLinksResolve(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, request := range []validate.Request{
		{Path: filepath.Join("testdata", "inherit.ad.hcl"), Profile: "source"},
		{Path: filepath.Join(root, "examples", "minimal.ad.hcl"), Profile: "source"},
		{Path: filepath.Join(root, "examples", "architecture.ad.hcl"), Profile: "architecture"},
		{Path: filepath.Join(root, "examples", "project", "model.ad.project.hcl"), Profile: "source"},
		{Path: filepath.Join(root, "testdata", "spec", "examples", "requirements_hcl.ad.hcl"), Profile: "source", Candidate: "decision.assessment_contracts.candidate.go_hcl_target", Context: "context.semantic_changes"},
	} {
		page := buildFixture(t, request)
		var out bytes.Buffer
		if err := Render(&out, page); err != nil {
			t.Fatal(err)
		}
		ids := map[string]int{}
		for _, match := range idAttribute.FindAllSubmatch(out.Bytes(), -1) {
			ids[string(match[1])]++
		}
		for id, count := range ids {
			if count > 1 {
				t.Errorf("%s: id %q occurs %d times", request.Path, id, count)
			}
		}
		for _, match := range hrefAttribute.FindAllSubmatch(out.Bytes(), -1) {
			if ids[string(match[1])] != 1 {
				t.Errorf("%s: link #%s has no target", request.Path, match[1])
			}
		}
	}
}

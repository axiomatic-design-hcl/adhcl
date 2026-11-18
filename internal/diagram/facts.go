package diagram

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

// Fact is one sentence with a label. Its parts are plain text, record links,
// classification badges, or page links.
type Fact struct {
	Label string
	Parts []Part
}

// Part is one piece of a fact. Ref.ID, Cls, or Href is set, or none for plain text.
type Part struct {
	Text, Cls, Href string
	Ref             Ref
}

func txt(s string) Part        { return Part{Text: s} }
func cls(c string) Part        { return Part{Cls: c, Text: c} }
func link(href, s string) Part { return Part{Href: href, Text: s} }
func refPart(r Ref) Part       { return Part{Ref: r} }

// Health holds the counts of the start tab.
type Health struct {
	Errors, Warnings, Kinds, ExpectHold, ExpectTotal, Evidence int
}

// OpenItem is one count of the open-items table.
type OpenItem struct {
	Text, Target string
	Count        int
}

// FindingGroup holds the findings of one code.
type FindingGroup struct {
	Code, Severity, Meaning string
	Count                   int
	Items                   []Finding
}

// GuideExamples are records of the current model that the guide shows for each domain.
type GuideExamples struct {
	Need, FR, DP, Process []Ref
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func (b *builder) health(page Page) Health {
	h := Health{Errors: page.Errors, Warnings: page.Warnings, Evidence: len(b.core.Evidence)}
	for _, group := range page.FindingGroups {
		if group.Severity == string(result.Warning) {
			h.Kinds++
		}
	}
	for _, e := range page.Expects {
		if e.Result == "not applicable" {
			continue
		}
		h.ExpectTotal++
		if e.Result == "holds" {
			h.ExpectHold++
		}
	}
	return h
}

func (b *builder) summary(page Page) []Fact {
	var facts []Fact
	if page.Errors > 0 {
		facts = append(facts, Fact{Label: labelTool, Parts: []Part{txt(fmt.Sprintf("The tool reported %s. Fix them before you rely on the rest of the page. ", plural(page.Errors, "error", "errors"))), link("#findings", "See the findings")}})
	}
	for _, m := range page.Matrices {
		switch {
		case m.Scope == string(b.graph.Design):
			parts := []Part{txt("The top level is "), cls(m.Classification), txt(". ")}
			if m.Classification == string(assessment.Decoupled) && len(m.Order) > 1 {
				parts = append(parts, txt("Set "))
				for i, r := range m.Order {
					if i > 0 {
						parts = append(parts, txt(" before "))
					}
					parts = append(parts, refPart(r))
				}
				parts = append(parts, txt(". "))
			} else {
				parts = append(parts, txt(m.Meaning+" "))
			}
			facts = append(facts, Fact{Label: labelTool, Parts: append(parts, link("#"+m.Anchor, "See the matrix"))})
		case m.Master:
			parts := []Part{txt("The full matrix of leaf FRs and leaf DPs is "), cls(m.Classification), txt(". ")}
			if m.Unknown > 0 {
				parts = append(parts, txt(fmt.Sprintf("%d of %d cells have no declared effect. ", m.Unknown, m.Total)))
			}
			facts = append(facts, Fact{Label: labelTool, Parts: append(parts, link("#"+m.Anchor, "See the master matrix"))})
		case m.Classification == string(assessment.Coupled):
			parts := []Part{txt("The branch under "), refPart(m.ScopeRef), txt(" is "), cls(m.Classification), txt(". ")}
			if len(m.Couplings) > 0 {
				parts = append(parts, txt("The model records the trade-off in "), refPart(m.Couplings[0]), txt(". "))
			} else {
				parts = append(parts, txt("The model has no coupling record for it. "))
			}
			facts = append(facts, Fact{Label: labelTool, Parts: append(parts, link("#"+m.Anchor, "See the matrix"))})
		}
	}
	if page.Health.ExpectTotal > 0 {
		facts = append(facts, Fact{Label: labelTool, Parts: []Part{txt(fmt.Sprintf("%d of %s hold. ", page.Health.ExpectHold, plural(page.Health.ExpectTotal, "expectation", "expectations"))), link("#expectations", "See the expectations")}})
	}
	var gaps []string
	if len(b.core.Evidence) == 0 {
		gaps = append(gaps, "The model has no evidence records.")
	}
	if len(b.core.Scenarios) == 0 {
		gaps = append(gaps, "The model has no acceptance scenarios.")
	}
	if page.Lists.Measurable == 0 && len(b.core.FRs) > 0 {
		gaps = append(gaps, "No requirement has a measurable criterion.")
	}
	if len(gaps) > 0 {
		parts := make([]Part, 0, len(gaps)+1)
		for _, g := range gaps {
			parts = append(parts, txt(g+" "))
		}
		facts = append(facts, Fact{Label: labelGap, Parts: append(parts, link("#risks", "See risks and evidence"))})
	}
	return facts
}

func (b *builder) openItems(page Page) []OpenItem {
	var items []OpenItem
	for _, m := range page.Matrices {
		if m.Master {
			items = append(items, OpenItem{Text: "Matrix cells with unknown effect", Count: m.Unknown, Target: m.Anchor})
		}
	}
	counts := map[string]int{}
	for _, f := range page.Findings {
		counts[f.Code]++
	}
	for _, row := range openItemCodes {
		items = append(items, OpenItem{Text: row.Text, Count: counts[row.Code], Target: row.Target})
	}
	provisional := 0
	for _, d := range page.Decisions {
		if d.Provisional {
			provisional++
		}
	}
	items = append(items, OpenItem{Text: "Provisional decisions", Count: provisional, Target: "a-decisions"})
	open := 0
	for _, a := range b.core.Assumptions {
		if a.Status == nil || *a.Status != model.AssumptionValidated {
			open++
		}
	}
	items = append(items, OpenItem{Text: "Assumptions not yet validated", Count: open, Target: "r-assumptions"})
	return append(items, OpenItem{Text: "Evidence records", Count: len(b.core.Evidence), Target: "r-evidence"})
}

func (b *builder) risks(page Page) []Fact {
	var facts []Fact
	for _, c := range page.Couplings {
		r := c.Ref
		if c.ResidualRisk != "" {
			facts = append(facts, Fact{Label: c.Label, Parts: []Part{refPart(r), txt(": " + c.ResidualRisk)}})
		} else {
			facts = append(facts, Fact{Label: labelGap, Parts: []Part{refPart(r), txt(" states no residual risk.")}})
		}
	}
	for _, d := range page.Decisions {
		if !d.Provisional {
			continue
		}
		r := d.Ref
		parts := []Part{refPart(r), txt(" is provisional.")}
		if d.Why != "" {
			parts = append(parts, txt(" "+d.Why))
		}
		facts = append(facts, Fact{Label: d.Label, Parts: parts})
	}
	for _, m := range page.Matrices {
		if m.Classification == string(assessment.Coupled) && len(m.Couplings) == 0 {
			facts = append(facts, Fact{Label: labelTool, Parts: []Part{txt("The branch under "), refPart(m.ScopeRef), txt(" is coupled, and the model has no coupling record for it.")}})
		}
		if m.Master && m.Classification == string(assessment.Unresolved) {
			facts = append(facts, Fact{Label: labelTool, Parts: []Part{txt("The master matrix is unresolved. A change to one leaf DP can affect other leaf FRs. See "), link("#t-impact", "change impact"), txt(".")}})
		}
	}
	return facts
}

// groupFindings groups findings by code: errors first, then the larger groups, then
// the order of first occurrence. It returns the guide text of each code in the same order.
func groupFindings(findings []Finding) ([]FindingGroup, []CodeGuide) {
	var groups []FindingGroup
	index := map[string]int{}
	for _, f := range findings {
		i, ok := index[f.Code]
		if !ok {
			i = len(groups)
			index[f.Code] = i
			groups = append(groups, FindingGroup{Code: f.Code, Severity: f.Severity, Meaning: codeGuides[f.Code].Meaning})
		}
		groups[i].Count++
		groups[i].Items = append(groups[i].Items, f)
	}
	slices.SortStableFunc(groups, func(a, b FindingGroup) int {
		if (a.Severity == string(result.Error)) != (b.Severity == string(result.Error)) {
			if a.Severity == string(result.Error) {
				return -1
			}
			return 1
		}
		return cmp.Compare(b.Count, a.Count)
	})
	codes := make([]CodeGuide, 0, len(groups))
	for _, g := range groups {
		guide := codeGuides[g.Code]
		guide.Code = g.Code
		codes = append(codes, guide)
	}
	return groups, codes
}

func (b *builder) guideExamples(page Page) GuideExamples {
	var g GuideExamples
	if len(page.Lists.Needs) > 0 {
		g.Need = []Ref{page.Lists.Needs[0].Ref}
	}
	if len(page.Lists.Capabilities) > 0 {
		g.FR = []Ref{page.Lists.Capabilities[0].Ref}
	}
	if len(page.Lists.DPs) > 0 {
		g.DP = []Ref{page.Lists.DPs[0].Ref}
	}
	if len(page.Lists.PVs) > 0 {
		g.Process = append(g.Process, page.Lists.PVs[0].Ref)
	}
	if len(page.Lists.Modules) > 0 {
		g.Process = append(g.Process, page.Lists.Modules[0].Ref)
	}
	return g
}

func (b *builder) tabs(page Page) []Tab {
	tabs := slices.Clone(tabDefinitions)
	l := page.Lists
	has := map[string]string{
		tabBusiness:     fmt.Sprintf("%s, %s, %s", plural(len(l.Needs), "need", "needs"), plural(len(l.Constraints), "constraint", "constraints"), plural(len(l.Stakeholders), "stakeholder", "stakeholders")),
		tabProduct:      fmt.Sprintf("%s, %s, %s", plural(len(l.FRs), "functional requirement", "functional requirements"), plural(len(l.Criteria), "acceptance criterion", "acceptance criteria"), plural(len(l.Scenarios), "scenario", "scenarios")),
		tabArchitecture: fmt.Sprintf("%s, %s, %s", plural(len(l.DPs), "design parameter", "design parameters"), plural(len(page.Matrices), "design matrix", "design matrices"), plural(len(page.Decisions), "decision", "decisions")),
		tabEngineering:  fmt.Sprintf("%s, %s, %s", plural(len(l.PVs), "process variable", "process variables"), plural(len(l.Modules), "module", "modules"), plural(len(page.Flows), "flow", "flows")),
		tabTrace:        "Trace map, change impact list, trace table",
		tabRisks:        fmt.Sprintf("%s, %s, %s", plural(len(l.Assumptions), "assumption", "assumptions"), plural(len(page.Couplings), "coupling record", "coupling records"), plural(len(page.Findings), "finding", "findings")),
		tabGuide:        "Axiomatic Design cheat sheet and page legend",
	}
	for i := range tabs {
		tabs[i].Has = has[tabs[i].ID]
	}
	return tabs
}

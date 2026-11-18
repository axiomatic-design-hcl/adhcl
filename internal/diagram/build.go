// Package diagram builds and renders a view of an assessed design.
package diagram

import (
	"errors"
	"fmt"
	"slices"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

// Page is the presentation data for one assessed design. It holds plain values only.
// The templates format it, and they compute nothing.
type Page struct {
	ID, Boundary, Level, Version, Candidate, Context string
	Files                                            []string
	Sources                                          []Item
	Tabs                                             []Tab
	Summary                                          []Fact
	Health                                           Health
	TopLevel                                         *Matrix
	Lists                                            Lists
	Matrices                                         []Matrix
	Decisions                                        []DecisionView
	Couplings                                        []CouplingView
	Expects                                          []ExpectRow
	Flows                                            []FlowView
	Junctions                                        []JunctionRow
	Cards                                            Cards
	Impacts                                          []assessment.Impact
	Trace                                            []TraceRow
	Open                                             []OpenItem
	Risks                                            []Fact
	Findings                                         []Finding
	FindingGroups                                    []FindingGroup
	Codes                                            []CodeGuide
	Guide                                            GuideExamples
	Errors, Warnings                                 int
	Map                                              TraceMap
	DefaultImpact                                    string
}

// Finding is a diagnostic with its source location.
type Finding struct {
	Code, Severity, Message, Location string
	Subject                           Ref
}

// Ref is a mention of a model record. Anchor is true when the record has a home row
// on the page, so that the mention can link to it.
type Ref struct {
	ID, Alias string
	Anchor    bool
}

// builder holds the indexes that every part of the page reads.
type builder struct {
	res      validate.Result
	core     *model.Core
	graph    *model.Graph
	aliases  map[model.ID]string
	anchors  map[model.ID]bool
	needs    map[model.ID]assessment.NeedTrace
	scopes   map[model.ID]*assessment.ScopeResult
	frs      map[model.ID]*model.FR
	dps      map[model.ID]*model.DP
	parentFR map[model.ID]model.ID // FR -> first FR that its parent DP controls
	numbers  map[model.ID]string   // FR and DP -> hierarchy number
	depths   map[model.ID]int      // sibling set scope -> level, from 1
}

func newBuilder(res validate.Result) *builder {
	graph := res.Snapshot.Graph
	b := &builder{res: res, core: &graph.Core, graph: graph, aliases: map[model.ID]string{}, anchors: map[model.ID]bool{}, scopes: map[model.ID]*assessment.ScopeResult{}, frs: map[model.ID]*model.FR{}, dps: map[model.ID]*model.DP{}, parentFR: map[model.ID]model.ID{}, numbers: map[model.ID]string{}, depths: map[model.ID]int{}}
	b.needs = assessment.EffectiveNeeds(b.core)
	for _, declaration := range b.core.Declarations() {
		b.anchors[declaration.ID] = declaration.ID != graph.Design
	}
	headers := func(h model.Header) {
		if h.Alias != nil {
			b.aliases[h.ID] = *h.Alias
		}
	}
	for i := range b.core.FRs {
		b.frs[b.core.FRs[i].ID] = &b.core.FRs[i]
		headers(b.core.FRs[i].Header)
	}
	for i := range b.core.DPs {
		b.dps[b.core.DPs[i].ID] = &b.core.DPs[i]
		headers(b.core.DPs[i].Header)
	}
	for _, module := range b.core.Modules {
		headers(module.Header)
	}
	for _, fr := range b.core.FRs {
		if fr.Parent == nil {
			continue
		}
		if dp := b.dps[fr.Parent.Target]; dp != nil && len(dp.Controls) > 0 {
			b.parentFR[fr.ID] = dp.Controls[0].Target
		}
	}
	for i := range res.Scopes {
		b.scopes[res.Scopes[i].Matrix.Scope] = &res.Scopes[i]
	}
	b.number(graph.Design, "", 1)
	return b
}

// number walks the sibling-set tree and gives each FR and DP its hierarchy number.
// A DP takes the number of the first FR that it controls. Inactive DPs get no number.
func (b *builder) number(scope model.ID, prefix string, depth int) {
	set, ok := b.graph.SiblingSet(scope)
	if !ok || b.depths[scope] != 0 {
		return
	}
	b.depths[scope] = depth
	for i, fr := range set.FRs {
		number := fmt.Sprintf("%s%d", prefix, i+1)
		b.numbers[fr] = number
		for _, dp := range set.DPs {
			if _, done := b.numbers[dp]; done || !b.controls(dp, fr) || !b.res.Snapshot.ActiveDPs[dp] {
				continue
			}
			b.numbers[dp] = number
			b.number(dp, number+".", depth+1)
		}
	}
}

func (b *builder) controls(dp, fr model.ID) bool {
	record := b.dps[dp]
	return record != nil && containsRef(record.Controls, fr)
}

func (b *builder) ref(id model.ID) Ref {
	return Ref{ID: string(id), Alias: b.aliases[id], Anchor: b.anchors[id]}
}

func (b *builder) refs(ids []model.ID) []Ref {
	out := make([]Ref, 0, len(ids))
	for _, id := range ids {
		out = append(out, b.ref(id))
	}
	return out
}

func (b *builder) modelRefs(refs []model.Ref) []Ref {
	out := make([]Ref, 0, len(refs))
	for _, r := range refs {
		out = append(out, b.ref(r.Target))
	}
	return out
}

// Build derives presentation data without changing the validation result.
func Build(res validate.Result) (Page, error) {
	if res.Snapshot == nil || res.Snapshot.Graph == nil {
		return Page{}, errors.New("core assessment is blocked")
	}
	b := newBuilder(res)
	page := Page{ID: string(b.graph.Design), Version: res.ToolVersion, Candidate: res.Candidate, Context: res.Context, Impacts: assessment.Impacts(res.Snapshot, res.Scopes)}
	for _, design := range b.core.Designs {
		if design.ID == b.graph.Design {
			page.Boundary = design.Boundary
			if design.Level != nil {
				page.Level = *design.Level
			}
			break
		}
	}
	page.Files = b.files()
	for _, diagnostic := range res.Findings {
		finding := Finding{Code: diagnostic.Code, Severity: string(diagnostic.Severity), Message: diagnostic.Message, Subject: b.ref(model.ID(diagnostic.Subject))}
		if diagnostic.Range != nil {
			finding.Location = location(*diagnostic.Range)
		}
		switch diagnostic.Severity {
		case result.Error:
			page.Errors++
		case result.Warning:
			page.Warnings++
		}
		page.Findings = append(page.Findings, finding)
	}
	page.Lists = b.lists(page.Impacts)
	page.Matrices = b.matrices()
	for i := range page.Matrices {
		if page.Matrices[i].Scope == string(b.graph.Design) {
			page.TopLevel = &page.Matrices[i]
		}
	}
	page.Decisions = b.decisions()
	page.Couplings = b.couplings()
	page.Expects = b.expects()
	for i := range page.Matrices {
		for _, e := range page.Expects {
			if e.ScopeID == page.Matrices[i].Scope {
				page.Matrices[i].Expects = append(page.Matrices[i].Expects, e)
			}
		}
	}
	page.Flows = b.flows()
	page.Junctions = b.junctions()
	page.Cards = b.cards()
	page.Sources = page.Cards.Sources
	page.Trace = b.traceRows()
	page.FindingGroups, page.Codes = groupFindings(page.Findings)
	page.Health = b.health(page)
	page.Summary = b.summary(page)
	page.Open = b.openItems(page)
	page.Risks = b.risks(page)
	page.Guide = b.guideExamples(page)
	page.Tabs = b.tabs(page)
	for _, impact := range page.Impacts {
		if len(impact.Affected) > 1 {
			page.DefaultImpact = impact.DP
			break
		}
	}
	page.Map = buildTraceMap(res, page.Impacts, b.needs)
	return page, nil
}

// files returns the distinct file names of the authored records, sorted.
func (b *builder) files() []string {
	var files []string
	for _, declaration := range b.core.Declarations() {
		if name := declaration.Range.Filename; name != "" && !slices.Contains(files, name) {
			files = append(files, name)
		}
	}
	slices.Sort(files)
	return files
}

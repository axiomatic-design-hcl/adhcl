package diagram

import (
	"fmt"
	"slices"
	"strings"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/hashicorp/hcl/v2"
)

// Lists holds one row slice for each list view of the page.
type Lists struct {
	Stakeholders []StakeholderRow
	Needs        []NeedRow
	Constraints  []ConstraintRow
	Capabilities []FRRow
	FRs          []FRRow
	Criteria     []CriterionRow
	Scenarios    []ScenarioRow
	DPs          []DPRow
	PVs          []PVRow
	Modules      []ModuleRow
	Assumptions  []AssumptionRow
	Impacts      []ImpactRow
	// Counts used by captions.
	Leaves, Measurable, Rejected, IncompleteModules int
	// LeavesWithoutScenario lists the active leaf FRs that no scenario verifies.
	LeavesWithoutScenario []Ref
}

type StakeholderRow struct {
	Ref
	Role, Label, Location string
	Needs                 []Ref
}

type NeedRow struct {
	Ref
	Statement, Why, Label, Location string
	Stakeholder, ServedBy           []Ref
}

type ConstraintRow struct {
	Ref
	Kind, Statement, Label, Location string
	Limits                           []Ref
}

type FRRow struct {
	Ref
	Number, Statement, Location string
	Depth                       int
	Leaf, Active, NoScenario    bool
	Needs, NeedFrom             []Ref
	NeedsDerived                bool
	Criteria, ControlledBy      []Ref
	BoundedBy                   []Ref
}

type CriterionRow struct {
	Ref
	FR                                  Ref
	Statement, Measure, Label, Location string
	Under, AllocatedFrom, Scenarios     []Ref
}

type ScenarioRow struct {
	Ref
	Kind, Description, Label, Location   string
	Steps                                []string
	Verifies, Vary, Affected, Unaffected []Ref
	Under, Evidence                      []Ref
}

type DPRow struct {
	Ref
	Number, Statement, Decision, Why, Location string
	Depth                                      int
	Active, Leaf                               bool
	Children, ChildMatrix                      string
	Controls, BoundedBy, HoldsIf, RealizedBy   []Ref
	OwnsState, ReadsState                      []Ref
}

type PVRow struct {
	Ref
	Kind, Note, Address, Label, Location string
	Realizes                             []Ref
}

type ModuleRow struct {
	Ref
	Failure, Location             string
	Incomplete                    bool
	Inputs, Outputs, Invariants   []string
	Realizes, Contains, Delivers  []Ref
	Owns, Depends, Checks, Within []Ref
}

type AssumptionRow struct {
	Ref
	Statement, Status, Label, Location string
	ReliedOn                           []Ref
	ReliedText                         []string
}

type ImpactRow struct {
	DP                                                 Ref
	Affected, Possible, Modules, PVs, Needs, Scenarios []Ref
	NeedsDerived                                       bool
}

func location(r hcl.Range) string {
	if r.Filename == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", r.Filename, r.Start.Line)
}

// originLabel gives the page label of an authored record.
func originLabel(origin *model.Origin) string {
	if origin != nil && *origin == model.OriginInferred {
		return labelInferred
	}
	return labelSource
}

func (b *builder) lists(impacts []assessment.Impact) Lists {
	var l Lists
	b.stakeholderRows(&l)
	b.needRows(&l)
	b.constraintRows(&l)
	b.frRows(&l)
	b.criterionRows(&l)
	b.scenarioRows(&l)
	b.dpRows(&l)
	b.pvRows(&l)
	b.moduleRows(&l)
	b.assumptionRows(&l)
	b.impactRows(&l, impacts)
	return l
}

func (b *builder) stakeholderRows(l *Lists) {
	for _, s := range b.core.Stakeholders {
		row := StakeholderRow{Ref: b.ref(s.ID), Role: s.Role, Label: originLabel(s.Origin), Location: location(s.Range)}
		for _, need := range b.core.Needs {
			if need.Stakeholder != nil && need.Stakeholder.Target == s.ID {
				row.Needs = append(row.Needs, b.ref(need.ID))
			}
		}
		l.Stakeholders = append(l.Stakeholders, row)
	}
}

func (b *builder) needRows(l *Lists) {
	for _, need := range b.core.Needs {
		row := NeedRow{Ref: b.ref(need.ID), Statement: need.Statement, Why: text(need.Why), Label: originLabel(need.Origin), Location: location(need.Range)}
		if need.Stakeholder != nil {
			row.Stakeholder = []Ref{b.ref(need.Stakeholder.Target)}
		}
		for _, fr := range b.core.FRs {
			if containsRef(fr.Need.Items, need.ID) {
				row.ServedBy = append(row.ServedBy, b.ref(fr.ID))
			}
		}
		l.Needs = append(l.Needs, row)
	}
}

func (b *builder) constraintRows(l *Lists) {
	for _, c := range b.core.Constraints {
		row := ConstraintRow{Ref: b.ref(c.ID), Kind: text(c.Kind), Statement: c.Statement, Label: originLabel(c.Origin), Location: location(c.Range)}
		for _, dp := range b.core.DPs {
			if containsRef(dp.BoundedBy, c.ID) {
				row.Limits = append(row.Limits, b.ref(dp.ID))
			}
		}
		for _, fr := range b.core.FRs {
			if containsRef(fr.BoundedBy, c.ID) {
				row.Limits = append(row.Limits, b.ref(fr.ID))
			}
		}
		l.Constraints = append(l.Constraints, row)
	}
}

// verifies reports whether a scenario verifies the FR or one of its criteria.
func verifies(scenario model.Scenario, fr *model.FR) bool {
	if containsRef(scenario.Verifies.Items, fr.ID) {
		return true
	}
	for _, criterion := range fr.Criteria {
		if containsRef(scenario.Verifies.Items, criterion.ID) {
			return true
		}
	}
	return false
}

// isLeaf reports whether no DP that controls the FR has a child sibling set.
func (b *builder) isLeafFR(fr model.ID) bool {
	for _, dp := range b.core.DPs {
		if containsRef(dp.Controls, fr) && len(b.graph.Children[dp.ID]) > 0 {
			return false
		}
	}
	return true
}

func (b *builder) frRow(fr *model.FR, depth int) FRRow {
	row := FRRow{Ref: b.ref(fr.ID), Number: b.numbers[fr.ID], Depth: depth, Statement: fr.Statement, Location: location(fr.Range), Leaf: b.isLeafFR(fr.ID), Active: b.res.Snapshot.ActiveFRs == nil || b.res.Snapshot.ActiveFRs[fr.ID]}
	trace := b.needs[fr.ID]
	row.Needs = b.refs(trace.Needs)
	row.NeedFrom = b.refs(trace.From)
	row.NeedsDerived = trace.Inherited()
	for _, criterion := range fr.Criteria {
		row.Criteria = append(row.Criteria, b.ref(criterion.ID))
	}
	for _, dp := range b.core.DPs {
		if containsRef(dp.Controls, fr.ID) && b.res.Snapshot.ActiveDPs[dp.ID] {
			row.ControlledBy = append(row.ControlledBy, b.ref(dp.ID))
		}
	}
	row.BoundedBy = b.modelRefs(fr.BoundedBy)
	if row.Leaf {
		row.NoScenario = true
		for _, scenario := range b.core.Scenarios {
			if verifies(scenario, fr) {
				row.NoScenario = false
				break
			}
		}
	}
	return row
}

// frRows lists the FRs in hierarchy order: each FR, then the FRs under its DPs.
func (b *builder) frRows(l *Lists) {
	seen := map[model.ID]bool{}
	var walk func(scope model.ID, depth int)
	walk = func(scope model.ID, depth int) {
		set, ok := b.graph.SiblingSet(scope)
		if !ok {
			return
		}
		for _, id := range set.FRs {
			fr := b.frs[id]
			if fr == nil || seen[id] {
				continue
			}
			seen[id] = true
			row := b.frRow(fr, depth)
			l.FRs = append(l.FRs, row)
			if depth == 0 {
				l.Capabilities = append(l.Capabilities, row)
			}
			for _, dp := range set.DPs {
				if b.controls(dp, id) {
					walk(dp, depth+1)
				}
			}
		}
	}
	walk(b.graph.Design, 0)
	for i := range b.core.FRs {
		if !seen[b.core.FRs[i].ID] {
			l.FRs = append(l.FRs, b.frRow(&b.core.FRs[i], 0))
		}
	}
	for _, row := range l.FRs {
		if row.Leaf && row.Active {
			l.Leaves++
			if row.NoScenario {
				l.LeavesWithoutScenario = append(l.LeavesWithoutScenario, row.Ref)
			}
		}
	}
}

func (b *builder) criterionRows(l *Lists) {
	for _, fr := range b.core.FRs {
		for _, criterion := range fr.Criteria {
			row := CriterionRow{Ref: b.ref(criterion.ID), FR: b.ref(fr.ID), Statement: text(criterion.Statement), Label: originLabel(criterion.Origin), Location: location(criterion.Range)}
			if criterion.Metric != nil && criterion.Op != nil && criterion.Value != nil {
				parts := []string{*criterion.Metric, string(*criterion.Op), criterionValue(*criterion.Value)}
				if criterion.Unit != nil {
					parts = append(parts, *criterion.Unit)
				}
				row.Measure = strings.Join(parts, " ")
				l.Measurable++
			}
			if criterion.Under != nil {
				row.Under = []Ref{b.ref(criterion.Under.Target)}
			}
			if criterion.AllocatedFrom != nil {
				row.AllocatedFrom = []Ref{b.ref(criterion.AllocatedFrom.Target)}
			}
			for _, scenario := range b.core.Scenarios {
				if containsRef(scenario.Verifies.Items, criterion.ID) {
					row.Scenarios = append(row.Scenarios, b.ref(scenario.ID))
				}
			}
			l.Criteria = append(l.Criteria, row)
		}
	}
}

func (b *builder) scenarioRows(l *Lists) {
	for _, s := range b.core.Scenarios {
		row := ScenarioRow{Ref: b.ref(s.ID), Kind: string(s.Kind), Description: text(s.Description), Label: originLabel(s.Origin), Location: location(s.Range), Steps: s.Steps.Items, Verifies: b.modelRefs(s.Verifies.Items), Vary: b.modelRefs(s.Vary), Affected: b.modelRefs(s.ExpectAffected), Unaffected: b.modelRefs(s.ExpectUnaffected)}
		if s.Under != nil {
			row.Under = []Ref{b.ref(s.Under.Target)}
		}
		for _, e := range b.core.Evidence {
			if e.Scenario.Target == s.ID {
				row.Evidence = append(row.Evidence, b.ref(e.ID))
			}
		}
		l.Scenarios = append(l.Scenarios, row)
	}
}

func (b *builder) dpRow(dp *model.DP, depth int) DPRow {
	row := DPRow{Ref: b.ref(dp.ID), Number: b.numbers[dp.ID], Depth: depth, Statement: dp.Responsibility, Decision: string(dp.Decision), Why: text(dp.Why), Location: location(dp.Range), Active: b.res.Snapshot.ActiveDPs[dp.ID], Controls: b.modelRefs(dp.Controls), BoundedBy: b.modelRefs(dp.BoundedBy), HoldsIf: b.modelRefs(dp.HoldsIf), OwnsState: b.modelRefs(dp.OwnsState.Items), ReadsState: b.modelRefs(dp.ReadsState)}
	if !row.Active {
		row.Number = ""
	}
	if scope := b.scopes[dp.ID]; scope != nil {
		row.Children, row.ChildMatrix = string(scope.Classification), matrixAnchor(dp.ID)
	} else {
		row.Leaf = len(b.graph.Children[dp.ID]) == 0
	}
	for _, module := range b.core.Modules {
		if containsRef(module.Realizes.Items, dp.ID) {
			row.RealizedBy = append(row.RealizedBy, b.ref(module.ID))
		}
	}
	for _, pv := range b.core.PVs {
		if containsRef(pv.Realizes, dp.ID) {
			row.RealizedBy = append(row.RealizedBy, b.ref(pv.ID))
		}
	}
	return row
}

// dpRows lists the DPs in hierarchy order. An inactive DP follows the active DP of the
// same FR, without a number.
func (b *builder) dpRows(l *Lists) {
	seen := map[model.ID]bool{}
	var walk func(scope model.ID, depth int)
	walk = func(scope model.ID, depth int) {
		set, ok := b.graph.SiblingSet(scope)
		if !ok {
			return
		}
		for _, fr := range set.FRs {
			var controllers []model.ID
			for _, dp := range b.core.DPs {
				if containsRef(dp.Controls, fr) && !seen[dp.ID] {
					controllers = append(controllers, dp.ID)
				}
			}
			slices.SortStableFunc(controllers, func(x, y model.ID) int {
				return boolRank(b.res.Snapshot.ActiveDPs[x]) - boolRank(b.res.Snapshot.ActiveDPs[y])
			})
			for _, id := range controllers {
				seen[id] = true
				l.DPs = append(l.DPs, b.dpRow(b.dps[id], depth))
				walk(id, depth+1)
			}
		}
	}
	walk(b.graph.Design, 0)
	for i := range b.core.DPs {
		if !seen[b.core.DPs[i].ID] {
			l.DPs = append(l.DPs, b.dpRow(&b.core.DPs[i], 0))
		}
	}
	for _, row := range l.DPs {
		if !row.Active {
			l.Rejected++
		}
	}
}

func boolRank(active bool) int {
	if active {
		return 0
	}
	return 1
}

func (b *builder) pvRows(l *Lists) {
	for _, pv := range b.core.PVs {
		l.PVs = append(l.PVs, PVRow{Ref: b.ref(pv.ID), Kind: text(pv.Kind), Note: text(pv.Note), Address: text(pv.Address), Label: originLabel(pv.Origin), Location: location(pv.Range), Realizes: b.modelRefs(pv.Realizes)})
	}
}

func (b *builder) moduleRows(l *Lists) {
	for _, m := range b.core.Modules {
		row := ModuleRow{Ref: b.ref(m.ID), Failure: text(m.FailureBehavior), Location: location(m.Range), Inputs: m.Inputs.Items, Outputs: m.Outputs.Items, Invariants: m.Invariants.Items, Realizes: b.modelRefs(m.Realizes.Items), Contains: b.modelRefs(m.Contains.Items), Owns: b.modelRefs(m.OwnsState.Items), Depends: b.modelRefs(m.Dependencies.Items), Checks: b.modelRefs(m.AcceptanceChecks.Items)}
		if m.Within != nil {
			row.Within = []Ref{b.ref(m.Within.Target)}
		}
		for _, r := range m.Realizes.Items {
			if dp := b.dps[r.Target]; dp != nil {
				row.Delivers = append(row.Delivers, b.modelRefs(dp.Controls)...)
			}
		}
		row.Incomplete = len(row.Inputs) == 0 || len(row.Outputs) == 0 || row.Failure == ""
		if row.Incomplete {
			l.IncompleteModules++
		}
		l.Modules = append(l.Modules, row)
	}
}

func (b *builder) assumptionRows(l *Lists) {
	for _, a := range b.core.Assumptions {
		row := AssumptionRow{Ref: b.ref(a.ID), Statement: a.Statement, Label: originLabel(a.Origin), Location: location(a.Range)}
		if a.Status != nil {
			row.Status = string(*a.Status)
		}
		for _, dp := range b.core.DPs {
			if containsRef(dp.HoldsIf, a.ID) {
				row.ReliedOn = append(row.ReliedOn, b.ref(dp.ID))
			}
		}
		for _, c := range b.core.Contexts {
			if containsRef(c.Assumes, a.ID) {
				row.ReliedOn = append(row.ReliedOn, b.ref(c.ID))
			}
		}
		for _, influence := range b.core.Influences {
			if containsRef(influence.HoldsIf, a.ID) {
				from := make([]string, 0, len(influence.From))
				for _, r := range influence.From {
					from = append(from, string(r.Target))
				}
				row.ReliedText = append(row.ReliedText, fmt.Sprintf("influence %s → %s (%s)", strings.Join(from, ", "), influence.To.Target, location(influence.Range)))
			}
		}
		l.Assumptions = append(l.Assumptions, row)
	}
}

func (b *builder) impactRows(l *Lists, impacts []assessment.Impact) {
	for _, impact := range impacts {
		row := ImpactRow{DP: b.ref(model.ID(impact.DP)), Affected: b.stringRefs(impact.Affected), Possible: b.stringRefs(impact.Possible), Modules: b.stringRefs(impact.Modules), PVs: b.stringRefs(impact.PVs), Needs: b.stringRefs(impact.Needs), Scenarios: b.stringRefs(impact.Scenarios)}
		for _, fr := range impact.Affected {
			if b.needs[model.ID(fr)].Inherited() {
				row.NeedsDerived = true
			}
		}
		l.Impacts = append(l.Impacts, row)
	}
}

func (b *builder) stringRefs(ids []string) []Ref {
	out := make([]Ref, 0, len(ids))
	for _, id := range ids {
		out = append(out, b.ref(model.ID(id)))
	}
	return out
}

// TraceRow follows one active leaf FR from its need to its module and scenario.
type TraceRow struct {
	Needs, CapabilityCriteria, Path, Criteria []Ref
	Capability                                Ref
	NeedsDerived                              bool
	DPs, Realized, Scenarios                  []Ref
}

func (b *builder) traceRows() []TraceRow {
	master := assessment.MasterScope(b.res.Scopes)
	if master == nil {
		return nil
	}
	var rows []TraceRow
	for _, id := range master.Matrix.FRs {
		fr := b.frs[id]
		if fr == nil {
			continue
		}
		trace := b.needs[id]
		row := TraceRow{Needs: b.refs(trace.Needs), NeedsDerived: trace.Inherited()}
		chain := []model.ID{id}
		for parent, ok := b.parentFR[id]; ok; parent, ok = b.parentFR[parent] {
			if slices.Contains(chain, parent) {
				break
			}
			chain = append([]model.ID{parent}, chain...)
		}
		row.Capability = b.ref(chain[0])
		if root := b.frs[chain[0]]; root != nil && len(chain) > 1 {
			for _, criterion := range root.Criteria {
				row.CapabilityCriteria = append(row.CapabilityCriteria, b.ref(criterion.ID))
			}
		}
		if len(chain) > 1 {
			row.Path = b.refs(chain[1:])
		}
		for _, criterion := range fr.Criteria {
			row.Criteria = append(row.Criteria, b.ref(criterion.ID))
		}
		if dp, ok := master.Pairing[id]; ok {
			row.DPs = []Ref{b.ref(dp)}
			for _, module := range b.core.Modules {
				if containsRef(module.Realizes.Items, dp) {
					row.Realized = append(row.Realized, b.ref(module.ID))
				}
			}
			for _, pv := range b.core.PVs {
				if containsRef(pv.Realizes, dp) {
					row.Realized = append(row.Realized, b.ref(pv.ID))
				}
			}
		}
		for _, scenario := range b.core.Scenarios {
			if verifies(scenario, fr) {
				row.Scenarios = append(row.Scenarios, b.ref(scenario.ID))
			}
		}
		rows = append(rows, row)
	}
	return rows
}

package assessment

import (
	"fmt"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/hashicorp/hcl/v2"
)

// Finding codes raised while resolving cells.
const (
	CodeConflictingInfluence    = "conflicting_influence"
	CodeUnresolvedApplicability = "unresolved_applicability"
)

// ResolveCells builds the FR-by-DP matrix of one sibling set. Only the active
// FRs and DPs of the set become axes, so cross-branch influences stay out.
// Each cell resolves from its applicable exact records first and falls back
// to the cells default of the set scope only when no such record exists.
// Severity is left to the result package.
func ResolveCells(snapshot *selection.Snapshot, set model.SiblingSet) (Matrix, []result.Finding) {
	matrix := newMatrix(snapshot, set)
	defaults := defaultFor(snapshot.Graph.Core.Cells, set.Scope)
	if len(defaults.records) > 0 {
		matrix.defaults = map[model.ID]scopeDefault{set.Scope: defaults}
	}
	findings := resolveExactCells(snapshot, &matrix)
	if defaults.conflict {
		findings = append(findings, defaultConflictFinding(set.Scope, defaults))
	}
	return matrix, findings
}

// newMatrix returns an empty matrix whose axes are the active FRs and DPs of
// the set, in authored order.
func newMatrix(snapshot *selection.Snapshot, set model.SiblingSet) Matrix {
	frs := activeIDs(set.FRs, snapshot.ActiveFRs)
	dps := activeIDs(set.DPs, snapshot.ActiveDPs)
	matrix := Matrix{
		Scope:    set.Scope,
		FRs:      frs,
		DPs:      dps,
		Cells:    make(map[model.ID]map[model.ID]Cell),
		frScopes: make(map[model.ID]model.ID, len(frs)),
		dpScopes: make(map[model.ID]model.ID, len(dps)),
	}
	for _, fr := range frs {
		matrix.frScopes[fr] = set.Scope
	}
	for _, dp := range dps {
		matrix.dpScopes[dp] = set.Scope
	}
	return matrix
}

// Cell resolves an exact fact, a default shared by both endpoints' sibling set,
// or an unknown. Looking up an absent coordinate never adds stored cells.
func (matrix Matrix) Cell(fr, dp model.ID) Cell {
	if cell, ok := matrix.Cells[fr][dp]; ok {
		return cell
	}
	cell := Cell{FR: fr, DP: dp}
	applyDefault(&cell, matrix.cellDefault(fr, dp))
	return cell
}

func (matrix Matrix) cellDefault(fr, dp model.ID) scopeDefault {
	frScope, hasFR := matrix.frScopes[fr]
	dpScope, hasDP := matrix.dpScopes[dp]
	if !hasFR || !hasDP || frScope != dpScope {
		return scopeDefault{}
	}
	return matrix.defaults[frScope]
}

// resolveExactCells visits only authored coordinates on the matrix axes.
// The first occurrence fixes finding order; all records for that coordinate
// resolve together, preserving conflicts and unresolved applicability.
func resolveExactCells(snapshot *selection.Snapshot, matrix *Matrix) []result.Finding {
	records := indexInfluences(snapshot.Influences)
	var findings []result.Finding
	var open []hcl.Range
	for _, influence := range snapshot.Influences {
		fr, dp := influence.To, influence.From
		if _, ok := matrix.frScopes[fr]; !ok {
			continue
		}
		if _, ok := matrix.dpScopes[dp]; !ok {
			continue
		}
		key := cellKey{fr: fr, dp: dp}
		exact, ok := records[key]
		if !ok {
			continue
		}
		delete(records, key)
		cell, cellOpen := resolveCell(fr, dp, exact, matrix.cellDefault(fr, dp))
		open = append(open, cellOpen...)
		if len(cell.Exact) == 0 && len(cell.Unknowns) == 0 && !cell.Unresolved {
			continue
		}
		if matrix.Cells[fr] == nil {
			matrix.Cells[fr] = make(map[model.ID]Cell)
		}
		matrix.Cells[fr][dp] = cell
		if hasExactConflict(cell) {
			findings = append(findings, conflictFinding(cell))
		}
	}
	if len(open) > 0 {
		findings = append(findings, unresolvedFinding(matrix.Scope, open))
	}
	return findings
}

// activeIDs keeps the active IDs in their authored order.
func activeIDs(ids []model.ID, active map[model.ID]bool) []model.ID {
	var kept []model.ID
	for _, id := range ids {
		if active[id] {
			kept = append(kept, id)
		}
	}
	return kept
}

// cellKey identifies one cell by its FR row and DP column.
type cellKey struct{ fr, dp model.ID }

// indexInfluences groups the applied influences by the cell they target.
func indexInfluences(influences []selection.AppliedInfluence) map[cellKey][]selection.AppliedInfluence {
	index := make(map[cellKey][]selection.AppliedInfluence)
	for _, influence := range influences {
		key := cellKey{fr: influence.To, dp: influence.From}
		index[key] = append(index[key], influence)
	}
	return index
}

// scopeDefault is the cells default that governs the unlisted cells of one scope.
type scopeDefault struct {
	records  []model.Cells
	effect   Effect // the agreed effect when every record agrees
	conflict bool   // the records disagree
}

// defaultFor collects the cells records that name the scope.
func defaultFor(cells []model.Cells, scope model.ID) scopeDefault {
	var found scopeDefault
	for _, record := range cells {
		if record.Scope.Target != scope {
			continue
		}
		found.add(record)
	}
	return found
}

func (defaults *scopeDefault) add(record model.Cells) {
	effect := effectOfUnlisted(record.Unlisted)
	if len(defaults.records) == 0 {
		defaults.effect = effect
	} else if effect != defaults.effect {
		defaults.conflict = true
	}
	defaults.records = append(defaults.records, record)
}

// exactRecords splits the influences of one cell by what they can establish.
type exactRecords struct {
	known       []selection.AppliedInfluence // applicable records with a material or none effect
	unknown     []selection.AppliedInfluence // applicable records whose effect is unknown
	open        []selection.AppliedInfluence // records whose applicability or premises stay unresolved
	conditional bool                         // a kept record relies on an assumed or unevaluated premise
}

// count reports how many records of any kind the cell has.
func (split exactRecords) count() int {
	return len(split.known) + len(split.unknown) + len(split.open)
}

// resolveCell resolves one cell. It also returns the ranges of the records
// that leave the cell open.
func resolveCell(fr, dp model.ID, influences []selection.AppliedInfluence, defaults scopeDefault) (Cell, []hcl.Range) {
	split := partitionExact(influences)
	cell := Cell{
		FR:          fr,
		DP:          dp,
		Exact:       split.known,
		Unknowns:    split.unknown,
		Conditional: split.conditional,
		Unresolved:  len(split.open) > 0,
	}
	if split.count() == 0 {
		applyDefault(&cell, defaults)
		return cell, nil
	}
	cell.Effect, cell.Conflict = exactEffect(split)
	return cell, rangesOf(split.open)
}

// partitionExact drops not applicable records and sorts the rest.
func partitionExact(influences []selection.AppliedInfluence) exactRecords {
	var split exactRecords
	for _, influence := range influences {
		if influence.Applicability == selection.Applicability("not_applicable") {
			continue
		}
		if isOpenRecord(influence) {
			split.open = append(split.open, influence)
			continue
		}
		if influence.Premises == selection.PremiseState("conditional") {
			split.conditional = true
		}
		if recordEffect(influence) == EffectUnknown {
			split.unknown = append(split.unknown, influence)
		} else {
			split.known = append(split.known, influence)
		}
	}
	return split
}

// isOpenRecord reports whether a record leaves its cell open. That happens when its
// applicability is unresolved or one of its premises is invalidated.
func isOpenRecord(influence selection.AppliedInfluence) bool {
	return influence.Applicability == selection.Applicability("unresolved") ||
		influence.Premises == selection.PremiseState("invalidated")
}

// exactEffect returns the effect of the cell from its exact records. Known
// records that disagree make a conflict. An open record or no known record
// leaves the effect unknown. Unknown records beside a known effect do not
// change the known effect.
func exactEffect(split exactRecords) (Effect, bool) {
	hasMaterial := hasEffect(split.known, EffectMaterial)
	hasNone := hasEffect(split.known, EffectNone)
	switch {
	case hasMaterial && hasNone:
		return EffectUnknown, true
	case !hasMaterial && !hasNone:
		return EffectUnknown, false
	case len(split.open) > 0:
		return EffectUnknown, false
	case hasMaterial:
		return EffectMaterial, false
	default:
		return EffectNone, false
	}
}

// hasEffect reports whether any record claims the given effect.
func hasEffect(influences []selection.AppliedInfluence, effect Effect) bool {
	for _, influence := range influences {
		if recordEffect(influence) == effect {
			return true
		}
	}
	return false
}

// applyDefault gives a cell with no applicable exact record the scope default.
// Disagreeing defaults make the cell a conflict. A cell with no default is unknown.
func applyDefault(cell *Cell, defaults scopeDefault) {
	cell.Effect = EffectUnknown
	if len(defaults.records) == 0 {
		return
	}
	cell.FromDefault = true
	cell.Conflict = defaults.conflict
	if !defaults.conflict {
		cell.Effect = defaults.effect
	}
}

// hasExactConflict reports whether a cell conflicts on its exact records.
// Default conflicts are reported at scope level instead.
func hasExactConflict(cell Cell) bool {
	return cell.Conflict && !cell.FromDefault
}

// recordEffect returns the effect a record claims. A record without an
// authored effect claims nothing, so it counts as unknown.
func recordEffect(influence selection.AppliedInfluence) Effect {
	if influence.Record == nil {
		return EffectUnknown
	}
	return effectOf(influence.Record.Effect)
}

// effectOf maps an authored effect to an assessment effect.
func effectOf(value model.Effect) Effect {
	switch value {
	case model.EffectMaterial:
		return EffectMaterial
	case model.EffectNone:
		return EffectNone
	default:
		return EffectUnknown
	}
}

// effectOfUnlisted maps an unlisted default to an assessment effect.
func effectOfUnlisted(value model.UnlistedEffect) Effect {
	if value == model.UnlistedNone {
		return EffectNone
	}
	return EffectUnknown
}

// rangesOf returns the source ranges of the given influences.
func rangesOf(influences []selection.AppliedInfluence) []hcl.Range {
	ranges := make([]hcl.Range, 0, len(influences))
	for _, influence := range influences {
		if influence.Record == nil {
			ranges = append(ranges, hcl.Range{})
			continue
		}
		ranges = append(ranges, influence.Record.Range)
	}
	return ranges
}

// rangesOfCells returns the source ranges of the given cells records.
func rangesOfCells(records []model.Cells) []hcl.Range {
	ranges := make([]hcl.Range, 0, len(records))
	for _, record := range records {
		ranges = append(ranges, record.Range)
	}
	return ranges
}

// conflictFinding reports a cell whose exact records contradict each other.
func conflictFinding(cell Cell) result.Finding {
	message := fmt.Sprintf("Requirement %s has both material and none influences from design parameter %s.", cell.FR, cell.DP)
	return cellFinding(CodeConflictingInfluence, string(cell.FR), message, rangesOf(cell.Exact))
}

// unresolvedFinding reports the open records of one scope.
func unresolvedFinding(scope model.ID, ranges []hcl.Range) result.Finding {
	message := fmt.Sprintf("Scope %s has influences whose applicability or premises are unresolved, so some of its cells stay unresolved.", scope)
	return cellFinding(CodeUnresolvedApplicability, string(scope), message, ranges)
}

// defaultConflictFinding reports cells defaults in one scope that disagree.
func defaultConflictFinding(scope model.ID, defaults scopeDefault) result.Finding {
	message := fmt.Sprintf("Cells defaults for scope %s disagree, so unlisted cells cannot take one default.", scope)
	return cellFinding(CodeConflictingInfluence, string(scope), message, rangesOfCells(defaults.records))
}

// cellFinding builds a finding with its subject and related ranges. The first
// related range is the primary range.
func cellFinding(code, subject, message string, ranges []hcl.Range) result.Finding {
	finding := result.Finding{Code: code, Subject: subject, Message: message, Related: ranges}
	if len(ranges) > 0 {
		finding.Range = &ranges[0]
	}
	return finding
}

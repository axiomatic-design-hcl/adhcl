package assessment

import (
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
)

// Effect is the resolved structural effect of one DP on one FR.
type Effect string

const (
	EffectMaterial Effect = "material"
	EffectNone     Effect = "none"
	EffectUnknown  Effect = "unknown"
)

// Cell is the resolved influence of one DP on one FR. It keeps every source
// record so that findings can point back to the authored influences.
type Cell struct {
	FR, DP      model.ID
	Effect      Effect
	Conflict    bool
	Exact       []selection.AppliedInfluence
	Unknowns    []selection.AppliedInfluence
	FromDefault bool
	Conditional bool
	Unresolved  bool
}

// Matrix is the FR-by-DP influence mapping of one scope. FRs are rows and DPs
// are columns, both in authored order. Cells stores only applicable exact facts;
// use Cell to resolve a coordinate's scoped default or implicit unknown.
type Matrix struct {
	Scope model.ID
	FRs   []model.ID
	DPs   []model.ID
	Cells map[model.ID]map[model.ID]Cell

	defaults map[model.ID]scopeDefault
	frScopes map[model.ID]model.ID
	dpScopes map[model.ID]model.ID
}

// Classification is the structural verdict for one scope.
type Classification string

const (
	Uncoupled  Classification = "uncoupled"
	Decoupled  Classification = "decoupled"
	Coupled    Classification = "coupled"
	Unresolved Classification = "unresolved"
)

// ScopeResult is the classification of one sibling set or the master axes.
// Unresolved covers every case that no concrete classification may claim,
// including ambiguous or incomplete pairings and conflicting cells.
type ScopeResult struct {
	Matrix           Matrix
	Pairing          map[model.ID]model.ID   // FR -> DP when exactly one complete pairing exists
	PairingWitnesses []map[model.ID]model.ID // two distinct complete pairings when ambiguous, not an exhaustive list
	Classification   Classification
	Order            []model.ID   // stable display order of DPs for uncoupled or decoupled scopes
	Cycles           [][]model.ID // known material precedence cycles, members in authored order
}

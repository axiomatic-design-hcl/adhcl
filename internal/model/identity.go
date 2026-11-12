package model

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
)

// ID is the full identity path of a labeled block, such as fr.build_db or
// fr.build_db.criterion.retention.
type ID string

// Kind names the namespace of an identity.
type Kind string

// Namespace kinds. Unlabeled blocks have no kind and no ID.
const (
	KindDesign      Kind = "design"
	KindSource      Kind = "source"
	KindStakeholder Kind = "stakeholder"
	KindNeed        Kind = "need"
	KindConstraint  Kind = "constraint"
	KindAssumption  Kind = "assumption"
	KindContext     Kind = "context"
	KindWorkload    Kind = "workload"
	KindState       Kind = "state"
	KindFR          Kind = "fr"
	KindCriterion   Kind = "criterion"
	KindDP          Kind = "dp"
	KindCells       Kind = "cells"
	KindExpect      Kind = "expect"
	KindCoupling    Kind = "coupling"
	KindDecision    Kind = "decision"
	KindCandidate   Kind = "candidate"
	KindEvidence    Kind = "evidence"
	KindPV          Kind = "pv"
	KindModule      Kind = "module"
	KindFlow        Kind = "flow"
	KindScenario    Kind = "scenario"
	KindExtension   Kind = "extension"
)

// Kind returns the namespace named by the path shape. A nested identity keeps
// its kind in the third segment, so fr.F.criterion.C has kind criterion.
func (id ID) Kind() Kind {
	segments := strings.Split(string(id), ".")
	if len(segments) == 4 {
		return Kind(segments[2])
	}
	return Kind(segments[0])
}

// Ref is one authored static traversal. Target is its path text, not yet checked
// against the declared identities.
type Ref struct {
	Target ID
	Range  hcl.Range
}

// Origin records where an authored fact came from. Criteria admit only user,
// proposed, and source. The decoder checks the set each block admits.
type Origin string

const (
	OriginUser     Origin = "user"
	OriginSource   Origin = "source"
	OriginProposed Origin = "proposed"
	OriginInferred Origin = "inferred"
)

// Header holds the identity and common metadata of a labeled block. Only the
// blocks that admit Cites, Origin, or Alias may set them. The decoder rejects
// any other use.
type Header struct {
	ID     ID
	File   string
	Range  hcl.Range
	Cites  []Ref
	Origin *Origin
	Alias  *string
}

// Unlabeled holds the position and common metadata of a block without a label.
type Unlabeled struct {
	File   string
	Range  hcl.Range
	Cites  []Ref
	Origin *Origin
}

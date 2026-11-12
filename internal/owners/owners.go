package owners

import (
	"fmt"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/hashicorp/hcl/v2"
)

// Status is the processing status of one owner.
type Status string

const (
	StatusComplete Status = "complete"
	StatusFailed   Status = "failed"
	StatusBlocked  Status = "blocked"
)

// Outcome is what an owner establishes for one attachment.
type Outcome string

const (
	OutcomeUnknown Outcome = "unknown"
	OutcomeValid   Outcome = "valid"
)

const (
	bindingCause  = "No caller-supplied observations are available."
	behaviorCause = "No external behavior model at the exact requested revision is available."
)

// AttachmentResult is the owner's result for one attachment.
type AttachmentResult struct {
	Attachment string
	Subject    model.ID
	Outcome    Outcome
	Cause      string
	Range      hcl.Range
	Revision   string
}

// OwnerResult is the independent result of one declared owner.
type OwnerResult struct {
	Extension      model.ID
	Contract       string
	Status         Status
	Results        []AttachmentResult
	Findings       []result.Finding
	Range          hcl.Range
	CoreSources    []SourceRevision
	CoreRevision   string
	Snapshot       string
	Invocation     string
	Implementation string
}

// SourceRevision keeps physical input identity separate from semantic subjects.
type SourceRevision struct {
	Path     string
	Revision string
}

// Assess runs every declared owner once and returns one result per declared
// extension, in manifest order. Owners never change the snapshot, the graph,
// or each other's results. A nil snapshot means selection failed, so every
// owner is blocked and no payload is processed.
func Assess(snapshot *selection.Snapshot, manifest model.Manifest, attachments model.Attachments) []OwnerResult {
	out := make([]OwnerResult, 0, len(manifest.Extensions))
	for _, ext := range manifest.Extensions {
		out = append(out, assessOwner(snapshot, ext, attachments))
	}
	return out
}

func assessOwner(snapshot *selection.Snapshot, ext model.Extension, attachments model.Attachments) OwnerResult {
	res := OwnerResult{Extension: ext.ID, Contract: ext.Contract, Status: StatusComplete, Range: ext.Range}
	spec, ok := lookupOwner(ext)
	if !ok {
		res.Status = StatusBlocked
		res.Findings = append(res.Findings, unavailableOwnerFinding(ext))
		return res
	}
	if snapshot == nil || snapshot.Graph == nil {
		res.Status = StatusBlocked
		return res
	}

	firstByIdentity := map[string]model.Attachment{}
	for _, att := range attachments {
		if att.Owner.Target != ext.ID {
			continue
		}
		subject := att.Subject.Target
		if subject.Kind() != spec.subject || !subjectActive(snapshot, subject, spec.subject) {
			continue
		}
		values, problem := decodePayload(spec.fields, att)
		if problem != nil {
			res.Findings = append(res.Findings, invalidAttachmentFinding(att, *problem))
			continue
		}
		identity := identityOf(ext.Role, att)
		if first, repeated := firstByIdentity[identity]; repeated {
			res.Findings = append(res.Findings, duplicateIdentityFinding(att, first))
			continue
		}
		firstByIdentity[identity] = att

		decision := judge(ext.Role, snapshot.Graph, att, values)
		res.Results = append(res.Results, AttachmentResult{
			Attachment: att.Label,
			Subject:    subject,
			Outcome:    decision.outcome,
			Cause:      decision.cause,
			Range:      att.Range,
		})
		res.Findings = append(res.Findings, decision.findings...)
	}
	return res
}

// verdict is one owner's decision for one valid attachment.
type verdict struct {
	outcome  Outcome
	cause    string
	findings []result.Finding
}

// judge decides the outcome for a valid payload. Binding and behavior
// observations are never available, so their outcome is unknown with the
// pinned cause. A valid view payload is valid without rendering HTML.
func judge(role model.ExtensionRole, graph *model.Graph, att model.Attachment, values payload) verdict {
	switch role {
	case model.ExtensionRoleBinding:
		v := verdict{
			outcome:  OutcomeUnknown,
			cause:    bindingCause,
			findings: []result.Finding{unavailableExternalFinding(att)},
		}
		if dup, ok := duplicateBindingFinding(graph, att, values); ok {
			v.findings = append(v.findings, dup)
		}
		return v
	case model.ExtensionRoleBehaviorLink:
		return verdict{
			outcome:  OutcomeUnknown,
			cause:    behaviorCause,
			findings: []result.Finding{unavailableExternalFinding(att)},
		}
	default:
		return verdict{outcome: OutcomeValid}
	}
}

// subjectActive reports whether an attachment subject is active in the
// snapshot. A PV is active when it realizes an active DP. A module is active
// when the snapshot lists it. The design is active when it is the graph design.
func subjectActive(snapshot *selection.Snapshot, subject model.ID, kind model.Kind) bool {
	switch kind {
	case model.KindPV:
		pv := findPV(snapshot.Graph, subject)
		if pv == nil {
			return false
		}
		for _, realized := range pv.Realizes {
			if snapshot.ActiveDPs[realized.Target] {
				return true
			}
		}
		return false
	case model.KindModule:
		_, active := snapshot.Modules[subject]
		return active
	case model.KindDesign:
		return snapshot.Graph.Design == subject
	}
	return false
}

// findPV returns the core PV with the given identity, or nil when none exists.
func findPV(graph *model.Graph, id model.ID) *model.PV {
	for i := range graph.Core.PVs {
		if graph.Core.PVs[i].ID == id {
			return &graph.Core.PVs[i]
		}
	}
	return nil
}

// identityOf names the result identity of an attachment. A view is unique
// per design. Other attachments are unique per label and subject.
func identityOf(role model.ExtensionRole, att model.Attachment) string {
	if role == model.ExtensionRoleView {
		return "view " + string(att.Subject.Target)
	}
	return att.Label + " " + string(att.Subject.Target)
}

// attachmentSubject returns the finding subject for an attachment label.
func attachmentSubject(label string) string {
	return "attachment." + label
}

// duplicateBindingFinding reports a binding whose address equals the address
// of its subject PV. The error names the PV block and the binding address.
func duplicateBindingFinding(graph *model.Graph, att model.Attachment, values payload) (result.Finding, bool) {
	pv := findPV(graph, att.Subject.Target)
	address := values["address"].value.AsString()
	if pv == nil || pv.Address == nil || *pv.Address != address {
		return result.Finding{}, false
	}
	pvRange := pv.Range
	return result.Finding{
		Code:    "duplicate_binding",
		Subject: attachmentSubject(att.Label),
		Message: fmt.Sprintf("The binding repeats the address %q already declared on %s.", address, pv.ID),
		Range:   &pvRange,
		Related: []hcl.Range{values["address"].valueRange},
	}, true
}

func unavailableOwnerFinding(ext model.Extension) result.Finding {
	rng := ext.Range
	return result.Finding{
		Code:     "unavailable_owner",
		Severity: result.Error,
		Subject:  string(ext.ID),
		Message:  fmt.Sprintf("Extension %s is not an available standard owner with this role and contract.", ext.ID),
		Range:    &rng,
	}
}

func invalidAttachmentFinding(att model.Attachment, problem payloadProblem) result.Finding {
	rng := problem.rng
	return result.Finding{
		Code:    "invalid_attachment",
		Subject: attachmentSubject(att.Label),
		Message: fmt.Sprintf("Attachment %q is invalid: %s.", att.Label, problem.message),
		Range:   &rng,
	}
}

func duplicateIdentityFinding(att, first model.Attachment) result.Finding {
	rng := att.Range
	return result.Finding{
		Code:    "duplicate_identity",
		Subject: attachmentSubject(att.Label),
		Message: fmt.Sprintf("Attachment %q repeats the result identity of attachment %q on %s.", att.Label, first.Label, att.Subject.Target),
		Range:   &rng,
		Related: []hcl.Range{first.Range},
	}
}

func unavailableExternalFinding(att model.Attachment) result.Finding {
	rng := att.Range
	return result.Finding{
		Code:    "unavailable_external_input",
		Subject: attachmentSubject(att.Label),
		Message: fmt.Sprintf("External input for attachment %q is unavailable. Its result stays unknown.", att.Label),
		Range:   &rng,
	}
}

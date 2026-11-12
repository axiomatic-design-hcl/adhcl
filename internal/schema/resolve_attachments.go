package schema

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

// ResolveAttachments checks each valid attachment's owner and subject references.
// The owner must be a manifest extension. The subject must be a declared identity of
// the kind that the owner's role admits. A nil graph skips the subject check, because
// declared identities are unknown without a graph.
func ResolveAttachments(graph *model.Graph, manifest model.Manifest, attachments model.Attachments) []result.Finding {
	extensions := make(map[model.ID]model.Extension, len(manifest.Extensions))
	for _, ext := range manifest.Extensions {
		if _, seen := extensions[ext.ID]; !seen {
			extensions[ext.ID] = ext
		}
	}
	var findings []result.Finding
	for _, att := range attachments {
		ext, ok := extensions[att.Owner.Target]
		if !ok {
			rng := att.Owner.Range
			findings = append(findings, newFinding(codeInvalidReference, string(att.Owner.Target), &rng, nil,
				fmt.Sprintf("%s is not a declared extension in the manifest. An attachment must name its owner.", att.Owner.Target)))
			continue
		}
		if graph == nil {
			continue
		}
		if f, bad := subjectFinding(graph, att, ext); bad {
			findings = append(findings, f)
		}
	}
	return findings
}

// subjectFinding reports a subject that is undeclared or has the wrong kind for the
// owner's role. It reports false when the subject is valid, or when the role has no
// subject kind.
func subjectFinding(graph *model.Graph, att model.Attachment, ext model.Extension) (result.Finding, bool) {
	want, ok := subjectKind(ext.Role)
	if !ok {
		return result.Finding{}, false
	}
	if graph.Has(att.Subject.Target) && att.Subject.Target.Kind() == want {
		return result.Finding{}, false
	}
	rng := att.Subject.Range
	return newFinding(codeInvalidReference, string(att.Subject.Target), &rng, nil,
		fmt.Sprintf("%s must name a declared %s, because its owner is a %s extension.", att.Subject.Target, want, ext.Role)), true
}

// subjectKind is the kind of identity that each extension role attaches to.
func subjectKind(role model.ExtensionRole) (model.Kind, bool) {
	switch role {
	case model.ExtensionRoleBinding:
		return model.KindPV, true
	case model.ExtensionRoleBehaviorLink:
		return model.KindModule, true
	case model.ExtensionRoleView:
		return model.KindDesign, true
	}
	return "", false
}

// CheckAttachmentLabels checks project-wide label uniqueness before envelope
// filtering. It needs neither a valid envelope nor a resolved core graph.
// The first declaration is primary, and later declarations retain related ranges.
func CheckAttachmentLabels(units []syntax.Unit) []result.Finding {
	var labels []string
	ranges := make(map[string][]hcl.Range)
	for _, unit := range units {
		if unit.Body == nil {
			continue
		}
		for _, block := range unit.Body.Blocks {
			if block.Type != "attachment" || len(block.Labels) != 1 {
				continue
			}
			label := block.Labels[0]
			if _, seen := ranges[label]; !seen {
				labels = append(labels, label)
			}
			ranges[label] = append(ranges[label], blockRange(block))
		}
	}
	var findings []result.Finding
	for _, label := range labels {
		group := ranges[label]
		if len(group) < 2 {
			continue
		}
		first := group[0]
		findings = append(findings, newFinding(codeDuplicateIdentity, label, &first, group[1:],
			fmt.Sprintf("The attachment label %s is declared more than once. Attachment labels must be unique.", label)))
	}
	return findings
}

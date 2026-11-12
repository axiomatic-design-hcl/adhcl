package validate

import (
	"strings"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/input"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/owners"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/schema"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

const (
	stageCore        = "core_assessment"
	stageComposition = "composition"
	statusBlocked    = "blocked"
	ownerStagePrefix = "owner:"
	extensionPrefix  = "extension."
)

// coreResult is what the core half of the pipeline produced. Fields that a
// failed step never reached keep their zero values.
type coreResult struct {
	status     string
	graph      *model.Graph
	snapshot   *selection.Snapshot
	scopes     []assessment.ScopeResult
	candidates []assessment.CandidateResult
	findings   []result.Finding
}

// assessCore runs the core steps: parse and decode the core sources, resolve
// the graph, select the snapshot, and assess it. Any failed step before the
// assessment leaves the core blocked. Resolve never runs after a syntax or
// decode error.
func assessCore(selected input.Selection, request Request) coreResult {
	units, parseFindings := parseSources(selected.Core)
	decoded, decodeFindings := schema.DecodeCore(units)
	core := coreResult{
		status:   statusBlocked,
		findings: concat(parseFindings, decodeFindings),
	}
	if !selected.CoreComplete || hasError(core.findings) {
		return core
	}

	graph, resolveFindings := schema.Resolve(decoded)
	core.findings = append(core.findings, resolveFindings...)
	if graph == nil {
		return core
	}
	core.graph = graph

	snapshot, selectFindings := selection.Select(graph, selection.Request{
		Candidate: model.ID(request.Candidate),
		Context:   model.ID(request.Context),
	})
	core.findings = append(core.findings, selectFindings...)
	if snapshot == nil {
		return core
	}
	core.snapshot = snapshot

	assessed := assessment.Assess(snapshot)
	core.scopes = scopesWithMaster(assessed)
	core.candidates = assessed.Candidates
	core.findings = append(core.findings, assessed.Findings...)
	core.status = statusComplete
	return core
}

// scopesWithMaster returns the sibling scopes followed by the master scope.
func scopesWithMaster(assessed assessment.Result) []assessment.ScopeResult {
	scopes := make([]assessment.ScopeResult, 0, len(assessed.Scopes)+1)
	scopes = append(scopes, assessed.Scopes...)
	return append(scopes, assessed.Master)
}

// assessOwners runs the owner steps for manifest input. Without a manifest it
// returns nothing, so no owner stage is reported. An owner that would otherwise
// be complete is blocked when some attachment was not read or did not parse,
// because the owner cannot then confirm that it saw every attachment.
func assessOwners(selected input.Selection, core coreResult) ([]owners.OwnerResult, []result.Finding, []Stage) {
	if selected.Manifest == nil {
		return nil, nil, nil
	}
	manifest, manifestFindings := schema.DecodeManifest(*selected.Manifest)
	units, parseFindings := parseSources(selected.Attachments)
	attachments, attachmentFindings := schema.DecodeAttachments(units)

	findings := concat(manifestFindings, parseFindings, attachmentFindings, schema.CheckAttachmentLabels(units))
	if core.graph != nil {
		findings = append(findings, schema.ResolveAttachments(core.graph, manifest, attachments)...)
	}

	ownerResults := owners.Assess(core.snapshot, manifest, attachments)
	intact := selected.AttachmentsComplete && !hasError(parseFindings)
	stages := make([]Stage, 0, len(ownerResults))
	for i := range ownerResults {
		if !intact && ownerResults[i].Status == owners.StatusComplete {
			ownerResults[i].Status = owners.StatusBlocked
		}
		findings = append(findings, ownerResults[i].Findings...)
		stages = append(stages, Stage{
			Name:   ownerStageName(ownerResults[i].Extension),
			Status: string(ownerResults[i].Status),
		})
	}
	return ownerResults, findings, stages
}

// ownerStageName names the stage of one owner by its label, which is the
// extension ID without its extension prefix.
func ownerStageName(extension model.ID) string {
	return ownerStagePrefix + strings.TrimPrefix(string(extension), extensionPrefix)
}

// parseSources parses each source and returns the units with their syntax findings.
func parseSources(sources []model.Source) ([]syntax.Unit, []result.Finding) {
	units := make([]syntax.Unit, 0, len(sources))
	var findings []result.Finding
	for _, source := range sources {
		unit := syntax.Parse(source)
		units = append(units, unit)
		findings = append(findings, unit.Findings...)
	}
	return units, findings
}

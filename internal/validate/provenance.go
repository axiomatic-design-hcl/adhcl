package validate

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/axiomatic-design-hcl/adhcl/internal/input"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/owners"
)

func sourceRevision(source model.Source) string {
	return fmt.Sprintf("%x", sha256.Sum256(source.Bytes))
}

func attachProvenance(selected input.Selection, core coreResult, request Request, results []owners.OwnerResult) {
	sources := make([]owners.SourceRevision, 0, len(selected.Core))
	for _, s := range selected.Core {
		sources = append(sources, owners.SourceRevision{Path: s.Path, Revision: sourceRevision(s)})
	}
	coreRevision := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%q", sources))))
	snapshotParts := []string{coreRevision, request.Candidate, request.Context}
	if core.snapshot != nil {
		snapshotParts = append(snapshotParts, string(core.snapshot.Context), string(core.snapshot.Workload), core.snapshot.Horizon)
		var members []string
		for id, active := range core.snapshot.ActiveDPs {
			if active {
				members = append(members, string(id))
			}
		}
		for id, active := range core.snapshot.ActiveFRs {
			if active {
				members = append(members, string(id))
			}
		}
		sort.Strings(members)
		snapshotParts = append(snapshotParts, members...)
	}
	snapshot := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%q", snapshotParts))))
	attachmentRevisions := make(map[string]string, len(selected.Attachments))
	invocationParts := []string{snapshot, request.Version, request.Profile}
	if selected.Manifest != nil {
		invocationParts = append(invocationParts, selected.Manifest.Source.Path, sourceRevision(selected.Manifest.Source))
	}
	for _, s := range selected.Attachments {
		revision := sourceRevision(s)
		attachmentRevisions[s.Path] = revision
		invocationParts = append(invocationParts, s.Path, revision)
	}
	invocation := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%q", invocationParts))))
	for i := range results {
		owner := &results[i]
		owner.CoreSources = sources
		owner.CoreRevision = coreRevision
		owner.Snapshot = snapshot
		owner.Invocation = invocation
		owner.Implementation = "adhcl " + request.Version
		for j := range owner.Results {
			a := &owner.Results[j]
			a.Revision = attachmentRevisions[a.Range.Filename]
		}
	}
}

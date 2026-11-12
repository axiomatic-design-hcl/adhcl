package schema

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

// blockDecoders maps each core block type to the function that decodes it.
// Each decoder reads one block and appends its records to the core.
var blockDecoders = map[string]func(*blockReader, *model.Core){
	"design":      decodeDesign,
	"source":      decodeSource,
	"stakeholder": decodeStakeholder,
	"need":        decodeNeed,
	"constraint":  decodeConstraint,
	"assumption":  decodeAssumption,
	"context":     decodeContext,
	"workload":    decodeWorkload,
	"state":       decodeState,
	"fr":          decodeFR,
	"dp":          decodeDP,
	"influence":   decodeInfluence,
	"cells":       decodeCells,
	"expect":      decodeExpect,
	"coupling":    decodeCoupling,
	"decision":    decodeDecision,
	"evidence":    decodeEvidence,
	"pv":          decodePV,
	"module":      decodeModule,
	"flow":        decodeFlow,
	"scenario":    decodeScenario,
}

// DecodeCore reads every core block in units into one core. It keeps every
// record in authored order, including duplicates, and reports each schema
// defect once as an invalid_structure finding. Native parse findings stay on
// their units.
func DecodeCore(units []syntax.Unit) (model.Core, []result.Finding) {
	var core model.Core
	var findings []result.Finding
	for _, unit := range units {
		decodeUnit(unit, &core, &findings)
	}
	return core, findings
}

// decodeUnit reads one parsed file. A unit without a body failed to parse, so it is skipped.
func decodeUnit(unit syntax.Unit, core *model.Core, findings *[]result.Finding) {
	if unit.Body == nil {
		return
	}
	for _, attr := range sortedAttributes(unit.Body.Attributes) {
		addFinding(findings, unit.Source.Path, attr.SrcRange, fmt.Sprintf("%q is not allowed at the top level of a file", attr.Name))
	}
	for _, block := range unit.Body.Blocks {
		decodeBlock(unit.Source.Path, block, core, findings)
	}
}

// decodeBlock dispatches one top-level block. A block type that is not in the
// schema, including removed types such as import, is reported once and its body is skipped.
func decodeBlock(file string, block *hclsyntax.Block, core *model.Core, findings *[]result.Finding) {
	decode, known := blockDecoders[block.Type]
	if !known {
		addFinding(findings, block.Type, blockHeaderRange(block), fmt.Sprintf("%q is not a block type in this schema", block.Type))
		return
	}
	reader := newBlockReader(file, block, findings)
	decode(reader, core)
	reader.finish()
}

package schema

import (
	"fmt"
	"math/big"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

// assumptionStatusNames lists the admitted assumption statuses.
var assumptionStatusNames = []string{
	string(model.AssumptionAssumed),
	string(model.AssumptionToValidate),
	string(model.AssumptionValidated),
	string(model.AssumptionInvalidated),
}

// decodeDesign reads one design block.
func decodeDesign(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindDesign), standardMeta)
	boundary := r.requiredString("boundary")
	level := r.optionalString("level")
	workload := r.optionalRef("workload")
	core.Designs = append(core.Designs, model.Design{
		Header:   header,
		Boundary: boundary,
		Level:    level,
		Workload: workload,
	})
}

// decodeSource reads one source block. Sources carry no cites, origin, or alias.
func decodeSource(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindSource), metadata{})
	title := r.requiredString("title")
	doi := r.optionalString("doi")
	url := r.optionalString("url")
	pages := r.optionalString("pages")
	core.Sources = append(core.Sources, model.SourceRecord{
		Header: header,
		Title:  title,
		DOI:    doi,
		URL:    url,
		Pages:  pages,
	})
}

// decodeStakeholder reads one stakeholder block.
func decodeStakeholder(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindStakeholder), standardMeta)
	role := r.requiredString("role")
	core.Stakeholders = append(core.Stakeholders, model.Stakeholder{Header: header, Role: role})
}

// decodeNeed reads one need block.
func decodeNeed(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindNeed), standardMeta)
	statement := r.requiredString("statement")
	why := r.optionalString("why")
	stakeholder := r.optionalRef("stakeholder")
	core.Needs = append(core.Needs, model.Need{
		Header:      header,
		Statement:   statement,
		Why:         why,
		Stakeholder: stakeholder,
	})
}

// decodeConstraint reads one constraint block.
func decodeConstraint(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindConstraint), standardMeta)
	statement := r.requiredString("statement")
	kind := r.optionalString("kind")
	core.Constraints = append(core.Constraints, model.Constraint{
		Header:    header,
		Statement: statement,
		Kind:      kind,
	})
}

// decodeAssumption reads one assumption block. A missing or invalid status defaults to to_validate.
func decodeAssumption(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindAssumption), standardMeta)
	statement := r.requiredString("statement")
	status := model.AssumptionToValidate
	if chosen := r.optionalEnum("status", assumptionStatusNames...); chosen != nil {
		status = model.AssumptionStatus(*chosen)
	}
	core.Assumptions = append(core.Assumptions, model.Assumption{
		Header:    header,
		Statement: statement,
		Status:    &status,
	})
}

// decodeContext reads one context block. A context may not exclude itself.
func decodeContext(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindContext), standardMeta)
	description := r.requiredString("description")
	assumes, _ := field(r, "assumes", false, refsOf)
	excludes, _ := field(r, "excludes", false, refsExcluding(header.ID))
	workload := r.optionalRef("workload")
	core.Contexts = append(core.Contexts, model.Context{
		Header:      header,
		Description: description,
		Assumes:     assumes,
		Excludes:    excludes,
		Workload:    workload,
	})
}

// refsExcluding accepts a list of references that does not include self.
func refsExcluding(self model.ID) func(hclsyntax.Expression) ([]model.Ref, string) {
	return func(expr hclsyntax.Expression) ([]model.Ref, string) {
		refs, problem := refsOf(expr)
		if problem != "" {
			return nil, problem
		}
		for _, ref := range refs {
			if ref.Target == self {
				return nil, fmt.Sprintf("must not include %s", self)
			}
		}
		return refs, ""
	}
}

// decodeWorkload reads one workload block. Ranges are optional; invalid ranges leave them absent.
func decodeWorkload(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindWorkload), standardMeta)
	description := r.requiredString("description")
	ranges := optionalField(r, "ranges", rangeDimensionsOf)
	workload := model.Workload{Header: header, Description: description}
	if ranges != nil {
		workload.Ranges = model.Optional[model.RangeDimension]{Present: true, Items: *ranges}
	}
	core.Workloads = append(core.Workloads, workload)
}

// decodeState reads one state block.
func decodeState(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindState), standardMeta)
	description := r.requiredString("description")
	core.States = append(core.States, model.State{Header: header, Description: description})
}

// rangeDimensionsOf reads an object whose keys name dimensions. Each value is an
// object with min and max, and optionally unit. Keys are kept in authored order.
func rangeDimensionsOf(expr hclsyntax.Expression) ([]model.RangeDimension, string) {
	object, ok := expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil, "must be an object whose keys name dimensions"
	}
	dimensions := make([]model.RangeDimension, 0, len(object.Items))
	for _, item := range object.Items {
		name, problem := objectKeyOf(item.KeyExpr)
		if problem != "" {
			return nil, "each dimension name " + problem
		}
		dimension, problem := rangeDimensionOf(name, item.ValueExpr)
		if problem != "" {
			return nil, problem
		}
		dimensions = append(dimensions, dimension)
	}
	return dimensions, ""
}

// rangeDimensionOf reads one dimension object with min, max, and optional unit.
// The first problem is returned in plain English.
func rangeDimensionOf(name string, expr hclsyntax.Expression) (model.RangeDimension, string) {
	object, ok := expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return model.RangeDimension{}, fmt.Sprintf("dimension %q must be an object with min and max", name)
	}
	fields, problem := dimensionFields(name, object)
	if problem != "" {
		return model.RangeDimension{}, problem
	}
	lowest, problem := dimensionNumber(name, fields, "min")
	if problem != "" {
		return model.RangeDimension{}, problem
	}
	highest, problem := dimensionNumber(name, fields, "max")
	if problem != "" {
		return model.RangeDimension{}, problem
	}
	if lowest.Cmp(highest) > 0 {
		return model.RangeDimension{}, fmt.Sprintf("dimension %q has min greater than max", name)
	}
	dimension := model.RangeDimension{
		Name:  name,
		Min:   lowest,
		Max:   highest,
		Range: object.Range(),
	}
	if unitExpr, hasUnit := fields["unit"]; hasUnit {
		unit, problem := stringOf(unitExpr)
		if problem != "" {
			return model.RangeDimension{}, fmt.Sprintf("dimension %q unit %s", name, problem)
		}
		dimension.Unit = &unit
	}
	return dimension, ""
}

// dimensionFields maps each key of a dimension object to its value. It rejects
// keys other than min, max, and unit, and rejects a key that appears twice.
func dimensionFields(name string, object *hclsyntax.ObjectConsExpr) (map[string]hclsyntax.Expression, string) {
	fields := make(map[string]hclsyntax.Expression, len(object.Items))
	for _, item := range object.Items {
		key, problem := objectKeyOf(item.KeyExpr)
		if problem != "" {
			return nil, fmt.Sprintf("dimension %q has an invalid key, %s", name, problem)
		}
		if key != "min" && key != "max" && key != "unit" {
			return nil, fmt.Sprintf("dimension %q has unknown key %q", name, key)
		}
		if _, repeated := fields[key]; repeated {
			return nil, fmt.Sprintf("dimension %q repeats key %q", name, key)
		}
		fields[key] = item.ValueExpr
	}
	return fields, ""
}

// dimensionNumber reads a required min or max value from a dimension object.
func dimensionNumber(name string, fields map[string]hclsyntax.Expression, key string) (*big.Float, string) {
	expr, ok := fields[key]
	if !ok {
		return nil, fmt.Sprintf("dimension %q needs %s", name, key)
	}
	number, problem := numberOf(expr)
	if problem != "" {
		return nil, fmt.Sprintf("dimension %q %s %s", name, key, problem)
	}
	return number, ""
}

// objectKeyOf reads an object key that is a bare word or a string literal.
func objectKeyOf(expr hclsyntax.Expression) (string, string) {
	if key, ok := expr.(*hclsyntax.ObjectConsKeyExpr); ok {
		expr = key.Wrapped
	}
	if word := hcl.ExprAsKeyword(expr); word != "" {
		return word, ""
	}
	return stringOf(expr)
}

package owners

import (
	"strings"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

// owner is one fixed local owner. Its role, contract, and subject kind come
// from the schema; a manifest cannot change them.
type owner struct {
	role     model.ExtensionRole
	contract string
	subject  model.Kind
	fields   []payloadField
}

// registry lists the three standard owners, keyed by their registry label.
var registry = map[string]owner{
	"terraform_binding": {
		role:     model.ExtensionRoleBinding,
		contract: "requirements_hcl/terraform_binding/1",
		subject:  model.KindPV,
		fields: []payloadField{
			nonBlankString("address"),
			nonBlankString("observation_source"),
		},
	},
	"event_model_link": {
		role:     model.ExtensionRoleBehaviorLink,
		contract: "requirements_hcl/event_model_link/1",
		subject:  model.KindModule,
		fields: []payloadField{
			nonBlankString("model"),
			nonBlankString("revision"),
			oneOf("target_kind", "slice"),
			nonBlankString("target_label"),
		},
	},
	"html_view": {
		role:     model.ExtensionRoleView,
		contract: "requirements_hcl/html_view/1",
		subject:  model.KindDesign,
		fields: []payloadField{
			oneOf("layout", "requirements_first", "matrices_first"),
			exactlyTrue("show_source_locations"),
		},
	},
}

// lookupOwner returns the registry entry for a declared extension. It reports
// false when the label is unknown or the role or contract does not match.
func lookupOwner(ext model.Extension) (owner, bool) {
	label := strings.TrimPrefix(string(ext.ID), "extension.")
	spec, ok := registry[label]
	if !ok || spec.role != ext.Role || spec.contract != ext.Contract {
		return owner{}, false
	}
	return spec, true
}

package owners

import (
	"fmt"
	"strings"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// payloadField is one required key of a closed owner payload. The valid
// function runs only on known, non-null values.
type payloadField struct {
	name  string
	want  string
	valid func(cty.Value) bool
}

func nonBlankString(name string) payloadField {
	return payloadField{
		name: name,
		want: "a non-blank string",
		valid: func(v cty.Value) bool {
			return v.Type().Equals(cty.String) && strings.TrimSpace(v.AsString()) != ""
		},
	}
}

func oneOf(name string, allowed ...string) payloadField {
	return payloadField{
		name: name,
		want: "one of " + strings.Join(allowed, ", "),
		valid: func(v cty.Value) bool {
			if !v.Type().Equals(cty.String) {
				return false
			}
			for _, option := range allowed {
				if v.AsString() == option {
					return true
				}
			}
			return false
		},
	}
}

func exactlyTrue(name string) payloadField {
	return payloadField{
		name: name,
		want: "the boolean value true",
		valid: func(v cty.Value) bool {
			return v.Type().Equals(cty.Bool) && v.True()
		},
	}
}

// payloadProblem is the first defect in a payload and the source range to report.
type payloadProblem struct {
	message string
	rng     hcl.Range
}

// payloadValue is one accepted value and the range of its data field.
type payloadValue struct {
	value      cty.Value
	valueRange hcl.Range
}

// payload maps each required field name to its accepted value.
type payload map[string]payloadValue

// decodePayload checks attachment data against the closed field list. It
// rejects unknown keys, repeated keys, nulls, unknown values, invalid values,
// and missing keys. It returns the accepted values, or the first problem.
// A missing key is reported at the attachment range, because the data object
// has no range of its own.
func decodePayload(fields []payloadField, attachment model.Attachment) (payload, *payloadProblem) {
	spec := make(map[string]payloadField, len(fields))
	for _, field := range fields {
		spec[field.name] = field
	}
	values := make(payload, len(fields))
	for _, item := range attachment.Data {
		field, known := spec[item.Key]
		if !known {
			return nil, &payloadProblem{
				message: fmt.Sprintf("field %q is not part of this owner's payload", item.Key),
				rng:     item.KeyRange,
			}
		}
		if _, repeated := values[item.Key]; repeated {
			return nil, &payloadProblem{
				message: fmt.Sprintf("field %q appears more than once", item.Key),
				rng:     item.KeyRange,
			}
		}
		if item.Value.IsNull() || !item.Value.IsKnown() || !field.valid(item.Value) {
			return nil, &payloadProblem{
				message: fmt.Sprintf("field %q must be %s", item.Key, field.want),
				rng:     item.ValueRange,
			}
		}
		values[item.Key] = payloadValue{value: item.Value, valueRange: item.ValueRange}
	}
	for _, field := range fields {
		if _, ok := values[field.name]; !ok {
			return nil, &payloadProblem{
				message: fmt.Sprintf("required field %q is missing", field.name),
				rng:     attachment.Range,
			}
		}
	}
	return values, nil
}

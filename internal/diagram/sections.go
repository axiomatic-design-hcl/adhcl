package diagram

import (
	"fmt"
	"strings"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

// Cards holds the record kinds that have no list view.
type Cards struct {
	Sources, States, Cells, Contexts, Workloads, Evidence, Owners []Item
}

// Item is one record shown as a card with a field list.
type Item struct {
	ID, Text, Location, Label string
	Anchor                    bool
	Fields                    []Field
}

// Field is one named value of a card. Label marks a value that the tool assessed.
type Field struct{ Name, Value, Label string }

func (b *builder) item(h model.Header, text string) Item {
	i := Item{ID: string(h.ID), Text: text, Location: location(h.Range), Label: originLabel(h.Origin), Anchor: b.anchors[h.ID]}
	i.add("Cites", refs(h.Cites))
	return i
}

func (i *Item) add(name, value string) {
	if value != "" {
		i.Fields = append(i.Fields, Field{Name: name, Value: value})
	}
}

func text(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func refs(rs []model.Ref) string {
	v := make([]string, 0, len(rs))
	for _, r := range rs {
		v = append(v, string(r.Target))
	}
	return strings.Join(v, ", ")
}

func (b *builder) cards() Cards {
	var c Cards
	for _, record := range b.core.Sources {
		mapped := b.item(record.Header, record.Title)
		mapped.add("DOI", text(record.DOI))
		mapped.add("URL", text(record.URL))
		mapped.add("Pages", text(record.Pages))
		c.Sources = append(c.Sources, mapped)
	}
	for _, record := range b.core.States {
		mapped := b.item(record.Header, record.Description)
		for _, dp := range b.core.DPs {
			if containsRef(dp.OwnsState.Items, record.ID) {
				mapped.add("Owned by", string(dp.ID))
			}
		}
		c.States = append(c.States, mapped)
	}
	for _, record := range b.core.Cells {
		mapped := b.item(record.Header, "Unlisted cells: "+string(record.Unlisted))
		mapped.add("Scope", string(record.Scope.Target))
		mapped.add("Because", text(record.Because))
		c.Cells = append(c.Cells, mapped)
	}
	for _, record := range b.core.Contexts {
		mapped := b.item(record.Header, record.Description)
		mapped.add("Assumes", refs(record.Assumes))
		mapped.add("Excludes", refs(record.Excludes))
		if record.Workload != nil {
			mapped.add("Workload", string(record.Workload.Target))
		}
		c.Contexts = append(c.Contexts, mapped)
	}
	for _, record := range b.core.Workloads {
		mapped := b.item(record.Header, record.Description)
		for _, r := range record.Ranges.Items {
			if r.Min != nil && r.Max != nil {
				mapped.add(r.Name, strings.TrimSpace(r.Min.Text('g', -1)+" – "+r.Max.Text('g', -1)+" "+text(r.Unit)))
			}
		}
		c.Workloads = append(c.Workloads, mapped)
	}
	for _, record := range b.core.Evidence {
		mapped := b.item(record.Header, record.Event)
		mapped.add("Scenario", string(record.Scenario.Target))
		mapped.add("Candidate", string(record.Candidate.Target))
		mapped.add("Context", string(record.Context.Target))
		mapped.add("Workload", string(record.Workload.Target))
		mapped.add("Horizon", record.Horizon)
		mapped.add("Basis", string(record.Basis))
		mapped.add("Version", record.Version)
		mapped.add("Dependence", string(record.Dependence))
		mapped.add("Sampling", record.Sampling)
		mapped.add("Confidence method", string(record.ConfidenceMethod))
		if record.Confidence != nil {
			mapped.add("Confidence", record.Confidence.Text('g', -1))
		}
		if record.Trials != nil {
			mapped.add("Trials", record.Trials.Text('g', -1))
		}
		if record.Failures != nil {
			mapped.add("Failures", record.Failures.Text('g', -1))
		}
		c.Evidence = append(c.Evidence, mapped)
	}
	for _, record := range b.res.Owners {
		mapped := Item{ID: string(record.Extension), Text: string(record.Status), Label: labelTool}
		mapped.add("Contract", record.Contract)
		for _, r := range record.Results {
			mapped.add(r.Attachment, fmt.Sprintf("%s: %s %s", r.Subject, r.Outcome, r.Cause))
		}
		c.Owners = append(c.Owners, mapped)
	}
	return c
}

func criterionValue(v model.CriterionValue) string {
	switch v.Kind {
	case model.ValueNumber, model.ValueString, model.ValueBool:
		return scalarValue(v.Scalar)
	case model.ValueList:
		values := make([]string, 0, len(v.List))
		for _, s := range v.List {
			values = append(values, scalarValue(s))
		}
		return strings.Join(values, ", ")
	case model.ValueInterval:
		if v.Interval.Min != nil && v.Interval.Max != nil {
			return v.Interval.Min.Text('g', -1) + " – " + v.Interval.Max.Text('g', -1)
		}
	}
	return ""
}

func scalarValue(v model.Scalar) string {
	switch v.Kind {
	case model.ValueNumber:
		if v.Number != nil {
			return v.Number.Text('g', -1)
		}
	case model.ValueString:
		return v.Text
	case model.ValueBool:
		return fmt.Sprint(v.Flag)
	}
	return ""
}

func information(value assessment.Information) string {
	if value.Status == assessment.StatusFinite {
		return fmt.Sprintf("%.6g", value.Value)
	}
	return value.Status
}

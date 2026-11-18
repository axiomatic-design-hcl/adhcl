package diagram

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

func TestRenderEscapesSourceAndIsDeterministic(t *testing.T) {
	r := validate.Run(validate.Request{Path: filepath.Join("..", "..", "examples", "architecture.ad.hcl"), Profile: "source"})
	r.Snapshot.Graph.Core.Needs[0].Statement = `<script>alert("injected")</script>`
	page, err := Build(r)
	if err != nil {
		t.Fatal(err)
	}
	var a, b bytes.Buffer
	if err = Render(&a, page); err != nil {
		t.Fatal(err)
	}
	if err = Render(&b, page); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("render is not deterministic")
	}
	if strings.Contains(a.String(), `<script>alert`) || !strings.Contains(a.String(), `&lt;script&gt;`) {
		t.Fatal("source was not escaped")
	}
}

func TestRenderPresentationContract(t *testing.T) {
	r := validate.Run(validate.Request{Path: filepath.Join("..", "..", "examples", "architecture.ad.hcl"), Profile: "source"})
	page, err := Build(r)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Render(&out, page); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<section class="panel" id="start">`, `<section class="panel" id="business">`, `<section class="panel" id="product">`,
		`<section class="panel" id="architecture">`, `<section class="panel" id="engineering">`, `<section class="panel" id="trace">`,
		`<section class="panel" id="risks">`, `<section class="panel" id="guide">`,
		`<span class="tag tool">adhcl</span>`,
		`<span class="tag gap">Not in source</span>`,
		`id="impact-data" type="application/json"`, `"impacts":`, `"initial":`,
		`<text class="map-label" x="230" y="20">REQUIREMENTS</text>`,
		`width="190" height="96"`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("rendered page missing %q", want)
		}
	}
	if len(page.Impacts) == 0 {
		t.Fatal("example has no impacts")
	}
	data, err := json.Marshal(page.Impacts[0])
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"DP", "Affected", "Possible", "Needs", "Scenarios", "Modules", "PVs"} {
		if _, ok := fields[name]; !ok {
			t.Errorf("impact JSON missing field %q", name)
		}
	}
}

func TestRenderUsesTraceMapGeometry(t *testing.T) {
	r := validate.Run(validate.Request{Path: filepath.Join("..", "..", "examples", "architecture.ad.hcl"), Profile: "source"})
	page, err := Build(r)
	if err != nil {
		t.Fatal(err)
	}
	page.Map.NodeWidth = 211
	page.Map.NodeHeight = 103
	page.Map.Columns[1] = 251
	page.Map.RuleStart = 251
	page.Map.RuleEnd = 701
	page.Map.Rules = []int{42}
	var out bytes.Buffer
	if err := Render(&out, page); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`width="211" height="103"`,
		`<text class="map-label" x="251" y="20">REQUIREMENTS</text>`,
		`<line x1="251" x2="701" y1="42" y2="42"`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("rendered page missing %q", want)
		}
	}
}

func TestMapNodesDoNotOverlap(t *testing.T) {
	r := validate.Run(validate.Request{Path: filepath.Join("..", "..", "examples", "architecture.ad.hcl"), Profile: "source"})
	p, err := Build(r)
	if err != nil {
		t.Fatal(err)
	}
	for i, a := range p.Map.Nodes {
		for _, b := range p.Map.Nodes[i+1:] {
			if a.X == b.X && a.Y < b.Y+96 && b.Y < a.Y+96 {
				t.Errorf("overlapping nodes %s and %s", a.ID, b.ID)
			}
		}
	}
}

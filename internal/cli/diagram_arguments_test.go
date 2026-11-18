package cli

import "testing"

func TestDiagramArguments(t *testing.T) {
	for _, tc := range []struct {
		args         []string
		output, path string
		valid        bool
	}{
		{[]string{"diagram", "model.ad.hcl"}, "", "model.ad.hcl", true},
		{[]string{"diagram", "-o", "out.html", "model.ad.hcl"}, "out.html", "model.ad.hcl", true},
		{[]string{"diagram", "--output=out.html", "--", "-model"}, "out.html", "-model", true},
		{[]string{"diagram", "--output", "out.html", "model.ad.hcl"}, "out.html", "model.ad.hcl", true},
		{[]string{"diagram"}, "", "", false},
		{[]string{"diagram", "-o"}, "", "", false},
		{[]string{"diagram", "-ox", "model"}, "", "", false},
		{[]string{"diagram", "-o", "x", "--output=y", "model"}, "", "", false},
		{[]string{"validate", "-o", "x", "model"}, "", "", false},
		{[]string{"validate", "--output=x", "model"}, "", "", false},
		{[]string{"validate", "--output", "x", "model"}, "", "", false},
		{[]string{"diagram", "-o=x", "model"}, "", "", false},
		{[]string{"diagram", "--output=x", "-o", "y", "model"}, "", "", false},
	} {
		got, err := parseArguments(tc.args, "test")
		if (err == nil) != tc.valid {
			t.Errorf("%v: error %v", tc.args, err)
			continue
		}
		if tc.valid && (got.request.Output != tc.output || got.request.Path != tc.path || got.command != "diagram") {
			t.Errorf("%v: got %+v", tc.args, got.request)
		}
	}
}

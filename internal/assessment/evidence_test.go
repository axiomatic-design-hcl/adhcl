package assessment

import (
	"math"
	"testing"
)

// evidenceTolerance bounds the allowed absolute error for expected numeric values.
const evidenceTolerance = 1e-9

func TestWilsonIntervalFiftyTrialsFiveFailuresAt95Percent(t *testing.T) {
	// Given
	// The success fraction is p = 45/50 = 0.9 and z = 1.959963984540054.
	// z^2 = 3.8414588206941254, so 1 + z^2/50 = 1.0768291764138824.
	// center = (0.9 + z^2/100) / 1.0768291764138824 = 0.8714609603466565.
	// half = z * sqrt(0.9*0.1/50 + z^2/10000) / 1.0768291764138824 = 0.08506327472145307.
	// low = 0.7863976856252035 and high = 0.9565242350681096.
	// When
	low, high, err := WilsonInterval(50, 5, 0.95)

	// Then
	if err != nil {
		t.Fatalf("WilsonInterval(50, 5, 0.95) returned error %v, want none", err)
	}
	requireNear(t, "low", low, 0.7863976856252035)
	requireNear(t, "high", high, 0.9565242350681096)
}

func TestWilsonIntervalAllSuccessKeepsLowerUncertainty(t *testing.T) {
	// Given
	// With 20 trials and 0 failures, p = 1 and the Wilson closed form gives
	// low = 20 / (20 + z^2) = 0.8388748419471806.
	// The upper bound is exactly 1 in the formula (center + half = 1).
	// Clipping only absorbs roundoff, so the bound stays at 1 within tolerance.
	// When
	low, high, err := WilsonInterval(20, 0, 0.95)

	// Then
	if err != nil {
		t.Fatalf("WilsonInterval(20, 0, 0.95) returned error %v, want none", err)
	}
	requireNear(t, "low", low, 0.8388748419471806)
	requireNear(t, "high", high, 1)
	if !(low < 1) {
		t.Errorf("low is %.16g, want less than 1", low)
	}
}

func TestWilsonIntervalAllFailureKeepsUpperUncertainty(t *testing.T) {
	// Given
	// With 20 trials and 20 failures, p = 0 and the Wilson closed form gives
	// low = 0 after clipping roundoff.
	// high = z^2 / (20 + z^2) = 0.16112515805281938.
	// When
	low, high, err := WilsonInterval(20, 20, 0.95)

	// Then
	if err != nil {
		t.Fatalf("WilsonInterval(20, 20, 0.95) returned error %v, want none", err)
	}
	requireNear(t, "low", low, 0)
	requireNear(t, "high", high, 0.16112515805281938)
	if !(high > 0) {
		t.Errorf("high is %.16g, want greater than 0", high)
	}
}

func TestWilsonIntervalSingleSuccessTrial(t *testing.T) {
	// Given
	// With 1 trial and 0 failures, p = 1 and the Wilson closed form gives
	// low = 1 / (1 + z^2) = 0.20654931437723745.
	// The upper bound is exactly 1 in the formula.
	// When
	low, high, err := WilsonInterval(1, 0, 0.95)

	// Then
	if err != nil {
		t.Fatalf("WilsonInterval(1, 0, 0.95) returned error %v, want none", err)
	}
	requireNear(t, "low", low, 0.20654931437723745)
	requireNear(t, "high", high, 1)
}

func TestWilsonIntervalRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name       string
		trials     int64
		failures   int64
		confidence float64
	}{
		{name: "zero trials", trials: 0, failures: 0, confidence: 0.95},
		{name: "negative failures", trials: 20, failures: -1, confidence: 0.95},
		{name: "failures above trials", trials: 20, failures: 21, confidence: 0.95},
		{name: "zero confidence", trials: 20, failures: 5, confidence: 0},
		{name: "unit confidence", trials: 20, failures: 5, confidence: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			// The input in this case is outside the accepted range for trials,
			// failures, or confidence.

			// When
			_, _, err := WilsonInterval(tc.trials, tc.failures, tc.confidence)

			// Then
			if err == nil {
				t.Errorf("WilsonInterval(%d, %d, %g) returned no error, want an error", tc.trials, tc.failures, tc.confidence)
			}
		})
	}
}

func TestInformationBoundsFullSuccessRangeGivesZeroBestAndOneBitWorst(t *testing.T) {
	// Given
	// The success probability range is [0.5, 1].
	// Best uses the high probability: -log2(1) = 0.
	// Worst uses the low probability: -log2(0.5) = 1.
	// When
	best, worst := InformationBounds(0.5, 1)

	// Then
	if best.Status != "finite" {
		t.Fatalf("best status is %q, want \"finite\"", best.Status)
	}
	requireNear(t, "best value", best.Value, 0)
	if worst.Status != "finite" {
		t.Fatalf("worst status is %q, want \"finite\"", worst.Status)
	}
	requireNear(t, "worst value", worst.Value, 1)
}

func TestInformationBoundsZeroLowGivesInfiniteWorst(t *testing.T) {
	// Given
	// The low success probability is 0, so -log2(0) is infinite.
	// When
	_, worst := InformationBounds(0, 1)

	// Then
	if worst.Status != "infinite" {
		t.Errorf("worst status is %q, want \"infinite\"", worst.Status)
	}
}

func TestInformationBoundsUsesHighForBestAndLowForWorst(t *testing.T) {
	// Given
	// The success probability range is [0.25, 0.5].
	// Best uses the high probability: -log2(0.5) = 1.
	// Worst uses the low probability: -log2(0.25) = 2.
	// When
	best, worst := InformationBounds(0.25, 0.5)

	// Then
	if best.Status != "finite" {
		t.Fatalf("best status is %q, want \"finite\"", best.Status)
	}
	requireNear(t, "best value", best.Value, 1)
	if worst.Status != "finite" {
		t.Fatalf("worst status is %q, want \"finite\"", worst.Status)
	}
	requireNear(t, "worst value", worst.Value, 2)
}

func TestInformationBoundsUnknownLowGivesUnknownWorst(t *testing.T) {
	// Given
	// NaN stands for an unknown input because the bounds are float64 values.
	// The low probability is unknown, so the worst bound is unknown.
	// When
	_, worst := InformationBounds(math.NaN(), 0.5)

	// Then
	if worst.Status != "unknown" {
		t.Errorf("worst status is %q, want \"unknown\"", worst.Status)
	}
}

func TestInformationBoundsUnknownHighGivesUnknownBest(t *testing.T) {
	// Given
	// NaN stands for an unknown input because the bounds are float64 values.
	// The high probability is unknown, so the best bound is unknown.
	// When
	best, _ := InformationBounds(0.5, math.NaN())

	// Then
	if best.Status != "unknown" {
		t.Errorf("best status is %q, want \"unknown\"", best.Status)
	}
}

// requireNear fails the test unless got is within evidenceTolerance of want.
// A NaN got always fails, because the comparison is written so that NaN is not within tolerance.
func requireNear(t *testing.T, name string, got, want float64) {
	t.Helper()
	if !(math.Abs(got-want) <= evidenceTolerance) {
		t.Errorf("%s is %.16g, want %.16g within %g", name, got, want, evidenceTolerance)
	}
}

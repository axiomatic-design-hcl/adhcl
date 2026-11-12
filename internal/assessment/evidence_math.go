package assessment

import (
	"fmt"
	"math"
)

// Information statuses for machine-readable bounds.
const (
	StatusFinite   = "finite"
	StatusInfinite = "infinite"
	StatusUnknown  = "unknown"
)

// Information is the information content of a probability, in bits.
// Value is meaningful only when Status is StatusFinite.
type Information struct {
	Status string
	Value  float64
}

// Breakpoints and coefficients of Acklam's rational approximation to the
// inverse normal CDF. Coefficients are listed from highest to lowest degree.
const (
	acklamLowerBreak = 0.02425
	acklamUpperBreak = 1 - acklamLowerBreak
)

var (
	acklamCentralNumerator = []float64{
		-3.969683028665376e+01, 2.209460984245205e+02, -2.759285104469687e+02,
		1.383577518672690e+02, -3.066479806614716e+01, 2.506628277459239e+00,
	}
	acklamCentralDenominator = []float64{
		-5.447609879822406e+01, 1.615858368580409e+02, -1.556989798598866e+02,
		6.680131188771972e+01, -1.328068155288572e+01, 1,
	}
	acklamTailNumerator = []float64{
		-7.784894002430293e-03, -3.223964580411365e-01, -2.400758277161838e+00,
		-2.549732539343734e+00, 4.374664141464968e+00, 2.938163982698783e+00,
	}
	acklamTailDenominator = []float64{
		7.784695709041462e-03, 3.224671290700398e-01, 2.445134137142996e+00,
		3.754408661907416e+00, 1,
	}
)

// WilsonInterval returns the two-sided Wilson score interval, without continuity
// correction, for the success fraction of trials with failures.
func WilsonInterval(trials, failures int64, confidence float64) (low, high float64, err error) {
	if err = validateWilsonInput(trials, failures, confidence); err != nil {
		return 0, 0, err
	}
	z := normalQuantile((1 + confidence) / 2)
	n := float64(trials)
	p := float64(trials-failures) / n
	zSquared := z * z
	denominator := 1 + zSquared/n
	center := (p + zSquared/(2*n)) / denominator
	half := z * math.Sqrt(p*(1-p)/n+zSquared/(4*n*n)) / denominator
	return clipUnit(center - half), clipUnit(center + half), nil
}

// InformationBounds converts a probability range into information bounds.
// The best bound comes from the high probability and the worst from the low one.
func InformationBounds(low, high float64) (best, worst Information) {
	return informationFromProbability(high), informationFromProbability(low)
}

func validateWilsonInput(trials, failures int64, confidence float64) error {
	switch {
	case trials < 1:
		return fmt.Errorf("trials must be at least 1, got %d", trials)
	case failures < 0:
		return fmt.Errorf("failures must not be negative, got %d", failures)
	case failures > trials:
		return fmt.Errorf("failures must not exceed trials, got %d failures in %d trials", failures, trials)
	case !(confidence > 0 && confidence < 1):
		return fmt.Errorf("confidence must be greater than 0 and less than 1, got %g", confidence)
	}
	return nil
}

// clipUnit absorbs roundoff that moves a probability bound just outside [0, 1].
func clipUnit(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

// informationFromProbability returns -log2(p). A probability outside [0, 1] is
// not a probability, so its information is unknown.
func informationFromProbability(probability float64) Information {
	switch {
	case math.IsNaN(probability), probability < 0, probability > 1:
		return Information{Status: StatusUnknown}
	case probability == 0:
		return Information{Status: StatusInfinite}
	case probability == 1:
		return Information{Status: StatusFinite, Value: 0}
	default:
		return Information{Status: StatusFinite, Value: -math.Log2(probability)}
	}
}

// normalQuantile returns the standard normal quantile for probability, which
// must lie strictly between 0 and 1.
func normalQuantile(probability float64) float64 {
	return refineNormalQuantile(acklamApproximation(probability), probability)
}

func acklamApproximation(probability float64) float64 {
	switch {
	case probability < acklamLowerBreak:
		return lowerTailApproximation(probability)
	case probability > acklamUpperBreak:
		return -lowerTailApproximation(1 - probability)
	default:
		return centralApproximation(probability)
	}
}

func lowerTailApproximation(probability float64) float64 {
	q := math.Sqrt(-2 * math.Log(probability))
	return evaluatePolynomial(acklamTailNumerator, q) / evaluatePolynomial(acklamTailDenominator, q)
}

func centralApproximation(probability float64) float64 {
	q := probability - 0.5
	r := q * q
	return evaluatePolynomial(acklamCentralNumerator, r) * q / evaluatePolynomial(acklamCentralDenominator, r)
}

// refineNormalQuantile applies one Halley step to an approximate quantile x.
func refineNormalQuantile(x, probability float64) float64 {
	residual := 0.5*math.Erfc(-x/math.Sqrt2) - probability
	u := residual * math.Sqrt(2*math.Pi) * math.Exp(x*x/2)
	return x - u/(1+x*u/2)
}

// evaluatePolynomial evaluates coefficients, highest degree first, with Horner's method.
func evaluatePolynomial(coefficients []float64, x float64) float64 {
	result := 0.0
	for _, coefficient := range coefficients {
		result = result*x + coefficient
	}
	return result
}

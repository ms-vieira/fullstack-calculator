package calculator

import (
	"math"
	"testing"
)

func TestPercentageRoundsOnlyOnce(t *testing.T) {
	calculator := New()
	tests := []struct {
		name string
		a, b float64
		want float64
	}{
		{"underflow boundary", 1e-150, 2.4703282292062328e-172, math.SmallestNonzeroFloat64},
		{"negative underflow boundary", -1e-150, 2.4703282292062328e-172, -math.SmallestNonzeroFloat64},
		{"above underflow boundary", 1e-150, math.Nextafter(2.4703282292062328e-172, math.Inf(1)), math.SmallestNonzeroFloat64},
		{"overflow boundary", 4.172657595118774e272, 4.308268996155542e37, math.MaxFloat64},
		{"negative overflow boundary", -4.172657595118774e272, 4.308268996155542e37, -math.MaxFloat64},
		{"below overflow boundary", 4.172657595118774e272, math.Nextafter(4.308268996155542e37, 0), math.Nextafter(math.MaxFloat64, 0)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := calculator.Calculate(Percent, test.a, test.b)
			if err != nil || got != test.want {
				t.Fatalf("got %.17g, %v; want %.17g", got, err, test.want)
			}
		})
	}
}

func TestPercentageStillRejectsOutOfRangeResults(t *testing.T) {
	calculator := New()
	_, err := calculator.Calculate(Percent, 1e-150, math.Nextafter(2.4703282292062328e-172, 0))
	assertCode(t, err, "NUMERIC_UNDERFLOW", "")
	_, err = calculator.Calculate(Percent, 4.172657595118774e272, math.Nextafter(4.308268996155542e37, math.Inf(1)))
	assertCode(t, err, "NUMERIC_OVERFLOW", "")
}

func TestPercentageRejectsNonfiniteOperands(t *testing.T) {
	_, err := (Percentage{}).Calculate(math.NaN(), 1)
	assertCode(t, err, "INVALID_NUMBER", "a")
	_, err = (Percentage{}).Calculate(1, math.Inf(1))
	assertCode(t, err, "INVALID_NUMBER", "b")
}

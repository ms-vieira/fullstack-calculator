package calculator

import (
	"math"
	"testing"
)

func TestAdvancedOperations(t *testing.T) {
	calculator := New()
	cases := []struct {
		name     string
		op       Operation
		operands []float64
		want     float64
	}{
		{"power", Power, []float64{2, 10}, 1024},
		{"negative exponent", Power, []float64{2, -3}, 0.125},
		{"negative base odd", Power, []float64{-2, 3}, -8},
		{"negative base even", Power, []float64{-2, 4}, 16},
		{"negative base negative exponent", Power, []float64{-2, -3}, -0.125},
		{"fractional exponent", Power, []float64{4, 0.5}, 2},
		{"zero exponent", Power, []float64{math.MaxFloat64, 0}, 1},
		{"zero base", Power, []float64{0, 2}, 0},
		{"negative zero base", Power, []float64{math.Copysign(0, -1), 3}, 0},
		{"smallest power", Power, []float64{2, -1074}, math.SmallestNonzeroFloat64},
		{"square root", Sqrt, []float64{9}, 3},
		{"zero square root", Sqrt, []float64{0}, 0},
		{"negative zero square root", Sqrt, []float64{math.Copysign(0, -1)}, 0},
		{"subnormal square root", Sqrt, []float64{math.SmallestNonzeroFloat64}, math.Sqrt(math.SmallestNonzeroFloat64)},
		{"maximum square root", Sqrt, []float64{math.MaxFloat64}, math.Sqrt(math.MaxFloat64)},
		{"percentage", Percent, []float64{10, 200}, 20},
		{"negative percentage", Percent, []float64{-10, 200}, -20},
		{"negative base percentage", Percent, []float64{10, -200}, -20},
		{"percentage above 100", Percent, []float64{150, 200}, 300},
		{"zero percentage", Percent, []float64{0, math.MaxFloat64}, 0},
		{"percentage of zero", Percent, []float64{-10, 0}, 0},
		{"avoid intermediate overflow", Percent, []float64{1e308, 2}, 2e306},
		{"maximum percentage", Percent, []float64{100, math.MaxFloat64}, math.MaxFloat64},
		{"maximum percentage reversed", Percent, []float64{math.MaxFloat64, 100}, math.MaxFloat64},
		{"avoid intermediate underflow", Percent, []float64{math.SmallestNonzeroFloat64, 100}, math.SmallestNonzeroFloat64},
		{"smallest percentage of 100", Percent, []float64{100, math.SmallestNonzeroFloat64}, math.SmallestNonzeroFloat64},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for attempt := 0; attempt < 3; attempt++ {
				got, err := calculator.Calculate(tc.op, tc.operands...)
				if err != nil || math.IsInf(got, 0) || math.IsNaN(got) || math.Abs(got-tc.want) > math.Abs(tc.want)*1e-14 {
					t.Fatalf("got %g, %v; want %g", got, err, tc.want)
				}
				if got == 0 && math.Signbit(got) {
					t.Fatal("negative zero was not normalized")
				}
			}
		})
	}
}

func TestAdvancedOperationErrors(t *testing.T) {
	calculator := New()
	cases := []struct {
		name, code, field string
		op                Operation
		operands          []float64
	}{
		{"zero to zero", "INVALID_DOMAIN", "b", Power, []float64{0, 0}},
		{"zero to negative", "INVALID_DOMAIN", "b", Power, []float64{0, -1}},
		{"negative zero to negative", "INVALID_DOMAIN", "b", Power, []float64{math.Copysign(0, -1), -1}},
		{"non-real power", "INVALID_DOMAIN", "b", Power, []float64{-4, 0.5}},
		{"power overflow", "NUMERIC_OVERFLOW", "", Power, []float64{1e308, 2}},
		{"power underflow", "NUMERIC_UNDERFLOW", "", Power, []float64{2, -1075}},
		{"negative square root", "INVALID_DOMAIN", "a", Sqrt, []float64{-1}},
		{"nonfinite unary input", "INVALID_NUMBER", "a", Sqrt, []float64{math.NaN()}},
		{"infinite unary input", "INVALID_NUMBER", "a", Sqrt, []float64{math.Inf(1)}},
		{"percentage overflow", "NUMERIC_OVERFLOW", "", Percent, []float64{math.MaxFloat64, math.MaxFloat64}},
		{"percentage underflow", "NUMERIC_UNDERFLOW", "", Percent, []float64{math.SmallestNonzeroFloat64, 1}},
		{"missing unary operand", "INVALID_REQUEST", "", Sqrt, nil},
		{"extra unary operand", "INVALID_REQUEST", "", Sqrt, []float64{4, 2}},
		{"missing binary operand", "INVALID_REQUEST", "", Power, []float64{2}},
		{"extra binary operand", "INVALID_REQUEST", "", Percent, []float64{2, 3, 4}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := calculator.Calculate(tc.op, tc.operands...)
			assertCode(t, err, tc.code, tc.field)
		})
	}
}

func TestOperandCount(t *testing.T) {
	calculator := New()
	for _, op := range []Operation{Add, Subtract, Multiply, Divide, Power, Percent} {
		if calculator.OperandCount(op) != 2 {
			t.Errorf("%s must require two operands", op)
		}
	}
	if calculator.OperandCount(Sqrt) != 1 || calculator.OperandCount("unknown") != 0 {
		t.Fatal("incorrect unary or unknown operand count")
	}
}

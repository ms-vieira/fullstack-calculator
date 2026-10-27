package calculator

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"strings"
	"testing"
)

func assertCode(t *testing.T, err error, expected, field string) {
	t.Helper()
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != expected || appErr.Field != field {
		t.Fatalf("error = %#v, want code=%s field=%s", err, expected, field)
	}
}

func TestParseNumber(t *testing.T) {
	tests := []struct {
		name, input, code string
		want              float64
	}{
		{name: "zero", input: "0", want: 0},
		{name: "negative zero", input: "-0", want: 0},
		{name: "negative decimal", input: "-12.5", want: -12.5},
		{name: "outer spaces", input: " 12.5 ", want: 12.5},
		{name: "scientific", input: "1E-3", want: 0.001},
		{name: "positive exponent", input: "1e+3", want: 1000},
		{name: "subnormal", input: "5e-324", want: math.SmallestNonzeroFloat64},
		{name: "zero with small exponent", input: "0e-400", want: 0},
		{name: "negative zero with small exponent", input: "-0.0e-400", want: 0},
		{name: "empty", input: "", code: "REQUIRED_FIELD"},
		{name: "blank", input: "  ", code: "REQUIRED_FIELD"},
		{name: "numeric prefix", input: "12abc", code: "INVALID_NUMBER"},
		{name: "hex", input: "0x10", code: "INVALID_NUMBER"},
		{name: "comma", input: "1,5", code: "INVALID_NUMBER"},
		{name: "separator", input: "1_000", code: "INVALID_NUMBER"},
		{name: "NaN", input: "NaN", code: "INVALID_NUMBER"},
		{name: "infinity", input: "Infinity", code: "INVALID_NUMBER"},
		{name: "leading decimal point", input: ".5", code: "INVALID_NUMBER"},
		{name: "trailing decimal point", input: "5.", code: "INVALID_NUMBER"},
		{name: "plus sign", input: "+5", code: "INVALID_NUMBER"},
		{name: "leading zero", input: "01", code: "INVALID_NUMBER"},
		{name: "unfinished exponent", input: "1e-", code: "INVALID_NUMBER"},
		{name: "positive overflow", input: "1e309", code: "NUMBER_OUT_OF_RANGE"},
		{name: "negative overflow", input: "-1e309", code: "NUMBER_OUT_OF_RANGE"},
		{name: "positive underflow", input: "1e-400", code: "NUMERIC_UNDERFLOW"},
		{name: "negative underflow", input: "-1e-400", code: "NUMERIC_UNDERFLOW"},
		{name: "mantissa zeroes", input: "0.00001e-400", code: "NUMERIC_UNDERFLOW"},
		{name: "maximum length", input: "0." + strings.Repeat("0", 125) + "1", want: 1e-126},
		{name: "above maximum length", input: "0." + strings.Repeat("0", 126) + "1", code: "INPUT_TOO_LONG"},
		{name: "spaces count toward limit", input: strings.Repeat(" ", 128) + "1", code: "INPUT_TOO_LONG"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseNumber(test.input, "a")
			if test.code != "" {
				assertCode(t, err, test.code, "a")
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("ParseNumber(%q) = %g, %v; want %g", test.input, got, err, test.want)
			}
		})
	}
}

func TestCalculate(t *testing.T) {
	calculator := New()
	tests := []struct {
		name string
		op   Operation
		a, b float64
		want float64
	}{
		{"add", Add, 10, 5, 15},
		{"subtract order", Subtract, 5, 10, -5},
		{"multiply negatives", Multiply, -3, -4, 12},
		{"divide order", Divide, -10, 4, -2.5},
		{"zero add", Add, 0, 0, 0},
		{"zero product", Multiply, 0, -42, 0},
		{"zero numerator", Divide, 0, 5, 0},
		{"cancellation", Subtract, 1e300, 1e300, 0},
		{"binary approximation", Add, 0.1, 0.2, 0.30000000000000004},
		{"repeating decimal", Divide, 1, 3, 0.3333333333333333},
		{"subnormal result", Divide, math.SmallestNonzeroFloat64, 1, math.SmallestNonzeroFloat64},
		{"small valid divisor", Divide, 1e-300, 1e-300, 1},
		{"large integer approximation", Add, 9007199254740992, 1, 9007199254740992},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for attempt := 0; attempt < 3; attempt++ {
				got, err := calculator.Calculate(test.op, test.a, test.b)
				if err != nil || got != test.want {
					t.Fatalf("Calculate(%s, %g, %g) = %g, %v; want %g", test.op, test.a, test.b, got, err, test.want)
				}
				if got == 0 && math.Signbit(got) {
					t.Fatal("result must normalize negative zero")
				}
			}
		})
	}
	result, err := calculator.Calculate(Multiply, math.Copysign(0, -1), 1)
	if err != nil || math.Signbit(result) {
		t.Fatalf("negative zero was not normalized: %g, %v", result, err)
	}
}

func TestCalculateErrors(t *testing.T) {
	calculator := New()
	tests := []struct {
		name        string
		op          Operation
		a, b        float64
		code, field string
	}{
		{"operation", "unknown", 1, 2, "INVALID_OPERATION", "operation"},
		{"NaN first", Add, math.NaN(), 2, "INVALID_NUMBER", "a"},
		{"infinite first", Add, math.Inf(1), 2, "INVALID_NUMBER", "a"},
		{"infinite second", Add, 1, math.Inf(-1), "INVALID_NUMBER", "b"},
		{"NaN second", Add, 1, math.NaN(), "INVALID_NUMBER", "b"},
		{"divide by zero", Divide, 5, 0, "DIVISION_BY_ZERO", "b"},
		{"zero by zero", Divide, 0, 0, "DIVISION_BY_ZERO", "b"},
		{"divide by negative zero", Divide, 5, math.Copysign(0, -1), "DIVISION_BY_ZERO", "b"},
		{"product overflow", Multiply, 1e308, 10, "NUMERIC_OVERFLOW", ""},
		{"sum overflow", Add, 1e308, 1e308, "NUMERIC_OVERFLOW", ""},
		{"difference overflow", Subtract, -1e308, 1e308, "NUMERIC_OVERFLOW", ""},
		{"division overflow", Divide, 1, 1e-320, "NUMERIC_OVERFLOW", ""},
		{"product underflow", Multiply, 1e-300, 1e-300, "NUMERIC_UNDERFLOW", ""},
		{"division underflow", Divide, 1e-300, 1e100, "NUMERIC_UNDERFLOW", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for attempt := 0; attempt < 3; attempt++ {
				_, err := calculator.Calculate(test.op, test.a, test.b)
				assertCode(t, err, test.code, test.field)
			}
		})
	}
}

func TestSharedNumericCases(t *testing.T) {
	body, err := os.ReadFile("../../../testdata/numeric-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures struct {
		Cases []struct {
			Name  string  `json:"name"`
			Input string  `json:"input"`
			Value float64 `json:"value"`
			Code  string  `json:"code"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(body, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures.Cases) == 0 {
		t.Fatal("shared numeric fixtures must not be empty")
	}
	for _, test := range fixtures.Cases {
		t.Run(test.Name, func(t *testing.T) {
			value, err := ParseNumber(test.Input, "a")
			if test.Code != "" {
				assertCode(t, err, test.Code, "a")
			} else if err != nil || value != test.Value {
				t.Fatalf("got %g, %v; want %g", value, err, test.Value)
			}
		})
	}
}

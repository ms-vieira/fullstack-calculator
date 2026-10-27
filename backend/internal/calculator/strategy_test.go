package calculator

import (
	"errors"
	"math"
	"testing"
)

func TestStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy BinaryOperation
		a, b     float64
		want     float64
	}{
		{"addition", Addition{}, -12.5, 2, -10.5},
		{"subtraction preserves order", Subtraction{}, 2, 12.5, -10.5},
		{"multiplication", Multiplication{}, -3, -4, 12},
		{"division preserves order", Division{}, -10, 4, -2.5},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.strategy.Calculate(test.a, test.b)
			if err != nil || got != test.want {
				t.Fatalf("got %g, %v; want %g", got, err, test.want)
			}
		})
	}
}

func TestDivisionByZero(t *testing.T) {
	for _, a := range []float64{0, 5} {
		for _, b := range []float64{0, math.Copysign(0, -1)} {
			_, err := (Division{}).Calculate(a, b)
			assertCode(t, err, "DIVISION_BY_ZERO", "b")
		}
	}
}

type binaryOperationFunc func(float64, float64) (float64, error)

func (f binaryOperationFunc) Calculate(a, b float64) (float64, error) {
	return f(a, b)
}

func TestValidOperation(t *testing.T) {
	calculator := New()
	for _, operation := range []Operation{Add, Subtract, Multiply, Divide, Power, Sqrt, Percent} {
		if !calculator.ValidOperation(operation) {
			t.Errorf("supported operation %q was rejected", operation)
		}
	}
	for _, operation := range []Operation{"", "log", "ADD", "unknown"} {
		if calculator.ValidOperation(operation) {
			t.Errorf("unsupported operation %q was accepted", operation)
		}
	}
}

func TestCalculatorDispatchesStrategy(t *testing.T) {
	const operation Operation = "custom"
	var calls int
	calculator := &Calculator{operations: map[Operation]operationDefinition{
		operation: {strategy: binaryOperationFunc(func(a, b float64) (float64, error) {
			calls++
			if a != 2 || b != 3 {
				t.Fatalf("strategy received %g, %g; want 2, 3", a, b)
			}
			return 17, nil
		})},
	}}
	if !calculator.ValidOperation(operation) {
		t.Fatal("operation validation must use the same registry as dispatch")
	}
	got, err := calculator.Calculate(operation, 2, 3)
	if err != nil || got != 17 || calls != 1 {
		t.Fatalf("got result=%g error=%v calls=%d; want 17, nil, 1", got, err, calls)
	}
}

func TestCalculatorRejectsInvalidInputBeforeStrategy(t *testing.T) {
	tests := []struct {
		name        string
		operation   Operation
		a, b        float64
		code, field string
	}{
		{"unknown operation", "unknown", 1, 2, "INVALID_OPERATION", "operation"},
		{"nonfinite first operand", Add, math.Inf(1), 2, "INVALID_NUMBER", "a"},
		{"nonfinite second operand", Add, 1, math.NaN(), "INVALID_NUMBER", "b"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var calls int
			calculator := &Calculator{operations: map[Operation]operationDefinition{
				Add: {strategy: binaryOperationFunc(func(float64, float64) (float64, error) {
					calls++
					return 3, nil
				})},
			}}
			_, err := calculator.Calculate(test.operation, test.a, test.b)
			assertCode(t, err, test.code, test.field)
			if calls != 0 {
				t.Fatal("invalid input reached the strategy")
			}
		})
	}
}

func TestCalculatorChecksStrategyResults(t *testing.T) {
	tests := []struct {
		name, code string
		result     float64
	}{
		{"overflow", "NUMERIC_OVERFLOW", math.Inf(1)},
		{"invalid result", "INVALID_RESULT", math.NaN()},
		{"negative zero", "", math.Copysign(0, -1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calculator := &Calculator{operations: map[Operation]operationDefinition{
				Add: {strategy: binaryOperationFunc(func(float64, float64) (float64, error) {
					return test.result, nil
				})},
			}}
			got, err := calculator.Calculate(Add, 1, 2)
			if test.code != "" {
				assertCode(t, err, test.code, "")
			} else if err != nil || got != 0 || math.Signbit(got) {
				t.Fatalf("got %g, %v; want normalized zero", got, err)
			}
		})
	}
}

func TestCalculatorPropagatesStrategyError(t *testing.T) {
	expected := errors.New("strategy failed")
	calculator := &Calculator{operations: map[Operation]operationDefinition{
		Add: {strategy: binaryOperationFunc(func(float64, float64) (float64, error) {
			return math.NaN(), expected
		})},
	}}
	got, err := calculator.Calculate(Add, 1, 2)
	if got != 0 || !errors.Is(err, expected) {
		t.Fatalf("got %g, %v; want original strategy error", got, err)
	}
}

package calculator

import "math"

type operationDefinition struct {
	strategy        BinaryOperation
	unaryStrategy   UnaryOperation
	detectUnderflow bool
}

type Calculator struct {
	operations map[Operation]operationDefinition
}

func New() *Calculator {
	return &Calculator{
		operations: map[Operation]operationDefinition{
			Add:      {strategy: Addition{}},
			Subtract: {strategy: Subtraction{}},
			Multiply: {strategy: Multiplication{}, detectUnderflow: true},
			Divide:   {strategy: Division{}, detectUnderflow: true},
			Power:    {strategy: Exponentiation{}, detectUnderflow: true},
			Sqrt:     {unaryStrategy: SquareRoot{}},
			Percent:  {strategy: Percentage{}, detectUnderflow: true},
		},
	}
}

func (c *Calculator) ValidOperation(operation Operation) bool {
	_, ok := c.operations[operation]
	return ok
}

func (c *Calculator) OperandCount(operation Operation) int {
	definition, ok := c.operations[operation]
	if !ok {
		return 0
	}
	if definition.unaryStrategy != nil {
		return 1
	}
	return 2
}

func (c *Calculator) Calculate(operation Operation, operands ...float64) (float64, error) {
	definition, ok := c.operations[operation]
	if !ok {
		return 0, &Error{Code: "INVALID_OPERATION", Message: "Choose a supported operation.", Field: "operation"}
	}
	if len(operands) != c.OperandCount(operation) {
		return 0, &Error{Code: "INVALID_REQUEST", Message: "Provide exactly the operands required by the operation."}
	}
	a, b := operands[0], 0.0
	if len(operands) == 2 {
		b = operands[1]
	}
	if err := validateOperands(a, b); err != nil {
		return 0, err
	}
	var result float64
	var err error
	if definition.unaryStrategy != nil {
		result, err = definition.unaryStrategy.Calculate(a)
	} else {
		result, err = definition.strategy.Calculate(a, b)
	}
	if err != nil {
		return 0, err
	}
	return validateResult(result, a, b, definition.detectUnderflow)
}

func validateOperands(a, b float64) error {
	if math.IsNaN(a) || math.IsInf(a, 0) {
		return &Error{Code: "INVALID_NUMBER", Message: "The first number must be finite.", Field: "a"}
	}
	if math.IsNaN(b) || math.IsInf(b, 0) {
		return &Error{Code: "INVALID_NUMBER", Message: "The second number must be finite.", Field: "b"}
	}
	return nil
}

func validateResult(result, a, b float64, detectUnderflow bool) (float64, error) {
	if math.IsInf(result, 0) {
		return 0, &Error{Code: "NUMERIC_OVERFLOW", Message: "The result exceeds the supported numeric range."}
	}
	if math.IsNaN(result) {
		return 0, &Error{Code: "INVALID_RESULT", Message: "The operation did not produce a valid result."}
	}
	if result == 0 {
		if detectUnderflow && a != 0 && b != 0 {
			return 0, &Error{Code: "NUMERIC_UNDERFLOW", Message: "The nonzero result is too small to represent."}
		}
		return 0, nil
	}
	return result, nil
}

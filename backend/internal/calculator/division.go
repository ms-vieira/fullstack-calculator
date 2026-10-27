package calculator

type Division struct{}

func (Division) Calculate(a, b float64) (float64, error) {
	if b == 0 {
		return 0, &Error{Code: "DIVISION_BY_ZERO", Message: "Cannot divide by zero.", Field: "b"}
	}
	return a / b, nil
}

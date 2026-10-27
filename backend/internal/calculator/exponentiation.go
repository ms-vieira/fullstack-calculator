package calculator

import "math"

type Exponentiation struct{}

func (Exponentiation) Calculate(a, b float64) (float64, error) {
	if a == 0 && b <= 0 {
		return 0, &Error{Code: "INVALID_DOMAIN", Message: "Zero requires a positive exponent.", Field: "b"}
	}
	if a < 0 && b != math.Trunc(b) {
		return 0, &Error{Code: "INVALID_DOMAIN", Message: "A negative base requires an integer exponent.", Field: "b"}
	}
	return math.Pow(a, b), nil
}

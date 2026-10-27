package calculator

import "math"

type SquareRoot struct{}

func (SquareRoot) Calculate(a float64) (float64, error) {
	if a < 0 {
		return 0, &Error{Code: "INVALID_DOMAIN", Message: "Square root requires a non-negative number.", Field: "a"}
	}
	return math.Sqrt(a), nil
}

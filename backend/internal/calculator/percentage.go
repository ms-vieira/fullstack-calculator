package calculator

import "math/big"

type Percentage struct{}

func (Percentage) Calculate(a, b float64) (float64, error) {
	if err := validateOperands(a, b); err != nil {
		return 0, err
	}
	percentage := new(big.Rat).SetFloat64(a)
	percentage.Mul(percentage, new(big.Rat).SetFloat64(b))
	percentage.Quo(percentage, big.NewRat(100, 1))
	result, _ := percentage.Float64()
	return result, nil
}

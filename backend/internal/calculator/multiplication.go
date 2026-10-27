package calculator

type Multiplication struct{}

func (Multiplication) Calculate(a, b float64) (float64, error) {
	return a * b, nil
}

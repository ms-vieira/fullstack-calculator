package calculator

type Subtraction struct{}

func (Subtraction) Calculate(a, b float64) (float64, error) {
	return a - b, nil
}

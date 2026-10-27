package calculator

type Addition struct{}

func (Addition) Calculate(a, b float64) (float64, error) {
	return a + b, nil
}

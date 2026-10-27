package calculator

type Operation string

const (
	Add      Operation = "add"
	Subtract Operation = "subtract"
	Multiply Operation = "multiply"
	Divide   Operation = "divide"
	Power    Operation = "power"
	Sqrt     Operation = "sqrt"
	Percent  Operation = "percent"
)

type BinaryOperation interface {
	Calculate(a, b float64) (float64, error)
}

type UnaryOperation interface {
	Calculate(a float64) (float64, error)
}

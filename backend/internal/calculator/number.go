package calculator

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

const MaxNumberLength = 128

var numberPattern = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)

func ParseNumber(original, field string) (float64, error) {
	if len(original) > MaxNumberLength {
		return 0, &Error{"INPUT_TOO_LONG", "Numbers must contain at most 128 characters.", field}
	}
	token := strings.TrimSpace(original)
	if token == "" {
		return 0, &Error{"REQUIRED_FIELD", "This field is required.", field}
	}
	if !numberPattern.MatchString(token) {
		return 0, &Error{"INVALID_NUMBER", "Enter a valid decimal number using a decimal point.", field}
	}
	value, err := strconv.ParseFloat(token, 64)
	if math.IsInf(value, 0) {
		return 0, &Error{"NUMBER_OUT_OF_RANGE", "This number exceeds the supported numeric range.", field}
	}
	if err != nil || math.IsNaN(value) {
		return 0, &Error{"INVALID_NUMBER", "Enter a valid finite number.", field}
	}
	if value == 0 && hasNonzeroMantissa(token) {
		return 0, &Error{"NUMERIC_UNDERFLOW", "This nonzero number is too small to represent.", field}
	}
	return value, nil
}

func hasNonzeroMantissa(token string) bool {
	for _, character := range token {
		if character == 'e' || character == 'E' {
			break
		}
		if character >= '1' && character <= '9' {
			return true
		}
	}
	return false
}

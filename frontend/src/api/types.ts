export const operations = [
  { value: 'add', label: 'Addition', symbol: '+', arity: 2, hint: 'Add the two numbers.' },
  { value: 'subtract', label: 'Subtraction', symbol: '−', arity: 2, hint: 'Subtract the second number from the first.' },
  { value: 'multiply', label: 'Multiplication', symbol: '×', arity: 2, hint: 'Multiply the two numbers.' },
  { value: 'divide', label: 'Division', symbol: '÷', arity: 2, hint: 'Divide the first number by the second. The divisor must be non-zero.' },
  { value: 'power', label: 'Exponentiation', symbol: '^', arity: 2, hint: 'Raise the first number to the second. Negative bases need an integer exponent; zero needs a positive exponent.' },
  { value: 'sqrt', label: 'Square Root', symbol: '√', arity: 1, hint: 'Square root of the first number, which must be non-negative. No second number is needed.' },
  { value: 'percent', label: 'Percentage', symbol: '%', arity: 2, hint: 'The first number is a percentage of the second: 10% of 200 is 20. Negative percentages are supported.' },
] as const;

export type Operation = (typeof operations)[number]['value'];
export type Field = 'operation' | 'a' | 'b';

export type CalculationRequest =
  | { operation: 'sqrt'; a: number; b?: never }
  | { operation: Exclude<Operation, 'sqrt'>; a: number; b: number };

export interface Issue {
  code: string;
  message: string;
  field?: Field;
}

const messages: Record<string, string> = {
  REQUIRED_FIELD: 'Enter a number.',
  INVALID_NUMBER: 'Use a number such as -12.5 or 1e3, with a decimal point.',
  INVALID_OPERATION: 'Choose a supported operation.',
  INPUT_TOO_LONG: 'Use no more than 128 characters.',
  NUMBER_OUT_OF_RANGE: 'This number is outside the supported range.',
  NUMERIC_OVERFLOW: 'The result is too large to represent. Try smaller numbers.',
  NUMERIC_UNDERFLOW: 'This value is too small to represent without becoming zero.',
  DIVISION_BY_ZERO: 'Cannot divide by zero. Enter a non-zero second number.',
  INVALID_RESULT: 'The calculation did not produce a supported number.',
  INVALID_DOMAIN: 'This operation is outside the supported real-number domain. Check the operation rules.',
  TIMEOUT: 'The calculation took too long. Please try again.',
  NETWORK_ERROR: 'Could not reach the calculator. Check your connection and try again.',
  SERVICE_UNAVAILABLE: 'The calculator is temporarily unavailable. Please try again shortly.',
  RATE_LIMITED: 'Too many calculations. Please wait and try again.',
  HTTP_ERROR: 'The calculator could not complete the request. Please try again.',
  INVALID_RESPONSE: 'The calculator returned an unexpected response. Please try again.',
};

export function messageFor(code: string): string {
  return Object.hasOwn(messages, code)
    ? messages[code]
    : 'The calculation could not be completed. Please try again.';
}

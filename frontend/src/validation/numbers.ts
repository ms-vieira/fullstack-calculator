import { messageFor, operations, type CalculationRequest, type Field, type Issue } from '../api/types';

const numberPattern = /^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$/;
const maxLength = 128;

type ParsedNumber = { ok: true; value: number } | { ok: false; issue: Issue };

function invalid(code: string, field: Field): ParsedNumber {
  return { ok: false, issue: issueFor(code, field) };
}

function issueFor(code: string, field: Field): Issue {
  return { code, message: messageFor(code), field };
}

export function parseNumber(input: string, field: 'a' | 'b'): ParsedNumber {
  if (input.length > maxLength) return invalid('INPUT_TOO_LONG', field);
  const token = input.trim();
  if (token === '') return invalid('REQUIRED_FIELD', field);
  if (!numberPattern.test(token)) return invalid('INVALID_NUMBER', field);

  const value = Number(token);
  if (!Number.isFinite(value)) return invalid('NUMBER_OUT_OF_RANGE', field);
  const mantissa = token.split(/[eE]/, 1)[0];
  if (value === 0 && /[1-9]/.test(mantissa)) return invalid('NUMERIC_UNDERFLOW', field);
  return { ok: true, value: value === 0 ? 0 : value };
}

export interface FormValues {
  operation: string;
  a: string;
  b: string;
}

export type ValidationResult =
  | { ok: true; request: CalculationRequest }
  | { ok: false; issues: Partial<Record<Field, Issue>> };

export function validateCalculation(values: FormValues): ValidationResult {
  const issues: Partial<Record<Field, Issue>> = {};
  const operation = operations.find(({ value }) => value === values.operation);
  if (!operation) {
    issues.operation = issueFor('INVALID_OPERATION', 'operation');
  }

  const a = parseNumber(values.a, 'a');
  const b = operation?.arity === 1 ? { ok: true as const, value: 0 } : parseNumber(values.b, 'b');
  if (!a.ok) issues.a = a.issue;
  if (!b.ok) issues.b = b.issue;
  if (!operation || !a.ok || !b.ok) return { ok: false, issues };
  if (operation.value === 'divide' && b.value === 0) {
    issues.b = issueFor('DIVISION_BY_ZERO', 'b');
  }
  if (operation.value === 'sqrt' && a.value < 0) {
    issues.a = issueFor('INVALID_DOMAIN', 'a');
  }
  if (operation.value === 'power' && ((a.value === 0 && b.value <= 0) || (a.value < 0 && !Number.isInteger(b.value)))) {
    issues.b = issueFor('INVALID_DOMAIN', 'b');
  }
  if (Object.keys(issues).length > 0) return { ok: false, issues };
  if (operation.value === 'sqrt') return { ok: true, request: { operation: 'sqrt', a: a.value } };
  return { ok: true, request: { operation: operation.value, a: a.value, b: b.value } };
}

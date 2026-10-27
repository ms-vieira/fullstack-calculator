import { describe, expect, it } from 'vitest';
import cases from '../../../testdata/numeric-cases.json';
import { parseNumber, validateCalculation } from './numbers';

describe('parseNumber', () => {
  for (const test of cases.cases) {
    it(test.name, () => {
      const parsed = parseNumber(test.input, 'a');
      if (test.code) {
        expect(parsed.ok).toBe(false);
        if (!parsed.ok) expect(parsed.issue.code).toBe(test.code);
      } else {
        expect(parsed.ok).toBe(true);
        if (parsed.ok) expect(parsed.value).toBe(test.value);
      }
    });
  }

  it('checks the original length before trimming and preserves long pasted input', () => {
    const parsed = parseNumber(`${' '.repeat(128)}1`, 'a');
    expect(parsed).toMatchObject({ ok: false, issue: { code: 'INPUT_TOO_LONG', field: 'a' } });
  });

  it('accepts a valid 128-character decimal and rejects 129 characters', () => {
    expect(parseNumber(`0.${'0'.repeat(125)}1`, 'a').ok).toBe(true);
    expect(parseNumber(`0.${'0'.repeat(126)}1`, 'a')).toMatchObject({ ok: false, issue: { code: 'INPUT_TOO_LONG' } });
  });
});

describe('validateCalculation', () => {
  it.each(['add', 'subtract', 'multiply', 'divide', 'power', 'percent'])('accepts %s with negative decimal operands', (operation) => {
    expect(validateCalculation({ operation, a: '-12.5', b: '2' })).toEqual({ ok: true, request: { operation, a: -12.5, b: 2 } });
  });

  it.each(['0', '-0', '0e-400'])('rejects division by %s', (b) => {
    expect(validateCalculation({ operation: 'divide', a: '0', b })).toMatchObject({ ok: false, issues: { b: { code: 'DIVISION_BY_ZERO' } } });
  });

  it('accepts a small nonzero divisor and leaves calculation to the API', () => {
    expect(validateCalculation({ operation: 'divide', a: '1', b: '5e-324' }).ok).toBe(true);
  });

  it('reports both invalid operands without making an API request', () => {
    expect(validateCalculation({ operation: 'add', a: '', b: 'abc' })).toMatchObject({ ok: false, issues: { a: { code: 'REQUIRED_FIELD' }, b: { code: 'INVALID_NUMBER' } } });
  });

  it('rejects unknown operations', () => {
    expect(validateCalculation({ operation: 'log', a: '4', b: '2' })).toMatchObject({ ok: false, issues: { operation: { code: 'INVALID_OPERATION' } } });
  });

  it.each(['', 'abc', '1e309', '1'.repeat(129)])('omits the unused second operand for square root', (b) => {
    expect(validateCalculation({ operation: 'sqrt', a: '9', b })).toEqual({ ok: true, request: { operation: 'sqrt', a: 9 } });
  });

  it.each(['0', '-0', '5e-324'])('accepts non-negative square root input %s', (a) => {
    expect(validateCalculation({ operation: 'sqrt', a, b: '' }).ok).toBe(true);
  });

  it.each(['-1', '-5e-324'])('rejects a negative square root input %s', (a) => {
    expect(validateCalculation({ operation: 'sqrt', a, b: '' })).toMatchObject({ ok: false, issues: { a: { code: 'INVALID_DOMAIN' } } });
  });

  it.each(['', 'Infinity', '1e-400'])('validates the required square root input %s', (a) => {
    expect(validateCalculation({ operation: 'sqrt', a, b: '' }).ok).toBe(false);
  });

  it.each([['0', '0'], ['-0', '-1'], ['0', '-2'], ['-4', '0.5']])('rejects power outside the real domain: %s ^ %s', (a, b) => {
    expect(validateCalculation({ operation: 'power', a, b })).toMatchObject({ ok: false, issues: { b: { code: 'INVALID_DOMAIN' } } });
  });

  it.each([['2', '-3'], ['4', '0.5'], ['-2', '3'], ['1e308', '0']])('accepts power inputs %s ^ %s', (a, b) => {
    expect(validateCalculation({ operation: 'power', a, b }).ok).toBe(true);
  });

  it.each(['power', 'percent'])('requires a second operand for %s', (operation) => {
    expect(validateCalculation({ operation, a: '2', b: '' })).toMatchObject({ ok: false, issues: { b: { code: 'REQUIRED_FIELD' } } });
  });
});

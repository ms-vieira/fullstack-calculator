import { describe, expect, it } from 'vitest';
import { messageFor } from './types';

describe('error message lookup', () => {
  it.each(['constructor', 'toString', '__proto__'])('treats inherited property %s as an unknown code', (code) => {
    expect(messageFor(code)).toBe('The calculation could not be completed. Please try again.');
  });
  it('keeps known domain messages', () => {
    expect(messageFor('DIVISION_BY_ZERO')).toContain('Cannot divide by zero');
  });
});

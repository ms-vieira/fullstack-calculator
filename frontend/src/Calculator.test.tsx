import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { calculate, CalculationError } from './api/client';
import { Calculator } from './Calculator';

vi.mock('./api/client', async (importOriginal) => ({
  ...await importOriginal<typeof import('./api/client')>(),
  calculate: vi.fn(),
}));

const mockCalculate = vi.mocked(calculate);
let container: HTMLDivElement;
let root: Root | null;

function input(id: string) { return container.querySelector<HTMLInputElement>(`#${id}`)!; }
function submitButton() { return container.querySelector<HTMLButtonElement>('button[type="submit"]')!; }
function result() { return container.querySelector('.result-value')?.textContent; }

async function change(id: string, value: string) {
  const element = container.querySelector<HTMLInputElement | HTMLSelectElement>(`#${id}`)!;
  const prototype = element instanceof HTMLSelectElement ? HTMLSelectElement.prototype : HTMLInputElement.prototype;
  await act(async () => {
    Object.getOwnPropertyDescriptor(prototype, 'value')!.set!.call(element, value);
    element.dispatchEvent(new Event(element instanceof HTMLSelectElement ? 'change' : 'input', { bubbles: true }));
  });
}

async function fill(a = '10', b = '2') { await change('operand-a', a); await change('operand-b', b); }
async function submit() { await act(async () => { container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })); }); }
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej; });
  return { promise, resolve, reject };
}

beforeEach(async () => {
  mockCalculate.mockReset().mockResolvedValue(12);
  container = document.createElement('div');
  document.body.append(container);
  root = createRoot(container);
  await act(async () => { root!.render(<Calculator />); });
});

afterEach(async () => {
  if (root) await act(async () => { root!.unmount(); });
  root = null;
  container.remove();
});

describe('calculator screen', () => {
  it.each([['power', '2', '10', 1024], ['percent', '10', '200', 20]] as const)('submits %s and shows the API result', async (operation, a, b, value) => {
    mockCalculate.mockResolvedValue(value);
    await fill(a, b);
    await change('operation', operation);
    await submit();
    expect(mockCalculate).toHaveBeenCalledWith({ operation, a: Number(a), b: Number(b) }, expect.anything());
    expect(result()).toBe(String(value));
  });

  it('hides the unused operand and sends only a for square root', async () => {
    mockCalculate.mockResolvedValue(3);
    await fill('9', 'invalid stale value');
    await change('operation', 'sqrt');
    expect(container.querySelector('#operand-b')).toBeNull();
    expect(container.querySelector('#operation-help')?.textContent).toContain('No second number');
    await submit();
    expect(mockCalculate).toHaveBeenCalledWith({ operation: 'sqrt', a: 9 }, expect.anything());
    expect(result()).toBe('3');
    await change('operation', 'add');
    expect(input('operand-b').value).toBe('invalid stale value');
    await submit();
    expect(mockCalculate).toHaveBeenCalledTimes(1);
    expect(input('operand-b').getAttribute('aria-invalid')).toBe('true');
  });

  it.each([['sqrt', '-1', '0', 'a'], ['power', '0', '0', 'b'], ['power', '-4', '0.5', 'b']] as const)('blocks invalid %s domain inputs before the API call', async (operation, a, b, field) => {
    await fill(a, b);
    await change('operation', operation);
    await submit();
    expect(mockCalculate).not.toHaveBeenCalled();
    expect(input(`operand-${field}`).getAttribute('aria-invalid')).toBe('true');
    expect(container.querySelector(`#error-${field}`)?.textContent).toContain('real-number domain');
  });

  it('cancels an old binary request when switching to a unary operation', async () => {
    const pending = deferred<number>();
    mockCalculate.mockReturnValueOnce(pending.promise).mockResolvedValueOnce(3);
    await fill('9', '2');
    await submit();
    const signal = mockCalculate.mock.calls[0][1]!.signal!;
    await change('operation', 'sqrt');
    expect(signal.aborted).toBe(true);
    await submit();
    expect(result()).toBe('3');
    await act(async () => { pending.resolve(11); });
    expect(result()).toBe('3');
  });

  it('shows visible labels and validates required inputs before calling the API', async () => {
    expect(container.querySelector('label[for="operand-a"]')?.textContent).toBe('First number');
    await submit();
    expect(mockCalculate).not.toHaveBeenCalled();
    expect(input('operand-a').getAttribute('aria-invalid')).toBe('true');
    expect(container.querySelectorAll('[role="alert"]')).toHaveLength(2);
  });

  it.each(['0', '-0'])('rejects division by %s on the screen', async (b) => {
    await fill('10', b);
    await change('operation', 'divide');
    await submit();
    expect(mockCalculate).not.toHaveBeenCalled();
    expect(container.querySelector('#error-b')?.textContent).toContain('Cannot divide by zero');
  });

  it('sends negative decimals and shows the result returned by the API, including zero', async () => {
    mockCalculate.mockResolvedValue(0);
    await fill('-1.5', '1.5');
    await submit();
    expect(mockCalculate).toHaveBeenCalledWith({ operation: 'add', a: -1.5, b: 1.5 }, expect.objectContaining({ signal: expect.any(AbortSignal) }));
    expect(result()).toBe('0');
  });

  it('accepts intermediate typing but rejects an invalid completed number', async () => {
    await fill('-', '12abc');
    expect(input('operand-a').value).toBe('-');
    await submit();
    expect(mockCalculate).not.toHaveBeenCalled();
    expect(container.querySelectorAll('[role="alert"]')).toHaveLength(2);
  });

  it('does not truncate pasted content', async () => {
    await fill('1'.repeat(129), '2');
    await submit();
    expect(input('operand-a').value).toHaveLength(129);
    expect(mockCalculate).not.toHaveBeenCalled();
    expect(container.querySelector('#error-a')?.textContent).toContain('128');
  });

  it('locks duplicate submissions synchronously', async () => {
    const pending = deferred<number>();
    mockCalculate.mockReturnValue(pending.promise);
    await fill();
    await act(async () => {
      const form = container.querySelector('form')!;
      form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
      form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    });
    expect(mockCalculate).toHaveBeenCalledTimes(1);
    expect(submitButton().disabled).toBe(true);
    await act(async () => { pending.resolve(12); });
    expect(submitButton().disabled).toBe(false);
  });

  it('ignores old success and finally while a new request is pending', async () => {
    const old = deferred<number>();
    const current = deferred<number>();
    mockCalculate.mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise);
    await fill();
    await submit();
    await change('operand-a', '20');
    expect(mockCalculate.mock.calls[0][1]?.signal?.aborted).toBe(true);
    await submit();
    await act(async () => { old.resolve(12); });
    expect(submitButton().disabled).toBe(true);
    expect(result()).toBe('…');
    await act(async () => { current.resolve(22); });
    expect(result()).toBe('22');
  });

  it('ignores an old failure and finally after a new request starts', async () => {
    const old = deferred<number>();
    const current = deferred<number>();
    mockCalculate.mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise);
    await fill();
    await submit();
    await change('operand-b', '3');
    await submit();
    await act(async () => { old.reject(new CalculationError('NETWORK_ERROR')); });
    expect(container.querySelector('[role="alert"]')).toBeNull();
    expect(submitButton().disabled).toBe(true);
    await act(async () => { current.resolve(13); });
    expect(result()).toBe('13');
  });

  it('clears and invalidates a pending response', async () => {
    const pending = deferred<number>();
    mockCalculate.mockReturnValue(pending.promise);
    await fill();
    await submit();
    await act(async () => { container.querySelector<HTMLButtonElement>('button[type="button"]')!.click(); });
    expect(input('operand-a').value).toBe('');
    expect(mockCalculate.mock.calls[0][1]?.signal?.aborted).toBe(true);
    await act(async () => { pending.resolve(12); });
    expect(result()).toBe('—');
    expect(submitButton().disabled).toBe(false);
  });

  it('cancels a pending request when unmounted', async () => {
    const pending = deferred<number>();
    mockCalculate.mockReturnValue(pending.promise);
    await fill();
    await submit();
    const signal = mockCalculate.mock.calls[0][1]?.signal;
    await act(async () => { root!.unmount(); });
    root = null;
    expect(signal?.aborted).toBe(true);
    await act(async () => { pending.resolve(12); });
  });

  it.each(['TIMEOUT', 'NETWORK_ERROR', 'INVALID_RESPONSE'])('keeps inputs and allows retry after %s', async (code) => {
    mockCalculate.mockRejectedValueOnce(new CalculationError(code)).mockResolvedValueOnce(12);
    await fill();
    await submit();
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
    expect(input('operand-a').value).toBe('10');
    expect(submitButton().disabled).toBe(false);
    await submit();
    expect(result()).toBe('12');
    expect(container.querySelector('[role="alert"]')).toBeNull();
  });

  it('associates API field errors with the input', async () => {
    mockCalculate.mockRejectedValueOnce(new CalculationError('NUMBER_OUT_OF_RANGE', 'a'));
    await fill();
    await submit();
    expect(input('operand-a').getAttribute('aria-describedby')).toContain('error-a');
    expect(container.querySelector('#error-a')?.textContent).toContain('outside the supported range');
  });
});

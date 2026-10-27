import { afterEach, describe, expect, it, vi } from 'vitest';
import { calculate } from './client';

const request = { operation: 'add' as const, a: 2, b: 3 };
const jsonResponse = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

afterEach(() => { vi.useRealTimers(); });

describe('calculator HTTP client', () => {
  it('serializes a unary request without a second operand', async () => {
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ result: 3 }));
    await expect(calculate({ operation: 'sqrt', a: 9 }, { fetcher })).resolves.toBe(3);
    expect(JSON.parse(fetcher.mock.calls[0][1]!.body as string)).toEqual({ operation: 'sqrt', a: 9 });
  });

  it('sends the contract and reads a finite result', async () => {
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ result: 5 }));
    expect(await calculate(request, { fetcher })).toBe(5);
    expect(fetcher).toHaveBeenCalledWith('/api/calculations', expect.objectContaining({ method: 'POST', body: JSON.stringify(request), headers: { 'Content-Type': 'application/json', Accept: 'application/json' }, signal: expect.any(AbortSignal) }));
  });

  it('preserves zero and normalizes negative zero', async () => {
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ result: -0 }));
    expect(Object.is(await calculate(request, { fetcher }), 0)).toBe(true);
  });

  it('returns domain error codes and fields', async () => {
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ error: { code: 'DIVISION_BY_ZERO', message: 'Cannot divide by zero.', field: 'b' } }, 400));
    await expect(calculate(request, { fetcher })).rejects.toMatchObject({ code: 'DIVISION_BY_ZERO', field: 'b' });
  });

  it.each([null, {}, { result: null }, { result: '5' }, { result: [5] }])('rejects malformed success payload %j', async (payload) => {
    await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(jsonResponse(payload)) })).rejects.toMatchObject({ code: 'INVALID_RESPONSE' });
  });

  it('rejects a non-finite result supplied by a transport', async () => {
    const response = jsonResponse({});
    vi.spyOn(response, 'json').mockResolvedValue({ result: Infinity });
    await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(response) })).rejects.toMatchObject({ code: 'INVALID_RESPONSE' });
  });

  it('rejects invalid JSON and non-JSON content types', async () => {
    for (const response of [new Response('oops', { headers: { 'Content-Type': 'application/json' } }), new Response('<html>error</html>')]) {
      await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(response) })).rejects.toMatchObject({ code: 'INVALID_RESPONSE' });
    }
  });

  it('validates error envelopes and supplies safe text for unknown errors', async () => {
    await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ error: 'Oops' }, 400)) })).rejects.toMatchObject({ code: 'INVALID_RESPONSE', httpStatus: 400 });
    await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ error: { code: 'UNKNOWN', message: 'private stack trace', field: 'anything' } }, 400)) })).rejects.toMatchObject({ code: 'UNKNOWN', field: undefined, httpStatus: 400, message: 'The calculation could not be completed. Please try again.' });
  });

  it.each([500, 502, 503, 504])('classifies HTTP %s before inspecting the response format', async (status) => {
    for (const contentType of ['text/html', 'application/json']) {
      const response = new Response('upstream failure', { status, headers: { 'Content-Type': contentType, 'Retry-After': '10' } });
      await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(response) })).rejects.toMatchObject({
        code: 'SERVICE_UNAVAILABLE', httpStatus: status, retryAfter: '10', message: expect.stringContaining('temporarily unavailable'),
      });
    }
  });

  it('preserves rate limiting and an HTTP-date Retry-After header', async () => {
    const retryAfter = 'Wed, 28 Oct 2026 12:00:00 GMT';
    const response = new Response('Slow down', { status: 429, headers: { 'Retry-After': retryAfter } });
    await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(response) })).rejects.toMatchObject({ code: 'RATE_LIMITED', httpStatus: 429, retryAfter });
  });

  it('cancels unread error response bodies', async () => {
    const cancel = vi.fn();
    const body = new ReadableStream({ cancel });
    const response = new Response(body, { status: 503 });
    await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(response) })).rejects.toMatchObject({ code: 'SERVICE_UNAVAILABLE' });
    expect(cancel).toHaveBeenCalledOnce();
  });

  it('preserves the HTTP failure when discarding its body fails', async () => {
    const cancel = vi.fn().mockRejectedValue(new Error('Body cleanup failed'));
    const response = new Response(new ReadableStream({ cancel }), { status: 503 });
    await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(response) })).rejects.toMatchObject({ code: 'SERVICE_UNAVAILABLE', httpStatus: 503 });
    expect(cancel).toHaveBeenCalledOnce();
  });

  it('distinguishes an HTTP failure from a malformed successful response', async () => {
    const response = new Response('<html>Not found</html>', { status: 404, headers: { 'Content-Type': 'text/html' } });
    await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(response) })).rejects.toMatchObject({ code: 'HTTP_ERROR', httpStatus: 404 });
    await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ result: null })) })).rejects.toMatchObject({ code: 'INVALID_RESPONSE', httpStatus: 200 });
  });

  it('distinguishes network failures', async () => {
    await expect(calculate(request, { fetcher: vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Network failed')) })).rejects.toMatchObject({ code: 'NETWORK_ERROR' });
  });

  it('times out a transport even if it ignores abort', async () => {
    vi.useFakeTimers();
    const fetcher = vi.fn<typeof fetch>().mockImplementation(() => new Promise(() => {}));
    const promise = calculate(request, { fetcher });
    const assertion = expect(promise).rejects.toMatchObject({ code: 'TIMEOUT' });
    await vi.advanceTimersByTimeAsync(5_000);
    await assertion;
    const options = fetcher.mock.calls[0][1];
    expect((options?.signal as AbortSignal).aborted).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
  });

  it('includes the response-body read in the timeout', async () => {
    vi.useFakeTimers();
    const response = jsonResponse({ result: 5 });
    vi.spyOn(response, 'json').mockImplementation(() => new Promise(() => {}));
    const promise = calculate(request, { fetcher: vi.fn<typeof fetch>().mockResolvedValue(response) });
    const assertion = expect(promise).rejects.toMatchObject({ code: 'TIMEOUT' });
    await vi.advanceTimersByTimeAsync(5_000);
    await assertion;
  });

  it('cancels pending calls and skips calls already cancelled', async () => {
    const controller = new AbortController();
    const fetcher = vi.fn<typeof fetch>().mockImplementation(() => new Promise(() => {}));
    const promise = calculate(request, { fetcher, signal: controller.signal });
    const assertion = expect(promise).rejects.toMatchObject({ code: 'CANCELLED' });
    controller.abort();
    await assertion;
    await expect(calculate(request, { fetcher, signal: controller.signal })).rejects.toMatchObject({ code: 'CANCELLED' });
    expect(fetcher).toHaveBeenCalledTimes(1);
  });
});

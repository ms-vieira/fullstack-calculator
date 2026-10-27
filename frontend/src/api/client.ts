import { messageFor, type CalculationRequest, type Field } from './types';

export class CalculationError extends Error {
  constructor(public readonly code: string, public readonly field?: Field, public readonly httpStatus?: number, public readonly retryAfter?: string) {
    super(messageFor(code));
    this.name = 'CalculationError';
  }
}

interface ClientOptions {
  signal?: AbortSignal;
  timeoutMs?: number;
  fetcher?: typeof fetch;
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function readResult(payload: unknown, response: Response): number {
  if (!isObject(payload) || typeof payload.result !== 'number' || !Number.isFinite(payload.result)) {
    throw responseError(response, 'INVALID_RESPONSE');
  }
  return Object.is(payload.result, -0) ? 0 : payload.result;
}

function responseError(response: Response, code: string, field?: Field): CalculationError {
  return new CalculationError(code, field, response.status, response.headers.get('Retry-After') ?? undefined);
}

function discardBody(response: Response) {
  void response.body?.cancel().catch(() => {});
}

function readError(payload: unknown, response: Response): CalculationError {
  if (!isObject(payload) || !isObject(payload.error) || typeof payload.error.code !== 'string' || typeof payload.error.message !== 'string') {
    return responseError(response, 'INVALID_RESPONSE');
  }
  const { code, field } = payload.error;
  return responseError(response, code, field === 'a' || field === 'b' || field === 'operation' ? field : undefined);
}

export async function calculate(request: CalculationRequest, options: ClientOptions = {}): Promise<number> {
  if (options.signal?.aborted) throw new CalculationError('CANCELLED');
  const controller = new AbortController();
  let rejectCancellation: (error: CalculationError) => void = () => {};
  const cancellation = new Promise<never>((_, reject) => { rejectCancellation = reject; });
  const cancel = (code: string) => {
    rejectCancellation(new CalculationError(code));
    controller.abort();
  };
  const onAbort = () => cancel('CANCELLED');
  options.signal?.addEventListener('abort', onAbort, { once: true });
  const timer = setTimeout(() => cancel('TIMEOUT'), options.timeoutMs ?? 5_000);

  const performRequest = async (): Promise<number> => {
    try {
      const response = await (options.fetcher ?? fetch)('/api/calculations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify(request),
        signal: controller.signal,
      });
      if (response.status === 429 || response.status >= 500) {
        discardBody(response);
        throw responseError(response, response.status === 429 ? 'RATE_LIMITED' : 'SERVICE_UNAVAILABLE');
      }
      if (response.headers.get('Content-Type')?.split(';', 1)[0].trim().toLowerCase() !== 'application/json') {
        discardBody(response);
        throw responseError(response, response.status === 200 ? 'INVALID_RESPONSE' : 'HTTP_ERROR');
      }
      let payload: unknown;
      try {
        payload = await response.json();
      } catch {
        throw responseError(response, 'INVALID_RESPONSE');
      }
      if (response.status !== 200) throw readError(payload, response);
      return readResult(payload, response);
    } catch (error) {
      if (error instanceof CalculationError) throw error;
      throw new CalculationError('NETWORK_ERROR');
    }
  };

  try {
    return await Promise.race([performRequest(), cancellation]);
  } finally {
    clearTimeout(timer);
    options.signal?.removeEventListener('abort', onAbort);
  }
}

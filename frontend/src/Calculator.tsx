import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import { calculate, CalculationError } from './api/client';
import { operations, type Field, type Issue } from './api/types';
import { validateCalculation, type FormValues } from './validation/numbers';

const initialValues: FormValues = { a: '', b: '', operation: 'add' };

export function Calculator() {
  const [values, setValues] = useState<FormValues>(initialValues);
  const [issues, setIssues] = useState<Partial<Record<Field, Issue>>>({});
  const [result, setResult] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const generation = useRef(0);
  const activeRequest = useRef<{ id: number; controller: AbortController } | null>(null);

  const invalidate = useCallback(() => {
    generation.current += 1;
    activeRequest.current?.controller.abort();
    activeRequest.current = null;
  }, []);

  useEffect(() => () => invalidate(), [invalidate]);

  function change(field: Field, value: string) {
    invalidate();
    setValues((current) => ({ ...current, [field]: value }));
    setIssues({});
    setResult(null);
    setError(null);
    setLoading(false);
  }

  function clear() {
    invalidate();
    setValues(initialValues);
    setIssues({});
    setResult(null);
    setError(null);
    setLoading(false);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (activeRequest.current) return;
    const validation = validateCalculation(values);
    setResult(null);
    setError(null);
    if (!validation.ok) {
      setIssues(validation.issues);
      return;
    }
    setIssues({});
    const id = ++generation.current;
    const controller = new AbortController();
    activeRequest.current = { id, controller };
    setLoading(true);
    try {
      const calculated = await calculate(validation.request, { signal: controller.signal });
      if (activeRequest.current?.id === id) setResult(calculated);
    } catch (cause) {
      if (activeRequest.current?.id !== id) return;
      const failure = cause instanceof CalculationError ? cause : new CalculationError('NETWORK_ERROR');
      if (failure.code === 'CANCELLED') return;
      if (failure.field) setIssues({ [failure.field]: { code: failure.code, message: failure.message, field: failure.field } });
      else setError(failure.message);
    } finally {
      if (activeRequest.current?.id === id) {
        activeRequest.current = null;
        setLoading(false);
      }
    }
  }

  const selectedOperation = operations.find(({ value }) => value === values.operation) ?? operations[0];

  return (
    <main className="page">
      <header className="page-header">
        <a className="wordmark" href="./" aria-label="Fullstack Calculator home">calculator<span aria-hidden="true">.</span></a>
        <span className="assessment-label">Technical assessment</span>
      </header>

      <section className="calculator" aria-labelledby="calculator-title">
        <div className="card-heading">
          <span className="eyebrow">A little clarity in every calculation</span>
          <h1 id="calculator-title">Let’s work it out.</h1>
          <p>Seven operations. One clear result.</p>
        </div>

        <form onSubmit={(event) => void submit(event)} noValidate aria-label="Calculator">
          <div className={`operands${selectedOperation.arity === 1 ? ' unary' : ''}`}>
            <div className="field">
              <label htmlFor="operand-a">First number</label>
              <input id="operand-a" name="a" type="text" inputMode="decimal" autoComplete="off" spellCheck={false}
                placeholder="e.g. -12.5" value={values.a} onChange={(event) => change('a', event.target.value)}
                aria-invalid={Boolean(issues.a)} aria-describedby={issues.a ? 'error-a number-help' : 'number-help'} />
              {issues.a && <p className="field-error" id="error-a" role="alert">{issues.a.message}</p>}
            </div>
            {selectedOperation.arity === 2 && <span className="operation-symbol" aria-hidden="true">{selectedOperation.symbol}</span>}
            {selectedOperation.arity === 2 && <div className="field">
              <label htmlFor="operand-b">Second number</label>
              <input id="operand-b" name="b" type="text" inputMode="decimal" autoComplete="off" spellCheck={false}
                placeholder="e.g. 2" value={values.b} onChange={(event) => change('b', event.target.value)}
                aria-invalid={Boolean(issues.b)} aria-describedby={issues.b ? 'error-b number-help' : 'number-help'} />
              {issues.b && <p className="field-error" id="error-b" role="alert">{issues.b.message}</p>}
            </div>}
          </div>
          <p className="input-help" id="number-help">Use a decimal point. Negative numbers and notation like 1e3 are welcome.</p>

          <div className="field operation-field">
            <label htmlFor="operation">Operation</label>
            <select id="operation" value={values.operation} onChange={(event) => change('operation', event.target.value)}
              aria-invalid={Boolean(issues.operation)} aria-describedby={issues.operation ? 'error-operation operation-help' : 'operation-help'}>
              {operations.map(({ value, label, symbol }) => <option key={value} value={value}>{label} ({symbol})</option>)}
            </select>
            {issues.operation && <p className="field-error" id="error-operation" role="alert">{issues.operation.message}</p>}
            <p className="input-help" id="operation-help">{selectedOperation.hint}</p>
          </div>

          <div className="actions">
            <button className="button-primary" type="submit" disabled={loading}>
              {loading ? <><span className="spinner" aria-hidden="true" />Calculating…</> : <>Calculate <span aria-hidden="true">↗</span></>}
            </button>
            <button className="button-secondary" type="button" onClick={clear}>Clear</button>
          </div>
        </form>

        <div className={`result-panel${result !== null ? ' has-result' : ''}`} role="status" aria-live="polite" aria-atomic="true" aria-busy={loading}>
          <span className="result-label">{loading ? 'Working on it' : 'Your result'}</span>
          <output className="result-value">{loading ? '…' : result !== null ? String(Object.is(result, -0) ? 0 : result) : '—'}</output>
          <span className="result-caption">{loading ? 'Waiting for the calculator.' : result !== null ? 'All worked out.' : 'Enter your numbers to get started.'}</span>
        </div>
        {error && <p className="request-error" role="alert">{error}</p>}
        <p className="precision-note">Uses approximate floating-point arithmetic. Very large integers and some decimals may lose precision.</p>
      </section>
      <footer className="page-footer"><span>Built with React, TypeScript & Go</span><span>Simple by design.</span></footer>
    </main>
  );
}

import { chromium } from 'playwright-core';
import AxeBuilder from '@axe-core/playwright';
import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync } from 'node:fs';

const baseURL = process.env.CALCULATOR_URL ?? 'http://127.0.0.1:4173';
const validCases = [
  ['add', '10', '5', 15],
  ['subtract', '5', '10', -5],
  ['multiply', '-2.5', '4', -10],
  ['divide', '10', '4', 2.5],
  ['divide', '0', '5', 0],
  ['power', '2', '10', 1024],
  ['power', '2', '-3', 0.125],
  ['sqrt', '81', null, 9],
  ['sqrt', '0', null, 0],
  ['percent', '10', '200', 20],
  ['percent', '-10', '200', -20],
  ['percent', '1e308', '2', 2e306],
  ['percent', '1e-150', '2.4703282292062328e-172', 5e-324],
  ['percent', '4.172657595118774e272', '4.308268996155542e37', Number.MAX_VALUE],
];
const invalidCases = [
  ['divide', '5', '0', 'Cannot divide by zero'],
  ['multiply', '1e308', '10', 'too large'],
  ['multiply', '1e-300', '1e-300', 'too small'],
  ['add', '1e-400', '1', 'too small'],
  ['add', '12abc', '1', 'Use a number'],
  ['sqrt', '-1', null, 'real-number domain'],
  ['power', '0', '0', 'real-number domain'],
  ['power', '0', '-1', 'real-number domain'],
  ['power', '-4', '0.5', 'real-number domain'],
  ['power', '1e308', '2', 'too large'],
  ['power', '2', '-1075', 'too small'],
  ['percent', '5e-324', '1', 'too small'],
];
const apiCases = [
  [{ operation: 'sqrt', a: 9 }, 200, 3],
  [{ operation: 'sqrt', a: 9, b: 0 }, 400, 'INVALID_REQUEST'],
  [{ operation: 'sqrt', a: -1 }, 400, 'INVALID_DOMAIN'],
  [{ operation: 'sqrt', a: null }, 400, 'INVALID_NUMBER'],
  [{ operation: 'power', a: 0, b: 0 }, 400, 'INVALID_DOMAIN'],
  [{ operation: 'power', a: -4, b: 0.5 }, 400, 'INVALID_DOMAIN'],
  [{ operation: 'percent', a: 5e-324, b: 1 }, 400, 'NUMERIC_UNDERFLOW'],
  [{ operation: 'percent', a: 100, b: 1e308 }, 200, 1e308],
  [{ operation: 'percent', a: 1e-150, b: 2.4703282292062328e-172 }, 200, 5e-324],
  [{ operation: 'percent', a: 4.172657595118774e272, b: 4.308268996155542e37 }, 200, Number.MAX_VALUE],
  [{ operation: 'percent', a: 10 }, 400, 'REQUIRED_FIELD'],
  [{ operation: 'log', a: 1, b: 2 }, 400, 'INVALID_OPERATION'],
];

async function waitForServer() {
  for (let attempt = 0; attempt < 60; attempt++) {
    try {
      if ((await fetch(baseURL)).ok) return;
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw new Error('Calculator server did not become ready');
}

async function setInputs(page, operation, a, b) {
  await page.getByLabel('Operation', { exact: true }).selectOption(operation);
  await page.getByLabel('First number', { exact: true }).fill(a);
  if (b !== null) await page.getByLabel('Second number', { exact: true }).fill(b);
  else assert.equal(await page.getByLabel('Second number', { exact: true }).count(), 0);
}

async function checkAccessibility(page) {
  const analysis = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze();
  assert.equal(analysis.violations.length, 0, JSON.stringify(analysis.violations));
  return analysis.violations;
}

async function main() {
  mkdirSync('e2e-results', { recursive: true });
  await waitForServer();
  for (const [body, status, expected] of apiCases) {
    const response = await fetch(new URL('/api/calculations', baseURL), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    assert.equal(response.status, status, JSON.stringify(body));
    assert.match(response.headers.get('Content-Type'), /application\/json/);
    const payload = await response.json();
    assert.equal(status === 200 ? payload.result : payload.error.code, expected);
  }

  const browser = await chromium.launch({ headless: true });
  try {
    const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
    const page = await context.newPage();
    const issues = [];
    page.on('pageerror', (error) => issues.push(error.message));
    await page.goto(baseURL);
    for (const [operation, a, b, expected] of validCases) {
      await setInputs(page, operation, a, b);
      await page.getByRole('button', { name: 'Calculate' }).click();
      await page.waitForFunction((value) => document.querySelector('output')?.textContent === value, String(expected));
    }
    for (const [operation, a, b, message] of invalidCases) {
      await setInputs(page, operation, a, b);
      await page.getByRole('button', { name: 'Calculate' }).click();
      await page.getByRole('alert').filter({ hasText: message }).waitFor();
    }
    const unavailableCases = [[503, 'temporarily unavailable'], [429, 'Too many calculations']];
    for (const [status, message] of unavailableCases) {
      await page.route('**/api/calculations', (route) => route.fulfill({ status, contentType: 'text/html', headers: { 'Retry-After': '10' }, body: 'Upstream failure' }));
      try {
        await setInputs(page, 'add', '2', '3');
        await page.getByRole('button', { name: 'Calculate' }).click();
        await page.getByRole('alert').filter({ hasText: message }).waitFor();
        assert.equal(await page.getByLabel('First number', { exact: true }).inputValue(), '2');
        assert.equal(await page.getByLabel('Second number', { exact: true }).inputValue(), '3');
      } finally {
        await page.unroute('**/api/calculations');
      }
      await page.getByRole('button', { name: 'Calculate' }).click();
      await page.waitForFunction(() => document.querySelector('output')?.textContent === '5');
    }
    await page.getByRole('button', { name: 'Clear', exact: true }).click();
    const desktopViolations = await checkAccessibility(page);
    await page.screenshot({ path: 'e2e-results/desktop.png', fullPage: true });
    const layout = [];
    const operationValues = await page.locator('#operation option').evaluateAll((options) => options.map((option) => option.value));
    assert.equal(operationValues.length, 7);
    for (const operation of operationValues) {
      await page.getByLabel('Operation', { exact: true }).selectOption(operation);
      for (const width of [320, 768, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        const dimensions = await page.evaluate(() => ({ scroll: document.documentElement.scrollWidth, viewport: innerWidth }));
        layout.push({ operation, width, ...dimensions });
        assert(dimensions.scroll <= width, JSON.stringify(layout.at(-1)));
      }
    }
    await page.getByLabel('Operation', { exact: true }).selectOption('sqrt');
    const unaryDesktopViolations = await checkAccessibility(page);
    await page.screenshot({ path: 'e2e-results/sqrt-desktop.png', fullPage: true });
    await page.setViewportSize({ width: 320, height: 900 });
    const unaryMobileViolations = await checkAccessibility(page);
    await page.screenshot({ path: 'e2e-results/sqrt-mobile.png', fullPage: true });
    await page.getByLabel('Operation', { exact: true }).selectOption('percent');
    const mobileViolations = await checkAccessibility(page);
    await page.screenshot({ path: 'e2e-results/mobile.png', fullPage: true });
    assert.equal(issues.length, 0);
    const summary = {
      integrationCases: validCases.length + invalidCases.length + apiCases.length + unavailableCases.length,
      uiSuccessCases: validCases.length,
      uiErrorCases: invalidCases.length,
      apiCases: apiCases.length,
      unavailableCases: unavailableCases.length,
      pageErrors: issues,
      layout,
      desktopViolations,
      mobileViolations,
      unaryDesktopViolations,
      unaryMobileViolations,
    };
    writeFileSync('e2e-results/summary.json', JSON.stringify(summary, null, 2));
    console.log(JSON.stringify(summary, null, 2));
  } finally {
    await browser.close();
  }
}

main().catch((error) => { console.error(error); process.exitCode = 1; });

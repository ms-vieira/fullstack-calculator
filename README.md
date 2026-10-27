# Fullstack Calculator

A full-stack calculator built with React, TypeScript and Go. The browser validates input and calls a stateless REST API for seven arithmetic operations. The calculation core is pure; repeated requests have no business side effects.

## Run with Docker

Install Docker with Compose support (Docker Desktop on macOS/Windows) and start the Docker engine. Go and Node.js are only needed for development outside Docker.

From the repository root:

```sh
docker compose up --build
```

Open http://127.0.0.1:8080. The same Go process serves the compiled React interface and `/api/calculations`. Stop with Ctrl+C, then remove the container with:

```sh
docker compose down
```

Alternatively, build and run directly:

```sh
docker build -t fullstack-calculator:local .
docker run --rm --name calculator -p 127.0.0.1:8080:8080 fullstack-calculator:local
```

Stop it with Ctrl+C or `docker stop calculator` from another terminal. If port 8080 is occupied, use `-p 127.0.0.1:8081:8080` and open http://127.0.0.1:8081. This changes the host port; the container still listens on 8080. Compose's host port can be changed in `compose.yaml`.

The Dockerfile has three stages: Node 24 builds the frontend from the lockfile, Go 1.27 compiles a static server binary, and a minimal final image receives only that binary and the frontend build. The container runs as UID/GID 65532. Compose also uses a read-only filesystem. No database, bind mount or separate frontend process is required.

`PORT` defaults to 8080. `STATIC_DIR` selects the frontend build directory and is set to `/app/static` in the image. Without `STATIC_DIR`, the server runs in API-only mode for local development. An invalid static directory fails startup. Static files support GET/HEAD; missing files and directories return 404, while `/api` routes retain JSON errors. The UI has no client-side routes requiring an HTML fallback.

The `.dockerignore` allowlist includes only build inputs, excluding local documentation, Git history, dependencies, generated reports and environment files outside those source inputs. If Docker reports that it cannot connect to the daemon, start Docker Desktop before retrying.

## Local development requirements

- Go 1.27.3 or newer (the CI and Docker build both use 1.27.3).
- Node.js 22.12 or newer and npm.
- Two terminals for local development. No database or external service is required.

## Run locally

Terminal 1, from the repository root:

```sh
cd backend
go run ./cmd/server
```

Terminal 2:

```sh
cd frontend
npm ci
npm run dev
```

Open the local URL printed by Vite (normally http://127.0.0.1:5173). The backend listens on port 8080 by default (override with PORT). If changing PORT, update the proxy target in frontend/vite.config.ts as well. Vite forwards /api requests to http://127.0.0.1:8080.

### Preview the frontend build

Keep the backend running:

```sh
cd frontend
npm run build
npm run preview
```

Open the preview URL printed by Vite (normally http://127.0.0.1:4173). Both dev and preview use the API proxy. The build output is static: when deploying it through a different server, configure that server to proxy /api to Go on the same origin. A frontend API URL alone does not authorize cross-origin requests; direct cross-origin hosting requires an explicit CORS policy and preflight handling.

## API

```sh
curl -i http://127.0.0.1:8080/api/calculations \
  -H 'Content-Type: application/json' \
  --data '{"operation":"divide","a":10,"b":2}'
```

HTTP 200:

```json
{"result":5}
```

```sh
curl -i http://127.0.0.1:8080/api/calculations \
  -H 'Content-Type: application/json' \
  --data '{"operation":"divide","a":10,"b":0}'
```

HTTP 400:

```json
{"error":{"code":"DIVISION_BY_ZERO","message":"Cannot divide by zero.","field":"b"}}
```

| Operation | Operands | Meaning | Example result |
| --- | --- | --- | --- |
| `add` | `a`, `b` | a + b | 10 + 5 = 15 |
| `subtract` | `a`, `b` | a − b | 5 − 10 = −5 |
| `multiply` | `a`, `b` | a × b | −2.5 × 4 = −10 |
| `divide` | `a`, `b` | a ÷ b | 10 ÷ 4 = 2.5 |
| `power` | `a`, `b` | a raised to b | 2^10 = 1024 |
| `sqrt` | `a` only | Square root of a | √81 = 9 |
| `percent` | `a`, `b` | a% of b = a × b / 100 | 10% of 200 = 20 |

Every request requires `operation` and `a`. Binary operations also require `b`; `sqrt` requires `b` to be omitted, including when its value would be zero or null. The UI hides the unused operand and does not submit it. Numbers must be JSON numbers, not strings. Unknown or duplicate fields, malformed JSON and additional JSON documents are rejected. Maximum request body: 4 KiB. Maximum numeric token/input length: 128 characters.

Examples of the advanced operations:

```sh
curl http://127.0.0.1:8080/api/calculations \
  -H 'Content-Type: application/json' \
  --data '{"operation":"power","a":2,"b":10}'

curl http://127.0.0.1:8080/api/calculations \
  -H 'Content-Type: application/json' \
  --data '{"operation":"sqrt","a":81}'

curl http://127.0.0.1:8080/api/calculations \
  -H 'Content-Type: application/json' \
  --data '{"operation":"percent","a":10,"b":200}'
```

These return `{"result":1024}`, `{"result":9}`, and `{"result":20}`, respectively.

Expected failures use a stable error code with an optional field. Invalid input/domain errors use 400; unknown routes 404; unsupported methods 405 (Allow: POST); oversized bodies 413; incompatible Content-Type 415.

## Numeric policy

JavaScript number and Go float64 use approximate binary floating-point arithmetic. Negative numbers, decimals, zero and representable subnormals are supported. Decimal input uses a dot; scientific notation is supported. Inputs must match the entire documented numeric grammar.

Accepted syntax: `^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`.

- Reject division by positive or negative zero.
- Reject non-finite inputs and overflowed results.
- Detect nonzero textual input that rounds to zero during parsing; ParseFloat can return zero with no error for 1e-400.
- Reject nonzero multiplication, division, power, or percentage results that round to zero.
- Square root accepts non-negative inputs. Negative inputs return `INVALID_DOMAIN` on `a`.
- Powers accept negative and fractional exponents for positive bases. A negative base requires an integer exponent. By policy, zero requires a positive exponent: `0^0` and zero to a negative exponent return `INVALID_DOMAIN` on `b`.
- Percentages accept negative values, zero, and percentages above 100. The parsed binary64 operands are multiplied and divided by 100 using exact rational intermediates, then rounded once to float64. This avoids intermediate overflow/underflow and double rounding at the numeric boundaries. Inputs and returned results still follow the floating-point policy; decimal text is not preserved as arbitrary-precision decimal arithmetic.
- Allow legitimate zero results and normalize negative zero.
- Preserve the returned floating-point value in the display rather than silently rounding to a fixed number of decimal places.
- Very large integers and long decimals are approximate; integers above 2^53 − 1 are not guaranteed exact. For example, 0.1 + 0.2 can display as 0.30000000000000004. This is not an arbitrary-precision financial calculator.

Validation occurs independently at the UI boundary, HTTP boundary and pure calculation core. Shared numeric fixtures are in testdata/numeric-cases.json.

## Concurrency and resilience

The API keeps no calculation/session state. It can evaluate duplicate requests independently, with no accumulated result or persistent business effect. This is safe repetition for this endpoint; it does not promise exactly-once execution or make POST generically idempotent.

The UI prevents duplicate pending submissions synchronously, cancels invalidated requests and uses request generations to ignore old responses, old errors and old cleanup. Editing, clearing or unmounting invalidates pending work. A five-second client timeout includes consuming the response body; failed requests preserve inputs for manual retry.

The API adapter classifies HTTP 5xx responses as temporary service failures and HTTP 429 as rate limiting, even when a proxy returns HTML. Errors preserve `httpStatus` and the raw `Retry-After` header when available. Unread error bodies are cancelled. These metadata support diagnostics; retries remain manual.

The server configures header/read/write/idle timeouts and graceful shutdown. These are transport controls, not a deadline that preempts arbitrary computation.

## Browser verification

With the API and preview running, install Chromium once and execute the real-browser checks:

```sh
node frontend/node_modules/playwright-core/cli.js install chromium
node frontend/scripts/e2e.mjs
```

The browser CI job builds the Docker image, runs it on host port 4173 with a read-only filesystem, executes the same browser and accessibility checks, and publishes screenshots as an artifact. It also verifies that the image runs as a non-root user and that SIGTERM shutdown exits successfully.

## CI and image delivery

GitHub Actions runs backend formatting, static analysis and race tests, frontend lint/type checking, tests and builds, and browser integration against the Docker image. The image is built for verification inside CI. Publishing it to a registry and deploying to an external environment are not configured.

## Verify

```sh
cd backend
go vet ./...
go test -race ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

```sh
cd frontend
npm run typecheck
npm run lint
npm test
npm run build
```

Frontend coverage is included in the locked development dependencies:

```sh
cd frontend
npm run coverage
```

Coverage is measured with V8. The frontend excludes only the main.tsx bootstrap, test files and test setup. Backend coverage includes the server bootstrap. Component tests use React act/createRoot and jsdom.

Measured statement coverage on October 27, 2026:

| Scope | Coverage |
| --- | ---: |
| Backend calculator | 97.6% |
| Backend HTTP API, including static serving | 94.8% |
| Backend overall, including server bootstrap | 82.8% |
| Frontend, excluding bootstrap and test files | 99.38% |

Generated HTML coverage reports are available locally at `backend/coverage.html` and `frontend/coverage/index.html`. CI is configured in .github/workflows/ci.yml; completed runs and browser artifacts are available in [GitHub Actions](https://github.com/ms-vieira/fullstack-calculator/actions/workflows/ci.yml).

Each CI run also publishes `backend-coverage` (the Go coverage profile and standalone HTML report) and `frontend-coverage` (the HTML report, LCOV data, and JSON summary). Download and extract the corresponding artifact from the workflow run to inspect detailed coverage.

## Design and scope

- React + TypeScript: local form state and typed API adapter.
- Go standard library: strict flat JSON handling and a pure calculator using the Strategy pattern.
- CSS tokens for a purple, mint and neutral palette, with accessible contrast, keyboard navigation and responsive layout.
- No persistence is needed for a stateless calculation.
- The four required operations plus exponentiation, square root, and percentage are implemented. Docker packages the frontend and API together for straightforward execution.

`Calculator` selects stateless strategies from a private registry built once per HTTP handler. `Addition`, `Subtraction`, `Multiplication`, `Division`, `Exponentiation`, and `Percentage` implement `BinaryOperation`. `SquareRoot` implements `UnaryOperation`. The registry determines the operand count used by both the core and HTTP validation. `Calculate(operation, operands...)` enforces the exact count and shared finite-input/result checks; each strategy owns its domain rules. Parsing lives separately in `number.go`.

## AI assistance

AI was explicitly allowed by the assessment. AI assistance is described in [AI_USAGE.md](AI_USAGE.md). Matheus must review and be able to explain the code; automated checks do not substitute for that review.

## Delivery status

The automated frontend and backend checks, real HTTP/proxy integration, lifecycle checks and dependency audit pass. Chromium browser integration, responsive widths and automated accessibility scans also pass in Linux CI; screenshots were reviewed. Manual keyboard and 200% zoom checks were not separately homologated. This is not a claim of exhaustive correctness.

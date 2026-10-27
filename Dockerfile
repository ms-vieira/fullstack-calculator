FROM node:24.22.1-bookworm-slim AS frontend
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
COPY testdata/numeric-cases.json /build/testdata/numeric-cases.json
RUN npm run build

FROM golang:1.27.3-bookworm AS backend
WORKDIR /build/backend
COPY backend/go.mod ./
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /build/server ./cmd/server

FROM scratch
WORKDIR /app
COPY --from=backend /build/server /app/server
COPY --from=frontend /build/frontend/dist /app/static
ENV PORT=8080 STATIC_DIR=/app/static
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/app/server"]

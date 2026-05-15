# WASAText

WASAText is a Sapienza WASA project: a small messaging app with a Go backend, SQLite storage, and a Vue frontend.

## Structure

- `cmd/webapi/`: server entry point, CORS, static frontend serving.
- `service/api/`: HTTP handlers and JSON responses.
- `service/database/`: in-memory and SQLite data layer.
- `doc/api.yaml`: OpenAPI specification.
- `webui/`: Vue frontend source built with Vite.
- `Dockerfile.backend`: backend image.
- `Dockerfile.frontend`: frontend nginx image.
- `docker-compose.yml`: two-container local deployment.

## Run Locally

```bash
go run ./cmd/webapi
```

Then open:

```text
http://localhost:3000
```

## Frontend

```bash
cd webui
npm install
npm run build
```

## Tests

```bash
go test ./...
```

## Docker

```bash
docker compose up --build
```


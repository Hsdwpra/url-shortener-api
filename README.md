# url-shortener-api

URL shortener with click analytics. REST API written in Go (chi + SQLite), fully containerized.

## Endpoints

| Method | Path                  | Description                     |
|--------|-----------------------|---------------------------------|
| POST   | `/api/shorten`        | Create a short URL              |
| GET    | `/{code}`             | Redirect to the long URL (302)  |
| GET    | `/api/{code}/stats`   | Click count + 50 recent clicks  |

## Run locally (Go)

```bash
mkdir -p data
go run .
```

## Run with Docker

```bash
docker compose up --build -d
```

## Example

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://polinema.ac.id"}'

# {"code":"AbC123","short_url":"http://localhost:8080/AbC123",...}

curl http://localhost:8080/AbC123            # redirects
curl http://localhost:8080/api/AbC123/stats  # click analytics
```

## Stack

- Go 1.23, chi router
- SQLite (WAL mode) — `urls` + `clicks` tables with indexed lookups
- Multi-stage Docker build (~15 MB final image)
- Non-blocking click recording (goroutine) so redirects stay fast

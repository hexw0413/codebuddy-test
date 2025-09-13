# CS2 Auto Trading Monorepo (Go + Python + React)

This scaffold provides:
- Go API: Steam OpenID login, CORS, sample market endpoint, NATS subscribe
- Python worker: APScheduler + NATS publisher for mock orders
- React (Vite): dashboard with a simple chart and Steam login button

## Quick start (no Docker)

1) Copy env file
```bash
cp .env.example .env
```

2) Start NATS (optional external)
- If you have Docker locally, run: `docker run --rm -p 4222:4222 -p 8222:8222 nats:2.10-alpine -js -m 8222`
- Or change `NATS_URL` to a reachable server

3) Run Go API
```bash
cd services/api
export $(grep -v '^#' ../../.env | xargs -d '\n')
API_PORT=8080 go run ./cmd/api
```

4) Run Web (dev)
```bash
cd services/web
npm install
npm run dev
```

5) Run Worker
```bash
cd services/worker
pip install -r requirements.txt
python -m app.main
```

Open:
- Frontend: http://localhost:5173
- API: http://localhost:8080
- NATS monitoring (if running local Docker): http://localhost:8222

## Notes
- Steam login uses OpenID redirect to validate and sets a session cookie.
- Connectors for BUFF/悠悠有品 are stubs; replace with real implementations and add secrets.
- For production, prefer Docker Compose and add a DB (e.g., Postgres) for persistence.
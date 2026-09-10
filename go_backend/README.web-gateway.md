# SpotiFLAC Web Gateway

The Web gateway is a small HTTP host around the same Go extension/provider runtime used by the mobile application. It does not duplicate provider search or stream-resolution logic.

## Endpoints

- `GET /health` — gateway status and loaded metadata-provider count
- `GET /search?q=<query>&limit=<1..50>` — searches enabled metadata providers
- `POST /resolve` — resolves a provider track to a short-lived HTTP(S) stream

When `SPOTIFLAC_GATEWAY_TOKEN` is set, all endpoints require `Authorization: Bearer <token>`.

## Required configuration

`SPOTIFLAC_EXTENSION_STORAGE_KEY` is required. It must decode to exactly 32 bytes using standard Base64. Generate a new key once and keep it stable for that gateway data volume:

```bash
openssl rand -base64 32
```

Environment variables:

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `SPOTIFLAC_EXTENSION_STORAGE_KEY` | yes | — | Encrypt extension settings and credentials |
| `SPOTIFLAC_GATEWAY_TOKEN` | strongly recommended on public hosts | empty | Protect server-to-server gateway requests |
| `SPOTIFLAC_GATEWAY_ADDR` | no | `127.0.0.1:8787` | HTTP listen address |
| `SPOTIFLAC_EXTENSIONS_DIR` | no | `./extensions` | Installed `.spx` extension packages |
| `SPOTIFLAC_DATA_DIR` | no | `./data` | Persistent extension state |

A healthy gateway with `providerCount: 0` is reachable but cannot search until at least one enabled metadata-provider extension is installed.

## Run directly

From `go_backend/`:

```bash
go build -o web-gateway ./cmd/web-gateway
SPOTIFLAC_EXTENSION_STORAGE_KEY='<base64-key>' \
SPOTIFLAC_GATEWAY_TOKEN='<long-random-token>' \
SPOTIFLAC_GATEWAY_ADDR=':8787' \
./web-gateway
```

## Docker

Build from `go_backend/`:

```bash
docker build -f Dockerfile.web-gateway -t spotiflac-web-gateway .
```

Run with persistent provider state and an extensions directory:

```bash
docker run --rm -p 8787:8787 \
  -e SPOTIFLAC_EXTENSION_STORAGE_KEY='<base64-key>' \
  -e SPOTIFLAC_GATEWAY_TOKEN='<long-random-token>' \
  -v "$PWD/extensions:/app/extensions" \
  -v spotiflac-gateway-data:/app/data \
  spotiflac-web-gateway
```

Test readiness from the host with the same bearer token:

```bash
curl -H 'Authorization: Bearer <long-random-token>' http://127.0.0.1:8787/health
```

## Vercel Web connection

The long-running Go gateway is separate from the Vercel Next.js deployment. Deploy this container to a container/VPS host with HTTPS, then configure the Vercel project (Root Directory `webapp`) with:

```text
SPOTIFLAC_WEB_GATEWAY_URL=https://your-gateway.example
SPOTIFLAC_WEB_GATEWAY_TOKEN=<same token as SPOTIFLAC_GATEWAY_TOKEN>
```

Do not expose either gateway token as a `NEXT_PUBLIC_*` value. Next.js calls the gateway only from server routes, so provider credentials and stream headers stay server-side.

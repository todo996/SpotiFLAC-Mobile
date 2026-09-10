# SpotiFLAC Web

Independent Next.js/PWA client for SpotiFLAC. It lives under `webapp/` so the existing Flutter Android/iOS implementation remains intact.

## Current capabilities

- Responsive desktop, laptop, tablet and mobile UI
- Installable PWA shell with offline fallback
- Provider-backed search through server routes
- Full-track stream resolution through the shared Go extension runtime
- Same-origin audio proxy with Range/If-Range forwarding for seeking
- Server-side provider headers so auth material is not exposed to browser JavaScript
- HTML5 player with play/pause, seek, previous/next, shuffle and repeat
- Media Session integration for OS/browser media controls
- Persistent queue, volume, shuffle and repeat settings across PWA/browser reloads
- Add to queue, remove from queue, clear queue and play-all flows
- Secure server-side download proxy
- Preview URL fallback when a full stream temporarily cannot be resolved
- End-to-end health endpoint that reports gateway reachability, provider count and capabilities
- Provider readiness panel showing each source as ready, disabled, missing configuration, waiting for verification, or failed

## Development

```bash
npm install
npm run typecheck
npm run build
npm run dev
```

Open `http://localhost:3000`. Search/stream/download features require the included Go gateway plus at least one enabled metadata-provider extension.

## Included provider gateway

The repository includes `go_backend/cmd/web-gateway`. It reuses the same extension manager, metadata search, provider priority/de-duplication and stream resolver used by the mobile backend instead of implementing a second provider stack.

See `go_backend/README.web-gateway.md` for direct and Docker deployment. The gateway exposes:

- `GET /health`
- `GET /providers`
- `GET /search?q=...`
- `POST /resolve`

`GET /providers` is intentionally read-only and sanitized. It reports configuration completeness and auth readiness without returning setting values, provider tokens, OAuth state or other credentials. Provider credentials remain encrypted on the gateway host.

## Vercel deployment

Create a Vercel project from this repository and set **Root Directory** to `webapp`. Next.js settings can remain auto-detected.

The Go gateway is a long-running service and should be deployed separately on a container/VPS host with HTTPS. Configure these server-only environment variables in Vercel:

- `SPOTIFLAC_WEB_GATEWAY_URL` — HTTPS base URL of the included Go gateway.
- `SPOTIFLAC_WEB_GATEWAY_TOKEN` — same bearer token configured as `SPOTIFLAC_GATEWAY_TOKEN` on the gateway.
- `SPOTIFLAC_WEB_FORCE_STREAM_PROXY=1` — optional; forces playback through `/api/stream`. Proxying is already selected automatically when provider request headers or plain HTTP require it.

Never expose `SPOTIFLAC_WEB_GATEWAY_TOKEN` through a `NEXT_PUBLIC_*` variable.

## Gateway contract

`GET /search?q=<query>` returns `tracks`. Each track includes an extension-backed `provider_id`, which is later sent to the resolver with the provider-native track ID.

`POST /resolve` request:

```json
{
  "providerId": "provider-id",
  "trackId": "provider-track-id",
  "quality": "optional-quality"
}
```

Typical response:

```json
{
  "success": true,
  "url": "https://cdn.example/audio.flac",
  "headers": {
    "Authorization": "Bearer short-lived-provider-token"
  },
  "content_type": "audio/flac",
  "expires_at_ms": 1770000000000,
  "provider": "provider-id",
  "quality": "FLAC"
}
```

`headers` remain server-side. `/api/stream` and `/api/download` consume them and never serialize them to browser JavaScript.

## Web API routes

- `GET /api/health` — PWA readiness plus live gateway/provider readiness
- `GET /api/providers` — sanitized provider inventory used by the readiness panel
- `GET /api/search?q=...` — validated search adapter
- `POST /api/resolve` — validated stream-resolution adapter
- `GET|HEAD /api/stream?...` — same-origin stream proxy with media Range support
- `GET /api/download?...` — same-origin attachment/download proxy

## Provider configuration and authentication

The browser is not used as a credential store. Required API keys, cookies or provider login state live in the gateway's encrypted persistent data directory. The Web UI reports which source still needs configuration or verification so an operator can complete that setup on the gateway without exposing credentials to every Web user.

Qobuz Web and direct TIDAL streams are supported by the shared compatibility resolver. Amazon legacy streams can require decryption/container conversion, so the provider panel reports that limitation instead of advertising direct playback that may fail.

## Architecture rule

The browser UI must not execute the native Flutter/gomobile filesystem or download stack directly. Provider work that requires extension execution, secret request headers, local filesystem access or other native capabilities runs in the included Go Web gateway. This keeps the browser/Vercel deployment separated from native-only capabilities while preserving one shared provider runtime.

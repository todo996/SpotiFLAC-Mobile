# SpotiFLAC Web

Independent Next.js/PWA client for SpotiFLAC. It lives under `webapp/` so the existing Flutter Android/iOS implementation remains intact.

## Current capabilities

- Responsive desktop, laptop, tablet and mobile UI
- Installable PWA shell with offline fallback
- Provider-backed search through server routes
- Full-track stream resolution through the Web gateway
- Same-origin audio proxy with Range/If-Range forwarding for seeking
- Server-side provider headers so auth material is not exposed to browser JavaScript
- HTML5 player with play/pause, seek, previous/next, shuffle and repeat
- Media Session integration for OS/browser media controls
- Persistent queue, volume, shuffle and repeat settings across PWA/browser reloads
- Add to queue, remove from queue, clear queue and play-all flows
- Secure server-side download proxy that resolves the provider source without exposing provider credentials
- Preview URL fallback when a full stream temporarily cannot be resolved
- Health endpoint that reports Web-gateway readiness and supported capabilities

## Development

```bash
npm install
npm run typecheck
npm run build
npm run dev
```

Open `http://localhost:3000`. Search/stream/download features require a configured provider gateway; the UI and PWA shell can still load without it.

## Vercel deployment

Create a Vercel project from this repository and set **Root Directory** to `webapp`. Next.js settings can remain auto-detected.

Configure these server-side environment variables in Vercel:

- `SPOTIFLAC_WEB_GATEWAY_URL` — base URL of the gateway. The gateway must expose `GET /search?q=...` and `POST /resolve`.
- `SPOTIFLAC_WEB_GATEWAY_TOKEN` — optional bearer token used only between the Next.js server routes and the gateway.
- `SPOTIFLAC_WEB_FORCE_STREAM_PROXY=1` — optional. Forces all playback through `/api/stream`. Without it, the proxy is selected automatically when a provider requires custom request headers or returns plain HTTP.

Never expose `SPOTIFLAC_WEB_GATEWAY_TOKEN` through a `NEXT_PUBLIC_*` variable.

## Gateway contract

### `GET /search?q=<query>`

The gateway should return either `tracks` or `results`. Each track should provide at least:

```json
{
  "id": "provider-track-id",
  "name": "Track name",
  "artist_name": "Artist",
  "provider_id": "provider-id"
}
```

Optional fields include `album_name`, `cover_url`, `duration_ms`, `quality`, `explicit` and `preview_url`.

### `POST /resolve`

Request:

```json
{
  "providerId": "provider-id",
  "trackId": "provider-track-id",
  "quality": "optional-quality"
}
```

Response:

```json
{
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

- `GET /api/health` — PWA/API readiness and gateway status
- `GET /api/search?q=...` — validated search adapter
- `POST /api/resolve` — validated stream-resolution adapter
- `GET|HEAD /api/stream?...` — same-origin stream proxy with media Range support
- `GET /api/download?...` — same-origin attachment/download proxy

## Architecture rule

The browser UI must not execute the native Flutter/gomobile filesystem or download stack directly. Provider work that requires extension execution, secret request headers, local filesystem access or other native capabilities belongs in a dedicated Web gateway/server adapter. This keeps the Web deployment compatible with Vercel while preserving the existing Android/iOS architecture.

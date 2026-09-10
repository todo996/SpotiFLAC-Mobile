# SpotiFLAC Web

Independent web/PWA client for SpotiFLAC. It is intentionally isolated under `webapp/` so the existing Flutter Android/iOS implementation is not redesigned or rewritten.

## Targets

- Desktop / laptop browsers
- Tablet browsers
- Mobile browsers
- Installable PWA shell
- Vercel deployment

## Development

```bash
npm install
npm run typecheck
npm run build
npm run dev
```

For Vercel, set **Root Directory** to `webapp` and use the detected Next.js build settings.

## Provider gateway

The browser UI does not execute `.sflx` packages. Search and stream resolution go through a dedicated gateway configured with server-side environment variables:

- `SPOTIFLAC_WEB_GATEWAY_URL` — base URL of the gateway that exposes `/search` and `/resolve`.
- `SPOTIFLAC_WEB_GATEWAY_TOKEN` — optional bearer token sent only from Next.js server routes to the gateway.
- `SPOTIFLAC_WEB_FORCE_STREAM_PROXY=1` — optional. Forces every resolved audio source through the same-origin `/api/stream` route. Normally the proxy is selected only when a provider requires request headers or returns plain HTTP.

Provider stream request headers returned by the gateway are never serialized to browser JavaScript. `/api/stream` keeps them server-side and forwards media range requests (`Range`, `If-Range`) so supported sources remain seekable.

## Architecture rule

Browser UI must call dedicated server/API adapters for provider work that cannot safely run in the browser. Native Flutter/gomobile filesystem and extension download behavior must not be copied into the browser runtime.

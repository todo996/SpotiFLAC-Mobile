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

## Architecture rule

Browser UI must call dedicated server/API adapters for provider work that cannot safely run in the browser. Native Flutter/gomobile filesystem and extension download behavior must not be copied into the browser runtime.

import { NextResponse } from "next/server";
import { isWebGatewayConfigured } from "@/lib/web-gateway";

export const dynamic = "force-dynamic";

export function GET() {
  return NextResponse.json(
    {
      ok: true,
      app: "spotiflac-web",
      platform: "web-pwa",
      gatewayConfigured: isWebGatewayConfigured(),
      capabilities: {
        search: true,
        streaming: true,
        rangeProxy: true,
        download: true,
        mediaSession: true,
        pwa: true,
      },
    },
    { headers: { "Cache-Control": "private, no-store" } },
  );
}

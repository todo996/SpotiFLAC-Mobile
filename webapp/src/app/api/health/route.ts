import { NextResponse } from "next/server";
import { gatewayHealth } from "@/lib/web-gateway";

export const dynamic = "force-dynamic";

export async function GET() {
  const gateway = await gatewayHealth();
  return NextResponse.json(
    {
      ok: true,
      app: "spotiflac-web",
      platform: "web-pwa",
      gatewayConfigured: gateway.configured,
      gatewayReachable: gateway.reachable,
      gatewayReady: gateway.ready,
      gatewayProviderCount: gateway.providerCount,
      gatewayCapabilities: gateway.capabilities,
      gatewayError: gateway.error,
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

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
      gatewayStatus: gateway.status,
      gatewayProviderCount: gateway.providerCount,
      gatewayAuth: gateway.auth,
      gatewayCapabilities: gateway.capabilities,
      gatewayError: gateway.error,
      gateway: {
        configured: gateway.configured,
        reachable: gateway.reachable,
        ready: gateway.ready,
        status: gateway.status,
        providerCount: gateway.providerCount,
        auth: gateway.auth,
        capabilities: gateway.capabilities,
        error: gateway.error,
      },
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

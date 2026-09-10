import { NextRequest, NextResponse } from "next/server";
import { gatewayResolveStream, isWebGatewayConfigured } from "@/lib/web-gateway";

export const dynamic = "force-dynamic";

export async function POST(request: NextRequest) {
  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "Yêu cầu không hợp lệ." }, { status: 400 });
  }

  const data = body && typeof body === "object" ? (body as Record<string, unknown>) : {};
  const providerId = String(data.providerId ?? data.provider_id ?? "").trim();
  const trackId = String(data.trackId ?? data.track_id ?? "").trim();
  const quality = String(data.quality ?? "").trim();
  if (!providerId || !trackId) {
    return NextResponse.json({ error: "Thiếu nguồn nhạc hoặc mã bài hát." }, { status: 400 });
  }
  if (!isWebGatewayConfigured()) {
    return NextResponse.json(
      {
        error: "Nguồn phát nhạc cho Web chưa được cấu hình.",
        code: "WEB_GATEWAY_NOT_CONFIGURED",
      },
      { status: 503 },
    );
  }

  try {
    const stream = await gatewayResolveStream({ providerId, trackId, quality });
    const requiresProxy =
      Object.keys(stream.headers ?? {}).length > 0 ||
      stream.url.toLowerCase().startsWith("http://") ||
      process.env.SPOTIFLAC_WEB_FORCE_STREAM_PROXY === "1";

    let playbackUrl = stream.url;
    if (requiresProxy) {
      const proxyUrl = new URL("/api/stream", request.url);
      proxyUrl.searchParams.set("providerId", providerId);
      proxyUrl.searchParams.set("trackId", trackId);
      if (quality) proxyUrl.searchParams.set("quality", quality);
      playbackUrl = proxyUrl.toString();
    }

    // Provider headers are intentionally omitted. They are consumed only by
    // /api/stream so bearer tokens, cookies, signed values, or custom headers
    // never become visible to browser JavaScript.
    return NextResponse.json(
      {
        url: playbackUrl,
        contentType: stream.contentType,
        expiresAtMs: stream.expiresAtMs,
        provider: stream.provider,
        quality: stream.quality,
        proxied: requiresProxy,
      },
      { headers: { "Cache-Control": "private, no-store" } },
    );
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "Không thể lấy nguồn phát nhạc." },
      { status: 502 },
    );
  }
}

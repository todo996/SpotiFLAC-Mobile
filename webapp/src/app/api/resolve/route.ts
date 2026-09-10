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
    return NextResponse.json(stream, {
      headers: { "Cache-Control": "private, no-store" },
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "Không thể lấy nguồn phát nhạc." },
      { status: 502 },
    );
  }
}

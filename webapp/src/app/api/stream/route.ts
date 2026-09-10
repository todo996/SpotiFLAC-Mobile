import { NextRequest, NextResponse } from "next/server";
import { gatewayResolveStream, isWebGatewayConfigured } from "@/lib/web-gateway";

export const dynamic = "force-dynamic";

const passthroughRequestHeaders = [
  "range",
  "if-range",
  "if-none-match",
  "if-modified-since",
  "accept",
] as const;

const passthroughResponseHeaders = [
  "accept-ranges",
  "content-length",
  "content-range",
  "content-type",
  "etag",
  "last-modified",
] as const;

function badRequest(message: string) {
  return NextResponse.json(
    { error: message },
    { status: 400, headers: { "Cache-Control": "private, no-store" } },
  );
}

async function proxyStream(request: NextRequest): Promise<Response> {
  const providerId = request.nextUrl.searchParams.get("providerId")?.trim() ?? "";
  const trackId = request.nextUrl.searchParams.get("trackId")?.trim() ?? "";
  const quality = request.nextUrl.searchParams.get("quality")?.trim() ?? "";

  if (!providerId || !trackId) {
    return badRequest("Thiếu nguồn nhạc hoặc mã bài hát.");
  }
  if (providerId.length > 200 || trackId.length > 1_000 || quality.length > 200) {
    return badRequest("Thông tin nguồn phát nhạc quá dài.");
  }
  if (!isWebGatewayConfigured()) {
    return NextResponse.json(
      {
        error: "Nguồn phát nhạc cho Web chưa được cấu hình.",
        code: "WEB_GATEWAY_NOT_CONFIGURED",
      },
      { status: 503, headers: { "Cache-Control": "private, no-store" } },
    );
  }

  try {
    // Resolve on every media request rather than accepting an arbitrary URL
    // from the browser. This keeps signed URLs and provider credentials on the
    // server and naturally refreshes them when the browser retries a range.
    let stream = await gatewayResolveStream({ providerId, trackId, quality });
    if (stream.expiresAtMs && stream.expiresAtMs <= Date.now() + 15_000) {
      stream = await gatewayResolveStream({ providerId, trackId, quality });
    }

    const upstreamHeaders = new Headers(stream.headers);
    upstreamHeaders.set("Accept-Encoding", "identity");
    for (const name of passthroughRequestHeaders) {
      const value = request.headers.get(name);
      if (value) upstreamHeaders.set(name, value);
    }

    const upstream = await fetch(stream.url, {
      method: request.method,
      headers: upstreamHeaders,
      cache: "no-store",
      redirect: "follow",
    });

    const responseHeaders = new Headers({
      "Cache-Control": "private, no-store",
      "X-Content-Type-Options": "nosniff",
    });
    for (const name of passthroughResponseHeaders) {
      const value = upstream.headers.get(name);
      if (value) responseHeaders.set(name, value);
    }
    if (!responseHeaders.has("Content-Type") && stream.contentType) {
      responseHeaders.set("Content-Type", stream.contentType);
    }

    // Keep the upstream status intact. In particular, audio players depend on
    // 206 + Content-Range for seeking and 416 when a range is invalid.
    return new Response(request.method === "HEAD" ? null : upstream.body, {
      status: upstream.status,
      statusText: upstream.statusText,
      headers: responseHeaders,
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "Không thể truyền luồng âm thanh." },
      { status: 502, headers: { "Cache-Control": "private, no-store" } },
    );
  }
}

export async function GET(request: NextRequest) {
  return proxyStream(request);
}

export async function HEAD(request: NextRequest) {
  return proxyStream(request);
}

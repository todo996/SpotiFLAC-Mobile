import { NextRequest, NextResponse } from "next/server";
import { gatewayResolveStream, isWebGatewayConfigured } from "@/lib/web-gateway";

export const dynamic = "force-dynamic";

function badRequest(message: string) {
  return NextResponse.json(
    { error: message },
    { status: 400, headers: { "Cache-Control": "private, no-store" } },
  );
}

function safeFilename(value: string): string {
  const trimmed = value.trim().replace(/[\\/:*?"<>|\u0000-\u001f\u007f]+/g, "_");
  const withoutTrailingDots = trimmed.replace(/[. ]+$/g, "");
  return (withoutTrailingDots || "SpotiFLAC-track").slice(0, 180);
}

function encodeFilename(value: string): string {
  return encodeURIComponent(value)
    .replace(/['()]/g, escape)
    .replace(/\*/g, "%2A");
}

export async function GET(request: NextRequest) {
  const providerId = request.nextUrl.searchParams.get("providerId")?.trim() ?? "";
  const trackId = request.nextUrl.searchParams.get("trackId")?.trim() ?? "";
  const quality = request.nextUrl.searchParams.get("quality")?.trim() ?? "";
  const requestedName = request.nextUrl.searchParams.get("filename") ?? "SpotiFLAC-track";

  if (!providerId || !trackId) {
    return badRequest("Thiếu nguồn nhạc hoặc mã bài hát.");
  }
  if (providerId.length > 200 || trackId.length > 1_000 || quality.length > 200) {
    return badRequest("Thông tin bài hát quá dài.");
  }
  if (!isWebGatewayConfigured()) {
    return NextResponse.json(
      {
        error: "Nguồn nhạc cho Web chưa được cấu hình.",
        code: "WEB_GATEWAY_NOT_CONFIGURED",
      },
      { status: 503, headers: { "Cache-Control": "private, no-store" } },
    );
  }

  try {
    let stream = await gatewayResolveStream({ providerId, trackId, quality });
    if (stream.expiresAtMs && stream.expiresAtMs <= Date.now() + 15_000) {
      stream = await gatewayResolveStream({ providerId, trackId, quality });
    }

    const upstreamHeaders = new Headers(stream.headers);
    upstreamHeaders.set("Accept-Encoding", "identity");
    const upstream = await fetch(stream.url, {
      headers: upstreamHeaders,
      cache: "no-store",
      redirect: "follow",
    });

    if (!upstream.ok || !upstream.body) {
      return NextResponse.json(
        { error: `Không thể tải tệp âm thanh (${upstream.status}).` },
        { status: 502, headers: { "Cache-Control": "private, no-store" } },
      );
    }

    const contentType = upstream.headers.get("content-type") || stream.contentType || "application/octet-stream";
    const extension = contentType.includes("flac")
      ? ".flac"
      : contentType.includes("mpeg")
        ? ".mp3"
        : contentType.includes("mp4") || contentType.includes("m4a")
          ? ".m4a"
          : contentType.includes("ogg") || contentType.includes("opus")
            ? ".opus"
            : "";
    const baseName = safeFilename(requestedName).replace(/\.(flac|mp3|m4a|aac|ogg|opus|wav)$/i, "");
    const filename = `${baseName}${extension}`;

    const headers = new Headers({
      "Cache-Control": "private, no-store",
      "Content-Type": contentType,
      "Content-Disposition": `attachment; filename="SpotiFLAC-track${extension}"; filename*=UTF-8''${encodeFilename(filename)}`,
      "X-Content-Type-Options": "nosniff",
    });
    const contentLength = upstream.headers.get("content-length");
    if (contentLength) headers.set("Content-Length", contentLength);

    return new Response(upstream.body, { status: 200, headers });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "Không thể tải bài hát." },
      { status: 502, headers: { "Cache-Control": "private, no-store" } },
    );
  }
}

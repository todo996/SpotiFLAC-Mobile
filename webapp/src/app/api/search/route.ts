import { NextRequest, NextResponse } from "next/server";
import { gatewaySearch, isWebGatewayConfigured } from "@/lib/web-gateway";

export const dynamic = "force-dynamic";

export async function GET(request: NextRequest) {
  const query = request.nextUrl.searchParams.get("q")?.trim() ?? "";
  if (!query) {
    return NextResponse.json({ error: "Vui lòng nhập tên bài hát, album, nghệ sĩ hoặc liên kết." }, { status: 400 });
  }
  if (query.length > 500) {
    return NextResponse.json({ error: "Nội dung tìm kiếm quá dài." }, { status: 400 });
  }
  if (!isWebGatewayConfigured()) {
    return NextResponse.json(
      {
        error: "Nguồn nhạc cho Web chưa được cấu hình.",
        code: "WEB_GATEWAY_NOT_CONFIGURED",
      },
      { status: 503 },
    );
  }

  try {
    const result = await gatewaySearch(query);
    return NextResponse.json(result, {
      headers: { "Cache-Control": "private, no-store" },
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "Không thể tìm kiếm lúc này." },
      { status: 502 },
    );
  }
}

import { NextResponse } from "next/server";
import { fetchProviderInventory } from "@/lib/provider-inventory";

export const dynamic = "force-dynamic";

export async function GET() {
  try {
    const inventory = await fetchProviderInventory();
    return NextResponse.json(inventory, {
      headers: { "Cache-Control": "private, no-store" },
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : "Provider inventory is unavailable.";
    return NextResponse.json(
      { providers: [], count: 0, error: message },
      {
        status: message === "WEB_GATEWAY_NOT_CONFIGURED" ? 503 : 502,
        headers: { "Cache-Control": "private, no-store" },
      },
    );
  }
}

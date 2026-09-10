import { normalizeTrack, type ResolvedWebStream, type SearchResponse } from "@/lib/music";

function gatewayBase(): string | null {
  const value = process.env.SPOTIFLAC_WEB_GATEWAY_URL?.trim();
  return value ? value.replace(/\/$/, "") : null;
}

function gatewayHeaders(): HeadersInit {
  const headers: Record<string, string> = { Accept: "application/json" };
  const token = process.env.SPOTIFLAC_WEB_GATEWAY_TOKEN?.trim();
  if (token) headers.Authorization = `Bearer ${token}`;
  return headers;
}

async function readJson(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    throw new Error("Web gateway returned an invalid JSON response.");
  }
}

export function isWebGatewayConfigured(): boolean {
  return gatewayBase() !== null;
}

export async function gatewaySearch(query: string): Promise<SearchResponse> {
  const base = gatewayBase();
  if (!base) {
    throw new Error("WEB_GATEWAY_NOT_CONFIGURED");
  }

  const url = new URL(`${base}/search`);
  url.searchParams.set("q", query);
  const response = await fetch(url, {
    headers: gatewayHeaders(),
    cache: "no-store",
    signal: AbortSignal.timeout(20_000),
  });
  const payload = await readJson(response);
  if (!response.ok) {
    const data = payload as Record<string, unknown> | null;
    throw new Error(String(data?.error ?? `Search gateway failed (${response.status}).`));
  }

  const data = (payload ?? {}) as Record<string, unknown>;
  const rawTracks = Array.isArray(data.tracks)
    ? data.tracks
    : Array.isArray(data.results)
      ? data.results
      : [];
  return {
    tracks: rawTracks.map(normalizeTrack).filter((track) => track !== null),
    provider: typeof data.provider === "string" ? data.provider : undefined,
    message: typeof data.message === "string" ? data.message : undefined,
  };
}

export async function gatewayResolveStream(input: {
  providerId: string;
  trackId: string;
  quality?: string;
}): Promise<ResolvedWebStream> {
  const base = gatewayBase();
  if (!base) {
    throw new Error("WEB_GATEWAY_NOT_CONFIGURED");
  }

  const response = await fetch(`${base}/resolve`, {
    method: "POST",
    headers: { ...gatewayHeaders(), "Content-Type": "application/json" },
    body: JSON.stringify(input),
    cache: "no-store",
    signal: AbortSignal.timeout(20_000),
  });
  const payload = (await readJson(response)) as Record<string, unknown> | null;
  if (!response.ok) {
    throw new Error(String(payload?.error ?? `Stream resolver failed (${response.status}).`));
  }

  const url = String(payload?.url ?? "").trim();
  const parsed = URL.parse(url);
  if (!parsed || (parsed.protocol !== "https:" && parsed.protocol !== "http:")) {
    throw new Error("Resolver returned an invalid stream URL.");
  }
  return {
    url,
    contentType: typeof payload?.content_type === "string" ? payload.content_type : undefined,
    expiresAtMs: Number.isFinite(Number(payload?.expires_at_ms))
      ? Number(payload?.expires_at_ms)
      : undefined,
    provider: typeof payload?.provider === "string" ? payload.provider : undefined,
    quality: typeof payload?.quality === "string" ? payload.quality : undefined,
  };
}

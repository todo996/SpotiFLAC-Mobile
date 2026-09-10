import { normalizeTrack, type ResolvedWebStream, type SearchResponse } from "@/lib/music";

export interface WebGatewayHealth {
  configured: boolean;
  reachable: boolean;
  ready: boolean;
  providerCount: number;
  capabilities: {
    search: boolean;
    resolve: boolean;
  };
  error?: string;
}

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

function normalizeStreamHeaders(value: unknown): Readonly<Record<string, string>> {
  if (!value || typeof value !== "object" || Array.isArray(value)) return {};

  const result: Record<string, string> = {};
  const blocked = new Set([
    "connection",
    "content-length",
    "host",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailer",
    "transfer-encoding",
    "upgrade",
  ]);

  for (const [rawKey, rawValue] of Object.entries(value as Record<string, unknown>)) {
    const key = rawKey.trim();
    const headerValue = String(rawValue ?? "");
    const lower = key.toLowerCase();
    if (!key || blocked.has(lower)) continue;
    if (key.includes("\r") || key.includes("\n") || headerValue.includes("\r") || headerValue.includes("\n")) {
      throw new Error("Resolver returned an invalid stream request header.");
    }
    if (Object.keys(result).length >= 32 || key.length > 128 || headerValue.length > 8_192) {
      throw new Error("Resolver returned an oversized stream request header set.");
    }
    result[key] = headerValue;
  }
  return Object.freeze(result);
}

export function isWebGatewayConfigured(): boolean {
  return gatewayBase() !== null;
}

export async function gatewayHealth(): Promise<WebGatewayHealth> {
  const base = gatewayBase();
  const unavailable = (error?: string): WebGatewayHealth => ({
    configured: base !== null,
    reachable: false,
    ready: false,
    providerCount: 0,
    capabilities: { search: false, resolve: false },
    ...(error ? { error } : {}),
  });
  if (!base) return unavailable();

  try {
    const response = await fetch(`${base}/health`, {
      headers: gatewayHeaders(),
      cache: "no-store",
      signal: AbortSignal.timeout(5_000),
    });
    const payload = (await readJson(response)) as Record<string, unknown> | null;
    if (!response.ok) {
      return unavailable(String(payload?.error ?? `Gateway health check failed (${response.status}).`));
    }

    const rawCount = Number(payload?.providerCount ?? payload?.provider_count ?? 0);
    const providerCount = Number.isFinite(rawCount) && rawCount > 0 ? Math.floor(rawCount) : 0;
    const rawCapabilities = payload?.capabilities && typeof payload.capabilities === "object"
      ? payload.capabilities as Record<string, unknown>
      : {};
    const capabilities = {
      search: rawCapabilities.search === true,
      resolve: rawCapabilities.resolve === true,
    };

    return {
      configured: true,
      reachable: true,
      ready: providerCount > 0 && capabilities.search && capabilities.resolve,
      providerCount,
      capabilities,
    };
  } catch (error) {
    return unavailable(error instanceof Error ? error.message : "Gateway health check failed.");
  }
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
    headers: normalizeStreamHeaders(payload?.headers),
    contentType:
      typeof payload?.content_type === "string"
        ? payload.content_type
        : typeof payload?.contentType === "string"
          ? payload.contentType
          : undefined,
    expiresAtMs: Number.isFinite(Number(payload?.expires_at_ms ?? payload?.expiresAtMs))
      ? Number(payload?.expires_at_ms ?? payload?.expiresAtMs)
      : undefined,
    provider: typeof payload?.provider === "string" ? payload.provider : undefined,
    quality: typeof payload?.quality === "string" ? payload.quality : undefined,
  };
}

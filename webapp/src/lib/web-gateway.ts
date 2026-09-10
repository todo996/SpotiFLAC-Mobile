import { normalizeTrack, type ResolvedWebStream, type SearchResponse } from "@/lib/music";

export type WebGatewayStatus =
  | "not_configured"
  | "unreachable"
  | "authentication_required"
  | "authentication_failed"
  | "no_providers"
  | "degraded"
  | "ready";

export interface WebGatewayHealth {
  configured: boolean;
  reachable: boolean;
  ready: boolean;
  providerCount: number;
  status: WebGatewayStatus;
  auth: {
    required: boolean | null;
    authenticated: boolean;
    tokenConfigured: boolean;
  };
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

function gatewayToken(): string | null {
  const value = process.env.SPOTIFLAC_WEB_GATEWAY_TOKEN?.trim();
  return value || null;
}

function gatewayHeaders(): HeadersInit {
  const headers: Record<string, string> = { Accept: "application/json" };
  const token = gatewayToken();
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
  const tokenConfigured = gatewayToken() !== null;

  const state = (
    status: WebGatewayStatus,
    options: {
      configured?: boolean;
      reachable?: boolean;
      providerCount?: number;
      authRequired?: boolean | null;
      authenticated?: boolean;
      capabilities?: { search: boolean; resolve: boolean };
      error?: string;
    } = {},
  ): WebGatewayHealth => ({
    configured: options.configured ?? base !== null,
    reachable: options.reachable ?? false,
    ready: status === "ready",
    providerCount: options.providerCount ?? 0,
    status,
    auth: {
      required: options.authRequired ?? null,
      authenticated: options.authenticated ?? false,
      tokenConfigured,
    },
    capabilities: options.capabilities ?? { search: false, resolve: false },
    ...(options.error ? { error: options.error } : {}),
  });

  if (!base) return state("not_configured", { configured: false });

  try {
    const response = await fetch(`${base}/health`, {
      headers: gatewayHeaders(),
      cache: "no-store",
      signal: AbortSignal.timeout(5_000),
    });
    const payload = (await readJson(response)) as Record<string, unknown> | null;

    if (!response.ok) {
      const error = String(payload?.error ?? `Gateway health check failed (${response.status}).`);
      if (response.status === 401) {
        return state(tokenConfigured ? "authentication_failed" : "authentication_required", {
          reachable: true,
          authRequired: true,
          authenticated: false,
          error,
        });
      }
      return state("degraded", { reachable: true, error });
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
    const rawAuth = payload?.auth && typeof payload.auth === "object"
      ? payload.auth as Record<string, unknown>
      : {};
    const authRequired = typeof rawAuth.required === "boolean" ? rawAuth.required : null;
    const authenticated = rawAuth.authenticated !== false;
    const backendReady = typeof payload?.ready === "boolean" ? payload.ready : providerCount > 0;
    const ready = backendReady && providerCount > 0 && capabilities.search && capabilities.resolve && authenticated;

    if (ready) {
      return state("ready", {
        reachable: true,
        providerCount,
        authRequired,
        authenticated,
        capabilities,
      });
    }
    if (providerCount === 0) {
      return state("no_providers", {
        reachable: true,
        providerCount,
        authRequired,
        authenticated,
        capabilities,
      });
    }
    return state("degraded", {
      reachable: true,
      providerCount,
      authRequired,
      authenticated,
      capabilities,
      error: "Gateway is reachable but one or more required capabilities are unavailable.",
    });
  } catch (error) {
    return state("unreachable", {
      error: error instanceof Error ? error.message : "Gateway health check failed.",
    });
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

export type ProviderStatus =
  | "ready"
  | "disabled"
  | "error"
  | "needs_configuration"
  | "verification_pending";

export interface WebProviderSetting {
  key: string;
  label: string;
  type: string;
  required: boolean;
  secret: boolean;
  configured: boolean;
  options?: string[];
}

export interface WebProviderInfo {
  id: string;
  displayName: string;
  version: string;
  enabled: boolean;
  status: ProviderStatus;
  error?: string;
  capabilities: {
    metadata: boolean;
    download: boolean;
    lyrics: boolean;
    search: boolean;
    streamMode: "none" | "supported" | "extension_defined" | "requires_processing";
  };
  configuration: {
    complete: boolean;
    missingRequired: string[];
    settings: WebProviderSetting[];
  };
  auth: {
    required: boolean;
    authenticated: boolean;
    pending: boolean;
  };
}

export interface WebProviderInventory {
  providers: WebProviderInfo[];
  count: number;
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

export async function fetchProviderInventory(): Promise<WebProviderInventory> {
  const base = gatewayBase();
  if (!base) throw new Error("WEB_GATEWAY_NOT_CONFIGURED");

  const response = await fetch(`${base}/providers`, {
    headers: gatewayHeaders(),
    cache: "no-store",
    signal: AbortSignal.timeout(5_000),
  });
  const text = await response.text();
  let payload: unknown = null;
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      throw new Error("Web gateway returned invalid provider JSON.");
    }
  }
  if (!response.ok) {
    const error = payload && typeof payload === "object"
      ? String((payload as Record<string, unknown>).error ?? "")
      : "";
    throw new Error(error || `Provider inventory failed (${response.status}).`);
  }

  const data = payload && typeof payload === "object"
    ? payload as Record<string, unknown>
    : {};
  const providers = Array.isArray(data.providers)
    ? data.providers as WebProviderInfo[]
    : [];
  return { providers, count: providers.length };
}

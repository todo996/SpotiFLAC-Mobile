export type MusicItemType = "track" | "album" | "playlist" | "artist";

export interface WebTrack {
  id: string;
  name: string;
  artistName: string;
  albumName?: string;
  coverUrl?: string;
  durationMs?: number;
  providerId: string;
  quality?: string;
  explicit?: boolean;
  previewUrl?: string;
}

export interface SearchResponse {
  tracks: WebTrack[];
  provider?: string;
  message?: string;
}

export interface ResolveStreamRequest {
  providerId: string;
  trackId: string;
  quality?: string;
}

export interface ResolvedWebStream {
  url: string;
  contentType?: string;
  expiresAtMs?: number;
  provider?: string;
  quality?: string;
}

export interface PlayerQueueItem extends WebTrack {
  queueId: string;
}

export function normalizeTrack(input: unknown): WebTrack | null {
  if (!input || typeof input !== "object") return null;
  const data = input as Record<string, unknown>;
  const id = String(data.id ?? data.spotify_id ?? "").trim();
  const name = String(data.name ?? "").trim();
  const artistName = String(data.artistName ?? data.artist_name ?? data.artists ?? data.artist ?? "").trim();
  const providerId = String(data.providerId ?? data.provider_id ?? data.source ?? "").trim();
  if (!id || !name || !providerId) return null;

  const rawDuration = Number(data.durationMs ?? data.duration_ms ?? 0);
  return {
    id,
    name,
    artistName,
    albumName: String(data.albumName ?? data.album_name ?? data.album ?? "").trim() || undefined,
    coverUrl: String(data.coverUrl ?? data.cover_url ?? data.images ?? "").trim() || undefined,
    durationMs: Number.isFinite(rawDuration) && rawDuration > 0 ? rawDuration : undefined,
    providerId,
    quality: String(data.quality ?? data.audio_quality ?? "").trim() || undefined,
    explicit: data.explicit === true,
    previewUrl: String(data.previewUrl ?? data.preview_url ?? "").trim() || undefined,
  };
}

"use client";

import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { PwaInstallButton } from "@/components/pwa-install-button";
import type { PlayerQueueItem, SearchResponse, WebTrack } from "@/lib/music";

function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "00:00";
  const whole = Math.floor(seconds);
  const minutes = Math.floor(whole / 60);
  const remaining = whole % 60;
  return `${String(minutes).padStart(2, "0")}:${String(remaining).padStart(2, "0")}`;
}

function queueItem(track: WebTrack): PlayerQueueItem {
  const random = typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
  return { ...track, queueId: `${track.providerId}:${track.id}:${random}` };
}

async function readApiError(response: Response, fallback: string): Promise<string> {
  try {
    const payload = (await response.json()) as { error?: unknown };
    const message = String(payload.error ?? "").trim();
    return message || fallback;
  } catch {
    return fallback;
  }
}

export function WebMusicApp() {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const requestGeneration = useRef(0);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<WebTrack[]>([]);
  const [searching, setSearching] = useState(false);
  const [queue, setQueue] = useState<PlayerQueueItem[]>([]);
  const [currentIndex, setCurrentIndex] = useState(-1);
  const [playing, setPlaying] = useState(false);
  const [loading, setLoading] = useState(false);
  const [position, setPosition] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [message, setMessage] = useState("Tìm một bài hát để bắt đầu nghe.");
  const [error, setError] = useState<string | null>(null);

  const current = currentIndex >= 0 && currentIndex < queue.length ? queue[currentIndex] : null;

  const resolveAndPlay = useCallback(async (item: PlayerQueueItem, index: number) => {
    const generation = ++requestGeneration.current;
    const audio = audioRef.current;
    if (!audio) return;

    setLoading(true);
    setError(null);
    setMessage(`Đang chuẩn bị ${item.name}…`);
    setPosition(0);
    setDuration(item.durationMs ? item.durationMs / 1000 : 0);

    try {
      const response = await fetch("/api/resolve", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          providerId: item.providerId,
          trackId: item.id,
          quality: item.quality ?? "",
        }),
      });
      if (!response.ok) {
        throw new Error(await readApiError(response, "Không thể lấy nguồn phát nhạc."));
      }
      const stream = (await response.json()) as { url?: unknown };
      const url = String(stream.url ?? "").trim();
      if (!/^https?:\/\//i.test(url)) {
        throw new Error("Nguồn nhạc trả về không hợp lệ.");
      }
      if (generation !== requestGeneration.current) return;

      audio.pause();
      audio.src = url;
      audio.load();
      setCurrentIndex(index);
      await audio.play();
      if (generation !== requestGeneration.current) return;
      setPlaying(true);
      setMessage(`Đang phát ${item.name}`);
    } catch (cause) {
      if (generation !== requestGeneration.current) return;
      setPlaying(false);
      const detail = cause instanceof Error ? cause.message : "Không thể phát bài hát này.";
      setError(detail);
      setMessage("Phát nhạc bị gián đoạn.");
    } finally {
      if (generation === requestGeneration.current) setLoading(false);
    }
  }, []);

  const playIndex = useCallback(async (index: number) => {
    if (index < 0 || index >= queue.length) return;
    await resolveAndPlay(queue[index], index);
  }, [queue, resolveAndPlay]);

  const playTrackNow = useCallback(async (track: WebTrack) => {
    const item = queueItem(track);
    setQueue([item]);
    await resolveAndPlay(item, 0);
  }, [resolveAndPlay]);

  const playAllResults = useCallback(async () => {
    if (results.length === 0) return;
    const items = results.map(queueItem);
    setQueue(items);
    await resolveAndPlay(items[0], 0);
  }, [results, resolveAndPlay]);

  const playNext = useCallback(async () => {
    if (queue.length === 0) return;
    const next = currentIndex < 0 ? 0 : currentIndex + 1;
    if (next >= queue.length) {
      setPlaying(false);
      setMessage("Đã phát hết hàng chờ.");
      return;
    }
    await resolveAndPlay(queue[next], next);
  }, [currentIndex, queue, resolveAndPlay]);

  const playPrevious = useCallback(async () => {
    const audio = audioRef.current;
    if (audio && audio.currentTime > 3) {
      audio.currentTime = 0;
      return;
    }
    if (currentIndex > 0) await resolveAndPlay(queue[currentIndex - 1], currentIndex - 1);
  }, [currentIndex, queue, resolveAndPlay]);

  const togglePlayback = useCallback(async () => {
    const audio = audioRef.current;
    if (!audio) return;
    if (!current) {
      if (queue.length > 0) await playIndex(0);
      else if (results.length > 0) await playAllResults();
      return;
    }
    if (audio.paused) {
      try {
        await audio.play();
        setPlaying(true);
      } catch {
        await resolveAndPlay(current, currentIndex);
      }
    } else {
      audio.pause();
      setPlaying(false);
    }
  }, [current, currentIndex, playAllResults, playIndex, queue.length, resolveAndPlay, results.length]);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;
    const onTime = () => setPosition(audio.currentTime || 0);
    const onDuration = () => setDuration(Number.isFinite(audio.duration) ? audio.duration : 0);
    const onPlay = () => setPlaying(true);
    const onPause = () => setPlaying(false);
    const onEnded = () => void playNext();
    const onError = () => {
      if (!audio.src) return;
      setPlaying(false);
      setLoading(false);
      setError("Trình duyệt không thể phát nguồn âm thanh này.");
    };
    audio.addEventListener("timeupdate", onTime);
    audio.addEventListener("durationchange", onDuration);
    audio.addEventListener("loadedmetadata", onDuration);
    audio.addEventListener("play", onPlay);
    audio.addEventListener("pause", onPause);
    audio.addEventListener("ended", onEnded);
    audio.addEventListener("error", onError);
    return () => {
      audio.removeEventListener("timeupdate", onTime);
      audio.removeEventListener("durationchange", onDuration);
      audio.removeEventListener("loadedmetadata", onDuration);
      audio.removeEventListener("play", onPlay);
      audio.removeEventListener("pause", onPause);
      audio.removeEventListener("ended", onEnded);
      audio.removeEventListener("error", onError);
    };
  }, [playNext]);

  useEffect(() => {
    const audio = audioRef.current;
    if (audio) audio.volume = volume;
  }, [volume]);

  useEffect(() => {
    if (!("mediaSession" in navigator)) return;
    if (current) {
      navigator.mediaSession.metadata = new MediaMetadata({
        title: current.name,
        artist: current.artistName,
        album: current.albumName ?? "",
        artwork: current.coverUrl ? [{ src: current.coverUrl }] : [],
      });
      navigator.mediaSession.playbackState = playing ? "playing" : "paused";
    } else {
      navigator.mediaSession.metadata = null;
      navigator.mediaSession.playbackState = "none";
    }

    const handlers: Array<[MediaSessionAction, MediaSessionActionHandler]> = [
      ["play", () => void togglePlayback()],
      ["pause", () => void togglePlayback()],
      ["previoustrack", () => void playPrevious()],
      ["nexttrack", () => void playNext()],
      ["seekto", (details) => {
        const audio = audioRef.current;
        if (!audio || details.seekTime == null) return;
        audio.currentTime = details.seekTime;
      }],
    ];
    for (const [action, handler] of handlers) {
      try { navigator.mediaSession.setActionHandler(action, handler); } catch { /* unsupported action */ }
    }
    return () => {
      for (const [action] of handlers) {
        try { navigator.mediaSession.setActionHandler(action, null); } catch { /* unsupported action */ }
      }
    };
  }, [current, playNext, playPrevious, playing, togglePlayback]);

  useEffect(() => {
    if (!("mediaSession" in navigator) || !Number.isFinite(duration) || duration <= 0) return;
    try {
      navigator.mediaSession.setPositionState({
        duration,
        playbackRate: audioRef.current?.playbackRate ?? 1,
        position: Math.min(Math.max(position, 0), duration),
      });
    } catch { /* browser may reject transient invalid media state */ }
  }, [duration, position]);

  async function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = query.trim();
    if (!normalized) return;
    setSearching(true);
    setError(null);
    setMessage(`Đang tìm “${normalized}”…`);
    try {
      const response = await fetch(`/api/search?q=${encodeURIComponent(normalized)}`, { cache: "no-store" });
      if (!response.ok) throw new Error(await readApiError(response, "Không thể tìm kiếm lúc này."));
      const payload = (await response.json()) as SearchResponse;
      const tracks = Array.isArray(payload.tracks) ? payload.tracks : [];
      setResults(tracks);
      setMessage(tracks.length > 0 ? `Tìm thấy ${tracks.length} bài hát.` : "Không tìm thấy bài hát phù hợp.");
    } catch (cause) {
      const detail = cause instanceof Error ? cause.message : "Không thể tìm kiếm lúc này.";
      setResults([]);
      setError(detail);
      setMessage("Tìm kiếm chưa hoàn tất.");
    } finally {
      setSearching(false);
    }
  }

  const queueLabel = useMemo(() => queue.length > 0 ? `${queue.length} bài` : "Trống", [queue.length]);

  return (
    <main className="app-shell">
      <audio ref={audioRef} preload="metadata" />
      <aside className="sidebar" aria-label="Điều hướng chính">
        <div className="brand"><span className="brand-mark">S</span><span>SpotiFLAC</span></div>
        <nav>
          <a className="nav-item active" href="#home">Trang chủ</a>
          <a className="nav-item" href="#results">Kết quả</a>
          <a className="nav-item" href="#queue">Hàng chờ</a>
          <a className="nav-item" href="#player">Trình phát</a>
        </nav>
      </aside>

      <section className="content" id="home">
        <header className="topbar">
          <div><p className="eyebrow">SPOTIFLAC WEB</p><h1>Nghe nhạc theo cách của bạn</h1></div>
          <PwaInstallButton />
        </header>

        <form className="search-card" onSubmit={submitSearch}>
          <label htmlFor="search">Tìm nhạc hoặc dán liên kết</label>
          <div className="search-row">
            <input id="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Tên bài hát, album, nghệ sĩ hoặc URL…" autoComplete="off" />
            <button type="submit" disabled={searching}>{searching ? "Đang tìm…" : "Tìm kiếm"}</button>
          </div>
          <p>{message}</p>
          {error ? <p className="inline-error" role="alert">{error}</p> : null}
        </form>

        <section id="results" aria-labelledby="results-title">
          <div className="section-title">
            <h2 id="results-title">Kết quả</h2>
            {results.length > 0 ? <button className="text-action" type="button" onClick={() => void playAllResults()}>Phát tất cả</button> : <span>Sẵn sàng trên PC · Tablet · Mobile</span>}
          </div>
          <div className="track-results">
            {results.length === 0 ? (
              <div className="empty-state"><strong>Chưa có kết quả</strong><span>Nhập tên bài hát hoặc dán liên kết ở phía trên.</span></div>
            ) : results.map((track) => (
              <article className="track-row" key={`${track.providerId}:${track.id}`}>
                <div className="result-art" aria-hidden="true">{track.coverUrl ? <img src={track.coverUrl} alt="" /> : "♪"}</div>
                <div className="result-meta"><strong>{track.name}</strong><span>{track.artistName || "Không rõ nghệ sĩ"}{track.albumName ? ` · ${track.albumName}` : ""}</span></div>
                <span className="result-quality">{track.quality || "Online"}</span>
                <button className="round-action primary-action" type="button" aria-label={`Phát ${track.name}`} onClick={() => void playTrackNow(track)}>▶</button>
                <button className="round-action" type="button" aria-label={`Thêm ${track.name} vào hàng chờ`} onClick={() => setQueue((items) => [...items, queueItem(track)])}>＋</button>
              </article>
            ))}
          </div>
        </section>

        <section id="queue" className="queue-section" aria-labelledby="queue-title">
          <div className="section-title"><h2 id="queue-title">Hàng chờ</h2><span>{queueLabel}</span></div>
          <div className="queue-list">
            {queue.length === 0 ? <div className="empty-state compact"><span>Hàng chờ đang trống.</span></div> : queue.map((item, index) => (
              <button type="button" className={`queue-item${index === currentIndex ? " active" : ""}`} key={item.queueId} onClick={() => void playIndex(index)}>
                <span>{index === currentIndex && playing ? "▮▮" : index + 1}</span><strong>{item.name}</strong><small>{item.artistName}</small>
              </button>
            ))}
          </div>
        </section>
      </section>

      <footer className="player" id="player" aria-label="Trình phát nhạc">
        <div className="track-placeholder">
          <div className="mini-art">{current?.coverUrl ? <img src={current.coverUrl} alt="" /> : "♪"}</div>
          <div><strong>{current?.name ?? "Chưa phát bài hát"}</strong><span>{current?.artistName ?? "Chọn một bài để bắt đầu"}</span></div>
        </div>
        <div className="transport">
          <div className="player-controls">
            <button type="button" aria-label="Bài trước" onClick={() => void playPrevious()}>‹</button>
            <button type="button" className="play" aria-label={playing ? "Tạm dừng" : "Phát"} onClick={() => void togglePlayback()} disabled={loading}>{loading ? "…" : playing ? "Ⅱ" : "▶"}</button>
            <button type="button" aria-label="Bài tiếp" onClick={() => void playNext()}>›</button>
          </div>
          <div className="timeline">
            <span>{formatTime(position)}</span>
            <input aria-label="Vị trí phát" type="range" min={0} max={Math.max(duration, 1)} step={0.25} value={Math.min(position, Math.max(duration, 1))} onChange={(event) => {
              const next = Number(event.target.value);
              const audio = audioRef.current;
              if (audio) audio.currentTime = next;
              setPosition(next);
            }} />
            <span>{formatTime(duration)}</span>
          </div>
        </div>
        <label className="volume-control"><span>Âm lượng</span><input aria-label="Âm lượng" type="range" min={0} max={1} step={0.05} value={volume} onChange={(event) => setVolume(Number(event.target.value))} /></label>
      </footer>
    </main>
  );
}

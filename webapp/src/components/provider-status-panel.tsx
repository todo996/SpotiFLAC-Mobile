"use client";

import { useEffect, useMemo, useState } from "react";
import type { WebProviderInfo, WebProviderInventory } from "@/lib/provider-inventory";
import styles from "./provider-status-panel.module.css";

function statusLabel(provider: WebProviderInfo): string {
  switch (provider.status) {
    case "ready":
      return "Sẵn sàng";
    case "disabled":
      return "Đã tắt";
    case "needs_configuration":
      return "Cần cấu hình";
    case "verification_pending":
      return "Cần xác minh";
    case "error":
      return "Có lỗi";
  }
}

function statusClass(provider: WebProviderInfo): string {
  if (provider.status === "ready") return `${styles.status} ${styles.ready}`;
  if (provider.status === "error") return `${styles.status} ${styles.bad}`;
  return `${styles.status} ${styles.warning}`;
}

function streamLabel(mode: WebProviderInfo["capabilities"]["streamMode"]): string | null {
  switch (mode) {
    case "supported":
      return "Phát trực tuyến";
    case "extension_defined":
      return "Stream theo tiện ích";
    case "requires_processing":
      return "Stream cần xử lý";
    default:
      return null;
  }
}

export function ProviderStatusPanel() {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [providers, setProviders] = useState<WebProviderInfo[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        const response = await fetch("/api/providers", { cache: "no-store" });
        const payload = (await response.json()) as WebProviderInventory & { error?: unknown };
        if (cancelled) return;
        setProviders(Array.isArray(payload.providers) ? payload.providers : []);
        setError(response.ok ? null : String(payload.error ?? "Không lấy được danh sách nguồn nhạc."));
      } catch {
        if (!cancelled) setError("Không lấy được danh sách nguồn nhạc.");
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void load();
    const interval = window.setInterval(() => void load(), 30_000);
    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, []);

  const readyCount = useMemo(
    () => providers.filter((provider) => provider.status === "ready").length,
    [providers],
  );

  return (
    <aside className={styles.panel} aria-label="Trạng thái nguồn nhạc Web">
      <button className={styles.toggle} type="button" onClick={() => setOpen((value) => !value)} aria-expanded={open}>
        <strong>Nguồn nhạc</strong>
        <span>{loading ? "Đang kiểm tra…" : `${readyCount}/${providers.length} sẵn sàng ${open ? "▲" : "▼"}`}</span>
      </button>
      {open ? (
        <div className={styles.body}>
          {error ? <p className={`${styles.message} ${styles.error}`}>{error}</p> : null}
          {!error && !loading && providers.length === 0 ? (
            <p className={styles.message}>Chưa có tiện ích nguồn nhạc nào trên gateway.</p>
          ) : null}
          {providers.map((provider) => {
            const stream = streamLabel(provider.capabilities.streamMode);
            return (
              <article className={styles.provider} key={provider.id}>
                <div className={styles.providerTop}>
                  <strong>{provider.displayName || provider.id}</strong>
                  <span className={statusClass(provider)}>{statusLabel(provider)}</span>
                </div>
                <div className={styles.meta}>
                  {provider.capabilities.metadata ? <span className={styles.badge}>Tìm kiếm</span> : null}
                  {provider.capabilities.download ? <span className={styles.badge}>Tải xuống</span> : null}
                  {provider.capabilities.lyrics ? <span className={styles.badge}>Lời bài hát</span> : null}
                  {stream ? <span className={styles.badge}>{stream}</span> : null}
                </div>
                {provider.status === "needs_configuration" ? (
                  <p className={styles.detail}>Thiếu: {provider.configuration.missingRequired.join(", ") || "cấu hình bắt buộc"}. Cấu hình được lưu ở gateway, không lưu trong trình duyệt.</p>
                ) : null}
                {provider.status === "verification_pending" ? (
                  <p className={styles.detail}>Nguồn đang chờ bước xác minh/đăng nhập ở gateway.</p>
                ) : null}
                {provider.capabilities.streamMode === "requires_processing" ? (
                  <p className={styles.detail}>Nguồn này có thể cần giải mã/chuyển container nên Web không giả định có thể phát trực tiếp.</p>
                ) : null}
                {provider.error ? <p className={`${styles.detail} ${styles.error}`}>{provider.error}</p> : null}
              </article>
            );
          })}
        </div>
      ) : null}
    </aside>
  );
}

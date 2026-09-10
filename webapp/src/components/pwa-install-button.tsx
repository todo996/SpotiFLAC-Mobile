"use client";

import { useEffect, useState } from "react";

type BeforeInstallPromptEvent = Event & {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
};

export function PwaInstallButton() {
  const [promptEvent, setPromptEvent] = useState<BeforeInstallPromptEvent | null>(null);
  const [isStandalone, setIsStandalone] = useState(false);

  useEffect(() => {
    const standalone =
      window.matchMedia("(display-mode: standalone)").matches ||
      ("standalone" in navigator && Boolean((navigator as Navigator & { standalone?: boolean }).standalone));
    setIsStandalone(standalone);

    const onBeforeInstall = (event: Event) => {
      event.preventDefault();
      setPromptEvent(event as BeforeInstallPromptEvent);
    };
    const onInstalled = () => {
      setPromptEvent(null);
      setIsStandalone(true);
    };

    window.addEventListener("beforeinstallprompt", onBeforeInstall);
    window.addEventListener("appinstalled", onInstalled);
    return () => {
      window.removeEventListener("beforeinstallprompt", onBeforeInstall);
      window.removeEventListener("appinstalled", onInstalled);
    };
  }, []);

  if (isStandalone) return null;

  const install = async () => {
    if (promptEvent) {
      await promptEvent.prompt();
      await promptEvent.userChoice;
      setPromptEvent(null);
      return;
    }

    // iOS/iPadOS does not expose beforeinstallprompt. Give the platform-native
    // Add to Home Screen guidance without blocking normal web use.
    window.alert("Trên iPhone/iPad: mở menu Chia sẻ của Safari và chọn ‘Thêm vào Màn hình chính’. Trên trình duyệt khác, dùng mục Cài đặt/Cài ứng dụng trong menu trình duyệt.");
  };

  return (
    <button className="install-button" type="button" onClick={install} aria-label="Cài SpotiFLAC PWA">
      Cài ứng dụng
    </button>
  );
}

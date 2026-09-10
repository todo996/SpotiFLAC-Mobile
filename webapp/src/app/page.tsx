import { ProviderStatusPanel } from "@/components/provider-status-panel";
import { WebMusicApp } from "@/components/web-music-app";

export default function HomePage() {
  return (
    <>
      <WebMusicApp />
      <ProviderStatusPanel />
    </>
  );
}

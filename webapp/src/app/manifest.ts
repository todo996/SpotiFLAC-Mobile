import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "SpotiFLAC Web",
    short_name: "SpotiFLAC",
    description: "SpotiFLAC for desktop, tablet, mobile and PWA",
    start_url: "/",
    scope: "/",
    display: "standalone",
    background_color: "#0c0d0f",
    theme_color: "#0c0d0f",
    orientation: "any",
    categories: ["music", "entertainment"],
    icons: [
      {
        src: "/icon.svg",
        sizes: "any",
        type: "image/svg+xml",
        purpose: "any",
      },
    ],
  };
}

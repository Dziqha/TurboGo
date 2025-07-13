import type { Metadata } from "next";

export const defaultMetadata: Metadata = {
  title: {
    default: "TurboGo Documentation",
    template: "%s | TurboGo Docs",
  },
  description:
    "Official documentation for TurboGo – a blazing fast Go framework.",
  keywords: ["Golang", "TurboGo", "framework", "documentation", "API", "web"],
  authors: [{ name: "TurboGo", url: "https://turbogo.web.id" }],
  creator: "Dziqha",
  publisher: "TurboGo",
  metadataBase: new URL("https://turbogo.web.id"),
  openGraph: {
    title: "TurboGo Documentation",
    description:
      "Official documentation for TurboGo – a blazing fast Go framework.",
    url: "https://turbogo.web.id/docs",
    siteName: "TurboGo Docs",
    images: [
      {
        url: "/images/cyclone.png",
        width: 1200,
        height: 630,
        alt: "TurboGo OG Image",
      },
    ],
    locale: "en_US",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "TurboGo Documentation",
    description:
      "Official documentation for TurboGo – a blazing fast Go framework.",
    site: "@turbogo_dev",
    creator: "@turbogo_dev",
    images: ["/images/cyclone.png"],
  },
  icons: {
    icon: "/images/cyclone.png",
    shortcut: "/images/cyclone.png",
    apple: "/images/cyclone.png",
  },
};

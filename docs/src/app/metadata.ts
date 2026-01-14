import type { Metadata } from "next";

export const defaultMetadata: Metadata = {
  title: {
    default: "TurboGo Documentation",
    template: "%s | TurboGo Docs",
  },
  description:
    "Official documentation for TurboGo – a blazing fast Go framework for building high-performance web applications and APIs.",
  keywords: [
    "Golang",
    "TurboGo",
    "framework",
    "documentation",
    "API",
    "web",
    "Go framework",
    "fast",
    "performance",
    "backend",
    "microservices",
  ],
  authors: [{ name: "TurboGo", url: "https://turbogo.web.id" }],
  creator: "Dziqha",
  publisher: "TurboGo",
  category: "technology",
  metadataBase: new URL("https://turbogo.web.id"),
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-video-preview": -1,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },

  verification: {
    google: "DzfDMtvAVr0wAel1ZJXnabbOJIrOAXpew_2C1SCPPn4",
  },

  alternates: {
    canonical: "https://turbogo.web.id",
    languages: {
      "en-US": "https://turbogo.web.id",
      "id-ID": "https://turbogo.web.id/id",
    },
  },

  openGraph: {
    title: "TurboGo Documentation",
    description:
      "Official documentation for TurboGo – a blazing fast Go framework for building high-performance web applications and APIs.",
    url: "https://turbogo.web.id/docs",
    siteName: "TurboGo Docs",
    images: [
      {
        url: "/images/ogturbogo.png",
        width: 1200,
        height: 630,
        alt: "TurboGo Framework - Blazing Fast Go Framework",
      },
    ],
    locale: "en_US",
    alternateLocale: ["id_ID"],
    type: "website",
  },

  twitter: {
    card: "summary_large_image",
    title: "TurboGo Documentation",
    description:
      "Official documentation for TurboGo – a blazing fast Go framework for building high-performance web applications and APIs.",
    site: "@turbogo_dev",
    creator: "@turbogo_dev",
    images: ["/images/ogturbogo.png"],
  },

  icons: {
    icon: [
      { url: "/images/icon.png" },
      { url: "/images/icon.png", sizes: "32x32", type: "image/png" },
      { url: "/images/icon.png", sizes: "16x16", type: "image/png" },
    ],
    shortcut: "/images/icon.png",
    apple: [
      { url: "/images/icon.png" },
      { url: "/images/icon.png", sizes: "180x180", type: "image/png" },
    ],
  },

  manifest: "/manifest.json",
};

import "@/app/global.css";
import { RootProvider } from "fumadocs-ui/provider";
import { Inter } from "next/font/google";
import type { ReactNode } from "react";
import { defaultMetadata } from "./metadata";

export const metadata = defaultMetadata;

const inter = Inter({
  subsets: ["latin"],
});

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className={inter.className} suppressHydrationWarning>
      <head>
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: JSON.stringify({
              "@context": "https://schema.org",
              "@type": "SoftwareApplication",
              name: "TurboGo",
              applicationCategory: "DeveloperApplication",
              operatingSystem: "Cross-platform",
              description:
                "A blazing fast Go framework for building high-performance web applications and APIs",
              url: "https://turbogo.web.id",
              author: {
                "@type": "Person",
                name: "Dziqha",
              },
              offers: {
                "@type": "Offer",
                price: "0",
                priceCurrency: "USD",
              },
              aggregateRating: {
                "@type": "AggregateRating",
                ratingValue: "5",
                ratingCount: "100",
              },
            }),
          }}
        />

        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: JSON.stringify({
              "@context": "https://schema.org",
              "@type": "Organization",
              name: "TurboGo",
              url: "https://turbogo.web.id",
              logo: "https://turbogo.web.id/images/cyclone.png",
              sameAs: ["https://twitter.com/turbogo_dev"],
              contactPoint: {
                "@type": "ContactPoint",
                contactType: "Developer Support",
                url: "https://turbogo.web.id",
              },
            }),
          }}
        />

        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: JSON.stringify({
              "@context": "https://schema.org",
              "@type": "WebSite",
              name: "TurboGo Documentation",
              url: "https://turbogo.web.id",
              potentialAction: {
                "@type": "SearchAction",
                target: "https://turbogo.web.id/search?q={search_term_string}",
                "query-input": "required name=search_term_string",
              },
            }),
          }}
        />
      </head>
      <body className="flex flex-col min-h-screen">
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}

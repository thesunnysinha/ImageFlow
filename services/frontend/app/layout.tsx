import type { Metadata, Viewport } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import AdsScript from "../components/AdsScript";
import { ADSENSE_CLIENT } from "../lib/ads";
import { PAGES } from "../lib/pages";
import { SITE_NAME, SITE_URL } from "../lib/site";
import "./globals.css";

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: { default: `${SITE_NAME}: free online image compressor`, template: "%s" },
  description: "Compress JPG, PNG and WebP images online for free. No sign-up, and your pictures never leave your device.",
  openGraph: { siteName: SITE_NAME, type: "website" },
  alternates: { canonical: "/" },
};

export const viewport: Viewport = { width: "device-width", initialScale: 1 };

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>
        <AdsScript client={ADSENSE_CLIENT} />
        <header className="site-header">
          <Link href="/" className="brand">{SITE_NAME}</Link>
          <nav aria-label="Tools">
            {PAGES.slice(0, 4).map((p) => <Link key={p.slug} href={`/${p.slug}`}>{p.h1.replace(/^Compress (an |a )?/i, "").replace(/ online$/i, "")}</Link>)}
          </nav>
        </header>
        <main>{children}</main>
        <footer className="site-footer">
          <nav aria-label="All tools">
            {PAGES.map((p) => <Link key={p.slug} href={`/${p.slug}`}>{p.h1}</Link>)}
          </nav>
          <p>
            <Link href="/privacy">Privacy policy</Link> · © {new Date().getFullYear()} {SITE_NAME}
          </p>
        </footer>
      </body>
    </html>
  );
}

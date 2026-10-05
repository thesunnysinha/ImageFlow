// The browser calls the relative path /api/v1. Locally the nginx proxy routes it to the backend. On Vercel, set
// API_ORIGIN (for example https://api.example.com) and Next.js forwards the request server to server, so the
// backend needs no CORS configuration. (The compressor itself needs no backend: it runs in the browser.)
const apiOrigin = process.env.API_ORIGIN?.replace(/\/+$/, "");

/** @type {import('next').NextConfig} */
const nextConfig = {
  poweredByHeader: false,
  distDir: process.env.NEXT_DIST_DIR || ".next", // lets the e2e test build an ads-enabled variant next to the normal build
  async rewrites() {
    return apiOrigin ? [{ source: "/api/v1/:path*", destination: `${apiOrigin}/api/v1/:path*` }] : [];
  },
  async headers() {
    return [{
      source: "/:path*",
      headers: [
        { key: "X-Content-Type-Options", value: "nosniff" },
        { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
        { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
      ],
    }];
  },
};

export default nextConfig;

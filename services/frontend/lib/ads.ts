// Ad configuration. A plain module (not a client component) so server and client code read the same values.
// NEXT_PUBLIC_* values are inlined at build time. With no client id, no ad script or markup is rendered.
export const ADSENSE_CLIENT = process.env.NEXT_PUBLIC_ADSENSE_CLIENT ?? ""; // e.g. ca-pub-1234567890123456

export const AD_SLOTS = {
  top: process.env.NEXT_PUBLIC_ADSENSE_SLOT_TOP ?? "",
  content: process.env.NEXT_PUBLIC_ADSENSE_SLOT_CONTENT ?? "",
  bottom: process.env.NEXT_PUBLIC_ADSENSE_SLOT_BOTTOM ?? "",
} as const;

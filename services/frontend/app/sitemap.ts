import type { MetadataRoute } from "next";
import { PAGES } from "../lib/pages";
import { SITE_URL } from "../lib/site";

export default function sitemap(): MetadataRoute.Sitemap {
  return [
    { url: `${SITE_URL}/`, changeFrequency: "monthly", priority: 1 },
    ...PAGES.map((p) => ({ url: `${SITE_URL}/${p.slug}`, changeFrequency: "monthly" as const, priority: 0.8 })),
    { url: `${SITE_URL}/privacy`, changeFrequency: "yearly", priority: 0.2 },
  ];
}

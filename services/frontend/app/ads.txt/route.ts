// ads.txt tells ad buyers which sellers may sell this site's ad space. Rendered from the AdSense client id
// (NEXT_PUBLIC_ADSENSE_CLIENT=ca-pub-...); without it the file does not exist, so no wrong data is published.
export const dynamic = "force-static";

export function GET() {
  const client = process.env.NEXT_PUBLIC_ADSENSE_CLIENT ?? "";
  const match = /^ca-(pub-\d{8,20})$/.exec(client);
  if (!match) return new Response("Not found", { status: 404 });
  return new Response(`google.com, ${match[1]}, DIRECT, f08c47fec0942fa0\n`, { headers: { "Content-Type": "text/plain; charset=utf-8" } });
}

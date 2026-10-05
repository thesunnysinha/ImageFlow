"use client";

import { useEffect, useRef } from "react";
import { ADSENSE_CLIENT, AD_SLOTS } from "../lib/ads";

declare global {
  interface Window {
    adsbygoogle?: unknown[];
  }
}

/**
 * One responsive display ad. Renders nothing unless the AdSense client and this slot id are configured, so local
 * development and previews stay ad free. The box reserves its height up front so the page does not jump when the ad loads.
 * Never place these directly next to buttons: accidental clicks violate AdSense policy.
 */
export function AdSlot({ slot, minHeight = 280 }: { slot: keyof typeof AD_SLOTS; minHeight?: number }) {
  const ref = useRef<HTMLModElement>(null);
  const id = AD_SLOTS[slot];
  useEffect(() => {
    if (!ADSENSE_CLIENT || !id || !ref.current || ref.current.dataset.adsbygoogleStatus) return;
    try {
      (window.adsbygoogle = window.adsbygoogle || []).push({});
    } catch {
      // An ad blocker or a not-yet-loaded script must never break the page.
    }
  }, [id]);
  if (!ADSENSE_CLIENT || !id) return null;
  return (
    <aside className="ad" aria-label="Advertisement" style={{ minHeight }}>
      <span className="ad-label">Advertisement</span>
      <ins ref={ref} className="adsbygoogle" style={{ display: "block", minHeight }} data-ad-client={ADSENSE_CLIENT}
        data-ad-slot={id} data-ad-format="auto" data-full-width-responsive="true" />
    </aside>
  );
}

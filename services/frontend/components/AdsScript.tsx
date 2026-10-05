import Script from "next/script";

// Regions where Google requires consent before personalised ads or ad cookies: the EEA, the UK and Switzerland.
const CONSENT_REGIONS = [
  "AT", "BE", "BG", "HR", "CY", "CZ", "DK", "EE", "FI", "FR", "DE", "GR", "HU", "IS", "IE", "IT", "LV", "LI", "LT", "LU",
  "MT", "NL", "NO", "PL", "PT", "RO", "SK", "SI", "ES", "SE", "GB", "CH",
];

/**
 * Loads the AdSense tag after Google Consent Mode defaults are set to "denied" for the regions above. The consent
 * message itself comes from Google's certified CMP: switch it on in AdSense > Privacy & messaging. Renders nothing
 * without NEXT_PUBLIC_ADSENSE_CLIENT.
 */
export default function AdsScript({ client }: { client: string }) {
  if (!client) return null;
  const defaults = JSON.stringify({
    ad_storage: "denied", ad_user_data: "denied", ad_personalization: "denied", analytics_storage: "denied",
    region: CONSENT_REGIONS, wait_for_update: 500,
  });
  return (
    <>
      <Script id="consent-defaults" strategy="beforeInteractive">
        {`window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}gtag('consent','default',${defaults});`}
      </Script>
      <Script async strategy="afterInteractive" crossOrigin="anonymous"
        src={`https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=${encodeURIComponent(client)}`} />
    </>
  );
}

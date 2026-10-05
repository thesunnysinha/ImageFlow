import type { ToolPage } from "../lib/pages";
import { AdSlot } from "./Ads";
import JsonLd from "./JsonLd";
import Tool from "./Tool";
import { SITE_NAME, SITE_URL } from "../lib/site";

export default function PageContent({ page, path }: { page: Pick<ToolPage, "h1" | "intro" | "steps" | "tips" | "faq" | "preset" | "description">; path: string }) {
  return (
    <>
      <JsonLd data={{
        "@context": "https://schema.org", "@type": "WebApplication", name: page.h1, url: `${SITE_URL}${path}`,
        description: page.description, applicationCategory: "MultimediaApplication", operatingSystem: "Any",
        offers: { "@type": "Offer", price: "0", priceCurrency: "USD" }, publisher: { "@type": "Organization", name: SITE_NAME },
      }} />
      <JsonLd data={{
        "@context": "https://schema.org", "@type": "FAQPage",
        mainEntity: page.faq.map((f) => ({ "@type": "Question", name: f.q, acceptedAnswer: { "@type": "Answer", text: f.a } })),
      }} />

      <h1>{page.h1}</h1>
      <p className="lead">{page.intro[0]}</p>

      <Tool preset={page.preset} />

      {/* Ads sit below the tool and between content blocks, never next to its buttons. */}
      <AdSlot slot="content" />

      <section>
        <h2>How it works</h2>
        <ol>{page.steps.map((s) => <li key={s}>{s}</li>)}</ol>
        {page.intro.slice(1).map((p) => <p key={p}>{p}</p>)}
      </section>

      <section>
        <h2>Tips for the best result</h2>
        <ul>{page.tips.map((t) => <li key={t}>{t}</li>)}</ul>
      </section>

      <AdSlot slot="bottom" />

      <section>
        <h2>Frequently asked questions</h2>
        {page.faq.map((f) => (
          <details key={f.q}>
            <summary>{f.q}</summary>
            <p>{f.a}</p>
          </details>
        ))}
      </section>
    </>
  );
}

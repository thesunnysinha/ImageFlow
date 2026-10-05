import type { Metadata } from "next";
import { CONTACT_EMAIL, SITE_NAME } from "../../lib/site";

export const metadata: Metadata = {
  title: `Privacy Policy | ${SITE_NAME}`,
  description: `How ${SITE_NAME} handles your images and data, and how advertising works on this site.`,
  alternates: { canonical: "/privacy" },
};

// NOTE: a starting point, not legal advice. Have it reviewed for your jurisdiction before you publish.
export default function Privacy() {
  return (
    <>
      <h1>Privacy policy</h1>
      <p className="muted">Last updated: October 2026</p>

      <h2>Your images</h2>
      <p>
        The compression tools on this site run entirely in your web browser. The images you choose are processed on your own
        device and are <strong>not uploaded to our servers</strong>. We do not see, store or share them.
      </p>

      <h2>Information we collect</h2>
      <p>
        We do not ask you to create an account and we do not collect personal information through the tools. Like most
        websites, our hosting provider may keep standard server logs (such as IP address, browser type and the pages
        requested) for security and to keep the service running.
      </p>

      <h2>Advertising</h2>
      <p>
        This site is supported by advertising from Google AdSense. Google and its partners may use cookies and similar
        technologies to show ads, including ads based on your previous visits to this or other websites, and to measure
        how ads perform. In the European Economic Area, the United Kingdom and Switzerland, ads that use cookies or
        personal data are shown only if you agree in the consent message.
      </p>
      <ul>
        <li>You can change your choice at any time using the privacy settings link provided in the consent message, when shown.</li>
        <li>
          You can manage personalised advertising in your Google account at{" "}
          <a href="https://adssettings.google.com" rel="noopener noreferrer">adssettings.google.com</a>, or opt out of
          personalised ads from many providers at <a href="https://www.aboutads.info" rel="noopener noreferrer">aboutads.info</a>.
        </li>
        <li>
          More about how Google uses data from sites that use its services:{" "}
          <a href="https://policies.google.com/technologies/partner-sites" rel="noopener noreferrer">policies.google.com/technologies/partner-sites</a>.
        </li>
      </ul>

      <h2>Cookies</h2>
      <p>
        The tools themselves do not set cookies. Cookies are set only by our advertising partners, as described above, and
        by your browser if you choose to store settings.
      </p>

      <h2>Children</h2>
      <p>This site is not directed at children under 13 and we do not knowingly collect information from them.</p>

      <h2>Changes</h2>
      <p>We may update this policy. The date at the top shows when it last changed.</p>

      {CONTACT_EMAIL && (
        <>
          <h2>Contact</h2>
          <p>Questions about this policy: <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a></p>
        </>
      )}
    </>
  );
}

import type { Metadata } from "next";
import Link from "next/link";
import PageContent from "../components/PageContent";
import { PAGES } from "../lib/pages";

const home = {
  h1: "Free online image compressor",
  description: "Compress JPG, PNG and WebP images online for free. No sign-up, and your pictures never leave your device.",
  preset: {},
  intro: [
    "Make your photos and graphics smaller in seconds. Drop your images below, choose how small you want them, and download the result. It works for JPG, PNG and WebP, handles up to 20 images at once, and never uploads your pictures: the compression happens in your browser.",
    "Smaller images load faster on websites, fit into email attachments and pass the upload limits of forms and applications. Pick the tool that matches what you need from the list at the bottom of the page.",
  ],
  steps: [
    "Drop your images onto the box, or choose them from your device.",
    "Choose the format and quality, or type a maximum size in KB to get the best quality that fits.",
    "Press Compress and download the images one by one or all together as a ZIP.",
  ],
  tips: [
    "For photos, JPG or WebP at 70 to 80 percent quality is usually indistinguishable from the original and far smaller.",
    "For screenshots and logos with flat colours, PNG with reduced colours works well. WebP is smaller still.",
    "Resizing the longest side to what you really need often saves more space than lowering the quality.",
  ],
  faq: [
    { q: "Is it free?", a: "Yes. There is no sign-up, no watermark and no limit on how often you use it. The site is supported by advertising." },
    { q: "Are my images uploaded to a server?", a: "No. The images are compressed inside your browser on your device and are never sent to us." },
    { q: "Which formats are supported?", a: "JPG (JPEG), PNG and WebP, as input and as output. Photos from some phones are in HEIC format, which browsers cannot read yet: change the camera setting to “Most compatible” or share the photo as JPG." },
  ],
};

export const metadata: Metadata = {
  title: "Free Online Image Compressor: JPG, PNG, WebP | ImageFlow",
  description: home.description,
  alternates: { canonical: "/" },
};

export default function Home() {
  return (
    <>
      <PageContent page={home} path="/" />
      <section>
        <h2>More image tools</h2>
        <ul className="tool-list">
          {PAGES.map((p) => (
            <li key={p.slug}>
              <Link href={`/${p.slug}`}>{p.h1}</Link>
              <span className="muted small"> — {p.description}</span>
            </li>
          ))}
        </ul>
      </section>
    </>
  );
}

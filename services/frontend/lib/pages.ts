import type { Preset } from "../components/Tool";

export interface ToolPage {
  slug: string;
  title: string; // <title>, keep under about 60 characters
  h1: string;
  description: string; // meta description, about 150 characters
  intro: string[];
  steps: string[];
  tips: string[];
  faq: { q: string; a: string }[];
  preset: Preset;
}

export const PAGES: ToolPage[] = [
  {
    slug: "compress-jpg",
    title: "Compress JPG Images Online, Free and Private | ImageFlow",
    h1: "Compress JPG images online",
    description: "Make JPG photos much smaller without visible quality loss. Free, no sign-up, and your pictures never leave your device.",
    preset: { format: "image/jpeg", quality: 0.7 },
    intro: [
      "Photos straight from a phone or camera are often 3 to 8 MB each, which is far more than a website, an email or a form upload needs. This tool re-encodes your JPG files at a lower quality setting, which usually cuts the size by 50 to 80 percent while the picture still looks the same on a normal screen.",
      "Everything runs in your browser. The files are not uploaded to a server, so it is fast, works on a slow connection once the page has loaded, and is safe for personal photos and documents.",
    ],
    steps: [
      "Drop your JPG files onto the box, or choose them from your device.",
      "Leave the quality at 70%, or move the slider: higher keeps more detail, lower makes the file smaller.",
      "Press Compress, then download each image, or all of them as one ZIP file.",
    ],
    tips: [
      "70 to 80 percent is the sweet spot for photos. Below 50 percent you will start to see blocky areas in skies and skin.",
      "If you only need the picture for a screen, set the longest side to 1920 px. Resizing saves far more space than quality alone.",
      "Compress the original, not a copy that was already compressed. Repeated compression slowly degrades the image.",
    ],
    faq: [
      { q: "Does compressing reduce the quality?", a: "JPG is a lossy format, so any re-encoding discards some detail. At 70 to 80 percent the difference is rarely visible. The result also never ends up larger than your original: if re-encoding would not help, the original is kept." },
      { q: "Are my photos uploaded anywhere?", a: "No. The compression happens inside your browser on your own device. Nothing is sent to our servers." },
      { q: "Is there a size limit?", a: "You can add up to 20 images at a time, up to 25 MB each." },
      { q: "Will it keep the date and location in the photo?", a: "No. Re-encoding drops the hidden metadata such as GPS location and camera details, which also makes the files smaller and more private." },
    ],
  },
  {
    slug: "compress-png",
    title: "Compress PNG Images Online, Free and Private | ImageFlow",
    h1: "Compress PNG images online",
    description: "Shrink PNG screenshots and graphics by reducing colours, keeping transparency. Free, no sign-up, processed on your device.",
    preset: { format: "image/png", quality: 0.7 },
    intro: [
      "PNG files are lossless, so they are large by nature. The biggest saving comes from reducing the number of colours, which is what this tool does: it converts the image to a smaller palette, keeps transparency, and typically makes screenshots and logos 50 to 70 percent smaller.",
      "Photos are a poor fit for PNG. If your file is a photograph, try the WebP or JPG options instead and you will usually save much more.",
    ],
    steps: [
      "Drop your PNG files onto the box.",
      "Pick a quality: 80% and above keeps 256 colours, lower values use fewer colours and give smaller files.",
      "Press Compress and download the results.",
    ],
    tips: [
      "Flat graphics, screenshots and icons compress very well. Gradients may show banding at low quality.",
      "Need it smaller still? Choose WebP as the format: it supports transparency and is usually far smaller than PNG.",
      "Set a smaller longest side if the image is bigger than where you will use it.",
    ],
    faq: [
      { q: "Does it keep transparency?", a: "Yes. PNG output keeps the transparent areas of your image." },
      { q: "Why did my PNG hardly get smaller?", a: "Photos and images with many smooth gradients contain a lot of colour information. Try WebP or JPG for those, or lower the quality." },
      { q: "Is it really lossless?", a: "At 95 percent quality it is lossless. Below that the tool reduces the number of colours, which is a lossy step that is usually invisible for screenshots and graphics." },
      { q: "Are my files uploaded?", a: "No. Everything is processed in your browser on your device." },
    ],
  },
  {
    slug: "compress-image-to-100kb",
    title: "Compress Image to 100 KB Online | ImageFlow",
    h1: "Compress an image to 100 KB",
    description: "Get a JPG under 100 KB for forms, applications and uploads. The tool finds the best quality that fits. Free and private.",
    preset: { format: "image/jpeg", targetKB: 100 },
    intro: [
      "Many online forms, job portals and government sites reject pictures bigger than 100 KB. This tool searches for the highest quality that still fits under the limit, so you get the best-looking image that will be accepted instead of guessing a quality number.",
      "The compression runs on your device, so passport photos, signatures and documents are not uploaded anywhere.",
    ],
    steps: [
      "Drop the image onto the box. The limit is already set to 100 KB.",
      "Press Compress. If the result is still too large, choose a smaller longest side such as 800 px and press Compress again.",
      "Download the file and upload it to the form.",
    ],
    tips: [
      "Large photos need resizing as well: a 4000 px photo will not reach 100 KB without visible damage, but at 1280 px it will.",
      "Use JPG for photos and signatures on white. It is the format nearly every form accepts.",
      "Change the Max size box if the form asks for a different limit, for example 50 KB or 150 KB.",
    ],
    faq: [
      { q: "What if it says the image cannot get smaller?", a: "The picture is too large for the limit at any quality. Pick a smaller longest side (for example 1280 or 800 px) and compress again." },
      { q: "Will the result be exactly 100 KB?", a: "It will be at or just under 100 KB, using the highest quality that fits." },
      { q: "Is my photo or ID uploaded?", a: "No. Everything happens in your browser, nothing is sent to a server." },
    ],
  },
  {
    slug: "compress-image-to-200kb",
    title: "Compress Image to 200 KB Online | ImageFlow",
    h1: "Compress an image to 200 KB",
    description: "Reduce a photo to under 200 KB for email, forms and websites with the best quality that fits. Free, private, no sign-up.",
    preset: { format: "image/jpeg", targetKB: 200 },
    intro: [
      "200 KB is a common limit for profile pictures, application forms and web pages that should load quickly. This tool finds the highest JPG quality that keeps your image under that size.",
      "Your files stay on your device: the work is done by your browser, not by a server.",
    ],
    steps: [
      "Drop your image onto the box. The limit is preset to 200 KB.",
      "Press Compress. If the result is larger than the limit, reduce the longest side and try again.",
      "Download the compressed image.",
    ],
    tips: [
      "For websites, 1920 px on the longest side is plenty for full-width photos.",
      "You can change the Max size box to any limit between 10 KB and 20,000 KB.",
      "Compress several images at once and download them together as a ZIP.",
    ],
    faq: [
      { q: "Does it work for PNG files?", a: "The preset converts to JPG, which is the smallest option for photos. Choose PNG or WebP in the Format box if you need transparency." },
      { q: "How many images can I compress at once?", a: "Up to 20 at a time, up to 25 MB each." },
      { q: "Are my images uploaded?", a: "No. Compression happens locally in your browser." },
    ],
  },
  {
    slug: "convert-png-to-webp",
    title: "Convert PNG to WebP Online, Free and Private | ImageFlow",
    h1: "Convert PNG to WebP",
    description: "Turn PNG images into much smaller WebP files and keep transparency. Free, no sign-up, converted on your device.",
    preset: { format: "image/webp", quality: 0.8 },
    intro: [
      "WebP is a modern image format that is usually 25 to 70 percent smaller than PNG while keeping transparency, which makes web pages load faster. All current browsers can display it.",
      "The conversion runs in your browser, so your images are never uploaded.",
    ],
    steps: [
      "Drop your PNG files onto the box. The format is already set to WebP.",
      "Press Compress and check the sizes.",
      "Download the WebP files, individually or as a ZIP.",
    ],
    tips: [
      "80 percent is a good default. Use 90 or more for graphics with fine text or sharp edges.",
      "Some older apps and email clients cannot open WebP. Keep the PNG if you need to share it with those.",
    ],
    faq: [
      { q: "Does WebP keep transparency?", a: "Yes, WebP supports transparent areas just like PNG." },
      { q: "Is WebP supported everywhere?", a: "All current versions of Chrome, Firefox, Safari and Edge display WebP. Some older software does not, so keep the original if you are unsure." },
      { q: "Is the conversion lossless?", a: "No, this tool uses lossy WebP, which is what makes the files small. At 90 percent and above the difference is hard to see." },
    ],
  },
  {
    slug: "convert-jpg-to-webp",
    title: "Convert JPG to WebP Online, Free and Private | ImageFlow",
    h1: "Convert JPG to WebP",
    description: "Convert JPG photos to WebP and make them about 25 to 35 percent smaller at the same look. Free, no sign-up, on your device.",
    preset: { format: "image/webp", quality: 0.8 },
    intro: [
      "At the same visual quality, WebP photos are typically about a quarter to a third smaller than JPG. Switching your website images to WebP is one of the easiest ways to improve loading speed.",
      "The conversion happens in your browser. Your photos are not uploaded to any server.",
    ],
    steps: [
      "Drop your JPG files onto the box. The format is set to WebP.",
      "Press Compress, and compare the before and after sizes.",
      "Download the results, one by one or all together in a ZIP file.",
    ],
    tips: [
      "Start at 80 percent. If an image shows artefacts, raise it a little.",
      "Keep your original JPG files as a backup: converting between lossy formats loses a little detail each time.",
    ],
    faq: [
      { q: "Will the picture look different?", a: "At 80 percent or higher it should look the same on screen. You can compare the result before you use it." },
      { q: "Can I convert many files at once?", a: "Yes, up to 20 at a time, and download them together as a ZIP." },
      { q: "Are my photos uploaded?", a: "No. The conversion is done locally in your browser." },
    ],
  },
];

export const PAGE_BY_SLUG = new Map(PAGES.map((p) => [p.slug, p]));

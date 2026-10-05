import type { Metadata } from "next";
import { notFound } from "next/navigation";
import PageContent from "../../components/PageContent";
import { PAGES, PAGE_BY_SLUG } from "../../lib/pages";

export const dynamicParams = false;

export function generateStaticParams() {
  return PAGES.map((p) => ({ slug: p.slug }));
}

export async function generateMetadata({ params }: { params: Promise<{ slug: string }> }): Promise<Metadata> {
  const page = PAGE_BY_SLUG.get((await params).slug);
  if (!page) return {};
  return { title: page.title, description: page.description, alternates: { canonical: `/${page.slug}` } };
}

export default async function ToolPage({ params }: { params: Promise<{ slug: string }> }) {
  const page = PAGE_BY_SLUG.get((await params).slug);
  if (!page) notFound();
  return <PageContent page={page} path={`/${page.slug}`} />;
}

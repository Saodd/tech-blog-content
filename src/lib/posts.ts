import type { MarkdownHeading, MarkdownInstance } from 'astro';
import { z } from 'astro/zod';

const frontmatterSchema = z.object({
	title: z.string(),
	pubDate: z.string(),
	updateDate: z.string().optional(),
	description: z.string().default(''),
	tags: z.array(z.string()).default([]),
	keywords: z.string().default(''),
});

type Frontmatter = z.infer<typeof frontmatterSchema>;

export type Post = {
	slug: string;
	title: string;
	pubDate: string;
	updateDate: string;
	description: string;
	tags: string[];
	keywords: string;
	headings: MarkdownHeading[];
	Content: MarkdownInstance<Frontmatter>['Content'];
};

export type PostListItem = Pick<Post, 'slug' | 'title' | 'pubDate' | 'description' | 'tags'>;

// 筛选的年份手动调整。
// 注意：2022年及之前的文章可能包含本地图片，会导致构建失败，因此排除。
const modules = import.meta.glob<MarkdownInstance<Frontmatter>>('../../content/{2023,2024,2025,2026}/*.md', {
	eager: true,
});

function parsePost(filePath: string, post: MarkdownInstance<Frontmatter>): Post {
	const slug = filePath.replace('../../content/', '').replace(/\.md$/, '');
	const frontmatter = frontmatterSchema.parse(post.frontmatter);

	return {
		slug,
		title: frontmatter.title ?? slug,
		pubDate: frontmatter.pubDate ?? '',
		updateDate: frontmatter.updateDate ?? '',
		description: frontmatter.description ??  '',
		tags: frontmatter.tags,
		keywords: frontmatter.keywords,
		headings: post.getHeadings(),
		Content: post.Content,
	};
}

export function getPosts() {
	return Object.entries(modules)
		.map(([filePath, post]) => parsePost(filePath, post))
		.sort((a, b) => b.pubDate.localeCompare(a.pubDate));
}

export function getPostListItems(): PostListItem[] {
	return getPosts().map(({ slug, title, pubDate, description, tags }) => ({
		slug,
		title,
		pubDate,
		description,
		tags,
	}));
}

export function getPost(slug: string) {
	return getPosts().find((post) => post.slug === slug);
}

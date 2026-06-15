// @ts-check

import { defineConfig } from 'astro/config';
import react from '@astrojs/react';
import sitemap from '@astrojs/sitemap';
import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
	site: 'https://lewinblog.com',
	base: '/blog/page/',
	build: {
		format: 'file',
	},
	trailingSlash: 'never',
	integrations: [
		react(),
		sitemap({
			filter: (page) => /^https:\/\/lewinblog\.com\/blog\/page\/20\d{2}\/[^/]+\/?$/.test(page),
			namespaces: {
				news: false,
				xhtml: false,
				image: false,
				video: false,
			},
		}),
	],
	vite: {
		plugins: [tailwindcss()],
	},
});

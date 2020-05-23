import { getPostListItems } from '../lib/posts';

export function GET() {
	return new Response(JSON.stringify(getPostListItems(), null, 2), {
		headers: {
			'Content-Type': 'application/json; charset=utf-8',
		},
	});
}

import { redirect } from '@sveltejs/kit';
import type { PageServerLoadEvent } from './$types';

export const ssr = false;

export function load({ cookies, params, url }: PageServerLoadEvent) {
	const jwt = url.searchParams.get('jwt') || '';
	cookies.set('jwt', jwt, {
		secure: false,
		httpOnly: false,
		domain: process.env.COMMON_DOMAIN,
		path: '/'
	});
	throw redirect(302, '/' + params.slug);
}

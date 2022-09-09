import type { PageServerLoadEvent } from './$types';

// TODO: Verify environment variables are set at startup

export function load({ params }: PageServerLoadEvent): { id: string; domain: string } {
	const domain = process.env.CDN_DOMAIN || '';
	return {
		id: params.id,
		domain
	};
}

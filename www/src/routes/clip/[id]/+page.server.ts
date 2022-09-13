import type { PageServerLoadEvent } from './$types';

export function load({ params }: PageServerLoadEvent): {
	id: string;
	domain: string;
	mail: string;
} {
	const domain = process.env.CDN_DOMAIN || '';
	const mail = process.env.DEV_MAIL_ADDR || '';
	return {
		id: params.id,
		domain,
		mail
	};
}

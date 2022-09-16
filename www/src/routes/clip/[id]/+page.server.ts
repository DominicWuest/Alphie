import type { PageServerLoadEvent } from './$types';

export function load({ params }: PageServerLoadEvent): {
	id: string;
	domain: string;
	authenticationUrl: string;
	authorizationUrl: string;
	mail: string;
} {
	const domain = process.env.CDN_DOMAIN || '';
	const mail = process.env.DEV_MAIL_ADDR || '';
	const authenticationUrl = process.env.STUDENT_AUTH_PATH || '';
	const authorizationUrl = process.env.AUTHORIZATION_URL || '';

	return {
		id: params.id,
		domain,
		authenticationUrl,
		authorizationUrl,
		mail
	};
}

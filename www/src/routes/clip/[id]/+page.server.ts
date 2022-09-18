import type { PageServerLoadEvent } from './$types';

export function load({ params }: PageServerLoadEvent): {
	id: string;
	domain: string;
	authenticationUrl: string;
	authorizationUrl: string;
	mail: string;
	proto: string;
} {
	const domain = process.env.CDN_DOMAIN || '';
	const mail = process.env.DEV_MAIL_ADDR || '';
	const authenticationUrl = process.env.STUDENT_AUTH_PATH || '';
	const authorizationUrl = process.env.AUTHORIZATION_URL || '';
	const proto = process.env.HTTP_PROTO || '';

	return {
		id: params.id,
		domain,
		authenticationUrl,
		authorizationUrl,
		mail,
		proto
	};
}

import type { Handle, RequestEvent } from '@sveltejs/kit';
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import jwt from 'jsonwebtoken';

// TODO: Verify environment variables are set at startup

export const handle: Handle = async function ({ event, resolve }) {
	if (event.url.pathname.startsWith(process.env.STUDENT_AUTH_PATH || '')) {
		return studentAuth(event);
	}
	const res = await resolve(event);
	return res;
};

function studentAuth(event: RequestEvent): Response {
	// Return 200 OK if auth is disabled
	if (!process.env.STUDENT_AUTH_ENABLED) {
		return new Response();
	}
	// The JWT token
	const token = event.cookies.get('jwt') || '';
	console.log(token);
	return new Response('Unauthorized', { status: 401 });
}

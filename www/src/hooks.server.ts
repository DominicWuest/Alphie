import type { Handle, RequestEvent } from '@sveltejs/kit';
import jwt from 'jsonwebtoken';

export const handle: Handle = async function ({ event, resolve }) {
	if (event.url.pathname.startsWith(process.env.STUDENT_AUTH_PATH || '')) {
		return studentAuth(event);
	}
	const res = await resolve(event);
	return res;
};

type tokenProperties = {
	student: boolean;
	exp: number;
};

function studentAuth(event: RequestEvent): Response {
	// Return 200 OK if auth is disabled
	if (!process.env.STUDENT_AUTH_ENABLED) {
		return new Response();
	}
	// The JWT token
	const token = event.cookies.get('jwt') || '';
	if (!token) {
		return new Response('Unauthorized', { status: 401 });
	}

	let response = new Response();

	jwt.verify(token, process.env.JWT_PUBLIC_KEY || '', function (err, decoded) {
		// If token is invalid or expired
		if (err) {
			response = new Response('Unauthorized', { status: 401 });
			return;
		}
		const props: tokenProperties = <tokenProperties>(<unknown>decoded);
		// Ensuring the token is for a student
		if (!props.student) {
			response = new Response('Unauthorized', { status: 401 });
			return;
		}
	});
	return response;
}

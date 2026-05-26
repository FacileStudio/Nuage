import type { Handle } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

const API_URL = env.API_URL || 'http://localhost:4000';

export const handle: Handle = async ({ event, resolve }) => {
	if (event.url.pathname === '/webdav' || event.url.pathname.startsWith('/webdav/')) {
		const target = `${API_URL}${event.url.pathname}${event.url.search}`;

		const headers = new Headers(event.request.headers);
		headers.delete('host');
		headers.delete('connection');

		const init: RequestInit = {
			method: event.request.method,
			headers,
			redirect: 'manual',
		};

		if (event.request.method !== 'GET' && event.request.method !== 'HEAD') {
			init.body = event.request.body;
			// @ts-expect-error — needed for streaming request bodies
			init.duplex = 'half';
		}

		const response = await fetch(target, init);

		const responseHeaders = new Headers(response.headers);
		responseHeaders.delete('transfer-encoding');

		return new Response(response.body, {
			status: response.status,
			statusText: response.statusText,
			headers: responseHeaders,
		});
	}

	return resolve(event);
};

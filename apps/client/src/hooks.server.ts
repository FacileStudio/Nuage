import type { Handle } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import { STRIPPED_REQUEST_HEADERS, proxyResponseHeaders } from '$lib/server/proxy';

const API_URL = env.API_URL || 'http://localhost:4000';

export const handle: Handle = async ({ event, resolve }) => {
	if (event.url.pathname === '/webdav' || event.url.pathname.startsWith('/webdav/')) {
		const target = `${API_URL}${event.url.pathname}${event.url.search}`;

		const headers = new Headers(event.request.headers);
		for (const header of STRIPPED_REQUEST_HEADERS) {
			headers.delete(header);
		}
		headers.set('X-Forwarded-For', event.getClientAddress());
		headers.set('X-Forwarded-Proto', event.url.protocol.replace(':', ''));

		const init: RequestInit = {
			method: event.request.method,
			headers,
			redirect: 'manual',
			signal: event.request.signal
		};

		if (event.request.method !== 'GET' && event.request.method !== 'HEAD') {
			init.body = event.request.body;
			// @ts-expect-error — needed for streaming request bodies
			init.duplex = 'half';
		}

		let response: Response;
		try {
			response = await fetch(target, init);
		} catch {
			if (event.request.signal.aborted) {
				return new Response(null, { status: 499 });
			}
			return new Response('The API is unreachable.', { status: 502 });
		}

		const responseHeaders = proxyResponseHeaders(response.headers);

		return new Response(response.body, {
			status: response.status,
			statusText: response.statusText,
			headers: responseHeaders
		});
	}

	return resolve(event);
};

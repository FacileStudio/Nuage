import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';
import { STRIPPED_REQUEST_HEADERS, proxyResponseHeaders } from '$lib/server/proxy';

const API_URL = env.API_URL || 'http://localhost:4000';

export const fallback: RequestHandler = async ({ request, params, url }) => {
	const target = `${API_URL}/${params.path}${url.search}`;

	const headers = new Headers(request.headers);
	for (const header of STRIPPED_REQUEST_HEADERS) {
		headers.delete(header);
	}
	headers.set('X-Forwarded-Prefix', '/api');

	const init: RequestInit = {
		method: request.method,
		headers,
		redirect: 'manual',
	};

	if (request.method !== 'GET' && request.method !== 'HEAD') {
		init.body = request.body;
		// @ts-expect-error — needed for streaming request bodies
		init.duplex = 'half';
	}

	const response = await fetch(target, init);

	const responseHeaders = proxyResponseHeaders(response.headers);

	return new Response(response.body, {
		status: response.status,
		statusText: response.statusText,
		headers: responseHeaders,
	});
};

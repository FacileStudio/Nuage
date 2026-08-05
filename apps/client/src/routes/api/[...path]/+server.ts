import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';
import { STRIPPED_REQUEST_HEADERS, proxyResponseHeaders } from '$lib/server/proxy';

const API_URL = env.API_URL || 'http://localhost:4000';

export const fallback: RequestHandler = async ({ request, params, url, getClientAddress }) => {
	const target = `${API_URL}/${params.path}${url.search}`;

	const headers = new Headers(request.headers);
	for (const header of STRIPPED_REQUEST_HEADERS) {
		headers.delete(header);
	}
	headers.set('X-Forwarded-Prefix', '/api');
	headers.set('X-Forwarded-For', getClientAddress());
	headers.set('X-Forwarded-Proto', url.protocol.replace(':', ''));

	const init: RequestInit = {
		method: request.method,
		headers,
		redirect: 'manual',
		signal: request.signal
	};

	if (request.method !== 'GET' && request.method !== 'HEAD') {
		init.body = request.body;
		// @ts-expect-error — needed for streaming request bodies
		init.duplex = 'half';
	}

	let response: Response;
	try {
		response = await fetch(target, init);
	} catch (error) {
		if (request.signal.aborted) {
			return new Response(null, { status: 499 });
		}
		return new Response(
			JSON.stringify({ error: { code: 'upstream_unavailable', message: 'The API is unreachable.' } }),
			{ status: 502, headers: { 'Content-Type': 'application/json' } }
		);
	}

	const responseHeaders = proxyResponseHeaders(response.headers);

	return new Response(response.body, {
		status: response.status,
		statusText: response.statusText,
		headers: responseHeaders
	});
};

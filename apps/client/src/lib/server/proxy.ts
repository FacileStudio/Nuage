export const STRIPPED_REQUEST_HEADERS = [
	'host',
	'connection',
	'keep-alive',
	'te',
	'trailer',
	'transfer-encoding',
	'upgrade',
	'x-forwarded-for',
	'x-forwarded-host',
	'x-forwarded-proto',
	'x-real-ip'
];

export function proxyResponseHeaders(source: Headers): Headers {
	const headers = new Headers(source);
	const wasDecoded = headers.has('content-encoding');

	headers.delete('transfer-encoding');
	headers.delete('content-encoding');
	if (wasDecoded) headers.delete('content-length');

	return headers;
}

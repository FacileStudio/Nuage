import { afterEach, describe, expect, test } from 'bun:test';
import { ApiError, backend, isAuthError } from './backend';

const realFetch = globalThis.fetch;

function respondWith(status: number) {
	const seen: { headers: Headers; credentials?: RequestCredentials }[] = [];
	globalThis.fetch = (async (_input: unknown, init: RequestInit = {}) => {
		seen.push({ headers: new Headers(init.headers), credentials: init.credentials });
		return new Response(status === 200 ? '{"user":{}}' : '', { status });
	}) as typeof fetch;
	return seen;
}

afterEach(() => {
	globalThis.fetch = realFetch;
});

describe('apiFetch', () => {
	test('sends the CSRF header and the cookie, which is all an SSO session has', async () => {
		const seen = respondWith(200);
		await backend.me();
		expect(seen[0].headers.get('X-Facile-CSRF')).toBe('1');
		expect(seen[0].credentials).toBe('same-origin');
		expect(seen[0].headers.has('Authorization')).toBe(false);
	});

	test('still sends the bearer token a local login stored', async () => {
		const seen = respondWith(200);
		await backend.me('abc');
		expect(seen[0].headers.get('Authorization')).toBe('Bearer abc');
	});

	test('a refusal is an auth error, so the route guard signs the user out', async () => {
		respondWith(401);
		const error = await backend.me().catch((e) => e);
		expect(error).toBeInstanceOf(ApiError);
		expect(isAuthError(error)).toBe(true);
	});

	test('a server failure is not, so a blip does not sign anyone out', async () => {
		respondWith(500);
		expect(isAuthError(await backend.me().catch((e) => e))).toBe(false);
		expect(isAuthError(new Error('network'))).toBe(false);
	});
});

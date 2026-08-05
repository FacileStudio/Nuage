import { describe, expect, test } from 'bun:test';
import { proxyResponseHeaders } from './proxy';

describe('proxyResponseHeaders', () => {
	test('strips content-encoding and the now-wrong content-length after undici decodes the body', () => {
		const headers = proxyResponseHeaders(
			new Headers({
				'content-type': 'application/json',
				'content-encoding': 'gzip',
				'content-length': '128',
				'transfer-encoding': 'chunked'
			})
		);

		expect(headers.get('content-encoding')).toBeNull();
		expect(headers.get('content-length')).toBeNull();
		expect(headers.get('transfer-encoding')).toBeNull();
		expect(headers.get('content-type')).toBe('application/json');
	});

	test('keeps content-length and content-range on uncompressed partial content', () => {
		const headers = proxyResponseHeaders(
			new Headers({
				'content-length': '100',
				'content-range': 'bytes 0-99/4096',
				'accept-ranges': 'bytes'
			})
		);

		expect(headers.get('content-length')).toBe('100');
		expect(headers.get('content-range')).toBe('bytes 0-99/4096');
		expect(headers.get('accept-ranges')).toBe('bytes');
	});
});

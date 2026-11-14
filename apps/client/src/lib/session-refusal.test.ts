import { expect, test } from 'bun:test';
import { ApiError } from '@facile/client';
import { OWN_SESSION_REFUSAL, refusesOwnSession } from './session-refusal';

test('refusesOwnSession matches porte refusing a session no browser sign-in opened', () => {
	expect(refusesOwnSession(new ApiError(403, 'permission_denied', OWN_SESSION_REFUSAL))).toBe(true);
});

test('refusesOwnSession leaves any other 403 to its own message', () => {
	expect(refusesOwnSession(new ApiError(403, 'permission_denied', 'access denied'))).toBe(false);
	expect(refusesOwnSession(new ApiError(403, 'forbidden', OWN_SESSION_REFUSAL))).toBe(false);
	expect(refusesOwnSession(new ApiError(401, 'permission_denied', OWN_SESSION_REFUSAL))).toBe(false);
	expect(refusesOwnSession(null)).toBe(false);
});

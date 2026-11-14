/** porte's `mcpauth.ErrNotOwnSession` text, the same as tronc's `mcp.ErrDelegated`. */
export const OWN_SESSION_REFUSAL = "only the person's own session can confirm actions";

/**
 * Reports whether err is porte's refusal of a session no browser sign-in opened: a 403
 * `permission_denied` carrying exactly porte's text. Signing in again fixes only that one.
 * Any other 403, such as a permission the confirmed run checked, keeps its own message.
 */
export function refusesOwnSession(err: unknown): boolean {
	const refusal = err as { status?: unknown; code?: unknown; message?: unknown } | null;
	return refusal?.status === 403 && refusal.code === 'permission_denied' && refusal.message === OWN_SESSION_REFUSAL;
}

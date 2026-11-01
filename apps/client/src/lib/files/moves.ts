import { backend } from '$lib/backend';
import type { Entry } from './selection.svelte';

/**
 * Moves every entry into `target` (`null` is the root) and returns the ones that moved, so
 * an undo only tries to bring back what actually left.
 */
export async function moveEntries(
	token: string,
	entries: Entry[],
	target: number | null
): Promise<{ moved: Entry[]; failed: number }> {
	const results = await Promise.allSettled(
		entries.map((e) => backend.moveItem(token, e.type, e.id, target))
	);
	const moved = entries.filter((_, i) => results[i].status === 'fulfilled');
	return { moved, failed: entries.length - moved.length };
}

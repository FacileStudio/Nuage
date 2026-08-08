export type ParentLink = { id: number; parent_id: number | null };

/**
 * True when `folderId` is `ancestorId` or sits anywhere below it.
 *
 * This is what stops a folder being dropped into its own subtree, which would detach the
 * branch from the tree and strand every file under it. The `seen` guard is not paranoia: the
 * chain is rebuilt from whatever the API returned, and a cycle there would otherwise spin
 * this loop forever and hang the tab rather than surface the bad data.
 *
 * `folders` only needs to contain the links along the path being walked; an unknown parent
 * ends the walk, which fails open — the drop is allowed and the server has the final say.
 */
export function isSelfOrDescendant(
	folderId: number,
	ancestorId: number,
	folders: ParentLink[]
): boolean {
	const seen = new Set<number>();
	let cursor: number | null = folderId;

	while (cursor != null && !seen.has(cursor)) {
		if (cursor === ancestorId) return true;
		seen.add(cursor);
		cursor = folders.find((f) => f.id === cursor)?.parent_id ?? null;
	}
	return false;
}

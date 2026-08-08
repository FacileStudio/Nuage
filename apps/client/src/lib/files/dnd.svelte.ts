import type { Folder } from '$lib/backend';
import type { Entry, ItemType } from './selection.svelte';
import { isSelfOrDescendant } from './tree';

/** Custom MIME so a drag from inside the browser is never mistaken for a file drop. */
export const MOVE_TYPE = 'application/x-nuage-move';

/** `null` is the root of the tree; `'root'` is the breadcrumb standing in for it. */
export type DropTarget = number | 'root' | null;

export const isInternalDrag = (e: DragEvent): boolean =>
	e.dataTransfer?.types.includes(MOVE_TYPE) ?? false;

/**
 * Drag-to-move state for the file browser.
 *
 * The counter map is the load-bearing part: `dragenter`/`dragleave` fire every time the
 * pointer crosses a *child* element, so a boolean flag flickers the highlight the whole time
 * the pointer is over a row. Counting entries and exits per target is the only thing that
 * survives nested markup — the same rule muse's `Dropzone` documents.
 */
export class DragMove {
	item = $state<Entry | null>(null);
	target = $state<DropTarget>(null);

	#depth: Record<string, number> = {};

	start(e: DragEvent, type: ItemType, id: number, name: string): void {
		this.item = { type, id };
		if (!e.dataTransfer) return;
		e.dataTransfer.effectAllowed = 'move';
		e.dataTransfer.setData(MOVE_TYPE, JSON.stringify({ type, id }));
		e.dataTransfer.setData('text/plain', name);
	}

	end(): void {
		this.item = null;
		this.target = null;
		this.#depth = {};
	}

	enter(key: string, target: DropTarget): void {
		this.#depth[key] = (this.#depth[key] ?? 0) + 1;
		this.target = target;
	}

	leave(key: string, target: DropTarget): void {
		this.#depth[key] = (this.#depth[key] ?? 0) - 1;
		if (this.#depth[key] <= 0 && this.target === target) this.target = null;
	}

	/**
	 * A folder cannot be dropped into itself or into anything below it — that detaches the
	 * subtree and strands every file under it. The walk itself is `tree.ts`, so it can be
	 * tested without a rune runtime.
	 */
	wouldOrphan(targetFolderId: number, folders: Folder[]): boolean {
		if (this.item?.type !== 'folder') return false;
		return isSelfOrDescendant(targetFolderId, this.item.id, folders);
	}

	/** True when this folder is a legal drop for whatever is currently being dragged. */
	canDropOn(targetFolderId: number, folders: Folder[]): boolean {
		return this.item !== null && !this.wouldOrphan(targetFolderId, folders);
	}
}

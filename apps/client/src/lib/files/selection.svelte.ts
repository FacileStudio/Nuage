export type ItemType = 'file' | 'folder';
export type Entry = { type: ItemType; id: number };

export const itemKey = (type: ItemType, id: number): string => `${type}:${id}`;

export function parseKey(key: string): Entry {
	const [type, id] = key.split(':');
	return { type: type as ItemType, id: Number(id) };
}

/**
 * Multi-select for the file browser: click to replace, ctrl/cmd-click to add, shift-click to
 * extend from the last anchor.
 *
 * It owns the anchor index, which is the part that kept going wrong when this logic was
 * inlined three times in the page — the checkbox path, the row-click path and the grid-tile
 * path each maintained their own copy of `lastClickedIndex`, so a shift-click after a
 * checkbox click extended from whichever one had run last.
 */
export class Selection {
	keys = $state<string[]>([]);
	/** In select mode a plain click selects instead of opening. Driven by the toolbar. */
	mode = $state(false);

	#anchor = -1;
	#entries: () => Entry[];
	#lookup = $derived(new Set(this.keys));

	constructor(entries: () => Entry[]) {
		this.#entries = entries;
	}

	get count(): number {
		return this.keys.length;
	}

	get all(): boolean {
		const total = this.#entries().length;
		return total > 0 && this.keys.length === total;
	}

	has(type: ItemType, id: number): boolean {
		return this.#lookup.has(itemKey(type, id));
	}

	/** The keys, resolved back into typed ids. */
	entries(): Entry[] {
		return this.keys.map(parseKey);
	}

	#range(from: number, to: number, base: string[]): string[] {
		const entries = this.#entries();
		const next = [...base];
		const seen = new Set(next);
		for (let i = Math.min(from, to); i <= Math.max(from, to); i += 1) {
			const entry = entries[i];
			if (!entry) continue;
			const key = itemKey(entry.type, entry.id);
			if (!seen.has(key)) {
				seen.add(key);
				next.push(key);
			}
		}
		return next;
	}

	toggle(type: ItemType, id: number, index: number): void {
		const key = itemKey(type, id);
		this.keys = this.#lookup.has(key) ? this.keys.filter((k) => k !== key) : [...this.keys, key];
		this.#anchor = index;
	}

	extendTo(type: ItemType, id: number, index: number, additive: boolean): void {
		if (this.#anchor < 0) {
			this.toggle(type, id, index);
			return;
		}
		this.keys = this.#range(this.#anchor, index, additive ? this.keys : []);
	}

	replaceWith(type: ItemType, id: number, index: number): void {
		this.keys = [itemKey(type, id)];
		this.#anchor = index;
	}

	selectAll(): void {
		this.keys = this.#entries().map((e) => itemKey(e.type, e.id));
	}

	clear(): void {
		this.keys = [];
		this.#anchor = -1;
	}

	enter(): void {
		this.mode = true;
	}

	exit(): void {
		this.mode = false;
		this.clear();
	}
}

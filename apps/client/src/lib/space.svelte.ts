import type { Space } from '$lib/backend';

const SPACE_KEY = 'nuage.space_id';

let currentSpace = $state<Space | null>(null);

export function getSpaceStore() {
	return {
		get current() { return currentSpace; },
		get id() { return currentSpace?.id ?? null; },

		set(space: Space | null) {
			currentSpace = space;
			if (space) {
				localStorage.setItem(SPACE_KEY, String(space.id));
			} else {
				localStorage.removeItem(SPACE_KEY);
			}
		},

		getSavedId(): number | null {
			const raw = localStorage.getItem(SPACE_KEY);
			if (!raw) return null;
			const parsed = parseInt(raw, 10);
			return isNaN(parsed) ? null : parsed;
		},

		clear() {
			currentSpace = null;
			localStorage.removeItem(SPACE_KEY);
		}
	};
}

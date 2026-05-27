export type UndoAction = {
	label: string;
	execute: () => Promise<void>;
};

let current = $state<UndoAction | null>(null);
let timer: ReturnType<typeof setTimeout> | null = null;

const UNDO_WINDOW_MS = 6000;

export function pushUndo(action: UndoAction) {
	if (timer) clearTimeout(timer);
	current = action;
	timer = setTimeout(() => {
		current = null;
		timer = null;
	}, UNDO_WINDOW_MS);
}

export async function undoLast() {
	if (!current) return;
	const action = current;
	dismissUndo();
	await action.execute();
}

export function dismissUndo() {
	if (timer) clearTimeout(timer);
	current = null;
	timer = null;
}

export function getUndo() {
	return {
		get current() { return current; }
	};
}

import { icons, toast } from '@facile/muse';

export type UndoAction = {
	label: string;
	execute: () => Promise<void>;
};

/**
 * The undo window is the toast's lifetime — muse freezes a toast's countdown while the
 * pointer is on it, so the offer cannot expire on the way to its own button, which is the
 * one thing the hand-rolled banner this replaced got wrong.
 */
const UNDO_WINDOW_MS = 6000;

let current: UndoAction | null = null;
let toastId: string | null = null;

export function pushUndo(action: UndoAction) {
	/* Cleared first: dismissing runs the previous toast's `onDismiss` synchronously, and it
	   would otherwise null out the action we are about to install. */
	const previous = toastId;
	toastId = null;
	if (previous) toast.dismiss(previous);

	current = action;
	toastId = toast.neutral(action.label, {
		icon: icons.remove,
		duration: UNDO_WINDOW_MS,
		action: { label: 'Undo', onClick: () => void undoLast() },
		onDismiss: () => {
			current = null;
			toastId = null;
		}
	});
}

export async function undoLast() {
	const action = current;
	if (!action) return;
	dismissUndo();
	try {
		await action.execute();
	} catch {
		toast.danger('Could not undo that.');
	}
}

export function hasPending() {
	return current !== null;
}

export function dismissUndo() {
	const id = toastId;
	current = null;
	toastId = null;
	if (id) toast.dismiss(id);
}

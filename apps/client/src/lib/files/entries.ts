import type { Folder, NuageFile } from '$lib/backend';
import { fileIcon, icons } from '$lib/icons';
import type { ItemType } from './selection.svelte';

/**
 * One shape for both halves of a listing. The page used to carry four `{#each}` blocks —
 * folders and files, once for the grid and once for the table — that were the same markup
 * with a different icon expression and a `—` where the size went, so a fix to row behaviour
 * had to be made in four places and reliably was not.
 */
export type BrowserEntry = {
	type: ItemType;
	id: number;
	name: string;
	icon: string;
	/** `null` for folders: a folder's stored size is a rollup that goes stale, so it is not shown. */
	size: number | null;
	date: string;
	mime: string;
	isImage: boolean;
	raw: NuageFile | Folder;
};

const folderEntry = (folder: Folder): BrowserEntry => ({
	type: 'folder',
	id: folder.id,
	name: folder.name,
	icon: icons.folder,
	size: null,
	date: folder.created_at,
	mime: '',
	isImage: false,
	raw: folder
});

const fileEntry = (file: NuageFile): BrowserEntry => ({
	type: 'file',
	id: file.id,
	name: file.name,
	icon: fileIcon(file.name, file.mime_type),
	size: file.size,
	date: file.updated_at,
	mime: file.mime_type,
	isImage: file.mime_type.startsWith('image/'),
	raw: file
});

/** Folders first, then files — the order the selection indices are keyed to. */
export function toEntries(folders: Folder[], files: NuageFile[]): BrowserEntry[] {
	return [...folders.map(folderEntry), ...files.map(fileEntry)];
}

export const isPreviewable = (mime: string): boolean =>
	mime.startsWith('image/') ||
	mime.startsWith('video/') ||
	mime.startsWith('audio/') ||
	mime === 'application/pdf';

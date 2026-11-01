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
	isImage: previewKind(file.name, file.mime_type) === 'image',
	raw: file
});

/** Folders first, then files — the order the selection indices are keyed to. */
export function toEntries(folders: Folder[], files: NuageFile[]): BrowserEntry[] {
	return [...folders.map(folderEntry), ...files.map(fileEntry)];
}

/** The slice of a file the viewer needs, so owned files and public shares both fit. */
export type ViewerFile = {
	id: number;
	name: string;
	mime_type: string;
	size: number;
	updated_at?: string;
};

export type PreviewKind = 'image' | 'video' | 'audio' | 'pdf';

const PREVIEW_EXTENSIONS: Record<string, PreviewKind> = {
	jpg: 'image', jpeg: 'image', png: 'image', gif: 'image', webp: 'image', avif: 'image', svg: 'image',
	mp4: 'video', m4v: 'video', mov: 'video', webm: 'video', ogv: 'video',
	mp3: 'audio', m4a: 'audio', aac: 'audio', wav: 'audio', ogg: 'audio', oga: 'audio', opus: 'audio', flac: 'audio',
	pdf: 'pdf'
};

/**
 * Which viewer can show a file, from its MIME type or, when the type is the generic
 * `application/octet-stream` older uploads were stored with, from its extension.
 */
export function previewKind(name: string, mime: string): PreviewKind | null {
	if (mime.startsWith('image/')) return 'image';
	if (mime.startsWith('video/')) return 'video';
	if (mime.startsWith('audio/')) return 'audio';
	if (mime === 'application/pdf') return 'pdf';
	return PREVIEW_EXTENSIONS[name.split('.').pop()?.toLowerCase() ?? ''] ?? null;
}

export const isPreviewable = (file: { name: string; mime_type: string }): boolean =>
	previewKind(file.name, file.mime_type) !== null;

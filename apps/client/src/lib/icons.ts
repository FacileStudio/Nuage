import { icons } from '@facile/muse';
import { t } from '$lib/i18n/index.svelte';

/**
 * Glyphs muse's shared set does not carry because they only mean something in a file
 * manager. Same rules as muse's own map (CHARTE §8): Solar `linear` for chrome, `bold-duotone`
 * reserved for the brand mark — collected here so a call site never inlines a raw string and
 * the pack rule never has to be re-litigated per page.
 */
export const nuage = {
	brand: 'solar:cloud-bold-duotone',
	/* A bare tick, not muse's `icons.check` — that one is a *circled* check, which reads as a
	   status badge rather than a checkbox mark. MDI for the same reason CHARTE §8 gives for
	   plus and close: Solar's is muddy at 14px. */
	tick: 'mdi:check',
	share: 'solar:share-linear',
	shareOff: 'solar:link-broken-linear',
	link: 'solar:link-linear',
	newFolder: 'solar:add-folder-linear',
	folderOpen: 'solar:folder-open-linear',
	rename: 'solar:pen-linear',
	restore: 'solar:restart-linear',
	more: 'solar:menu-dots-linear',
	grid: 'solar:widget-linear',
	list: 'solar:list-linear',
	versions: 'solar:layers-linear',
	fullscreen: 'solar:full-screen-linear',
	camera: 'solar:camera-linear',
	login: 'solar:login-3-linear',
	lock: 'solar:lock-linear',
	memberAdd: 'solar:user-plus-linear',
	team: 'solar:users-group-rounded-linear',
	personal: 'solar:user-circle-linear',
	move: 'solar:move-to-folder-linear',
	parent: 'solar:arrow-left-up-linear',
	play: 'solar:play-bold',
	pause: 'solar:pause-bold',
	back10: 'solar:rewind-10-seconds-back-linear',
	forward10: 'solar:rewind-10-seconds-forward-linear',
	volume: 'solar:volume-loud-linear',
	muted: 'solar:volume-cross-linear',
	zoomIn: 'solar:magnifer-zoom-in-linear',
	zoomOut: 'solar:magnifer-zoom-out-linear',
	fitWidth: 'solar:maximize-square-minimalistic-linear',
	rotate: 'solar:refresh-linear',
	exitFullscreen: 'solar:quit-full-screen-linear',
	pip: 'solar:pip-2-linear',
	next: 'mdi:chevron-right',
	previous: 'mdi:chevron-left'
} as const;

/** File-type glyphs, keyed by the family `fileIcon` resolves a MIME type into. */
const FILE_TYPES = {
	image: 'solar:gallery-linear',
	video: 'solar:videocamera-record-linear',
	audio: 'solar:music-note-2-linear',
	pdf: 'solar:document-text-linear',
	archive: 'solar:zip-file-linear',
	sheet: 'solar:chart-square-linear',
	slides: 'solar:presentation-graph-linear',
	document: 'solar:document-linear',
	code: 'solar:code-linear',
	file: 'solar:file-linear'
} as const;

const EXTENSION_TYPES: Record<string, keyof typeof FILE_TYPES> = {
	jpg: 'image',
	jpeg: 'image',
	png: 'image',
	gif: 'image',
	webp: 'image',
	avif: 'image',
	svg: 'image',
	heic: 'image',
	mp4: 'video',
	m4v: 'video',
	mov: 'video',
	webm: 'video',
	mkv: 'video',
	mp3: 'audio',
	m4a: 'audio',
	wav: 'audio',
	ogg: 'audio',
	flac: 'audio',
	aac: 'audio',
	pdf: 'pdf',
	zip: 'archive',
	rar: 'archive',
	'7z': 'archive',
	tar: 'archive',
	gz: 'archive',
	bz2: 'archive',
	xz: 'archive',
	csv: 'sheet',
	xls: 'sheet',
	xlsx: 'sheet',
	ods: 'sheet',
	ppt: 'slides',
	pptx: 'slides',
	odp: 'slides',
	doc: 'document',
	docx: 'document',
	odt: 'document',
	rtf: 'document',
	md: 'document',
	txt: 'document'
};

/**
 * Picks a glyph from the MIME type, falling back to the extension. The type is the better
 * signal but the API stores whatever the browser guessed at upload, and browsers send
 * `application/octet-stream` for most of the interesting cases.
 */
export function fileIcon(name: string, mime = ''): string {
	if (mime.startsWith('image/')) return FILE_TYPES.image;
	if (mime.startsWith('video/')) return FILE_TYPES.video;
	if (mime.startsWith('audio/')) return FILE_TYPES.audio;
	if (mime === 'application/pdf') return FILE_TYPES.pdf;

	const extension = name.split('.').pop()?.toLowerCase() ?? '';
	return FILE_TYPES[EXTENSION_TYPES[extension] ?? 'file'];
}

const typeLabel = (type: keyof typeof FILE_TYPES): string => t(`type.${type}`);

/** The human name for whatever `fileIcon` classified the file as. */
export function fileTypeLabel(name: string, mime = ''): string {
	if (mime.startsWith('image/')) return typeLabel('image');
	if (mime.startsWith('video/')) return typeLabel('video');
	if (mime.startsWith('audio/')) return typeLabel('audio');
	if (mime === 'application/pdf') return typeLabel('pdf');

	const extension = name.split('.').pop()?.toLowerCase() ?? '';
	return typeLabel(EXTENSION_TYPES[extension] ?? 'file');
}

export { icons };

import { backend } from '$lib/backend';

export type UploadStatus = 'pending' | 'uploading' | 'done' | 'error';

export type UploadItem = {
	id: string;
	name: string;
	size: number;
	progress: number;
	status: UploadStatus;
	error?: string;
};

export type UploadTarget = {
	token: string;
	folderId: number | null;
	spaceId: number | null;
};

const CHUNK_SIZE = 5 * 1024 * 1024;

/**
 * Anything past this goes through the chunked session endpoints. A single PUT of a 2GB file
 * is one connection that cannot be resumed and that a proxy is free to cut; chunks survive
 * that, at the cost of hundreds of sequential requests — which is why the API's rate limiter
 * skips `/api/files/upload/*`.
 */
const CHUNKED_THRESHOLD = 10 * 1024 * 1024;

/** Shape muse's `UploadProgress` renders — `items` is handed to it directly. */
export class UploadQueue {
	items = $state<UploadItem[]>([]);
	active = $state(false);

	get done(): number {
		return this.items.filter((i) => i.status === 'done').length;
	}

	#patch(id: string, patch: Partial<UploadItem>): void {
		this.items = this.items.map((item) => (item.id === id ? { ...item, ...patch } : item));
	}

	/**
	 * Uploads sequentially and resolves once every file has settled. Failures are recorded on
	 * the item rather than thrown: one rejected file must not abandon the rest of a drop, and
	 * the row is what tells the user which one it was.
	 */
	async run(files: File[], target: UploadTarget): Promise<void> {
		if (files.length === 0) return;

		this.active = true;
		this.items = files.map((file, i) => ({
			id: `${Date.now()}-${i}-${file.name}`,
			name: file.name,
			size: file.size,
			progress: 0,
			status: 'pending' as const
		}));

		for (const [i, file] of files.entries()) {
			const { id } = this.items[i];
			this.#patch(id, { status: 'uploading' });
			try {
				if (file.size > CHUNKED_THRESHOLD) await this.#chunked(file, id, target);
				else await this.#simple(file, id, target);
				this.#patch(id, { progress: 100, status: 'done' });
			} catch (e) {
				this.#patch(id, {
					status: 'error',
					error: e instanceof Error && e.message ? e.message : 'Upload failed'
				});
			}
		}

		this.active = false;
	}

	/** Clears finished rows. The caller decides when — usually after reloading the listing. */
	reset(): void {
		this.items = [];
	}

	/**
	 * Drops one row. Failed uploads are deliberately kept on screen after a run — they are the
	 * only report of what did not make it — so there has to be a way to acknowledge one, or the
	 * strip stays until the next upload replaces it.
	 */
	remove(id: string): void {
		this.items = this.items.filter((item) => item.id !== id);
	}

	get failed(): UploadItem[] {
		return this.items.filter((i) => i.status === 'error');
	}

	async #simple(file: File, id: string, target: UploadTarget): Promise<void> {
		const formData = new FormData();
		formData.set('file', file);
		if (target.folderId != null) formData.set('folder_id', String(target.folderId));
		if (target.spaceId != null) formData.set('space_id', String(target.spaceId));

		await backend.uploadFileWithProgress(target.token, formData, (loaded) => {
			this.#patch(id, { progress: file.size === 0 ? 100 : (loaded / file.size) * 100 });
		});
	}

	async #chunked(file: File, id: string, target: UploadTarget): Promise<void> {
		const totalChunks = Math.ceil(file.size / CHUNK_SIZE);

		const session = await backend.initUpload(target.token, {
			file_name: file.name,
			mime_type: file.type || 'application/octet-stream',
			total_size: file.size,
			folder_id: target.folderId,
			space_id: target.spaceId
		});

		for (let part = 0; part < totalChunks; part += 1) {
			const start = part * CHUNK_SIZE;
			const end = Math.min(start + CHUNK_SIZE, file.size);
			await backend.uploadChunk(target.token, session.session_id, part + 1, file.slice(start, end));
			this.#patch(id, { progress: (end / file.size) * 100 });
		}

		await backend.completeUpload(target.token, session.session_id);
	}
}

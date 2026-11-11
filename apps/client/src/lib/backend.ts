import { ApiError, CSRF_HEADER, createApi, redirectToLogin } from '@facile/client';
import { isSafeUpgradeUrl, notePlanLimit } from '$lib/plan-limit';

export { ApiError };

const backendBaseUrl = '/api';

/*
 * porte carries a session on two transports: an Authorization header, which API tokens
 * use, and an HttpOnly cookie, which is all an SSO login leaves behind. The cookie is ambient,
 * so porte refuses a mutating request that carries one without the X-Facile-CSRF header, which
 * @facile/client's kernel adds to every method but GET, HEAD and OPTIONS.
 */
const csrfHeader = CSRF_HEADER;

const api = createApi({ base: backendBaseUrl, onUnauthorized: redirectToLogin });


export type UserProfile = {
	id: string;
	email: string;
	name: string;
	avatar_url: string;
	color: string;
	locale: string;
	created_at: string;
};

export type InvitationsResponse = {
	count: number;
	href?: string;
};

export type MeResponse = {
	user: UserProfile;
};

export type NuageFile = {
	id: number;
	facile_id: string;
	name: string;
	mime_type: string;
	size: number;
	folder_id: number | null;
	origin_app: string;
	linked_to: string;
	uploaded_by: number;
	created_at: string;
	updated_at: string;
};

export type Folder = {
	id: number;
	facile_id: string;
	name: string;
	size: number;
	parent_id: number | null;
	owner_id: number;
	created_at: string;
};

export type FolderResponse = Folder;

export type FolderDetailResponse = {
	folder: Folder;
	files: NuageFile[];
	folders: Folder[];
};

export type Share = {
	id: number;
	token: string;
	file_id: number | null;
	folder_id: number | null;
	shared_by: number;
	permission: string;
	expires_at: string | null;
	created_at: string;
	file?: NuageFile;
	folder?: Folder;
};

export type TrashItem = {
	type: 'file' | 'folder';
	id: number;
	facile_id: string;
	name: string;
	mime_type?: string;
	size?: number;
	deleted_at: string;
	expires_at?: string;
};

export type TrashResponse = {
	items: TrashItem[];
	retention_days: number;
};

export type ApiToken = {
	id: number;
	token?: string;
	name: string;
	created_at: string;
};

/** What one registered app stores for the signed-in user: usage only, never content. */
export type AppData = {
	app: string;
	name: string;
	active: boolean;
	objects: number;
	used_bytes: number;
	created_at: string;
};

export type QuotaResponse = {
	user_id: number;
	storage_used: number;
	storage_limit: number;
	percentage: number;
	source?: 'plan' | 'override' | 'instance';
	upgrade_url?: string;
};

export type InitUploadResponse = {
	session_id: string;
	expires_at: string;
};

export type ChunkResponse = {
	part_number: number;
	size: number;
	hash: string;
};

export type CompleteUploadResponse = {
	file: NuageFile;
};

/** What an anonymous visitor sees of a shared file: no owner, no folder, no origin. */
export type PublicFile = {
	id: number;
	facile_id: string;
	name: string;
	mime_type: string;
	size: number;
	updated_at: string;
};

export type PublicFolder = {
	id: number;
	facile_id: string;
	name: string;
	size: number;
};

export type PublicShareResponse = {
	token: string;
	permission: string;
	file: PublicFile | null;
	folder: PublicFolder | null;
};

export type PublicShareFilesResponse = {
	permission: string;
	files: PublicFile[];
	folders: PublicFolder[];
};

export type ActivityEntry = {
	id: number;
	user_id: number;
	event_type: string;
	resource_type: string;
	resource_id: number;
	resource_name: string;
	metadata: string;
	created_at: string;
	actor?: { id: number; name: string; email: string; avatar_url: string };
};

export type ActivityListResponse = {
	activities: ActivityEntry[];
	total: number;
	page: number;
	per_page: number;
};

export type Space = {
	id: number;
	facile_id: string;
	registre_space_id: string | null;
	name: string;
	description: string;
	role: string;
	created_at: string;
	updated_at: string;
};

export type SpaceMember = {
	id: number;
	user_id: number;
	role: string;
	joined_at: string;
	user?: {
		id: number;
		email: string;
		name: string;
		avatar_url: string;
		color: string;
	};
};

export type IdeaSource = { label: string; ref: string };

export type BulleIdea = {
	id: string;
	kind: string;
	title: string;
	why: string;
	draft: string;
	sources: IdeaSource[] | null;
	status: string;
	expires_at: string;
};

export type IdeasResponse = {
	ai_access: boolean;
	upgrade_url?: string;
	ideas: BulleIdea[];
	generating: boolean;
	last_batch_at: string | null;
};

export type RunState = 'running' | 'needs_approval' | 'done' | 'failed';

export type IdeaRunResponse = {
	id: string;
	state: RunState;
	summary: string;
	outputs: { call_id: string; tool: string; url: string; preview: string }[] | null;
	pending_approvals: { call_id: string; tool: string; args: unknown }[] | null;
	continue_url: string;
};

export type UploadProgressCallback = (loaded: number, total: number) => void;

/** An action an MCP client asked for, waiting for the person to confirm it. */
export type PendingAction = {
	id: string;
	kind: 'trash' | 'share';
	client_id: string;
	summary: string;
	created_at: string;
	expires_at: string;
	state: 'ready' | 'changed' | 'missing';
	expires_in_days?: number;
	item?: { type: 'file' | 'folder'; id: number; name: string; mime_type?: string; size?: number };
};

type ApiErrorPayload = {
	error?: { message?: string };
};

/**
 * isAuthError reports whether the API refused the credential, as opposed to
 * failing for any other reason.
 *
 * The distinction is the whole point: a route guard that treats a 500 or a
 * dropped connection as "logged out" throws the session away on a blip, and the
 * user lands on the login page holding a session that was fine.
 */
export function isAuthError(error: unknown): boolean {
	return error instanceof ApiError && (error.status === 401 || error.status === 403);
}

/**
 * The error for a failed upload request: an ApiError carrying the server's own message when
 * it sent one, so the UI can show it, and a plain Error otherwise, so the UI shows its own
 * localised wording instead of this English fallback.
 */
function uploadFailure(xhr: XMLHttpRequest, fallback: string): Error {
	try {
		const payload = JSON.parse(xhr.responseText);
		notePlanLimit(xhr.status, payload);
		if (payload?.error?.message) return new ApiError(xhr.status, '', payload.error.message);
	} catch {}
	return new Error(`${fallback} with status ${xhr.status}`);
}

/**
 * One API call through @facile/client's kernel. A 401 sends the browser to the login page unless
 * the call is quiet; a 402 raises the plan-limit banner unless the call is quiet. An upgrade link
 * survives only when it is safe to put in an href, and an empty body reads as an empty object.
 */
async function apiFetch<T>(path: string, options: RequestInit = {}, token?: string, quiet = false): Promise<T> {
	const headers = new Headers(options.headers);
	if (token) headers.set('Authorization', `Bearer ${token}`);
	try {
		return ((await api<T>(path, { ...options, headers }, { quiet })) ?? {}) as T;
	} catch (error) {
		if (!(error instanceof ApiError)) throw error;
		const upgradeUrl = error.status === 402 && isSafeUpgradeUrl(error.upgradeUrl) ? error.upgradeUrl : undefined;
		if (!quiet) notePlanLimit(error.status, { message: error.message, upgrade_url: upgradeUrl });
		throw new ApiError(error.status, error.code, error.message, upgradeUrl);
	}
}

const spaceQuery = (spaceId: number | null): string => (spaceId === null ? '' : `?space_id=${spaceId}`);

const isSvg = (file: { name: string; mime_type: string }): boolean =>
	file.mime_type === 'image/svg+xml' || file.name.toLowerCase().endsWith('.svg');

export const backend = {
	baseUrl: backendBaseUrl,

	listIdeas(spaceId: number | null) {
		return apiFetch<IdeasResponse>(`/ideas${spaceQuery(spaceId)}`, {}, undefined, true);
	},

	generateIdeas(spaceId: number | null) {
		return apiFetch<{ batch_id: string }>(`/ideas/generate${spaceQuery(spaceId)}`, { method: 'POST' }, undefined, true);
	},

	runIdea(id: string, spaceId: number | null) {
		return apiFetch<{ run_id: string }>(
			`/ideas/${encodeURIComponent(id)}/run${spaceQuery(spaceId)}`,
			{ method: 'POST' },
			undefined,
			true
		);
	},

	dismissIdea(id: string) {
		return apiFetch<object>(`/ideas/${encodeURIComponent(id)}/dismiss`, { method: 'POST' }, undefined, true);
	},

	ideaRun(runId: string) {
		return apiFetch<IdeaRunResponse>(`/ideas/runs/${encodeURIComponent(runId)}`, {}, undefined, true);
	},

	decideIdea(runId: string, callId: string, approve: boolean) {
		return apiFetch<object>(
			`/ideas/runs/${encodeURIComponent(runId)}/approvals/${encodeURIComponent(callId)}`,
			{ method: 'POST', body: JSON.stringify({ approve }) },
			undefined,
			true
		);
	},

	/**
	 * The authenticated profile, and the only honest answer to "is this browser
	 * signed in": the credential is the cookie an SSO callback left behind, and
	 * only the server can see it.
	 */
	me(token?: string) {
		return apiFetch<MeResponse>('/users/me', {}, token, true);
	},

	/**
	 * The signed-in person's pending Registre space invitations for the user menu. The API never
	 * fails this call for Registre's sake; without Registre the route is not mounted.
	 */
	invitations() {
		return apiFetch<InvitationsResponse>('/me/invitations', {}, undefined, true);
	},

	/**
	 * Ends the session server-side. Dropping the bearer token from localStorage is
	 * not enough on its own — an SSO login never put one there, and the cookie
	 * would sign the user straight back in on the next page load.
	 */
	logout(token?: string) {
		return apiFetch<{ logged_out: boolean }>('/auth/logout', { method: 'POST' }, token);
	},

	listFiles(token: string, params?: { folder_id?: number; search?: string; linked_to?: string; origin_app?: string; space_id?: number | null }) {
		const qs = new URLSearchParams();
		if (params?.folder_id != null) qs.set('folder_id', String(params.folder_id));
		if (params?.search) qs.set('search', params.search);
		if (params?.linked_to) qs.set('linked_to', params.linked_to);
		if (params?.origin_app) qs.set('origin_app', params.origin_app);
		if (params?.space_id != null) qs.set('space_id', String(params.space_id));
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<{ files: NuageFile[] }>(`/files${query}`, {}, token);
	},

	uploadFile(token: string, formData: FormData) {
		const headers = new Headers();
		headers.set(csrfHeader, '1');
		if (token) headers.set('Authorization', `Bearer ${token}`);
		return fetch(`${backendBaseUrl}/files`, {
			method: 'POST',
			body: formData,
			credentials: 'same-origin',
			headers
		}).then(async (r) => {
			if (!r.ok) {
				let payload: ApiErrorPayload | undefined;
				try {
					payload = (await r.json()) as ApiErrorPayload;
				} catch {
					payload = undefined;
				}
				notePlanLimit(r.status, payload);
				throw payload?.error?.message
					? new ApiError(r.status, '', payload.error.message)
					: new Error(`Upload failed with status ${r.status}`);
			}
			return (await r.json()) as NuageFile;
		});
	},

	getFile(token: string, id: number) {
		return apiFetch<NuageFile>(`/files/${id}`, {}, token);
	},

	/**
	 * A URL the browser fetches on its own — an <img src>, a window.open, an
	 * anchor's download. It carries no credential of its own because it cannot:
	 * none of those attach an Authorization header, and porte reads only the
	 * cookie and that header. The session cookie is what authenticates it.
	 */
	downloadUrl(id: number): string {
		return `${backendBaseUrl}/files/${id}/download`;
	},

	/**
	 * A cacheable `<img>` URL for a downscaled preview: it changes whenever the file does. An
	 * SVG is already tiny and the server cannot rasterise it, so it is served as it is.
	 */
	thumbnailUrl(file: NuageFile, width: 320 | 640 = 320): string {
		const version = encodeURIComponent(file.updated_at);
		if (isSvg(file)) return `${backendBaseUrl}/files/${file.id}/download?v=${version}`;
		return `${backendBaseUrl}/files/${file.id}/thumbnail?w=${width}&v=${version}`;
	},

	/** Both thumbnail widths for `<img srcset>`, or `undefined` where there is only one image. */
	thumbnailSrcset(file: NuageFile): string | undefined {
		if (isSvg(file)) return undefined;
		return `${backend.thumbnailUrl(file, 320)} 320w, ${backend.thumbnailUrl(file, 640)} 640w`;
	},

	/** Moves a file or folder. The API reads a missing target as "unchanged" and `0` as the root. */
	moveItem(token: string, type: 'file' | 'folder', id: number, folderId: number | null) {
		const target = folderId ?? 0;
		return type === 'file'
			? apiFetch<NuageFile>(`/files/${id}`, { method: 'PUT', body: JSON.stringify({ folder_id: target }) }, token)
			: apiFetch<FolderResponse>(`/folders/${id}`, { method: 'PUT', body: JSON.stringify({ parent_id: target }) }, token);
	},

	deleteFile(token: string, id: number) {
		return apiFetch<{ deleted: boolean }>(`/files/${id}`, { method: 'DELETE' }, token);
	},

	updateFile(token: string, id: number, data: { name: string }) {
		return apiFetch<NuageFile>(`/files/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}, token);
	},

	linkFile(token: string, id: number, data: { linked_to: string }) {
		return apiFetch<NuageFile>(`/files/${id}/link`, {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	createFolder(token: string, data: { name: string; parent_id?: number | null; space_id?: number | null }) {
		return apiFetch<FolderResponse>('/folders', {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	listFolders(token: string, params?: { parent_id?: number | null; space_id?: number | null }) {
		const qs = new URLSearchParams();
		if (params?.parent_id != null) qs.set('parent_id', String(params.parent_id));
		if (params?.space_id != null) qs.set('space_id', String(params.space_id));
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<{ folders: Folder[] }>(`/folders${query}`, {}, token);
	},

	getFolder(token: string, id: number) {
		return apiFetch<FolderDetailResponse>(`/folders/${id}`, {}, token);
	},

	updateFolder(token: string, id: number, data: { name: string }) {
		return apiFetch<FolderResponse>(`/folders/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}, token);
	},

	deleteFolder(token: string, id: number) {
		return apiFetch<{ deleted: boolean }>(`/folders/${id}`, { method: 'DELETE' }, token);
	},

	listTrash(token: string, params?: { space_id?: number | null }) {
		const qs = new URLSearchParams();
		if (params?.space_id !== undefined) {
			qs.set('space_id', params.space_id === null ? 'null' : String(params.space_id));
		}
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<TrashResponse>(`/trash${query}`, {}, token);
	},

	restoreItem(token: string, type: 'file' | 'folder', id: number) {
		return apiFetch<{}>(`/trash/${type}/${id}/restore`, { method: 'POST' }, token);
	},

	permanentDelete(token: string, type: 'file' | 'folder', id: number) {
		return apiFetch<{}>(`/trash/${type}/${id}`, { method: 'DELETE' }, token);
	},

	emptyTrash(token: string) {
		return apiFetch<{ deleted: number }>('/trash', { method: 'DELETE' }, token);
	},

	createShare(token: string, data: { file_id?: number; folder_id?: number; permission?: string; expires_at?: string; space_id?: number | null }) {
		return apiFetch<Share>('/shares', {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	listMyShares(token: string, params?: { space_id?: number | null }) {
		const qs = new URLSearchParams();
		if (params?.space_id !== undefined) {
			qs.set('space_id', params.space_id === null ? 'null' : String(params.space_id));
		}
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<{ shares: Share[] }>(`/shares/by-me${query}`, {}, token);
	},

	deleteShare(token: string, id: number) {
		return apiFetch<{}>(`/shares/${id}`, { method: 'DELETE' }, token);
	},

	/** Lists the actions MCP clients asked for that wait for the person. */
	listPending() {
		return apiFetch<{ actions: PendingAction[] }>('/mcp/pending');
	},

	/** Runs one waiting action once; a confirmed share answers its link. */
	confirmPending(id: string) {
		return apiFetch<{ status: string; url?: string }>(`/mcp/pending/${encodeURIComponent(id)}/confirm`, { method: 'POST' });
	},

	/** Closes one waiting action without running it. */
	dismissPending(id: string) {
		return apiFetch<{ status: string }>(`/mcp/pending/${encodeURIComponent(id)}/dismiss`, { method: 'POST' });
	},

	async getSettings(token: string): Promise<Record<string, string>> {
		const res = await apiFetch<{ settings?: { key: string; value: string }[] }>('/settings', {}, token);
		return Object.fromEntries((res.settings ?? []).map((setting) => [setting.key, setting.value]));
	},

	updateSettings(token: string, data: Record<string, string>) {
		return apiFetch<Record<string, string>>('/settings', {
			method: 'PUT',
			body: JSON.stringify({ settings: data })
		}, token);
	},

	testNook(token: string, data: { url: string; secret: string; enabled: boolean }) {
		return apiFetch<{ success: boolean; message?: string }>('/settings/test-nook', {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	getApiToken(token: string) {
		return apiFetch<{ tokens: ApiToken[] }>('/users/me/api-token', {}, token);
	},

	createApiToken(token: string, data: { name: string }) {
		return apiFetch<ApiToken>('/users/me/api-token', {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	deleteApiToken(token: string, tokenId: number) {
		return apiFetch<{}>(`/users/me/api-token/${tokenId}`, { method: 'DELETE' }, token);
	},

	listAppData(token: string) {
		return apiFetch<{ apps: AppData[] }>('/app-data', {}, token);
	},

	deleteAppData(token: string, app: string) {
		return apiFetch<{}>(`/app-data/${encodeURIComponent(app)}`, { method: 'DELETE' }, token);
	},

	getQuota(token: string) {
		return apiFetch<QuotaResponse>('/quota/me', {}, token);
	},

	uploadFileWithProgress(token: string, formData: FormData, onProgress?: UploadProgressCallback): Promise<NuageFile> {
		return new Promise((resolve, reject) => {
			const xhr = new XMLHttpRequest();
			xhr.open('POST', `${backendBaseUrl}/files`);
			xhr.setRequestHeader(csrfHeader, '1');
			if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
			if (onProgress) {
				xhr.upload.addEventListener('progress', (e) => {
					if (e.lengthComputable) onProgress(e.loaded, e.total);
				});
			}
			xhr.addEventListener('load', () => {
				if (xhr.status >= 200 && xhr.status < 300) {
					resolve(JSON.parse(xhr.responseText) as NuageFile);
				} else {
					reject(uploadFailure(xhr, 'Upload failed'));
				}
			});
			xhr.addEventListener('error', () => reject(new Error('Upload network error')));
			xhr.addEventListener('abort', () => reject(new Error('Upload aborted')));
			xhr.send(formData);
		});
	},

	initUpload(token: string, data: { file_name: string; mime_type: string; total_size: number; folder_id?: number | null; space_id?: number | null }) {
		return apiFetch<InitUploadResponse>('/files/upload/init', {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	uploadChunk(token: string, sessionId: string, partNumber: number, blob: Blob) {
		return new Promise<ChunkResponse>((resolve, reject) => {
			const xhr = new XMLHttpRequest();
			xhr.open('PUT', `${backendBaseUrl}/files/upload/${sessionId}/part/${partNumber}`);
			xhr.setRequestHeader(csrfHeader, '1');
			if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
			xhr.setRequestHeader('Content-Type', 'application/octet-stream');
			xhr.addEventListener('load', () => {
				if (xhr.status >= 200 && xhr.status < 300) {
					resolve(JSON.parse(xhr.responseText) as ChunkResponse);
				} else {
					reject(uploadFailure(xhr, 'Chunk upload failed'));
				}
			});
			xhr.addEventListener('error', () => reject(new Error('Chunk upload network error')));
			xhr.addEventListener('abort', () => reject(new Error('Chunk upload aborted')));
			xhr.send(blob);
		});
	},

	completeUpload(token: string, sessionId: string) {
		return apiFetch<CompleteUploadResponse>(`/files/upload/${sessionId}/complete`, {
			method: 'POST'
		}, token);
	},

	abortUpload(token: string, sessionId: string) {
		return apiFetch<{ aborted: boolean }>(`/files/upload/${sessionId}`, {
			method: 'DELETE'
		}, token);
	},

	getPublicShare(shareToken: string) {
		return apiFetch<PublicShareResponse>(`/shared/${shareToken}`, {}, undefined, true);
	},

	getPublicShareFiles(shareToken: string, folderId?: number) {
		const qs = new URLSearchParams();
		if (folderId != null) qs.set('folder_id', String(folderId));
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<PublicShareFilesResponse>(`/shared/${shareToken}/files${query}`, {}, undefined, true);
	},

	/** Downscaled preview of a shared image; an SVG is already small and is served as it is. */
	publicThumbnailUrl(shareToken: string, file: PublicFile, width: 320 | 640 = 320): string {
		if (isSvg(file)) return `${backendBaseUrl}/shared/${shareToken}/download/${file.id}`;
		const version = encodeURIComponent(file.updated_at ?? '');
		return `${backendBaseUrl}/shared/${shareToken}/thumbnail/${file.id}?w=${width}&v=${version}`;
	},

	publicThumbnailSrcset(shareToken: string, file: PublicFile): string | undefined {
		if (isSvg(file)) return undefined;
		return `${backend.publicThumbnailUrl(shareToken, file, 320)} 320w, ${backend.publicThumbnailUrl(shareToken, file, 640)} 640w`;
	},

	publicDownloadUrl(shareToken: string, fileId: number): string {
		return `${backendBaseUrl}/shared/${shareToken}/download/${fileId}`;
	},

	/**
	 * Every user on the instance, name-sorted. Any authenticated caller may read it — this is
	 * a single-tenant self-hosted tool where the directory is the point, and it is what makes
	 * picking a space member possible without an invite-by-email endpoint.
	 */
	listUsers(token: string) {
		return apiFetch<{ users: UserProfile[] }>('/users', {}, token);
	},

	listSpaces(token: string) {
		return apiFetch<{ spaces: Space[] }>('/spaces', {}, token);
	},

	getSpace(token: string, id: number) {
		return apiFetch<Space>(`/spaces/${id}`, {}, token);
	},

	updateSpace(token: string, id: number, data: { name?: string; description?: string }) {
		return apiFetch<Space>(`/spaces/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}, token);
	},

	deleteSpace(token: string, id: number) {
		return apiFetch<{ deleted: boolean }>(`/spaces/${id}`, { method: 'DELETE' }, token);
	},

	leaveSpace(token: string, id: number) {
		return apiFetch<{ left: boolean }>(`/spaces/${id}/leave`, { method: 'POST' }, token);
	},

	listSpaceMembers(token: string, spaceId: number) {
		return apiFetch<{ members: SpaceMember[] }>(`/spaces/${spaceId}/members`, {}, token);
	},

	addSpaceMember(token: string, spaceId: number, data: { user_id?: number; email?: string; role?: string }) {
		return apiFetch<SpaceMember>(`/spaces/${spaceId}/members`, {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	updateSpaceMember(token: string, spaceId: number, memberId: number, data: { role: string }) {
		return apiFetch<SpaceMember>(`/spaces/${spaceId}/members/${memberId}`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}, token);
	},

	removeSpaceMember(token: string, spaceId: number, memberId: number) {
		return apiFetch<{ deleted: boolean }>(`/spaces/${spaceId}/members/${memberId}`, { method: 'DELETE' }, token);
	},

	listActivity(token: string, params?: { page?: number; per_page?: number; event_type?: string; resource_type?: string; space_id?: number | null }) {
		const qs = new URLSearchParams();
		if (params?.page != null) qs.set('page', String(params.page));
		if (params?.per_page != null) qs.set('per_page', String(params.per_page));
		if (params?.event_type) qs.set('event_type', params.event_type);
		if (params?.resource_type) qs.set('resource_type', params.resource_type);
		if (params?.space_id !== undefined) {
			qs.set('space_id', params.space_id === null ? 'null' : String(params.space_id));
		}
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<ActivityListResponse>(`/activity/me${query}`, {}, token);
	}
};

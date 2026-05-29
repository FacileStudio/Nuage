const backendBaseUrl = '/api';

export type AuthConfig = {
	sso_only: boolean;
	oidc_enabled: boolean;
};

export type AuthResponse = {
	user_id: string;
	token: string;
};

export type UserProfile = {
	id: string;
	email: string;
	name: string;
	avatar_url: string;
	color: string;
	created_at: string;
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
};

export type TrashResponse = {
	items: TrashItem[];
};

export type ApiToken = {
	id: number;
	token?: string;
	name: string;
	created_at: string;
};

export type QuotaResponse = {
	user_id: number;
	storage_used: number;
	storage_limit: number;
	percentage: number;
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

export type PublicShareResponse = {
	token: string;
	permission: string;
	file: NuageFile | null;
	folder: Folder | null;
};

export type PublicShareFilesResponse = {
	permission: string;
	files: NuageFile[];
	folders: Folder[];
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
};

export type ActivityListResponse = {
	activities: ActivityEntry[];
	total: number;
	page: number;
	per_page: number;
};

export type UploadProgressCallback = (loaded: number, total: number) => void;

type ApiErrorPayload = {
	error?: { message?: string };
};

async function apiFetch<T>(path: string, options: RequestInit = {}, token?: string): Promise<T> {
	const headers = new Headers(options.headers);
	if (!headers.has('Content-Type') && options.body && !(options.body instanceof FormData)) {
		headers.set('Content-Type', 'application/json');
	}
	if (token) {
		headers.set('Authorization', `Bearer ${token}`);
	}
	const response = await fetch(`${backendBaseUrl}${path}`, { ...options, headers });
	if (!response.ok) {
		let payload: ApiErrorPayload | undefined;
		try {
			payload = (await response.json()) as ApiErrorPayload;
		} catch {
			payload = undefined;
		}
		throw new Error(payload?.error?.message || `Request failed with status ${response.status}`);
	}
	const text = await response.text();
	if (!text) return {} as T;
	return JSON.parse(text) as T;
}

export const backend = {
	baseUrl: backendBaseUrl,

	getAuthConfig() {
		return apiFetch<AuthConfig>('/auth/config');
	},

	register(email: string, password: string) {
		return apiFetch<AuthResponse>('/auth/register', {
			method: 'POST',
			body: JSON.stringify({ email, password })
		});
	},

	login(email: string, password: string) {
		return apiFetch<AuthResponse>('/auth/login', {
			method: 'POST',
			body: JSON.stringify({ email, password })
		});
	},

	me(token: string) {
		return apiFetch<MeResponse>('/users/me', {}, token);
	},

	listFiles(token: string, params?: { folder_id?: number; search?: string; linked_to?: string; origin_app?: string }) {
		const qs = new URLSearchParams();
		if (params?.folder_id != null) qs.set('folder_id', String(params.folder_id));
		if (params?.search) qs.set('search', params.search);
		if (params?.linked_to) qs.set('linked_to', params.linked_to);
		if (params?.origin_app) qs.set('origin_app', params.origin_app);
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<{ files: NuageFile[] }>(`/files${query}`, {}, token);
	},

	uploadFile(token: string, formData: FormData) {
		const headers = new Headers();
		headers.set('Authorization', `Bearer ${token}`);
		return fetch(`${backendBaseUrl}/files`, {
			method: 'POST',
			body: formData,
			headers
		}).then(async (r) => {
			if (!r.ok) {
				let payload: ApiErrorPayload | undefined;
				try {
					payload = (await r.json()) as ApiErrorPayload;
				} catch {
					payload = undefined;
				}
				throw new Error(payload?.error?.message || `Upload failed with status ${r.status}`);
			}
			return (await r.json()) as NuageFile;
		});
	},

	getFile(token: string, id: number) {
		return apiFetch<NuageFile>(`/files/${id}`, {}, token);
	},

	downloadUrl(token: string, id: number): string {
		return `${backendBaseUrl}/files/${id}/download?token=${encodeURIComponent(token)}`;
	},

	deleteFile(token: string, id: number) {
		return apiFetch<{ deleted: boolean }>(`/files/${id}`, { method: 'DELETE' }, token);
	},

	updateFile(token: string, id: number, data: { name?: string; folder_id?: number | null }) {
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

	createFolder(token: string, data: { name: string; parent_id?: number | null }) {
		return apiFetch<FolderResponse>('/folders', {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	listFolders(token: string, params?: { parent_id?: number | null }) {
		const qs = new URLSearchParams();
		if (params?.parent_id != null) qs.set('parent_id', String(params.parent_id));
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<{ folders: Folder[] }>(`/folders${query}`, {}, token);
	},

	getFolder(token: string, id: number) {
		return apiFetch<FolderDetailResponse>(`/folders/${id}`, {}, token);
	},

	updateFolder(token: string, id: number, data: { name?: string; parent_id?: number | null }) {
		return apiFetch<FolderResponse>(`/folders/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}, token);
	},

	deleteFolder(token: string, id: number) {
		return apiFetch<{ deleted: boolean }>(`/folders/${id}`, { method: 'DELETE' }, token);
	},

	listTrash(token: string) {
		return apiFetch<TrashResponse>('/trash', {}, token);
	},

	restoreItem(token: string, type: 'file' | 'folder', id: number) {
		return apiFetch<{}>(`/trash/${type}/${id}/restore`, { method: 'POST' }, token);
	},

	permanentDelete(token: string, type: 'file' | 'folder', id: number) {
		return apiFetch<{}>(`/trash/${type}/${id}`, { method: 'DELETE' }, token);
	},

	createShare(token: string, data: { file_id?: number; folder_id?: number; permission?: string; expires_at?: string }) {
		return apiFetch<Share>('/shares', {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	listMyShares(token: string) {
		return apiFetch<{ shares: Share[] }>('/shares/by-me', {}, token);
	},

	deleteShare(token: string, id: number) {
		return apiFetch<{}>(`/shares/${id}`, { method: 'DELETE' }, token);
	},

	getSettings(token: string) {
		return apiFetch<Record<string, string>>('/settings', {}, token);
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

	updateProfile(token: string, data: { name?: string; email?: string }) {
		return apiFetch<{ user: UserProfile }>('/users/me', {
			method: 'PATCH',
			body: JSON.stringify(data)
		}, token);
	},

	uploadAvatar(token: string, formData: FormData) {
		const headers = new Headers();
		headers.set('Authorization', `Bearer ${token}`);
		return fetch(`${backendBaseUrl}/users/me/avatar`, {
			method: 'POST',
			body: formData,
			headers
		}).then(async (r) => {
			if (!r.ok) {
				let payload: ApiErrorPayload | undefined;
				try {
					payload = (await r.json()) as ApiErrorPayload;
				} catch {
					payload = undefined;
				}
				throw new Error(payload?.error?.message || `Upload failed with status ${r.status}`);
			}
			return (await r.json()) as { avatar_url: string };
		});
	},

	deleteAvatar(token: string) {
		return apiFetch<{}>('/users/me/avatar', { method: 'DELETE' }, token);
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

	getQuota(token: string) {
		return apiFetch<QuotaResponse>('/quota/me', {}, token);
	},

	uploadFileWithProgress(token: string, formData: FormData, onProgress?: UploadProgressCallback): Promise<NuageFile> {
		return new Promise((resolve, reject) => {
			const xhr = new XMLHttpRequest();
			xhr.open('POST', `${backendBaseUrl}/files`);
			xhr.setRequestHeader('Authorization', `Bearer ${token}`);
			if (onProgress) {
				xhr.upload.addEventListener('progress', (e) => {
					if (e.lengthComputable) onProgress(e.loaded, e.total);
				});
			}
			xhr.addEventListener('load', () => {
				if (xhr.status >= 200 && xhr.status < 300) {
					resolve(JSON.parse(xhr.responseText) as NuageFile);
				} else {
					let msg = `Upload failed with status ${xhr.status}`;
					try {
						const payload = JSON.parse(xhr.responseText);
						if (payload?.error?.message) msg = payload.error.message;
					} catch {}
					reject(new Error(msg));
				}
			});
			xhr.addEventListener('error', () => reject(new Error('Upload network error')));
			xhr.addEventListener('abort', () => reject(new Error('Upload aborted')));
			xhr.send(formData);
		});
	},

	initUpload(token: string, data: { file_name: string; mime_type: string; total_size: number; folder_id?: number | null }) {
		return apiFetch<InitUploadResponse>('/files/upload/init', {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	uploadChunk(token: string, sessionId: string, partNumber: number, blob: Blob) {
		return new Promise<ChunkResponse>((resolve, reject) => {
			const xhr = new XMLHttpRequest();
			xhr.open('PUT', `${backendBaseUrl}/files/upload/${sessionId}/part/${partNumber}`);
			xhr.setRequestHeader('Authorization', `Bearer ${token}`);
			xhr.setRequestHeader('Content-Type', 'application/octet-stream');
			xhr.addEventListener('load', () => {
				if (xhr.status >= 200 && xhr.status < 300) {
					resolve(JSON.parse(xhr.responseText) as ChunkResponse);
				} else {
					let msg = `Chunk upload failed with status ${xhr.status}`;
					try {
						const payload = JSON.parse(xhr.responseText);
						if (payload?.error?.message) msg = payload.error.message;
					} catch {}
					reject(new Error(msg));
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
		return apiFetch<PublicShareResponse>(`/shared/${shareToken}`);
	},

	getPublicShareFiles(shareToken: string, folderId?: number) {
		const qs = new URLSearchParams();
		if (folderId != null) qs.set('folder_id', String(folderId));
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<PublicShareFilesResponse>(`/shared/${shareToken}/files${query}`);
	},

	publicDownloadUrl(shareToken: string, fileId: number): string {
		return `${backendBaseUrl}/shared/${shareToken}/download/${fileId}`;
	},

	syncProfile(token: string) {
		return apiFetch<{ synced: boolean }>('/auth/sync-profile', { method: 'POST' }, token);
	},

	listActivity(token: string, params?: { page?: number; per_page?: number; event_type?: string; resource_type?: string }) {
		const qs = new URLSearchParams();
		if (params?.page != null) qs.set('page', String(params.page));
		if (params?.per_page != null) qs.set('per_page', String(params.per_page));
		if (params?.event_type) qs.set('event_type', params.event_type);
		if (params?.resource_type) qs.set('resource_type', params.resource_type);
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<ActivityListResponse>(`/activity/me${query}`, {}, token);
	}
};

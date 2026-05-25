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
	shared_with: number | null;
	permission: string;
	expires_at: string | null;
	created_at: string;
	file?: NuageFile;
	folder?: Folder;
};

export type TrashResponse = {
	files: NuageFile[];
	folders: Folder[];
};

export type ApiToken = {
	id: number;
	token?: string;
	name: string;
	created_at: string;
};

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

	createShare(token: string, data: { file_id?: number; folder_id?: number; shared_with?: number; permission?: string; expires_at?: string }) {
		return apiFetch<Share>('/shares', {
			method: 'POST',
			body: JSON.stringify(data)
		}, token);
	},

	listSharedWithMe(token: string) {
		return apiFetch<{ shares: Share[] }>('/shares', {}, token);
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

	testNook(token: string) {
		return apiFetch<{ success: boolean; message?: string }>('/settings/test-nook', { method: 'POST' }, token);
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
	}
};

<script lang="ts">
	import { onMount, getContext } from 'svelte';
	import { backend, type UserProfile, type ApiToken } from '$lib/backend';

	const app = getContext<{ token: string; user: UserProfile | null; setUser: (u: UserProfile) => void }>('app');

	let activeTab = $state<'profile' | 'instance' | 'developers'>('profile');

	let profileName = $state('');
	let profileEmail = $state('');
	let profileSaving = $state(false);
	let profileMessage = $state('');

	let avatarUploading = $state(false);
	let avatarUrl = $derived(app.user?.avatar_url ?? '');

	let apiTokens = $state<ApiToken[]>([]);
	let newTokenName = $state('');
	let createdToken = $state<string | null>(null);
	let copiedToken = $state(false);
	let tokenLoading = $state(false);

	let instanceName = $state('');
	let nookWebhookUrl = $state('');
	let nookSecret = $state('');
	let nookEnabled = $state(false);
	let settingsSaving = $state(false);
	let settingsMessage = $state('');
	let testingNook = $state(false);
	let nookTestResult = $state<{ success: boolean; message?: string } | null>(null);

	onMount(async () => {
		if (app.user) {
			profileName = app.user.name ?? '';
			profileEmail = app.user.email ?? '';
		}
		await Promise.all([loadSettings(), loadApiTokens()]);
	});

	async function loadSettings() {
		try {
			const settings = await backend.getSettings(app.token);
			instanceName = settings.instance_name ?? '';
			nookWebhookUrl = settings.nook_webhook_url ?? '';
			nookSecret = settings.nook_webhook_secret ?? '';
			nookEnabled = settings.nook_enabled === 'true';
		} catch {}
	}

	async function loadApiTokens() {
		try {
			const res = await backend.getApiToken(app.token);
			apiTokens = res.tokens ?? [];
		} catch {
			apiTokens = [];
		}
	}

	async function saveProfile() {
		profileSaving = true;
		profileMessage = '';
		try {
			const res = await backend.updateProfile(app.token, { name: profileName });
			app.setUser(res.user);
			profileMessage = 'Profile updated';
		} catch (e: any) {
			profileMessage = e.message || 'Failed to update profile';
		}
		profileSaving = false;
		setTimeout(() => { profileMessage = ''; }, 3000);
	}

	async function handleAvatarUpload(e: Event) {
		const input = e.target as HTMLInputElement;
		if (!input.files?.length) return;
		avatarUploading = true;
		try {
			const formData = new FormData();
			formData.set('avatar', input.files[0]);
			const res = await backend.uploadAvatar(app.token, formData);
			if (app.user) {
				app.setUser({ ...app.user, avatar_url: res.avatar_url });
			}
		} catch {}
		avatarUploading = false;
		input.value = '';
	}

	async function removeAvatar() {
		avatarUploading = true;
		try {
			await backend.deleteAvatar(app.token);
			if (app.user) {
				app.setUser({ ...app.user, avatar_url: '' });
			}
		} catch {}
		avatarUploading = false;
	}

	async function createToken() {
		if (!newTokenName.trim()) return;
		tokenLoading = true;
		createdToken = null;
		try {
			const res = await backend.createApiToken(app.token, { name: newTokenName.trim() });
			createdToken = res.token;
			newTokenName = '';
			await loadApiTokens();
		} catch {}
		tokenLoading = false;
	}

	async function copyToken() {
		if (!createdToken) return;
		await navigator.clipboard.writeText(createdToken);
		copiedToken = true;
		setTimeout(() => { copiedToken = false; }, 2000);
	}

	async function revokeToken(id: number) {
		tokenLoading = true;
		try {
			await backend.deleteApiToken(app.token, id);
			await loadApiTokens();
			createdToken = null;
		} catch {}
		tokenLoading = false;
	}

	async function saveSettings() {
		settingsSaving = true;
		settingsMessage = '';
		try {
			await backend.updateSettings(app.token, {
				instance_name: instanceName,
				nook_webhook_url: nookWebhookUrl,
				nook_webhook_secret: nookSecret,
				nook_enabled: String(nookEnabled)
			});
			settingsMessage = 'Settings saved';
		} catch (e: any) {
			settingsMessage = e.message || 'Failed to save settings';
		}
		settingsSaving = false;
		setTimeout(() => { settingsMessage = ''; }, 3000);
	}

	async function testNookConnection() {
		testingNook = true;
		nookTestResult = null;
		try {
			const res = await backend.testNook(app.token, {
				url: nookWebhookUrl,
				secret: nookSecret,
				enabled: nookEnabled
			});
			nookTestResult = res;
		} catch (e: any) {
			nookTestResult = { success: false, message: e.message || 'Connection failed' };
		}
		testingNook = false;
		setTimeout(() => { nookTestResult = null; }, 5000);
	}

	function getInitials(value: string) {
		const parts = value.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase();
	}

	function handleTokenKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') createToken();
	}
</script>

<svelte:head>
	<title>Settings — Nuage</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<h1 class="text-lg font-semibold">Settings</h1>
		<div class="mt-4 flex gap-1">
			<button
				class="inline-flex h-9 items-center gap-2 rounded-md px-4 text-sm font-medium transition-colors {activeTab === 'profile' ? 'bg-foreground text-background' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
				onclick={() => activeTab = 'profile'}
			>
				<iconify-icon icon="solar:user-circle-linear" width="18"></iconify-icon>
				Profile
			</button>
			<button
				class="inline-flex h-9 items-center gap-2 rounded-md px-4 text-sm font-medium transition-colors {activeTab === 'developers' ? 'bg-foreground text-background' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
				onclick={() => activeTab = 'developers'}
			>
				<iconify-icon icon="solar:code-square-linear" width="18"></iconify-icon>
				Developers
			</button>
			<button
				class="inline-flex h-9 items-center gap-2 rounded-md px-4 text-sm font-medium transition-colors {activeTab === 'instance' ? 'bg-foreground text-background' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
				onclick={() => activeTab = 'instance'}
			>
				<iconify-icon icon="solar:server-linear" width="18"></iconify-icon>
				Instance
			</button>
		</div>
	</div>

	<div class="flex-1 overflow-auto px-4 py-6 md:px-8">
		{#if activeTab === 'profile'}
			<div class="mx-auto max-w-xl space-y-8">
				<div class="space-y-4">
					<h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Avatar</h2>
					<div class="flex items-center gap-4">
						{#if avatarUrl}
							<img src={avatarUrl} alt="Avatar" class="h-16 w-16 rounded-full border border-border object-cover" />
						{:else}
							<div class="flex h-16 w-16 items-center justify-center rounded-full border border-border bg-foreground text-lg font-semibold text-background">
								{getInitials(profileName || profileEmail)}
							</div>
						{/if}
						<div class="flex gap-2">
							<label class="inline-flex h-9 cursor-pointer items-center gap-2 rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent disabled:opacity-50">
								{#if avatarUploading}
									<div class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
								{:else}
									<iconify-icon icon="solar:camera-linear" width="16"></iconify-icon>
								{/if}
								Change
								<input type="file" accept="image/*" class="hidden" onchange={handleAvatarUpload} disabled={avatarUploading} />
							</label>
							{#if avatarUrl}
								<button
									class="inline-flex h-9 items-center gap-2 rounded-md border border-border bg-background px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:opacity-50"
									onclick={removeAvatar}
									disabled={avatarUploading}
								>
									Remove
								</button>
							{/if}
						</div>
					</div>
				</div>

				<div class="space-y-4">
					<h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Profile</h2>
					<div class="space-y-3">
						<div>
							<label for="profile-name" class="mb-1.5 block text-sm font-medium">Name</label>
							<input
								id="profile-name"
								type="text"
								bind:value={profileName}
								class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							/>
						</div>
						<div>
							<label for="profile-email" class="mb-1.5 block text-sm font-medium">Email</label>
							<input
								id="profile-email"
								type="email"
								value={profileEmail}
								disabled
								class="flex h-10 w-full rounded-md border border-input bg-muted px-3 py-2 text-sm text-muted-foreground"
							/>
						</div>
					</div>
					<div class="flex items-center gap-3">
						<button
							class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
							onclick={saveProfile}
							disabled={profileSaving}
						>
							{profileSaving ? 'Saving...' : 'Save profile'}
						</button>
						{#if profileMessage}
							<span class="text-sm text-muted-foreground">{profileMessage}</span>
						{/if}
					</div>
				</div>

			</div>

		{:else if activeTab === 'developers'}
			<div class="mx-auto max-w-xl space-y-8">
				<div class="space-y-2">
					<h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">API Tokens</h2>
					<p class="text-sm text-muted-foreground">Tokens are used to authenticate CLI tools and API integrations. The token value is only shown once at creation.</p>
				</div>

				{#if createdToken}
					<div class="rounded-md border border-emerald-500/30 bg-emerald-500/5 p-4">
						<p class="text-xs font-medium text-emerald-400">Token created — copy it now, it won't be shown again.</p>
						<div class="mt-2 flex items-center gap-2">
							<code class="flex-1 break-all rounded bg-background px-3 py-2 text-xs">{createdToken}</code>
							<button
								class="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-md border border-border px-3 text-xs font-medium transition-colors hover:bg-accent"
								onclick={copyToken}
							>
								<iconify-icon icon={copiedToken ? 'solar:check-read-linear' : 'solar:copy-linear'} width="14"></iconify-icon>
								{copiedToken ? 'Copied' : 'Copy'}
							</button>
						</div>
					</div>
				{/if}

				{#if apiTokens.length > 0}
					<div class="space-y-2">
						{#each apiTokens as tok}
							<div class="flex items-center justify-between rounded-md border border-border p-3">
								<div class="flex items-center gap-3">
									<iconify-icon icon="solar:key-linear" width="16" class="text-muted-foreground"></iconify-icon>
									<div>
										<p class="text-sm font-medium">{tok.name}</p>
										<p class="text-xs text-muted-foreground">Created {new Date(tok.created_at).toLocaleDateString()}</p>
									</div>
								</div>
								<button
									class="inline-flex h-8 items-center rounded-md px-3 text-xs font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:opacity-50"
									onclick={() => revokeToken(tok.id)}
									disabled={tokenLoading}
								>
									Revoke
								</button>
							</div>
						{/each}
					</div>
				{:else}
					<div class="rounded-md border border-dashed border-border p-6 text-center">
						<p class="text-sm text-muted-foreground">No API tokens yet.</p>
					</div>
				{/if}

				<div class="space-y-2">
					<label for="new-token-name" class="block text-sm font-medium">New token</label>
					<div class="flex items-center gap-2">
						<input
							id="new-token-name"
							type="text"
							placeholder="Token name (e.g. MacBook CLI)"
							bind:value={newTokenName}
							onkeydown={handleTokenKeydown}
							class="flex h-9 flex-1 rounded-md border border-input bg-background px-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
						<button
							class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
							onclick={createToken}
							disabled={tokenLoading || !newTokenName.trim()}
						>
							Generate
						</button>
					</div>
				</div>
			</div>

		{:else}
			<div class="mx-auto max-w-xl space-y-8">
				<div class="space-y-4">
					<h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Instance</h2>
					<div>
						<label for="instance-name" class="mb-1.5 block text-sm font-medium">Instance name</label>
						<input
							id="instance-name"
							type="text"
							bind:value={instanceName}
							placeholder="My Nuage"
							class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
					</div>
				</div>

				<div class="space-y-4">
					<h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Nook Integration</h2>
					<div class="space-y-3">
						<div>
							<label for="nook-url" class="mb-1.5 block text-sm font-medium">Webhook URL</label>
							<input
								id="nook-url"
								type="url"
								bind:value={nookWebhookUrl}
								placeholder="https://..."
								class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							/>
						</div>
						<div>
							<label for="nook-secret" class="mb-1.5 block text-sm font-medium">Secret</label>
							<input
								id="nook-secret"
								type="password"
								bind:value={nookSecret}
								placeholder="Shared secret"
								class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							/>
						</div>
						<div class="flex items-center gap-3">
							<button
								class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full transition-colors {nookEnabled ? 'bg-primary' : 'bg-muted'}"
								onclick={() => nookEnabled = !nookEnabled}
								role="switch"
								aria-checked={nookEnabled}
								aria-label="Toggle Nook integration"
							>
								<span class="pointer-events-none inline-block h-4 w-4 rounded-full bg-background shadow-sm transition-transform {nookEnabled ? 'translate-x-6' : 'translate-x-1'}"></span>
							</button>
							<span class="text-sm">Enabled</span>
						</div>
					</div>
					<div class="flex items-center gap-2">
						<button
							class="inline-flex h-9 items-center gap-2 rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent disabled:opacity-50"
							onclick={testNookConnection}
							disabled={testingNook || !nookWebhookUrl}
						>
							{#if testingNook}
								<div class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
							{:else}
								<iconify-icon icon="solar:bolt-linear" width="16"></iconify-icon>
							{/if}
							Test connection
						</button>
						{#if nookTestResult}
							<span class="text-sm {nookTestResult.success ? 'text-emerald-600' : 'text-destructive'}">
								{nookTestResult.success ? 'Connected!' : (nookTestResult.message ?? 'Failed')}
							</span>
						{/if}
					</div>
				</div>

				<div class="flex items-center gap-3 border-t border-border pt-6">
					<button
						class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
						onclick={saveSettings}
						disabled={settingsSaving}
					>
						{settingsSaving ? 'Saving...' : 'Save settings'}
					</button>
					{#if settingsMessage}
						<span class="text-sm text-muted-foreground">{settingsMessage}</span>
					{/if}
				</div>
			</div>
		{/if}
	</div>
</div>

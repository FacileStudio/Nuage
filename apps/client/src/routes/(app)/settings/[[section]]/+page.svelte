<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import {
		Alert,
		Avatar,
		Button,
		ConfirmModal,
		Divider,
		Drawer,
		EmptyState,
		Field,
		Input,
		OptionCards,
		SecretField,
		SettingsRow,
		SettingsSection,
		StatusDot,
		Switch,
		Table,
		Tabs,
		toast
	} from '@facile/muse';
	import { backend, type ApiToken, type UserProfile } from '$lib/backend';
	import { theme, type ThemePreference } from '$lib/theme.svelte';
	import { formatDate } from '$lib/format';
	import { icons, nuage } from '$lib/icons';

	const app = getContext<{
		token: string;
		user: UserProfile | null;
		setUser: (u: UserProfile) => void;
	}>('app');

	const SECTIONS = [
		{ id: 'profile', label: 'Profile', icon: icons.userCircle },
		{ id: 'appearance', label: 'Appearance', icon: icons.palette },
		{ id: 'api', label: 'API', icon: icons.key },
		{ id: 'nook', label: 'Nook', icon: icons.bolt },
		{ id: 'advanced', label: 'Advanced', icon: icons.server }
	] as const;

	/*
	 * The section is a real route, so a link to /settings/api opens on API, a reload keeps you
	 * there and browser-back walks the sections — none of which local `$state` tabs give you
	 * (CHARTE §14). An unknown slug falls back to Profile rather than rendering nothing.
	 */
	const section = $derived(
		SECTIONS.some((s) => s.id === page.params.section) ? page.params.section! : 'profile'
	);
	const tabs = SECTIONS.map((s) => ({ ...s, href: `/settings/${s.id}` }));

	let profileName = $state('');
	let profileSaving = $state(false);

	let avatarUploading = $state(false);
	let avatarInput = $state<HTMLInputElement | null>(null);
	// avatar_url arrives ready to use, whether it points at a file this instance serves or
	// straight at the SSO photo. Never prefix it with the API base URL.
	let avatarUrl = $derived(app.user?.avatar_url ?? '');
	let avatarFromSSO = $derived(app.user?.avatar_source === 'oidc');

	let apiTokens = $state<ApiToken[]>([]);
	let tokenDrawerOpen = $state(false);
	let newTokenName = $state('');
	let createdToken = $state<string | null>(null);
	let tokenSaving = $state(false);
	let revoking = $state<ApiToken | null>(null);

	let instanceName = $state('');
	let nookWebhookUrl = $state('');
	let nookSecret = $state('');
	let nookEnabled = $state(false);
	let settingsSaving = $state(false);
	let testingNook = $state(false);
	let nookTestResult = $state<{ success: boolean; message?: string } | null>(null);

	let loggingOut = $state(false);

	onMount(async () => {
		profileName = app.user?.name ?? '';
		await Promise.all([loadSettings(), loadApiTokens()]);
	});

	async function loadSettings() {
		try {
			const settings = await backend.getSettings(app.token);
			instanceName = settings.instance_name ?? '';
			nookWebhookUrl = settings.nook_webhook_url ?? '';
			nookSecret = settings.nook_webhook_secret ?? '';
			nookEnabled = settings.nook_enabled === 'true';
		} catch {
			toast.danger('Could not load instance settings.');
		}
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
		try {
			const res = await backend.updateProfile(app.token, { name: profileName });
			app.setUser(res.user);
			toast.success('Profile updated.');
		} catch (e) {
			toast.danger(message(e, 'Could not update your profile.'));
		}
		profileSaving = false;
	}

	async function handleAvatarUpload(e: Event) {
		const input = e.target as HTMLInputElement;
		if (!input.files?.length) return;
		avatarUploading = true;
		try {
			const formData = new FormData();
			formData.set('avatar', input.files[0]);
			// The response is the whole profile: avatar_url is derived server-side, so
			// picking it apart here would be a second place for the rule to drift.
			const res = await backend.uploadAvatar(app.token, formData);
			app.setUser(res.user);
			toast.success('Picture updated.');
		} catch (e) {
			toast.danger(message(e, 'Could not upload that picture.'));
		}
		avatarUploading = false;
		input.value = '';
	}

	async function removeAvatar() {
		avatarUploading = true;
		try {
			const res = await backend.deleteAvatar(app.token);
			app.setUser(res.user);
		} catch (e) {
			toast.danger(message(e, 'Could not remove your picture.'));
		}
		avatarUploading = false;
	}

	function openTokenDrawer() {
		/* Reopening must never resurrect the previous token — the drawer body swaps to the
		   revealed value on success, and a stale one there is a credential leak. */
		newTokenName = '';
		createdToken = null;
		tokenDrawerOpen = true;
	}

	async function createToken() {
		if (!newTokenName.trim()) return;
		tokenSaving = true;
		try {
			const res = await backend.createApiToken(app.token, { name: newTokenName.trim() });
			createdToken = res.token ?? null;
			await loadApiTokens();
		} catch (e) {
			toast.danger(message(e, 'Could not create that token.'));
		}
		tokenSaving = false;
	}

	async function revokeToken() {
		const target = revoking;
		if (!target) return;
		await backend.deleteApiToken(app.token, target.id);
		await loadApiTokens();
		revoking = null;
		toast.success(`Revoked “${target.name}”.`);
	}

	async function saveSettings() {
		settingsSaving = true;
		try {
			await backend.updateSettings(app.token, {
				instance_name: instanceName,
				nook_webhook_url: nookWebhookUrl,
				nook_webhook_secret: nookSecret,
				nook_enabled: String(nookEnabled)
			});
			toast.success('Settings saved.');
		} catch (e) {
			toast.danger(message(e, 'Could not save settings.'));
		}
		settingsSaving = false;
	}

	async function testNookConnection() {
		testingNook = true;
		nookTestResult = null;
		try {
			nookTestResult = await backend.testNook(app.token, {
				url: nookWebhookUrl,
				secret: nookSecret,
				enabled: nookEnabled
			});
		} catch (e) {
			nookTestResult = { success: false, message: message(e, 'Connection failed.') };
		}
		testingNook = false;
	}

	function logout() {
		localStorage.removeItem('nuage.token');
		goto('/login');
	}

	function message(error: unknown, fallback: string): string {
		return error instanceof Error && error.message ? error.message : fallback;
	}

	const themeOptions = [
		{ value: 'system', label: 'System', icon: icons.monitor },
		{ value: 'light', label: 'Light', icon: icons.sun },
		{ value: 'dark', label: 'Dark', icon: icons.moon }
	];

	/*
	 * Not a boolean: "not connected" hides disabled, never-tested and failing behind one word,
	 * and those have three different fixes (CHARTE §14).
	 */
	const nookStatus = $derived.by(() => {
		if (!nookEnabled) return { tone: 'neutral' as const, label: 'Disabled', pulse: false };
		if (testingNook) return { tone: 'info' as const, label: 'Testing the connection…', pulse: true };
		if (!nookWebhookUrl) return { tone: 'warning' as const, label: 'Enabled, but no URL is set', pulse: false };
		if (nookTestResult?.success) return { tone: 'success' as const, label: 'Reached the endpoint', pulse: false };
		if (nookTestResult) {
			return { tone: 'danger' as const, label: nookTestResult.message ?? 'The endpoint did not answer', pulse: false };
		}
		return { tone: 'neutral' as const, label: 'Enabled — not tested since load', pulse: false };
	});
</script>

<svelte:head>
	<title>Settings — Nuage</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-10 px-4 py-6 md:px-8">
	<div class="flex flex-col gap-4">
		<h1 class="text-fc-2xl font-semibold text-fc-fg">Settings</h1>
		<Tabs items={tabs} value={section} label="Settings sections" />
		<Divider class="my-0" />
	</div>

	{#if section === 'profile'}
		<SettingsSection title="Picture" description="Shown next to your name across the app.">
			<div class="flex flex-wrap items-center gap-4">
				<Avatar name={profileName || app.user?.email || ''} src={avatarUrl} size="lg" />
				<!-- Uploading is the fallback, not a second option: when SSO supplies a photo
				     it is the one shown everywhere, so offering a file picker here would take
				     an upload the user would never see. Say where the photo lives instead. -->
				{#if avatarFromSSO}
					<div class="flex min-w-0 flex-col gap-1">
						<p class="text-fc-sm text-fc-fg">Your picture comes from single sign-on.</p>
						<p class="text-fc-xs text-fc-fg-muted">
							Change it on the login portal and it updates here within a few minutes.
						</p>
					</div>
				{:else}
					<div class="flex flex-wrap gap-2">
						<Button
							variant="outline"
							icon={nuage.camera}
							disabled={avatarUploading}
							onclick={() => avatarInput?.click()}
						>
							{avatarUploading ? 'Uploading…' : 'Change'}
						</Button>
						{#if avatarUrl}
							<Button
								variant="ghost-danger"
								icon={icons.remove}
								disabled={avatarUploading}
								onclick={removeAvatar}
							>
								Remove
							</Button>
						{/if}
						<input
							bind:this={avatarInput}
							type="file"
							accept="image/*"
							class="hidden"
							onchange={handleAvatarUpload}
						/>
					</div>
				{/if}
			</div>
		</SettingsSection>

		<SettingsSection title="Identity" description="How you appear to everyone sharing a space with you.">
			<SettingsRow label="Display name" stacked>
				<Field>
					<Input bind:value={profileName} placeholder="Your name" />
				</Field>
			</SettingsRow>
			<SettingsRow
				label="Email"
				description="Managed by single sign-on — change it on the login portal."
				stacked
			>
				<Field>
					<Input value={app.user?.email ?? ''} disabled />
				</Field>
			</SettingsRow>
			<SettingsRow>
				<Button icon={icons.check} disabled={profileSaving} onclick={saveProfile}>
					{profileSaving ? 'Saving…' : 'Save profile'}
				</Button>
			</SettingsRow>
		</SettingsSection>

		<SettingsSection title="Session" description="This browser only — your files stay where they are.">
			<SettingsRow label="Log out" description="You will need to sign in again to reach your files.">
				<Button variant="outline" icon={icons.logout} onclick={() => (loggingOut = true)}>
					Log out
				</Button>
			</SettingsRow>
		</SettingsSection>
	{:else if section === 'appearance'}
		<SettingsSection title="Theme" description="Applied to this browser.">
			<SettingsRow label="Colour scheme" description="Follow the system setting, or pin one." stacked>
				<OptionCards
					options={themeOptions}
					value={theme.preference}
					label="Colour scheme"
					onSelect={(value) => theme.set(value as ThemePreference)}
				/>
			</SettingsRow>
		</SettingsSection>
	{:else if section === 'api'}
		<SettingsSection
			title="API tokens"
			description="Authenticate the CLI, WebDAV clients and integrations. A token's value is shown once, at creation."
		>
			{#snippet actions()}
				<Button icon={icons.plus} onclick={openTokenDrawer}>New token</Button>
			{/snippet}

			{#if apiTokens.length === 0}
				<EmptyState
					bare
					icon={icons.key}
					title="No API tokens yet"
					description="Create one to mount Nuage over WebDAV or drive it from the CLI."
				>
					<Button icon={icons.plus} onclick={openTokenDrawer}>New token</Button>
				</EmptyState>
			{:else}
				<Table>
					<thead>
						<tr>
							<th scope="col">Name</th>
							<th scope="col">Created</th>
							<th scope="col" class="text-right"><span class="sr-only">Actions</span></th>
						</tr>
					</thead>
					<tbody>
						{#each apiTokens as tok (tok.id)}
							<tr>
								<td class="font-medium">{tok.name}</td>
								<td class="text-fc-fg-muted">{formatDate(tok.created_at)}</td>
								<td class="text-right">
									<Button
										variant="ghost-danger"
										size="sm"
										icon={icons.revoke}
										onclick={() => (revoking = tok)}
									>
										Revoke
									</Button>
								</td>
							</tr>
						{/each}
					</tbody>
				</Table>
			{/if}
		</SettingsSection>

		<SettingsSection title="WebDAV" description="Mount Nuage in Finder or any WebDAV client.">
			<SettingsRow label="Endpoint" description="Sign in with your email and an API token as the password." stacked>
				<SecretField value={`${page.url.origin}/webdav`} sensitive={false} />
			</SettingsRow>
		</SettingsSection>
	{:else if section === 'nook'}
		<SettingsSection
			title="Nook"
			description="Nuage posts file events to a Nook webhook so alerts land where the rest of the suite reports."
		>
			<SettingsRow label="Status">
				<StatusDot tone={nookStatus.tone} label={nookStatus.label} pulse={nookStatus.pulse} />
			</SettingsRow>
			<SettingsRow label="Enabled" description="Stops every outbound event when off.">
				<Switch bind:checked={nookEnabled} aria-label="Nook integration enabled" />
			</SettingsRow>
			<SettingsRow label="Webhook URL" stacked>
				<Field>
					<Input bind:value={nookWebhookUrl} type="url" placeholder="https://nook.example.com/hooks/nuage" />
				</Field>
			</SettingsRow>
			<SettingsRow label="Shared secret" description="Signs every payload so Nook can tell it came from here." stacked>
				<SecretField bind:value={nookSecret} editable placeholder="Not set" />
			</SettingsRow>
			<SettingsRow>
				<div class="flex flex-wrap gap-2">
					<Button icon={icons.check} disabled={settingsSaving} onclick={saveSettings}>
						{settingsSaving ? 'Saving…' : 'Save'}
					</Button>
					<Button
						variant="outline"
						icon={icons.bolt}
						disabled={testingNook || !nookWebhookUrl}
						onclick={testNookConnection}
					>
						Test connection
					</Button>
				</div>
			</SettingsRow>
		</SettingsSection>
	{:else}
		<SettingsSection title="Instance" description="Applies to everyone on this Nuage.">
			<SettingsRow label="Instance name" stacked>
				<Field>
					<Input bind:value={instanceName} placeholder="My Nuage" />
				</Field>
			</SettingsRow>
			<SettingsRow>
				<Button icon={icons.check} disabled={settingsSaving} onclick={saveSettings}>
					{settingsSaving ? 'Saving…' : 'Save'}
				</Button>
			</SettingsRow>
		</SettingsSection>

		<SettingsSection title="About" description="Facts worth having on hand when something breaks.">
			<SettingsRow label="API documentation" description="Every endpoint this instance serves.">
				<Button variant="outline" href="/docs" iconRight={icons.arrow}>Open</Button>
			</SettingsRow>
			<SettingsRow label="Account created">
				<span class="text-fc-sm text-fc-fg-muted">
					{app.user?.created_at ? formatDate(app.user.created_at) : '—'}
				</span>
			</SettingsRow>
		</SettingsSection>
	{/if}
</div>

<Drawer bind:open={tokenDrawerOpen} title="New API token" showClose>
	{#if createdToken}
		<div class="flex flex-col gap-4">
			<Alert tone="warning" title="Copy it now">
				This is the only time the token is shown. If you lose it, revoke it and make another.
			</Alert>
			<SecretField value={createdToken} autoHideMs={0} />
		</div>
	{:else}
		<Field label="Name" helper="Where you intend to use it — “MacBook Finder”, “backup cron”.">
			<Input bind:value={newTokenName} placeholder="MacBook Finder" />
		</Field>
	{/if}

	{#snippet footer()}
		{#if createdToken}
			<Button size="lg" class="w-full" onclick={() => (tokenDrawerOpen = false)}>Done</Button>
		{:else}
			<Button
				size="lg"
				class="w-full"
				icon={icons.plus}
				disabled={tokenSaving || !newTokenName.trim()}
				onclick={createToken}
			>
				{tokenSaving ? 'Generating…' : 'Generate token'}
			</Button>
		{/if}
	{/snippet}
</Drawer>

<ConfirmModal
	open={revoking !== null}
	title="Revoke this token?"
	description={revoking
		? `Anything still authenticating with “${revoking.name}” starts failing immediately, and it cannot be un-revoked.`
		: ''}
	confirmLabel="Revoke"
	tone="danger"
	onConfirm={revokeToken}
	onCancel={() => (revoking = null)}
/>

<ConfirmModal
	bind:open={loggingOut}
	title="Log out of Nuage?"
	description="Your files stay where they are. You will need to sign in again on this browser."
	confirmLabel="Log out"
	onConfirm={logout}
/>

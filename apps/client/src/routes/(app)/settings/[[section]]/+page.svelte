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
	import { ApiError, backend, type ApiToken, type Space, type UserProfile } from '$lib/backend';
	import { theme, type ThemePreference } from '$lib/theme.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { formatDate } from '$lib/format';
	import AppDataSection from '$lib/components/AppDataSection.svelte';
	import { icons, nuage } from '$lib/icons';

	const app = getContext<{
		token: string;
		user: UserProfile | null;
		setUser: (u: UserProfile) => void;
	}>('app');

	const SECTIONS = [
		{ id: 'profile', icon: icons.userCircle },
		{ id: 'appearance', icon: icons.palette },
		{ id: 'api', icon: icons.key },
		{ id: 'apps', icon: icons.widget },
		{ id: 'nook', icon: icons.bolt },
		{ id: 'advanced', icon: icons.server }
	] as const;

	/*
	 * The section is a real route, so a link to /settings/api opens on API, a reload keeps you
	 * there and browser-back walks the sections — none of which local `$state` tabs give you
	 * (CHARTE §14). An unknown slug falls back to Profile rather than rendering nothing.
	 */
	const section = $derived(
		SECTIONS.some((s) => s.id === page.params.section) ? page.params.section! : 'profile'
	);
	const tabs = $derived(
		SECTIONS.map((s) => ({ ...s, label: t(`settings.tab.${s.id}`), href: `/settings/${s.id}` }))
	);

	let profileName = $state('');
	let profileSaving = $state(false);

	const ACCOUNT_URL = 'https://sso.facile.studio/account';
	let avatarUrl = $derived(app.user?.avatar_url ?? '');

	let apiTokens = $state<ApiToken[]>([]);
	let spaces = $state<Space[]>([]);
	let tokenDrawerOpen = $state(false);
	let newTokenName = $state('');
	let createdToken = $state<string | null>(null);
	let tokenSaving = $state(false);
	let revoking = $state<ApiToken | null>(null);

	let instanceName = $state('');
	let isAdmin = $state(false);
	let trashRetention = $state('30');
	let nookWebhookUrl = $state('');
	let nookSecret = $state('');
	let nookEnabled = $state(false);
	let settingsSaving = $state(false);
	let testingNook = $state(false);
	let nookTestResult = $state<{ success: boolean; message?: string } | null>(null);

	let loggingOut = $state(false);

	onMount(async () => {
		profileName = app.user?.name ?? '';
		await Promise.all([loadSettings(), loadApiTokens(), loadSpaces()]);
	});

	async function loadSettings() {
		try {
			const settings = await backend.getSettings(app.token);
			instanceName = settings.instance_name ?? '';
			nookWebhookUrl = settings.nook_webhook_url ?? '';
			nookSecret = settings.nook_webhook_secret ?? '';
			nookEnabled = settings.nook_enabled === 'true';
			trashRetention = settings.trash_retention_days ?? '30';
			isAdmin = true;
		} catch (e) {
			if (!(e instanceof ApiError && e.status === 403)) toast.danger(t('settings.loadFailed'));
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

	async function loadSpaces() {
		try {
			const res = await backend.listSpaces(app.token);
			spaces = res.spaces ?? [];
		} catch {
			spaces = [];
		}
	}

	async function saveProfile() {
		profileSaving = true;
		try {
			const res = await backend.updateProfile(app.token, { name: profileName });
			app.setUser(res.user);
			toast.success(t('settings.profileUpdated'));
		} catch (e) {
			toast.danger(message(e, t('settings.profileFailed')));
		}
		profileSaving = false;
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
			toast.danger(message(e, t('settings.tokens.createFailed')));
		}
		tokenSaving = false;
	}

	async function revokeToken() {
		const target = revoking;
		if (!target) return;
		await backend.deleteApiToken(app.token, target.id);
		await loadApiTokens();
		revoking = null;
		toast.success(t('settings.tokens.revoked', { name: target.name }));
	}

	async function saveSettings() {
		settingsSaving = true;
		try {
			const values: Record<string, string> = {
				instance_name: instanceName,
				nook_webhook_url: nookWebhookUrl,
				nook_enabled: String(nookEnabled),
				trash_retention_days: trashRetention.trim()
			};
			if (!nookSecret.startsWith('*')) values.nook_webhook_secret = nookSecret;
			await backend.updateSettings(app.token, values);
			toast.success(t('settings.saved'));
		} catch (e) {
			toast.danger(message(e, t('settings.saveFailed')));
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
			nookTestResult = { success: false, message: message(e, t('settings.nook.failed')) };
		}
		testingNook = false;
	}

	/*
	 * The cookie is the half that matters. An SSO login never wrote a bearer token
	 * to localStorage, so clearing it alone leaves the session standing and the
	 * next page load signs the user straight back in — a logout button that does
	 * nothing. The redirect happens either way: a logout that fails server-side
	 * must still get them off the app.
	 */
	async function logout() {
		try {
			await backend.logout();
		} catch {}
		goto('/login');
	}

	function message(error: unknown, fallback: string): string {
		return error instanceof Error && error.message ? error.message : fallback;
	}

	const themeOptions = $derived([
		{ value: 'system', label: t('settings.theme.system'), icon: icons.monitor },
		{ value: 'light', label: t('settings.theme.light'), icon: icons.sun },
		{ value: 'dark', label: t('settings.theme.dark'), icon: icons.moon }
	]);

	/*
	 * Not a boolean: "not connected" hides disabled, never-tested and failing behind one word,
	 * and those have three different fixes (CHARTE §14).
	 */
	const nookStatus = $derived.by(() => {
		if (!nookEnabled) return { tone: 'neutral' as const, label: t('settings.nook.disabled'), pulse: false };
		if (testingNook) return { tone: 'info' as const, label: t('settings.nook.testing'), pulse: true };
		if (!nookWebhookUrl) return { tone: 'warning' as const, label: t('settings.nook.noUrl'), pulse: false };
		if (nookTestResult?.success) return { tone: 'success' as const, label: t('settings.nook.reached'), pulse: false };
		if (nookTestResult) {
			return { tone: 'danger' as const, label: nookTestResult.message ?? t('settings.nook.noAnswer'), pulse: false };
		}
		return { tone: 'neutral' as const, label: t('settings.nook.untested'), pulse: false };
	});
</script>

<svelte:head>
	<title>{t('settings.title')}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-10 px-4 py-6 md:px-8">
	<div class="flex flex-col gap-4">
		<h1 class="text-fc-2xl font-semibold text-fc-fg">{t('settings.heading')}</h1>
		<Tabs items={tabs} value={section} label={t('settings.sections')} />
		<Divider />
	</div>

	{#if section === 'profile'}
		<SettingsSection title={t('settings.picture.title')} description={t('settings.picture.description')}>
			<div class="flex flex-wrap items-center gap-4">
				<Avatar name={profileName || app.user?.email || ''} src={avatarUrl} size="lg" />
				<div class="flex min-w-0 flex-1 flex-col gap-1">
					<p class="text-fc-sm text-fc-fg">{t('settings.picture.source')}</p>
					<p class="text-fc-xs text-fc-fg-muted">{t('settings.picture.hint')}</p>
				</div>
				<Button variant="outline" icon={nuage.camera} href={ACCOUNT_URL} target="_blank" rel="noopener">
					{t('settings.picture.change')}
				</Button>
			</div>
		</SettingsSection>

		<SettingsSection title={t('settings.identity.title')} description={t('settings.identity.description')}>
			<SettingsRow label={t('settings.displayName')} stacked>
				<Field>
					<Input bind:value={profileName} placeholder={t('settings.namePlaceholder')} />
				</Field>
			</SettingsRow>
			<SettingsRow label={t('common.email')} description={t('settings.emailManaged')} stacked>
				<Field>
					<Input value={app.user?.email ?? ''} disabled />
				</Field>
			</SettingsRow>
			<SettingsRow>
				<Button icon={icons.check} disabled={profileSaving} onclick={saveProfile}>
					{profileSaving ? t('common.saving') : t('settings.saveProfile')}
				</Button>
			</SettingsRow>
		</SettingsSection>

		<SettingsSection title={t('settings.session.title')} description={t('settings.session.description')}>
			<SettingsRow label={t('settings.logout')} description={t('settings.logout.description')}>
				<Button variant="outline" icon={icons.logout} onclick={() => (loggingOut = true)}>
					{t('settings.logout')}
				</Button>
			</SettingsRow>
		</SettingsSection>
	{:else if section === 'appearance'}
		<SettingsSection title={t('settings.theme.title')} description={t('settings.theme.description')}>
			<SettingsRow label={t('settings.theme.scheme')} description={t('settings.theme.schemeHint')} stacked>
				<OptionCards
					options={themeOptions}
					value={theme.preference}
					label={t('settings.theme.scheme')}
					onSelect={(value) => theme.set(value as ThemePreference)}
				/>
			</SettingsRow>
		</SettingsSection>
	{:else if section === 'api'}
		<SettingsSection
			title={t('settings.tokens.title')}
			description={t('settings.tokens.description')}
		>
			{#snippet actions()}
				<Button icon={icons.plus} onclick={openTokenDrawer}>{t('settings.tokens.new')}</Button>
			{/snippet}

			{#if apiTokens.length === 0}
				<EmptyState
					bare
					icon={icons.key}
					title={t('settings.tokens.emptyTitle')}
					description={t('settings.tokens.emptyText')}
				>
					<Button icon={icons.plus} onclick={openTokenDrawer}>{t('settings.tokens.new')}</Button>
				</EmptyState>
			{:else}
				<Table>
					<thead>
						<tr>
							<th scope="col">{t('common.name')}</th>
							<th scope="col">{t('settings.tokens.created')}</th>
							<th scope="col" class="text-right"><span class="sr-only">{t('settings.tokens.actions')}</span></th>
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
										{t('common.revoke')}
									</Button>
								</td>
							</tr>
						{/each}
					</tbody>
				</Table>
			{/if}
		</SettingsSection>

		<SettingsSection title={t('settings.webdav.title')} description={t('settings.webdav.description')}>
			<SettingsRow label={t('settings.webdav.personal')} description={t('settings.webdav.personalText')} stacked>
				<SecretField value={`${page.url.origin}/webdav`} sensitive={false} />
			</SettingsRow>
			{#each spaces as space (space.id)}
				<SettingsRow label={space.name} description={t('settings.webdav.space')} stacked>
					<SecretField value={`${page.url.origin}/webdav/spaces/${space.id}`} sensitive={false} />
				</SettingsRow>
			{/each}
			<SettingsRow label={t('settings.webdav.index')} description={t('settings.webdav.indexText')} stacked>
				<SecretField value={`${page.url.origin}/webdav/spaces/`} sensitive={false} />
			</SettingsRow>
		</SettingsSection>
	{:else if section === 'apps'}
		<AppDataSection token={app.token} />
	{:else if section === 'nook'}
		<SettingsSection
			title="Nook"
			description={t('settings.nook.description')}
		>
			<SettingsRow label={t('settings.nook.status')}>
				<StatusDot tone={nookStatus.tone} label={nookStatus.label} pulse={nookStatus.pulse} />
			</SettingsRow>
			<SettingsRow label={t('settings.nook.enabled')} description={t('settings.nook.enabledText')}>
				<Switch bind:checked={nookEnabled} aria-label={t('settings.nook.switchLabel')} />
			</SettingsRow>
			<SettingsRow label={t('settings.nook.url')} stacked>
				<Field>
					<Input bind:value={nookWebhookUrl} type="url" placeholder="https://nook.example.com/hooks/nuage" />
				</Field>
			</SettingsRow>
			<SettingsRow label={t('settings.nook.secret')} description={t('settings.nook.secretText')} stacked>
				<SecretField bind:value={nookSecret} editable placeholder={t('settings.nook.notSet')} />
			</SettingsRow>
			<SettingsRow>
				<div class="flex flex-wrap gap-2">
					<Button icon={icons.check} disabled={settingsSaving} onclick={saveSettings}>
						{settingsSaving ? t('common.saving') : t('common.save')}
					</Button>
					<Button
						variant="outline"
						icon={icons.bolt}
						disabled={testingNook || !nookWebhookUrl}
						onclick={testNookConnection}
					>
						{t('settings.nook.test')}
					</Button>
				</div>
			</SettingsRow>
		</SettingsSection>
	{:else}
		<SettingsSection title={t('settings.instance.title')} description={t('settings.instance.description')}>
			<SettingsRow label={t('settings.instance.name')} stacked>
				<Field>
					<Input bind:value={instanceName} placeholder={t('settings.instancePlaceholder')} />
				</Field>
			</SettingsRow>
			{#if isAdmin}
				<SettingsRow
					label={t('settings.retention')}
					description={t('settings.retention.text')}
					stacked
				>
					<Field>
						<Input
							bind:value={trashRetention}
							type="number"
							min="0"
							max="3650"
							step="1"
							inputmode="numeric"
							aria-label={t('settings.retention.label')}
						/>
					</Field>
				</SettingsRow>
			{/if}
			<SettingsRow>
				<Button icon={icons.check} disabled={settingsSaving} onclick={saveSettings}>
					{settingsSaving ? t('common.saving') : t('common.save')}
				</Button>
			</SettingsRow>
		</SettingsSection>

		<SettingsSection title={t('settings.about.title')} description={t('settings.about.description')}>
			<SettingsRow label={t('settings.about.docs')} description={t('settings.about.docsText')}>
				<Button variant="outline" href="/docs" data-sveltekit-reload iconRight={icons.arrow}>{t('settings.about.open')}</Button>
			</SettingsRow>
			<SettingsRow label={t('settings.about.created')}>
				<span class="text-fc-sm text-fc-fg-muted">
					{app.user?.created_at ? formatDate(app.user.created_at) : '—'}
				</span>
			</SettingsRow>
		</SettingsSection>
	{/if}
</div>

<Drawer bind:open={tokenDrawerOpen} title={t('settings.tokens.drawerTitle')} showClose>
	{#if createdToken}
		<div class="flex flex-col gap-4">
			<Alert tone="warning" title={t('settings.tokens.copyTitle')}>{t('settings.tokens.copyText')}</Alert>
			<SecretField value={createdToken} autoHideMs={0} />
		</div>
	{:else}
		<Field label={t('common.name')} helper={t('settings.tokens.nameHelper')}>
			<Input bind:value={newTokenName} placeholder={t('settings.tokens.namePlaceholder')} />
		</Field>
	{/if}

	{#snippet footer()}
		{#if createdToken}
			<Button size="lg" class="w-full" onclick={() => (tokenDrawerOpen = false)}>{t('common.done')}</Button>
		{:else}
			<Button
				size="lg"
				class="w-full"
				icon={icons.plus}
				disabled={tokenSaving || !newTokenName.trim()}
				onclick={createToken}
			>
				{tokenSaving ? t('settings.tokens.generating') : t('settings.tokens.generate')}
			</Button>
		{/if}
	{/snippet}
</Drawer>

<ConfirmModal
	open={revoking !== null}
	title={t('settings.tokens.revokeTitle')}
	description={revoking ? t('settings.tokens.revokeText', { name: revoking.name }) : ''}
	confirmLabel={t('common.revoke')}
	tone="danger"
	onConfirm={revokeToken}
	onCancel={() => (revoking = null)}
/>

<ConfirmModal
	bind:open={loggingOut}
	title={t('settings.logout.confirmTitle')}
	description={t('settings.logout.confirmText')}
	confirmLabel={t('settings.logout')}
	onConfirm={logout}
/>

<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { SecretField, SettingsRow, SettingsSection, TokensSection, toast, type TokenDraft, type TokenItem } from '@facile/muse';
	import { backend, type ApiToken, type Space } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let { token }: { token: string } = $props();

	let apiTokens = $state<ApiToken[]>([]);
	let spaces = $state<Space[]>([]);

	const items = $derived<TokenItem[]>(apiTokens.map((tok) => ({ id: String(tok.id), name: tok.name })));

	onMount(() => Promise.all([loadApiTokens(), loadSpaces()]));

	async function loadApiTokens() {
		try {
			apiTokens = (await backend.getApiToken(token)).tokens ?? [];
		} catch {
			apiTokens = [];
		}
	}

	async function loadSpaces() {
		try {
			spaces = (await backend.listSpaces(token)).spaces ?? [];
		} catch {
			spaces = [];
		}
	}

	async function createToken(draft: Omit<TokenDraft, 'days'>): Promise<string> {
		const res = await backend.createApiToken(token, { name: draft.name });
		await loadApiTokens();
		return res.token ?? '';
	}

	async function revokeToken(target: TokenItem) {
		await backend.deleteApiToken(token, Number(target.id));
		await loadApiTokens();
		toast.success(t('settings.tokens.revoked', { name: target.name }));
	}
</script>

<TokensSection tokens={items} expiry={false} lastUsed={false} onCreate={createToken} onRevoke={revokeToken}>
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
</TokensSection>

<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { Alert, Button, EmptyState, SecretField, Spinner, toast } from '@facile/muse';
	import { ApiError, backend, type PendingAction } from '$lib/backend';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import PendingCard from '$lib/components/PendingCard.svelte';
	import { icons } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';
	import { refusesOwnSession } from '$lib/session-refusal';

	let actions = $state<PendingAction[]>([]);
	let links = $state<{ id: string; name: string; url: string }[]>([]);
	let loading = $state(true);
	let busy = $state('');
	let signInAgain = $state(false);

	const focused = $derived(page.url.searchParams.get('action') ?? '');
	const ordered = $derived([
		...actions.filter((action) => action.id === focused),
		...actions.filter((action) => action.id !== focused)
	]);

	onMount(load);

	/**
	 * porte's own-session refusal means the API would not take this credential for a confirmation:
	 * a cookie from before the browser-session mark, or a CLI or app session. The person signs in
	 * again in the browser. Any other 403 is the server's own answer and keeps its message.
	 */
	function refused(err: unknown): boolean {
		signInAgain = refusesOwnSession(err);
		return signInAgain;
	}

	/** The server's own words for any other 403, else the fallback text. */
	function reason(err: unknown, fallback: string): string {
		return err instanceof ApiError && err.status === 403 ? err.message : fallback;
	}

	/** Keeps this page for the return after the sign-in, as the app layout does. */
	function keepReturn() {
		try {
			sessionStorage.setItem('nuage.return', page.url.pathname + page.url.search);
		} catch {}
	}

	/** Loads the open actions. */
	async function load() {
		loading = true;
		try {
			actions = (await backend.listPending()).actions;
		} catch (err) {
			if (!refused(err)) toast.danger(reason(err, t('pending.loadFailed')));
		}
		loading = false;
	}

	/** Runs one action once, then drops it from the list; a new link stays shown. */
	async function confirm(action: PendingAction) {
		busy = action.id;
		try {
			const result = await backend.confirmPending(action.id);
			if (result.url) links = [...links, { id: action.id, name: action.item?.name ?? '', url: result.url }];
			toast.success(t(action.kind === 'share' ? 'pending.shared' : 'pending.trashed'));
			actions = actions.filter((item) => item.id !== action.id);
		} catch (err) {
			if (refused(err)) {
				busy = '';
				return;
			}
			toast.danger(err instanceof Error ? err.message : t('pending.failed'));
			await load();
		}
		busy = '';
	}

	/** Closes one action without running it. */
	async function dismiss(action: PendingAction) {
		busy = action.id;
		try {
			await backend.dismissPending(action.id);
			toast.success(t('pending.dismissed'));
			actions = actions.filter((item) => item.id !== action.id);
		} catch (err) {
			if (!refused(err)) toast.danger(err instanceof Error ? err.message : t('pending.failed'));
		}
		busy = '';
	}
</script>

<svelte:head>
	<title>{t('pending.title')}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-2xl flex-col gap-6 px-4 py-6 md:px-8">
	<PageHeader title={t('pending.heading')} description={t('pending.intro')} />

	{#if signInAgain}
		<Alert tone="warning" title={t('pending.signInAgain')}>
			<div class="flex flex-col items-start gap-3">
				<p>{t('pending.signInAgainBody')}</p>
				<Button href="{backend.baseUrl}/auth/oidc" size="sm" onclick={keepReturn}>{t('pending.signIn')}</Button>
			</div>
		</Alert>
	{/if}

	{#each links as link (link.id)}
		<SecretField label={`${t('pending.newLink')} · ${link.name}`} value={link.url} sensitive={false} />
	{/each}

	{#if loading}
		<div class="flex h-48 items-center justify-center"><Spinner /></div>
	{:else if actions.length === 0}
		<EmptyState icon={icons.check} title={t('pending.none')} description={t('pending.noneBody')} />
	{:else}
		<section class="flex flex-col gap-4">
			{#each ordered as action (action.id)}
				<PendingCard {action} busy={busy === action.id} onconfirm={confirm} ondismiss={dismiss} />
			{/each}
		</section>
	{/if}
</div>

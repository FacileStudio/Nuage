<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Button, MobileNav, SideBar } from '@facile/muse';
	import { backend, isAuthError, type QuotaResponse, type Space, type UserProfile } from '$lib/backend';
	import { rememberLocale, setSessionLocale, t } from '$lib/i18n/index.svelte';
	import { theme, type ThemePreference } from '$lib/theme.svelte';
	import { planName } from '$lib/plan';
	import { REGISTRE_SPACES_URL } from '$lib/registre';
	import { hasPending, undoLast } from '$lib/undo.svelte';
	import { getSpaceStore } from '$lib/space.svelte';
	import { icons, nuage } from '$lib/icons';
	import QuotaMeter from '$lib/components/QuotaMeter.svelte';
	import PlanLimitBanner from '$lib/components/PlanLimitBanner.svelte';

	let { children } = $props();

	const TOKEN_KEY = 'nuage.token';

	let token = $state('');
	let user = $state<UserProfile | null>(null);
	let loaded = $state(false);
	let failure = $state('');
	let quota = $state<QuotaResponse | null>(null);
	let spaces = $state<Space[]>([]);
	let collapsed = $state(false);

	function setUser(nextUser: UserProfile) {
		user = nextUser;
		setSessionLocale(nextUser.locale);
	}

	async function refreshQuota() {
		try {
			quota = await backend.getQuota(token);
		} catch {
			quota = null;
		}
	}

	const spaceStore = getSpaceStore();

	/*
	 * The space list lives here rather than in the switcher, because muse's `SideBar` renders
	 * the switcher itself — the rail owns the collapse transition that a fixed-width control
	 * inside it cannot survive on its own (CHARTE §10).
	 */
	async function loadSpaces() {
		try {
			const res = await backend.listSpaces(token);
			spaces = res.spaces ?? [];
		} catch {
			spaces = [];
			return;
		}

		const savedId = spaceStore.getSavedId();
		if (savedId === null) return;
		const match = spaces.find((s) => s.id === savedId);
		if (match) spaceStore.set(match);
		else spaceStore.clear();
	}

	function selectSpace(id: string | null) {
		const next = id === null ? null : (spaces.find((s) => String(s.id) === id) ?? null);
		if (next?.id === spaceStore.id) return;
		spaceStore.set(next);
		if (page.url.searchParams.has('folder')) void goto('/files', { replaceState: true });
	}

	setContext('app', {
		get token() { return token; },
		get user() { return user; },
		get space() { return spaceStore.current; },
		get spaceId() { return spaceStore.id; },
		get quota() { return quota; },
		setUser,
		refreshQuota
	});

	function handleUndoKeydown(e: KeyboardEvent) {
		if (e.repeat) return;
		if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;
		const mod = navigator.platform.includes('Mac') ? e.metaKey : e.ctrlKey;
		if (mod && e.key === 'z' && !e.shiftKey && hasPending()) {
			e.preventDefault();
			undoLast();
		}
	}

	/*
	 * The gate is the API's answer. An SSO callback signs the browser in with an HttpOnly cookie
	 * and no bearer token. A token left in localStorage by the retired password login is dropped:
	 * sent as a bearer header it would shadow the cookie once it expired.
	 *
	 * Only a refusal logs anyone out. A 500 or a dead connection says nothing
	 * about the session, and throwing it away over one strands the user on the
	 * login page holding a credential that was fine.
	 */
	onMount(() => {
		document.addEventListener('keydown', handleUndoKeydown);

		(async () => {
			localStorage.removeItem(TOKEN_KEY);
			try {
				const result = await backend.me();
				setUser(result.user);
				loaded = true;
				refreshQuota();
				loadSpaces();
			} catch (error) {
				if (isAuthError(error)) { goto('/login'); return; }
				failure = error instanceof Error && error.message ? error.message : t('common.unreachable');
			}
		})();

		return () => document.removeEventListener('keydown', handleUndoKeydown);
	});

	function isActive(href: string) {
		return page.url.pathname === href || page.url.pathname.startsWith(href + '/');
	}

	/*
	 * Settings is deliberately absent: it hangs off the user card via `userHref`, not off a
	 * permanent nav row that spends prime vertical space on the page people open least
	 * (CHARTE §14).
	 */
	const navPages = $derived([
		{ href: '/files', label: t('nav.files'), short: t('nav.files'), icon: icons.folder },
		{ href: '/shared', label: t('nav.shared'), short: t('nav.sharedShort'), icon: nuage.share },
		{ href: '/trash', label: t('nav.trash'), short: t('nav.trash'), icon: icons.remove },
		{ href: '/activity', label: t('nav.activity'), short: t('nav.activity'), icon: icons.history },
		{ href: '/spaces', label: t('nav.spaces'), short: t('nav.spaces'), icon: icons.space }
	].map((item) => ({ ...item, active: isActive(item.href) })));

	/* MobileNav's ceiling is six targets plus the avatar and its labels are tooltips, so the
	   bar takes the short spelling of the same destinations. */
	const mobilePages = $derived(navPages.map((item) => ({ ...item, label: item.short })));

	/**
	 * Clears the browser session. The cookie is the half that matters: an SSO sign-in never wrote a
	 * bearer token, so clearing local state alone would sign the person straight back in. The
	 * redirect happens either way, so a failed server call still leaves the app.
	 */
	async function signOut() {
		try {
			await backend.logout();
		} catch {}
		void goto('/login');
	}

	const userMenu = $derived({
		subscription: { plan: planName(quota), href: REGISTRE_SPACES_URL },
		helpHref: `${page.url.origin}/docs`,
		onSignOut: signOut,
		theme: theme.preference,
		onThemeChange: (next: ThemePreference) => theme.set(next),
		onLocaleChange: rememberLocale
	});

	const onSettings = $derived(isActive('/settings'));
	const identity = $derived(
		user
			? { name: user.name?.trim() || user.email, email: user.email, avatar: user.avatar_url || undefined }
			: undefined
	);
	const switcherSpaces = $derived(spaces.map((s) => ({ id: String(s.id), name: s.name })));
	const activeSpaceId = $derived(spaceStore.id === null ? null : String(spaceStore.id));
</script>

{#if loaded}
	<!-- One scroll container for the app, and it is the `<main>` below — the shell itself never
	     scrolls, so a flick past either end has nothing to chain into (CHARTE §7). -->
	<div class="flex h-dvh w-full gap-3 overflow-hidden p-3">
		<!--
		  The rail animates its own width from the tokens; this column matches it on the same
		  duration and curve so the quota card underneath tracks the tween instead of snapping
		  when it lands.
		-->
		<div
			class="hidden h-full min-h-0 shrink-0 flex-col gap-3 transition-[width] duration-300 ease-[var(--ease-fc)] md:flex"
			style="width: var({collapsed ? '--width-fc-nav-collapsed' : '--width-fc-nav-expanded'})"
		>
			<SideBar
				class="min-h-0 w-full flex-1"
				icon={nuage.brand}
				title="Nuage"
				bind:collapsed
				pages={navPages.filter((p) => p.href !== '/spaces')}
				spaces={switcherSpaces}
				{activeSpaceId}
				onSpaceSelect={selectSpace}
				manageSpacesHref="/spaces"
				user={identity}
				userHref="/settings"
				userActive={onSettings}
				{userMenu}
			/>
			{#if quota}
				<QuotaMeter {quota} {collapsed} />
			{/if}
		</div>

		<main class="min-w-0 flex-1 overflow-auto overscroll-contain rounded-fc-lg pb-24 md:pb-0">
			<PlanLimitBanner />
			{@render children()}
		</main>
	</div>

	<MobileNav items={mobilePages} user={identity} profileHref="/settings" profileActive={onSettings} {userMenu} />
{:else if failure}
	<div class="flex h-dvh flex-col items-center justify-center gap-3 p-6 text-center">
		<p class="text-fc-sm text-fc-fg-muted">{failure}</p>
		<Button variant="secondary" onclick={() => location.reload()}>{t('common.retry')}</Button>
	</div>
{/if}

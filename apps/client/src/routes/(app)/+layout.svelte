<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { MobileNav, SideBar } from '@facile/muse';
	import { backend, type QuotaResponse, type Space, type UserProfile } from '$lib/backend';
	import { hasPending, undoLast } from '$lib/undo.svelte';
	import { getSpaceStore } from '$lib/space.svelte';
	import { icons, nuage } from '$lib/icons';
	import QuotaMeter from '$lib/components/QuotaMeter.svelte';

	let { children } = $props();

	const TOKEN_KEY = 'nuage.token';

	let token = $state('');
	let user = $state<UserProfile | null>(null);
	let loaded = $state(false);
	let quota = $state<QuotaResponse | null>(null);
	let spaces = $state<Space[]>([]);
	let collapsed = $state(false);

	function setUser(nextUser: UserProfile) {
		user = nextUser;
	}

	async function refreshQuota() {
		if (!token) return;
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
		spaceStore.set(id === null ? null : (spaces.find((s) => String(s.id) === id) ?? null));
	}

	setContext('app', {
		get token() { return token; },
		get user() { return user; },
		get space() { return spaceStore.current; },
		get spaceId() { return spaceStore.id; },
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

	onMount(() => {
		document.addEventListener('keydown', handleUndoKeydown);

		(async () => {
			const stored = localStorage.getItem(TOKEN_KEY) ?? '';
			if (!stored) { goto('/login'); return; }
			try {
				const result = await backend.me(stored);
				token = stored;
				user = result.user;
				loaded = true;
				refreshQuota();
				loadSpaces();
				backend.syncProfile(stored).then(({ synced }) => {
					if (synced) backend.me(stored).then((r) => { user = r.user; });
				}).catch(() => {});
			} catch {
				goto('/login');
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
		{ href: '/files', label: 'Files', icon: icons.folder },
		{ href: '/shared', label: 'Shared links', icon: nuage.share },
		{ href: '/trash', label: 'Trash', icon: icons.remove },
		{ href: '/activity', label: 'Activity', icon: icons.history },
		{ href: '/spaces', label: 'Spaces', icon: icons.usersGroup }
	].map((item) => ({ ...item, active: isActive(item.href) })));

	/* MobileNav's ceiling is six targets plus the avatar and its labels are tooltips, so the
	   bar takes the short spelling of the same destinations. */
	const mobilePages = $derived(navPages.map((item) => ({ ...item, label: item.label.split(' ')[0] })));

	const onSettings = $derived(isActive('/settings'));
	const identity = $derived(
		user ? { name: user.name?.trim() || user.email, avatar: user.avatar_url || undefined } : undefined
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
				pages={navPages}
				spaces={switcherSpaces}
				{activeSpaceId}
				onSpaceSelect={selectSpace}
				manageSpacesHref="/spaces"
				user={identity}
				userHref="/settings"
				userActive={onSettings}
			/>
			{#if quota}
				<QuotaMeter {quota} {collapsed} />
			{/if}
		</div>

		<main class="min-w-0 flex-1 overflow-auto overscroll-contain rounded-fc-lg pb-24 md:pb-0">
			{@render children()}
		</main>
	</div>

	<MobileNav items={mobilePages} user={identity} profileHref="/settings" profileActive={onSettings} />
{/if}

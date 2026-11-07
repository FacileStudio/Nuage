<script lang="ts">
	import { getContext } from 'svelte';
	import { Avatar, Button, EmptyState, Spinner, toast } from '@facile/muse';
	import { backend, type ActivityEntry } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';
	import { formatDateTime, relativeTime } from '$lib/format';
	import { icons, nuage } from '$lib/icons';
	import { locale, t } from '$lib/i18n/index.svelte';
	import { displayName } from '$lib/people';
	import PageHeader from '$lib/components/PageHeader.svelte';

	const app = getContext<{ token: string }>('app');
	const spaceStore = getSpaceStore();

	const PER_PAGE = 30;

	let activities = $state<ActivityEntry[]>([]);
	let loading = $state(true);
	let loadingMore = $state(false);
	let currentPage = $state(1);
	let total = $state(0);

	let hasMore = $derived(activities.length < total);

	$effect(() => {
		const _spaceId = spaceStore.id;
		loadActivity();
	});

	async function loadActivity() {
		loading = true;
		try {
			const res = await backend.listActivity(app.token, {
				page: 1,
				per_page: PER_PAGE,
				space_id: spaceStore.id
			});
			activities = res.activities ?? [];
			total = res.total;
			currentPage = 1;
		} catch {
			activities = [];
			total = 0;
			toast.danger(t('activity.loadFailed'));
		}
		loading = false;
	}

	async function loadMore() {
		if (loadingMore || !hasMore) return;
		loadingMore = true;
		try {
			const nextPage = currentPage + 1;
			const res = await backend.listActivity(app.token, {
				page: nextPage,
				per_page: PER_PAGE,
				space_id: spaceStore.id
			});
			activities = [...activities, ...(res.activities ?? [])];
			total = res.total;
			currentPage = nextPage;
		} catch {
			toast.danger(t('activity.moreFailed'));
		}
		loadingMore = false;
	}

	type Tone = 'neutral' | 'info' | 'success' | 'warning' | 'danger';

	/*
	 * One table for glyph, wording and tone, so a new event type cannot pick up an icon in one
	 * place and a colour in another. The tones are the shared vocabulary (CHARTE §2) — the old
	 * map reached straight for stock Tailwind emerald and sky steps, fixed values that do not
	 * move with the theme and rendered as bright confetti in dark mode.
	 */
	const EVENTS = {
		'file.uploaded': { icon: icons.upload, tone: 'success' },
		'file.deleted': { icon: icons.remove, tone: 'warning' },
		'file.updated': { icon: nuage.rename, tone: 'info' },
		'file.versioned': { icon: nuage.versions, tone: 'info' },
		'file.restored': { icon: nuage.restore, tone: 'success' },
		'file.permanently_deleted': { icon: icons.remove, tone: 'danger' },
		'file.expired': { icon: icons.remove, tone: 'danger' },
		'folder.created': { icon: nuage.newFolder, tone: 'success' },
		'folder.updated': { icon: nuage.rename, tone: 'info' },
		'folder.deleted': { icon: icons.remove, tone: 'warning' },
		'folder.restored': { icon: nuage.restore, tone: 'success' },
		'folder.permanently_deleted': { icon: icons.remove, tone: 'danger' },
		'folder.expired': { icon: icons.remove, tone: 'danger' },
		'share.created': { icon: nuage.share, tone: 'info' },
		'share.revoked': { icon: nuage.shareOff, tone: 'warning' }
	} as const satisfies Record<string, { icon: string; tone: Tone }>;

	type EventType = keyof typeof EVENTS;

	const FALLBACK = { icon: icons.bolt, label: '', tone: 'neutral' as Tone };

	/* Tinted, never solid — the same 10% wash `Badge` and `Alert` use for the same tone. */
	const TINTS: Record<Tone, string> = {
		neutral: 'bg-fc-surface text-fc-fg-muted',
		info: 'bg-fc-info/10 text-fc-info',
		success: 'bg-fc-success/10 text-fc-success',
		warning: 'bg-fc-warning/10 text-fc-warning',
		danger: 'bg-fc-danger/10 text-fc-danger'
	};

	function describe(eventType: string): { icon: string; tone: Tone; label: string } {
		if (eventType in EVENTS) {
			const known = eventType as EventType;
			return { ...EVENTS[known], label: t(`activity.${known}`) };
		}
		return { ...FALLBACK, label: eventType };
	}

	function dateHeading(iso: string): string {
		const date = new Date(iso);
		const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
		const days = Math.floor((startOfDay(new Date()) - startOfDay(date)) / 86_400_000);

		if (days === 0) return t('activity.today');
		if (days === 1) return t('activity.yesterday');
		if (days < 7) return new Intl.DateTimeFormat(locale(), { weekday: 'long' }).format(date);
		return new Intl.DateTimeFormat(locale(), { dateStyle: 'long' }).format(date);
	}

	function groupByDate(entries: ActivityEntry[]): { heading: string; items: ActivityEntry[] }[] {
		const groups: { heading: string; items: ActivityEntry[] }[] = [];
		for (const entry of entries) {
			const heading = dateHeading(entry.created_at);
			if (groups.at(-1)?.heading !== heading) groups.push({ heading, items: [] });
			groups.at(-1)!.items.push(entry);
		}
		return groups;
	}

	let grouped = $derived(groupByDate(activities));
</script>

<svelte:head>
	<title>{t('activity.title')}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-6 md:px-8">
	<PageHeader title={t('activity.heading')} description={t('activity.description')} />

	{#if loading}
		<div class="flex h-64 items-center justify-center"><Spinner /></div>
	{:else if activities.length === 0}
		<EmptyState
			icon={icons.history}
			title={t('activity.emptyTitle')}
			description={t('activity.emptyText')}
		/>
	{:else}
		<div class="flex flex-col gap-10">
			{#each grouped as group (group.heading)}
				<section class="flex flex-col gap-4">
					<h2 class="text-fc-lg font-semibold text-fc-fg">{group.heading}</h2>
					<div class="flex flex-col gap-2">
						{#each group.items as entry (entry.id)}
							{@const event = describe(entry.event_type)}
							<div class="flex items-center gap-3 rounded-fc-md bg-fc-component p-3">
								<span
									class="flex size-8 shrink-0 items-center justify-center rounded-fc-pill {TINTS[event.tone]}"
								>
									<iconify-icon icon={event.icon} width="16" height="16" class="block"></iconify-icon>
								</span>
								<p class="min-w-0 flex-1 truncate text-fc-sm text-fc-fg">
									{#if entry.actor}<span class="text-fc-fg-muted">{displayName(entry.actor)} · </span>{/if}
									<span class="font-medium">{event.label}</span>
									{#if entry.resource_name}<span class="text-fc-fg-muted"> — {entry.resource_name}</span>{/if}
								</p>
								<time
									class="shrink-0 text-fc-xs text-fc-fg-muted"
									datetime={entry.created_at}
									title={formatDateTime(entry.created_at)}
								>
									{relativeTime(entry.created_at)}
								</time>
								{#if entry.actor}
									<Avatar
										name={displayName(entry.actor)}
										src={entry.actor.avatar_url || undefined}
										size="sm"
									/>
								{/if}
							</div>
						{/each}
					</div>
				</section>
			{/each}

			{#if hasMore}
				<div class="flex justify-center">
					<Button
						variant="outline"
						icon={icons.chevronDown}
						disabled={loadingMore}
						onclick={loadMore}
					>
						{loadingMore ? t('activity.loading') : t('activity.loadMore', { left: total - activities.length })}
					</Button>
				</div>
			{/if}
		</div>
	{/if}
</div>

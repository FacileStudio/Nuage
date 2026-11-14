<script lang="ts">
	import { flip } from 'svelte/animate';
	import { fade } from 'svelte/transition';
	import { Bar, Stage, Tile, type Act } from '@facile/mise-en-scene';
	import { Avatar, Icon, icons } from '@facile/muse';
	import { fileIcon } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

	const MEMBERS = ['Léa Martin', 'Sam Okafor', 'Inès Roy'];
	const TONES: Record<string, string> = {
		'Léa Martin': 'bg-(--color-fc-chart-3) text-white',
		'Sam Okafor': 'bg-(--color-fc-chart-2) text-white',
		'Inès Roy': 'bg-(--color-fc-chart-5) text-white'
	};
	const FOLDERS = ['56%', '44%', '62%'];
	const ARRIVALS = [
		{ file: 'pitch-deck.pdf', by: 'Sam Okafor' },
		{ file: 'logo-final.png', by: 'Inès Roy' }
	];

	let arrived = $state(0);

	const rows = $derived([
		...ARRIVALS.slice(0, arrived)
			.reverse()
			.map((row) => ({ key: row.file, ...row, w: '' })),
		...FOLDERS.map((w) => ({ key: w, file: '', by: '', w }))
	].slice(0, 3));

	function reset() {
		arrived = 0;
	}

	async function script(act: Act) {
		await act.wait(900);
		arrived = 1;
		await act.wait(1600);
		arrived = 2;
		await act.wait(2600);
		reset();
		await act.wait(700);
	}
</script>

<Tile title={t('landing.card.spaces.title')} body={t('landing.card.spaces.body')}>
	<Stage width={320} height={240} {script} {reset} poster={() => (arrived = 2)}>
		{#snippet children()}
			<div class="sp-panel absolute top-5 right-5 left-5 flex h-[190px] flex-col rounded-fc-lg bg-fc-page p-3.5">
				<div class="mb-2 flex items-center gap-2.5">
					<span class="flex size-8 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg-muted">
						<Icon icon={icons.space} size={16} />
					</span>
					<span class="min-w-0 flex-1 truncate text-fc-sm font-bold text-fc-fg">{t('landing.card.spaces.name')}</span>
					<span class="flex -space-x-1.5">
						{#each MEMBERS as name (name)}
							<span class="sp-ring rounded-fc-pill"><Avatar {name} size="sm" class="size-6 text-[10px] {TONES[name]}" /></span>
						{/each}
					</span>
				</div>
				<div class="flex flex-col gap-1">
					{#each rows as row (row.key)}
						<div animate:flip={{ duration: 360 }} in:fade={{ duration: 260 }} out:fade={{ duration: 160 }}>
							{#if row.file}
								<div class="flex h-9 items-center gap-2.5 rounded-fc-md bg-fc-surface px-2.5">
									<Icon icon={fileIcon(row.file)} size={16} class="shrink-0 text-fc-fg-muted" />
									<span class="min-w-0 flex-1 truncate text-fc-xs font-bold text-fc-fg">{row.file}</span>
									<Avatar name={row.by} size="sm" class="size-5 text-[9px] {TONES[row.by]}" />
								</div>
							{:else}
								<div class="flex h-9 items-center gap-2.5 px-2.5">
									<Icon icon={icons.folder} size={16} class="shrink-0 text-fc-fg-muted" />
									<Bar w={row.w} />
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{/snippet}
	</Stage>
</Tile>

<style>
	.sp-panel {
		overflow: hidden;
		box-shadow: 0 14px 30px -14px rgb(0 0 0 / 0.25);
	}
	.sp-ring {
		box-shadow: 0 0 0 2px var(--color-fc-page);
	}
</style>

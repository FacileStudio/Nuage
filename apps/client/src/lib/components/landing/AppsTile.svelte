<script lang="ts">
	import { Roll, Stage, Tile, type Act } from '@facile/mise-en-scene';
	import { Icon } from '@facile/muse';
	import { formatSize } from '$lib/format';
	import { fileIcon, nuage, suite } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

	const FILE = 'contrat-dupont.pdf';
	const MB = 1024 * 1024;

	let signed = $state(false);
	let chip = $state<'start' | 'moving' | 'done'>('start');
	let landed = $state(false);

	const apps = $derived([
		{ name: 'Plume', icon: suite.plume, tone: 'ap-plume', items: landed ? 4 : 3, bytes: (landed ? 1.3 : 1.1) * MB },
		{ name: 'Ardoise', icon: suite.ardoise, tone: 'ap-ardoise', items: 12, bytes: 2.8 * MB },
		{ name: 'Toile', icon: suite.toile, tone: 'ap-toile', items: 7, bytes: 5.4 * MB }
	]);

	function reset() {
		signed = false;
		chip = 'start';
		landed = false;
	}

	function poster() {
		signed = true;
		landed = true;
	}

	async function script(act: Act) {
		await act.wait(900);
		signed = true;
		await act.wait(1000);
		chip = 'moving';
		await act.wait(650);
		landed = true;
		await act.wait(250);
		chip = 'done';
		await act.wait(2600);
		signed = false;
		await act.wait(400);
		landed = false;
		await act.wait(700);
	}
</script>

<Tile title={t('landing.card.apps.title')} body={t('landing.card.apps.body')}>
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<div class="ap-card absolute top-4 right-6 left-6 flex h-14 items-center gap-2.5 rounded-fc-lg bg-fc-page px-3">
				<span class="ap-mark ap-plume"><Icon icon={suite.plume} size={16} /></span>
				<div class="min-w-0 flex-1">
					<div class="truncate text-fc-xs font-bold text-fc-fg">{FILE}</div>
					<div class="text-[11px] text-fc-fg-muted">Plume</div>
				</div>
				<span class="ap-badge" class:ap-show={signed}>{t('landing.card.apps.signed')}</span>
			</div>

			<div class="ap-chip ap-{chip}">
				<Icon icon={fileIcon(FILE)} size={14} />
			</div>

			<div class="ap-card absolute top-[88px] right-6 left-6 h-[136px] rounded-fc-lg bg-fc-page px-3 pt-2.5">
				<div class="mb-1.5 flex items-center gap-2 text-fc-xs font-bold text-fc-fg">
					<Icon icon={nuage.brand} size={14} />
					<span class="truncate">{t('appdata.title')}</span>
				</div>
				{#each apps as app (app.name)}
					<div class="ap-row" class:ap-lit={landed && app.name === 'Plume'}>
						<span class="ap-mark ap-small {app.tone}"><Icon icon={app.icon} size={13} /></span>
						<span class="min-w-0 flex-1 truncate text-fc-xs font-bold text-fc-fg">{app.name}</span>
						<span class="w-6 text-right text-[11px] text-fc-fg-muted"><Roll value={String(app.items)} /></span>
						<span class="w-14 text-right text-[11px] text-fc-fg-muted"><Roll value={formatSize(app.bytes)} /></span>
					</div>
				{/each}
			</div>
		{/snippet}
	</Stage>
</Tile>

<style>
	.ap-card {
		box-shadow: 0 14px 30px -14px rgb(0 0 0 / 0.25);
	}
	.ap-mark {
		display: flex;
		flex-shrink: 0;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border-radius: var(--radius-fc-md);
		color: white;
	}
	.ap-small {
		width: 24px;
		height: 24px;
		border-radius: var(--radius-fc-sm);
	}
	.ap-plume {
		background: var(--color-fc-chart-3);
	}
	.ap-ardoise {
		background: var(--color-fc-chart-2);
	}
	.ap-toile {
		background: var(--color-fc-chart-5);
	}
	.ap-badge {
		padding: 2px 8px;
		border-radius: 999px;
		background: color-mix(in oklab, var(--color-fc-success) 16%, transparent);
		color: var(--color-fc-success);
		font-size: 11px;
		font-weight: 700;
		opacity: 0;
		scale: 0.8;
		transition:
			opacity 0.25s ease,
			scale 0.3s cubic-bezier(0.2, 0.8, 0.2, 1);
	}
	.ap-show {
		opacity: 1;
		scale: 1;
	}
	.ap-row {
		position: relative;
		isolation: isolate;
		display: flex;
		align-items: center;
		gap: 8px;
		height: 32px;
		padding: 0 6px;
		border-radius: var(--radius-fc-md);
	}
	.ap-row::before {
		content: '';
		position: absolute;
		inset: 0;
		z-index: -1;
		border-radius: inherit;
		background: var(--color-fc-surface);
		opacity: 0;
		transition: opacity 0.4s ease;
	}
	.ap-lit::before {
		opacity: 1;
	}
	.ap-chip {
		position: absolute;
		top: 30px;
		left: 36px;
		z-index: 5;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: var(--radius-fc-sm);
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
		box-shadow: 0 8px 18px -6px rgb(0 0 0 / 0.35);
		opacity: 0;
		transition:
			translate 0.6s cubic-bezier(0.45, 0, 0.15, 1),
			opacity 0.2s ease;
	}
	.ap-moving {
		opacity: 1;
		translate: 0 92px;
	}
	.ap-done {
		translate: 0 92px;
	}
	@media (prefers-reduced-motion: reduce) {
		.ap-badge,
		.ap-row::before,
		.ap-chip {
			transition: none;
		}
	}
</style>

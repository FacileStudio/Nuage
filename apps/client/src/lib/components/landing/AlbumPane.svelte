<script lang="ts">
	import { fade, scale } from 'svelte/transition';
	import { Bar } from '@facile/mise-en-scene';
	import { Icon, icons } from '@facile/muse';
	import { nuage } from '$lib/icons';
	import Photo from './Photo.svelte';
	import { t } from '$lib/i18n/index.svelte';

	interface Props {
		compact: boolean;
		total: number;
		done: number;
		uploading: boolean;
		overlay: boolean;
		grid?: HTMLElement;
		shareButton?: HTMLElement;
	}

	let {
		compact,
		total,
		done,
		uploading,
		overlay,
		grid = $bindable(),
		shareButton = $bindable()
	}: Props = $props();

	const percent = $derived(Math.round((done / total) * 100));
	const NAMES = ['54%', '68%', '46%', '62%', '50%', '58%'];
</script>

<div class="ap-pane" class:ap-compact={compact}>
	<div class="ap-head">
		<nav class="ap-crumbs">
			<span class="ap-muted">{t('nav.files')}</span>
			<Icon icon={icons.arrow} size={14} class="ap-muted" />
			<span class="ap-here">{t('landing.album.folder')}</span>
		</nav>
		{#if !compact}
			<div class="ap-toggle">
				<span class="ap-mode"><Icon icon={nuage.list} size={14} /></span>
				<span class="ap-mode ap-on"><Icon icon={nuage.grid} size={14} /></span>
			</div>
		{/if}
	</div>

	<div class="ap-tools">
		<span class="ap-btn ap-primary"><Icon icon={icons.upload} size={14} />{t('files.upload')}</span>
		<span class="ap-btn ap-secondary" bind:this={shareButton}><Icon icon={nuage.share} size={14} />{t('files.menu.share')}</span>
		{#if uploading}
			<div class="ap-progress" transition:fade={{ duration: 200 }}>
				<div class="ap-progress-text">
					<span>{t('landing.album.progress', { done, total })}</span>
					{#if !compact}<span>{percent}%</span>{/if}
				</div>
				<div class="ap-track"><div class="ap-fill" style:width="{percent}%"></div></div>
			</div>
		{/if}
	</div>

	<div class="ap-grid" bind:this={grid}>
		{#each Array.from({ length: done }, (_, i) => i) as i (i)}
			<div class="ap-tile" in:scale={{ start: 0.85, duration: 320 }} out:fade={{ duration: 260 }}>
				<Photo scene={i} class="ap-photo" />
				<div class="ap-meta">
					<Bar w={NAMES[i]} />
					<Bar w="36%" />
				</div>
			</div>
		{/each}
	</div>

	{#if done === 0 && !overlay}
		<div class="ap-empty" transition:fade={{ duration: 200 }}>
			<Icon icon={icons.upload} size={24} />
			<span class="ap-here">{t('files.emptyTitle')}</span>
		</div>
	{/if}

	{#if overlay}
		<div class="ap-drop" transition:fade={{ duration: 160 }}>
			<Icon icon={icons.upload} size={28} />
			<span>{t('files.dropToUpload')}</span>
		</div>
	{/if}
</div>

<style>
	.ap-pane {
		position: relative;
		flex: 1;
		min-height: 0;
		padding: 16px;
		color: var(--color-fc-fg);
		font-family: var(--font-fc-body);
		font-size: 13px;
	}
	.ap-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 32px;
		margin-bottom: 8px;
	}
	.ap-crumbs {
		display: flex;
		align-items: center;
		gap: 6px;
	}
	.ap-muted,
	.ap-crumbs :global(.ap-muted) {
		color: var(--color-fc-fg-muted);
	}
	.ap-here {
		font-weight: 700;
	}
	.ap-toggle {
		display: flex;
		overflow: hidden;
		border-radius: var(--radius-fc-md);
		background: var(--color-fc-surface);
	}
	.ap-mode {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 30px;
		height: 30px;
		color: var(--color-fc-fg-muted);
	}
	.ap-on {
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
	}
	.ap-tools {
		display: flex;
		align-items: center;
		gap: 8px;
		height: 32px;
		margin-bottom: 12px;
	}
	.ap-btn {
		display: inline-flex;
		flex: none;
		align-items: center;
		gap: 6px;
		height: 30px;
		padding: 0 12px;
		border-radius: var(--radius-fc-pill);
		font-size: 12px;
		font-weight: 600;
		white-space: nowrap;
	}
	.ap-primary {
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
	}
	.ap-secondary {
		background: var(--color-fc-surface);
	}
	.ap-progress {
		display: flex;
		flex: 1;
		flex-direction: column;
		gap: 5px;
		min-width: 0;
		margin-left: 8px;
	}
	.ap-progress-text {
		display: flex;
		justify-content: space-between;
		gap: 8px;
		font-size: 11px;
		color: var(--color-fc-fg-muted);
		white-space: nowrap;
	}
	.ap-track {
		height: 6px;
		overflow: hidden;
		border-radius: 999px;
		background: var(--color-fc-surface);
	}
	.ap-fill {
		height: 100%;
		border-radius: 999px;
		background: var(--color-fc-accent);
		transition: width 0.3s ease;
	}
	.ap-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		align-content: start;
		align-items: start;
		gap: 12px;
		min-height: 290px;
	}
	.ap-compact .ap-grid {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
	.ap-compact .ap-tile:nth-child(n + 5) {
		display: none;
	}
	.ap-tile {
		display: flex;
		flex-direction: column;
		border-radius: var(--radius-fc-md);
		background: var(--mes-w-side, var(--color-fc-component));
	}
	.ap-tile :global(.ap-photo) {
		margin: 8px 8px 0;
		border-radius: var(--radius-fc-sm);
	}
	.ap-meta {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 10px 12px;
	}
	.ap-empty {
		position: absolute;
		inset: 120px 16px 16px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: flex-end;
		gap: 8px;
		padding-bottom: 40px;
		color: var(--color-fc-fg-muted);
		pointer-events: none;
	}
	.ap-drop {
		position: absolute;
		inset: 8px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: flex-end;
		gap: 8px;
		padding-bottom: 40px;
		border-radius: var(--radius-fc-lg);
		background: color-mix(in oklab, var(--color-fc-accent) 10%, transparent);
		font-weight: 600;
	}
</style>

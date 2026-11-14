<script lang="ts">
	import { fly, scale } from 'svelte/transition';
	import { Bar, grow } from '@facile/mise-en-scene';
	import { Icon, icons } from '@facile/muse';
	import { formatDate, formatSize } from '$lib/format';
	import { fileIcon, nuage } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

	interface Props {
		compact: boolean;
		uploaded: boolean;
		menu: boolean;
		copied: boolean;
		moreButton?: HTMLElement;
		shareItem?: HTMLElement;
	}

	let { compact, uploaded, menu, copied, moreButton = $bindable(), shareItem = $bindable() }: Props = $props();

	const FILE = 'brief.pdf';
	const BYTES = 254259;

	const rows = [
		{ icon: icons.folder, w: '46%' },
		{ icon: fileIcon('moodboard.png'), w: '58%' },
		{ icon: fileIcon('budget.xlsx'), w: '38%' },
		{ icon: fileIcon('contract.docx'), w: '52%' },
		{ icon: icons.folder, w: '34%' }
	];

	const size = $derived(formatSize(BYTES));
	const today = $derived(formatDate(new Date().toISOString()));
</script>

<div class="sd-pane" class:sd-compact={compact}>
	<div class="sd-head">
		<span class="sd-crumb">{t('nav.files')}</span>
	</div>
	<div class="sd-cols sd-muted">
		<span>{t('common.name')}</span>
		<span>{t('common.size')}</span>
		{#if !compact}<span>{t('common.modified')}</span>{/if}
		<span></span>
	</div>
	{#if uploaded}
		<div class="sd-grow" transition:grow>
			<div class="sd-row sd-new">
				<span class="sd-name">
					<Icon icon={fileIcon(FILE)} size={16} class="sd-icon" />
					<span class="sd-label">{FILE}</span>
				</span>
				<span class="sd-muted">{size}</span>
				{#if !compact}<span class="sd-muted">{today}</span>{/if}
				<span class="sd-more" bind:this={moreButton}><Icon icon={nuage.more} size={16} /></span>
			</div>
		</div>
	{/if}
	{#each compact ? rows.slice(0, 4) : rows as row (row.w)}
		<div class="sd-row">
			<span class="sd-name">
				<Icon icon={row.icon} size={16} class="sd-icon" />
				<Bar w={row.w} />
			</span>
			<Bar w="60%" />
			{#if !compact}<Bar w="70%" />{/if}
			<span></span>
		</div>
	{/each}
	{#if menu}
		<div class="sd-menu" class:sd-menu-compact={compact} transition:scale={{ start: 0.96, duration: 160 }}>
			<div class="sd-item" bind:this={shareItem}>
				<Icon icon={nuage.share} size={16} />
				<span>{t('files.menu.share')}</span>
			</div>
			<div class="sd-item"><Bar w="64%" /></div>
			<div class="sd-item"><Bar w="48%" /></div>
		</div>
	{/if}
	{#if copied}
		<div class="sd-toast" transition:fly={{ y: 12, duration: 320 }}>
			<Icon icon={nuage.link} size={16} />
			<span>{t('shared.copied')}</span>
		</div>
	{/if}
</div>

<style>
	.sd-pane {
		position: relative;
		flex: 1;
		min-height: 0;
		padding: 16px 16px 0;
		color: var(--color-fc-fg);
		font-family: var(--font-fc-body);
		font-size: 13px;
	}
	.sd-head {
		display: flex;
		align-items: center;
		height: 32px;
		margin-bottom: 8px;
	}
	.sd-crumb {
		font-family: var(--font-fc-title);
		font-size: 17px;
		font-weight: 700;
	}
	.sd-cols,
	.sd-row {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 64px 80px 32px;
		align-items: center;
		gap: 8px;
		padding: 0 10px;
	}
	.sd-compact .sd-cols,
	.sd-compact .sd-row {
		grid-template-columns: minmax(0, 1fr) 72px 32px;
		gap: 12px;
	}
	.sd-cols {
		height: 32px;
		font-size: 12px;
	}
	.sd-row {
		height: 44px;
		border-radius: 10px;
	}
	.sd-grow {
		padding-bottom: 2px;
	}
	.sd-new {
		background: var(--color-fc-surface);
	}
	.sd-muted {
		color: var(--color-fc-fg-muted);
	}
	.sd-name {
		display: flex;
		align-items: center;
		gap: 12px;
		min-width: 0;
	}
	.sd-name :global(.sd-icon) {
		flex: none;
		color: var(--color-fc-fg-muted);
	}
	.sd-label {
		overflow: hidden;
		font-weight: 700;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.sd-more {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border-radius: 8px;
		color: var(--color-fc-fg-muted);
	}
	.sd-menu {
		position: absolute;
		top: 132px;
		right: 24px;
		z-index: 2;
		width: 180px;
		padding: 4px 0;
		border-radius: 10px;
		background: var(--color-fc-component);
		box-shadow: 0 16px 36px -12px rgb(0 0 0 / 0.28), 0 4px 12px -4px rgb(0 0 0 / 0.12);
		transform-origin: top right;
	}
	.sd-menu-compact {
		right: 20px;
	}
	.sd-item {
		display: flex;
		align-items: center;
		gap: 10px;
		height: 36px;
		padding: 0 12px;
	}
	.sd-toast {
		position: absolute;
		bottom: 20px;
		left: 50%;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 16px;
		border-radius: 999px;
		background: var(--color-fc-inverse);
		color: var(--color-fc-on-inverse);
		white-space: nowrap;
		translate: -50% 0;
		box-shadow: 0 12px 28px -10px rgb(0 0 0 / 0.35);
	}
</style>

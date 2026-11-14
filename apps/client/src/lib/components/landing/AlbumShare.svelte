<script lang="ts">
	import { fade, fly, scale } from 'svelte/transition';
	import { Roll } from '@facile/mise-en-scene';
	import { Icon, icons } from '@facile/muse';
	import { nuage } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

	interface Props {
		open: boolean;
		week: boolean;
		copied: boolean;
		expiry?: HTMLElement;
		create?: HTMLElement;
	}

	let { open, week, copied, expiry = $bindable(), create = $bindable() }: Props = $props();
</script>

{#if open}
	<div class="as-scrim" transition:fade={{ duration: 200 }}>
		<div class="as-dialog" transition:scale={{ start: 0.96, duration: 200 }}>
			<p class="as-title">{t('share.title', { name: t('landing.album.folder') })}</p>
			<div class="as-field">
				<span class="as-label">{t('share.expires')}</span>
				<span class="as-select" bind:this={expiry}>
					<Roll value={week ? t('share.in7d') : t('share.never')} class="as-roll" />
					<Icon icon={icons.chevronDown} size={16} />
				</span>
			</div>
			<div class="as-foot">
				<span class="as-btn as-ghost">{t('common.cancel')}</span>
				<span class="as-btn as-primary" bind:this={create}>
					<Icon icon={nuage.link} size={14} />{t('share.create')}
				</span>
			</div>
		</div>
	</div>
{/if}

{#if copied}
	<div class="as-toast" transition:fly={{ y: 12, duration: 320 }}>
		<Icon icon={nuage.link} size={16} />
		<span>{t('shared.copied')}</span>
	</div>
{/if}

<style>
	.as-scrim {
		position: absolute;
		inset: 0;
		z-index: 3;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 16px;
		background: color-mix(in oklab, var(--color-fc-scrim) 45%, transparent);
		font-family: var(--font-fc-body);
	}
	.as-dialog {
		display: flex;
		flex-direction: column;
		gap: 14px;
		width: 100%;
		max-width: 300px;
		padding: 18px;
		border-radius: var(--radius-fc-lg);
		background: var(--color-fc-component);
		color: var(--color-fc-fg);
		box-shadow: 0 24px 48px -16px rgb(0 0 0 / 0.35);
	}
	.as-title {
		margin: 0;
		font-family: var(--font-fc-title);
		font-size: 15px;
		font-weight: 700;
	}
	.as-field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.as-label {
		font-size: 12px;
		font-weight: 600;
	}
	.as-select {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 40px;
		padding: 0 12px;
		border-radius: var(--radius-fc-md);
		background: var(--color-fc-field);
		font-size: 13px;
		color: var(--color-fc-fg-muted);
	}
	.as-select :global(.as-roll) {
		color: var(--color-fc-fg);
	}
	.as-foot {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}
	.as-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 32px;
		padding: 0 12px;
		border-radius: var(--radius-fc-pill);
		font-size: 12px;
		font-weight: 600;
		white-space: nowrap;
	}
	.as-ghost {
		color: var(--color-fc-fg-muted);
	}
	.as-primary {
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
	}
	.as-toast {
		position: absolute;
		bottom: 20px;
		left: 50%;
		z-index: 4;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 16px;
		border-radius: 999px;
		background: var(--color-fc-inverse);
		color: var(--color-fc-on-inverse);
		font-family: var(--font-fc-body);
		font-size: 13px;
		white-space: nowrap;
		translate: -50% 0;
		box-shadow: 0 12px 28px -10px rgb(0 0 0 / 0.35);
	}
</style>

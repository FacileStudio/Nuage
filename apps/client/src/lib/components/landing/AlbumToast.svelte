<script lang="ts">
	import { fly } from 'svelte/transition';
	import { Icon } from '@facile/muse';
	import { nuage } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

	let { show }: { show: boolean } = $props();
</script>

{#if show}
	<div class="at-toast" transition:fly={{ y: 12, duration: 320 }}>
		<Icon icon={nuage.link} size={16} />
		<span>{t('shared.copied')}</span>
	</div>
{/if}

<style>
	.at-toast {
		position: absolute;
		bottom: 6px;
		left: 50%;
		z-index: 20;
		display: flex;
		align-items: center;
		gap: 8px;
		height: 34px;
		padding: 0 16px;
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

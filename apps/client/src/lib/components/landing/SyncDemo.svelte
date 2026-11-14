<script lang="ts">
	import { tick } from 'svelte';
	import { fly } from 'svelte/transition';
	import { Bar, Caret, Cursor, Stage, Terminal, Window, cursorAt, type Act } from '@facile/mise-en-scene';
	import { Icon, icons } from '@facile/muse';
	import { nuage } from '$lib/icons';
	import SyncFiles from './SyncFiles.svelte';
	import { t } from '$lib/i18n/index.svelte';

	const FILE = 'brief.pdf';
	const COMMAND = 'nuage start';

	let typed = $state('');
	let started = $state(false);
	let uploaded = $state(false);
	let menu = $state(false);
	let copied = $state(false);
	let cursor = $state(cursorAt());
	let moreButton = $state<HTMLElement>();
	let shareItem = $state<HTMLElement>();

	function reset() {
		typed = '';
		started = false;
		uploaded = false;
		menu = false;
		copied = false;
		cursor.visible = false;
	}

	function poster() {
		typed = COMMAND;
		started = true;
		uploaded = true;
		copied = true;
	}

	async function press(act: Act, target: HTMLElement | undefined, fx: number, onPress: () => void) {
		await act.move(cursor, act.point(target, fx, 0.6));
		await act.click(cursor, onPress);
		await tick();
	}

	async function script(act: Act) {
		await act.wait(700);
		await act.type(COMMAND, (v) => (typed = v), 60);
		await act.wait(400);
		started = true;
		await act.wait(1100);
		uploaded = true;
		await act.wait(1300);
		cursor.visible = true;
		await press(act, moreButton, 0.5, () => (menu = true));
		await act.wait(500);
		await press(act, shareItem, 0.84, () => {
			menu = false;
			copied = true;
		});
		await act.wait(2200);
		copied = false;
		uploaded = false;
		cursor.visible = false;
		await act.wait(500);
		reset();
		await act.wait(300);
	}
</script>

{#snippet terminal(style: string)}
	<Terminal {style}>
		<div class="sd-line"><span class="sd-dim">$</span> {typed}{#if !started}<Caret />{/if}</div>
		{#if started}
			<div class="sd-line sd-dim" in:fly={{ y: 6, duration: 300 }}>daemon started, PID 4182</div>
		{/if}
		{#if uploaded}
			<div class="sd-line" in:fly={{ y: 6, duration: 300 }}>↑ uploaded {FILE} (248.3 KB)</div>
		{/if}
	</Terminal>
{/snippet}

<Stage
	width={960}
	height={520}
	compact={{ below: 640, width: 358, height: 570 }}
	{script}
	{reset}
	{poster}
	label={t('landing.demoLabel')}
>
	{#snippet children({ compact })}
		{#if compact}
			{@render terminal('top:12px;left:12px;width:334px;height:96px')}
			<Window style="top:124px;left:12px;width:334px;height:384px">
				<SyncFiles compact {uploaded} {menu} {copied} bind:moreButton bind:shareItem />
			</Window>
		{:else}
			{@render terminal('top:182px;left:32px;width:300px;height:96px')}
			<Window style="top:24px;left:356px;width:556px;height:412px">
				{#snippet sidebar()}
					<div class="sd-nav">
						<div class="sd-brand">
							<Icon icon={nuage.brand} size={18} />
							<span>Nuage</span>
						</div>
						<div class="sd-link sd-active">
							<Icon icon={icons.folder} size={16} />
							<span>{t('nav.files')}</span>
						</div>
						<div class="sd-link"><Icon icon={nuage.share} size={16} /><Bar w="58%" /></div>
						<div class="sd-link"><Icon icon={icons.remove} size={16} /><Bar w="44%" /></div>
						<div class="sd-link"><Icon icon={icons.space} size={16} /><Bar w="52%" /></div>
					</div>
				{/snippet}
				<SyncFiles compact={false} {uploaded} {menu} {copied} bind:moreButton bind:shareItem />
			</Window>
		{/if}
		<Cursor {cursor} />
	{/snippet}
</Stage>

<style>
	.sd-line {
		white-space: nowrap;
	}
	.sd-dim {
		color: var(--color-fc-on-inverse-muted);
	}
	.sd-nav {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 8px 10px;
		font-family: var(--font-fc-body);
		font-size: 13px;
		color: var(--color-fc-fg-muted);
	}
	.sd-brand {
		display: flex;
		align-items: center;
		gap: 8px;
		height: 36px;
		padding: 0 12px;
		margin-bottom: 4px;
		color: var(--color-fc-fg);
		font-family: var(--font-fc-title);
		font-weight: 700;
	}
	.sd-link {
		display: flex;
		align-items: center;
		gap: 10px;
		height: 36px;
		padding: 0 12px;
		border-radius: 10px;
	}
	.sd-active {
		background: var(--color-fc-surface);
		color: var(--color-fc-fg);
		font-weight: 700;
	}
</style>

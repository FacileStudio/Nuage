<script lang="ts">
	import { tick } from 'svelte';
	import { Bar, Cursor, Stage, Window, cursorAt, type Act, type Point } from '@facile/mise-en-scene';
	import { Icon, icons } from '@facile/muse';
	import { nuage } from '$lib/icons';
	import AlbumPane from './AlbumPane.svelte';
	import AlbumShare from './AlbumShare.svelte';
	import AlbumToast from './AlbumToast.svelte';
	import Photo from './Photo.svelte';
	import { t } from '$lib/i18n/index.svelte';

	let stack = $state<Point & { visible: boolean }>({ x: 0, y: 0, visible: false });
	let overlay = $state(false);
	let uploading = $state(false);
	let done = $state(0);
	let dialog = $state(false);
	let week = $state(false);
	let copied = $state(false);
	let cursor = $state(cursorAt());
	let grid = $state<HTMLElement>();
	let shareButton = $state<HTMLElement>();
	let expiry = $state<HTMLElement>();
	let create = $state<HTMLElement>();
	let from = $state<HTMLElement>();

	const total = 6;

	function place(to: Point) {
		cursor.x = to.x;
		cursor.y = to.y;
		stack.x = to.x;
		stack.y = to.y;
	}

	function home() {
		if (from) place({ x: from.offsetLeft, y: from.offsetTop });
	}

	function reset() {
		overlay = false;
		uploading = false;
		done = 0;
		dialog = false;
		week = false;
		copied = false;
		home();
		stack.visible = true;
		cursor.visible = true;
	}

	function poster() {
		stack.visible = false;
		done = total;
		copied = true;
	}

	async function press(act: Act, target: HTMLElement | undefined, onPress: () => void) {
		await act.move(cursor, act.point(target, 0.6, 0.6));
		await act.click(cursor, onPress);
		await tick();
	}

	async function script(act: Act) {
		reset();
		await act.wait(900);
		const drop = act.point(grid, 0.5, 0.3);
		stack.x = drop.x;
		stack.y = drop.y;
		await act.move(cursor, drop, 700);
		overlay = true;
		await act.wait(450);
		await act.click(cursor, () => {
			stack.visible = false;
			overlay = false;
			uploading = true;
		});
		await act.count(total, (v) => (done = v), 380);
		await act.wait(400);
		uploading = false;
		await act.wait(300);
		await press(act, shareButton, () => (dialog = true));
		await act.wait(500);
		await press(act, expiry, () => (week = true));
		await act.wait(700);
		await press(act, create, () => {
			dialog = false;
			copied = true;
		});
		await act.wait(2400);
		cursor.visible = false;
		await act.wait(350);
		copied = false;
		done = 0;
		home();
		await act.wait(950);
		stack.visible = true;
		cursor.visible = true;
		await act.wait(300);
	}
</script>

{#snippet sidebar()}
	<div class="ad-nav">
		<div class="ad-brand">
			<Icon icon={nuage.brand} size={18} />
			<span>Nuage</span>
		</div>
		<div class="ad-link ad-active">
			<Icon icon={icons.folder} size={16} />
			<span>{t('nav.files')}</span>
		</div>
		<div class="ad-link"><Icon icon={nuage.share} size={16} /><Bar w="58%" /></div>
		<div class="ad-link"><Icon icon={icons.remove} size={16} /><Bar w="44%" /></div>
		<div class="ad-link"><Icon icon={icons.space} size={16} /><Bar w="52%" /></div>
	</div>
{/snippet}

{#snippet body(compact: boolean)}
	<div class="ad-body">
		<AlbumPane {compact} {total} {done} {uploading} {overlay} bind:grid bind:shareButton />
		<AlbumShare open={dialog} {week} bind:expiry bind:create />
	</div>
{/snippet}

<Stage
	width={960}
	height={540}
	compact={{ below: 640, width: 358, height: 520 }}
	{script}
	{reset}
	{poster}
	label={t('landing.demoLabel')}
>
	{#snippet children({ compact })}
		{#if compact}
			<span class="ad-from" style="left:262px;top:270px" bind:this={from}></span>
			<Window style="top:12px;left:12px;width:334px;height:464px">
				{@render body(true)}
			</Window>
		{:else}
			<span class="ad-from" style="left:80px;top:300px" bind:this={from}></span>
			<Window style="top:16px;left:160px;width:640px;height:476px" {sidebar}>
				{@render body(false)}
			</Window>
		{/if}
		<div
			class="ad-stack"
			style:translate="{stack.x - 46}px {stack.y - 40}px"
			style:opacity={stack.visible ? 1 : 0}
		>
			<Photo scene={2} class="ad-card ad-card-1" />
			<Photo scene={4} class="ad-card ad-card-2" />
			<Photo scene={0} class="ad-card ad-card-3" />
			<span class="ad-count">{total}</span>
		</div>
		<AlbumToast show={copied} />
		<Cursor {cursor} />
	{/snippet}
</Stage>

<style>
	.ad-body {
		position: relative;
		display: flex;
		flex: 1;
		flex-direction: column;
		min-height: 0;
	}
	.ad-from {
		position: absolute;
		width: 1px;
		height: 1px;
	}
	.ad-stack {
		position: absolute;
		top: 0;
		left: 0;
		z-index: 20;
		width: 92px;
		height: 69px;
		pointer-events: none;
		transition:
			translate 0.7s cubic-bezier(0.45, 0, 0.1, 1),
			opacity 0.25s ease;
	}
	.ad-stack :global(.ad-card) {
		position: absolute;
		inset: 0;
		border-radius: var(--radius-fc-sm);
		box-shadow: 0 10px 24px -8px rgb(0 0 0 / 0.35);
	}
	.ad-stack :global(.ad-card-1) {
		rotate: -9deg;
	}
	.ad-stack :global(.ad-card-2) {
		rotate: 6deg;
	}
	.ad-count {
		position: absolute;
		top: -8px;
		right: -8px;
		display: flex;
		align-items: center;
		justify-content: center;
		min-width: 22px;
		height: 22px;
		padding: 0 6px;
		border-radius: 999px;
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
		font-family: var(--font-fc-body);
		font-size: 12px;
		font-weight: 700;
	}
	.ad-nav {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 8px 10px;
		font-family: var(--font-fc-body);
		font-size: 13px;
		color: var(--color-fc-fg-muted);
	}
	.ad-brand {
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
	.ad-link {
		display: flex;
		align-items: center;
		gap: 10px;
		height: 36px;
		padding: 0 12px;
		border-radius: 10px;
	}
	.ad-active {
		background: var(--color-fc-surface);
		color: var(--color-fc-fg);
		font-weight: 700;
	}
</style>

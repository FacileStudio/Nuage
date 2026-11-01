<script lang="ts">
	import { onDestroy } from 'svelte';
	import { fileIcon, nuage } from '$lib/icons';
	import { formatDuration } from '$lib/format';

	let { url, kind, name }: { url: string; kind: 'audio' | 'video'; name: string } = $props();

	let wrapper = $state<HTMLDivElement | null>(null);
	let media = $state<HTMLMediaElement | null>(null);
	let paused = $state(true);
	let currentTime = $state(0);
	let duration = $state(0);
	let volume = $state(1);
	let muted = $state(false);
	let playbackRate = $state(1);
	let buffered = $state<{ start: number; end: number }[]>([]);
	let fullscreen = $state(false);
	let idle = $state(false);
	let failed = $state(false);
	let idleTimer: ReturnType<typeof setTimeout> | null = null;

	const RATES = [0.5, 0.75, 1, 1.25, 1.5, 2];
	const played = $derived(duration ? (currentTime / duration) * 100 : 0);
	const loaded = $derived(duration && buffered.length ? (buffered[buffered.length - 1].end / duration) * 100 : 0);
	const hideChrome = $derived(kind === 'video' && idle && !paused);

	const toggle = () => (media?.paused ? void media.play() : media?.pause());
	const skip = (seconds: number) => {
		if (media) media.currentTime = Math.max(0, Math.min(duration, media.currentTime + seconds));
	};
	const cycleRate = () => (playbackRate = RATES[(RATES.indexOf(playbackRate) + 1) % RATES.length]);

	function wake() {
		idle = false;
		if (idleTimer) clearTimeout(idleTimer);
		idleTimer = setTimeout(() => (idle = true), 2500);
	}

	async function toggleFullscreen() {
		if (document.fullscreenElement) await document.exitFullscreen();
		else await wrapper?.requestFullscreen();
	}

	const SHORTCUTS: Record<string, () => void> = {
		' ': toggle,
		k: toggle,
		j: () => skip(-10),
		l: () => skip(10),
		ArrowLeft: () => skip(-5),
		ArrowRight: () => skip(5),
		m: () => (muted = !muted),
		f: () => kind === 'video' && void toggleFullscreen()
	};

	function onKeydown(e: KeyboardEvent) {
		if (e.target instanceof HTMLInputElement) return;
		const action = SHORTCUTS[e.key];
		if (!action) return;
		e.preventDefault();
		e.stopPropagation();
		action();
		wake();
	}

	onDestroy(() => idleTimer && clearTimeout(idleTimer));
</script>

<svelte:document onfullscreenchange={() => (fullscreen = document.fullscreenElement === wrapper)} />

{#snippet controls()}
	<div class="flex flex-col gap-1 sm:gap-2">
		<div class="flex items-center gap-2">
			<span class="w-10 shrink-0 text-right text-fc-xs text-fc-fg-muted tabular-nums sm:hidden">
				{formatDuration(currentTime)}
			</span>
			<div class="relative h-1.5 min-w-0 flex-1 rounded-fc-pill bg-fc-border">
				<div class="absolute inset-y-0 left-0 rounded-fc-pill bg-fc-fg-muted/40" style="width: {loaded}%"></div>
				<div class="absolute inset-y-0 left-0 rounded-fc-pill bg-fc-accent" style="width: {played}%"></div>
				<input
					type="range"
					min="0"
					max={duration || 0}
					step="0.1"
					bind:value={currentTime}
					aria-label="Seek"
					aria-valuetext="{formatDuration(currentTime)} of {formatDuration(duration)}"
					class="absolute -inset-y-2 inset-x-0 w-full cursor-pointer opacity-0 [@media(pointer:coarse)]:-inset-y-4"
				/>
			</div>
			<span class="w-10 shrink-0 text-fc-xs text-fc-fg-muted tabular-nums sm:hidden">
				{formatDuration(duration)}
			</span>
		</div>
		<div class="flex items-center gap-1 text-fc-fg">
			<button type="button" class="viewer-tool" aria-label="Back 10 seconds" onclick={() => skip(-10)}>
				<iconify-icon icon={nuage.back10} width="20" height="20" class="block"></iconify-icon>
			</button>
			<button
				type="button"
				class="mx-1 inline-flex size-11 shrink-0 items-center justify-center rounded-fc-pill bg-fc-accent text-fc-accent-fg transition-transform hover:scale-105 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
				aria-label={paused ? 'Play' : 'Pause'}
				onclick={toggle}
			>
				<iconify-icon icon={paused ? nuage.play : nuage.pause} width="20" height="20" class="block"></iconify-icon>
			</button>
			<button type="button" class="viewer-tool" aria-label="Forward 10 seconds" onclick={() => skip(10)}>
				<iconify-icon icon={nuage.forward10} width="20" height="20" class="block"></iconify-icon>
			</button>
			<span class="ml-2 hidden shrink-0 whitespace-nowrap text-fc-xs text-fc-fg-muted tabular-nums sm:inline">
				{formatDuration(currentTime)} / {formatDuration(duration)}
			</span>
			<div class="min-w-0 flex-1"></div>
			<button type="button" class="viewer-tool w-auto px-2 text-fc-xs font-medium whitespace-nowrap tabular-nums" aria-label="Playback speed" onclick={cycleRate}>
				{playbackRate}×
			</button>
			<button type="button" class="viewer-tool" aria-label={muted ? 'Unmute' : 'Mute'} onclick={() => (muted = !muted)}>
				<iconify-icon icon={muted || volume === 0 ? nuage.muted : nuage.volume} width="20" height="20" class="block"></iconify-icon>
			</button>
			<input
				type="range"
				min="0"
				max="1"
				step="0.05"
				bind:value={volume}
				aria-label="Volume"
				class="hidden w-20 accent-fc-accent sm:block"
			/>
			{#if kind === 'video'}
				<button type="button" class="viewer-tool" aria-label={fullscreen ? 'Exit full screen' : 'Full screen'} onclick={toggleFullscreen}>
					<iconify-icon icon={fullscreen ? nuage.exitFullscreen : nuage.fullscreen} width="20" height="20" class="block"></iconify-icon>
				</button>
			{/if}
		</div>
	</div>
{/snippet}

{#if failed}
	<p class="text-fc-sm text-fc-fg-muted">This file cannot be played in the browser. Download it to open it.</p>
{:else if kind === 'video'}
	<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
	<div
		bind:this={wrapper}
		role="group"
		aria-label="Video player: {name}"
		tabindex="0"
		class="group relative flex h-full w-full items-center justify-center overflow-hidden rounded-fc-lg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring {hideChrome ? 'cursor-none' : ''}"
		onkeydown={onKeydown}
		onpointermove={wake}
	>
		<!-- svelte-ignore a11y_media_has_caption -->
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<video
			bind:this={media}
			src={url}
			autoplay
			playsinline
			preload="metadata"
			bind:paused
			bind:currentTime
			bind:duration
			bind:volume
			bind:muted
			bind:playbackRate
			bind:buffered
			onclick={toggle}
			ondblclick={toggleFullscreen}
			onerror={() => (failed = true)}
			class="max-h-full max-w-full rounded-fc-md"
		></video>
		{#if paused}
			<button
				type="button"
				aria-label="Play"
				onclick={toggle}
				class="absolute inset-0 m-auto flex size-16 items-center justify-center rounded-fc-pill bg-fc-component/90 text-fc-fg shadow-lg backdrop-blur transition-transform hover:scale-105"
			>
				<iconify-icon icon={nuage.play} width="28" height="28" class="block"></iconify-icon>
			</button>
		{/if}
		<div
			class="absolute inset-x-3 bottom-[max(0.75rem,env(safe-area-inset-bottom))] rounded-fc-lg border border-fc-border bg-fc-component/90 px-3 pt-3 pb-1.5 shadow-lg backdrop-blur transition-opacity duration-300 {hideChrome
				? 'pointer-events-none opacity-0'
				: 'opacity-100'}"
		>
			{@render controls()}
		</div>
	</div>
{:else}
	<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
	<div
		role="group"
		aria-label="Audio player: {name}"
		tabindex="0"
		onkeydown={onKeydown}
		class="flex w-full max-w-md flex-col gap-4 rounded-fc-lg bg-fc-component p-4 sm:gap-6 sm:p-6 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
	>
		<div class="flex aspect-square max-h-[45dvh] w-full items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg-muted">
			<iconify-icon
				icon={fileIcon(name, 'audio/')}
				width="96"
				height="96"
				class="block transition-transform duration-700 {paused ? '' : 'scale-110'}"
			></iconify-icon>
		</div>
		<div class="min-w-0">
			<p class="truncate text-fc-lg font-semibold text-fc-fg" title={name}>{name}</p>
			<p class="text-fc-sm text-fc-fg-muted">{paused ? 'Paused' : 'Playing'}</p>
		</div>
		<audio
			bind:this={media}
			src={url}
			autoplay
			preload="metadata"
			bind:paused
			bind:currentTime
			bind:duration
			bind:volume
			bind:muted
			bind:playbackRate
			bind:buffered
			onerror={() => (failed = true)}
		></audio>
		{@render controls()}
	</div>
{/if}

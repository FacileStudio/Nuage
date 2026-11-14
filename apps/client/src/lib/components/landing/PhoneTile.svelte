<script lang="ts">
	import { fade, scale } from 'svelte/transition';
	import { Stage, Tile, type Act } from '@facile/mise-en-scene';
	import { Icon, icons } from '@facile/muse';
	import { nuage } from '$lib/icons';
	import Photo from './Photo.svelte';
	import { t } from '$lib/i18n/index.svelte';

	const SCENES = [1, 3, 0, 5, 2, 4];

	let tap = $state<number | null>(null);
	let viewing = $state<number | null>(null);

	function reset() {
		tap = null;
		viewing = null;
	}

	async function show(act: Act, index: number) {
		tap = index;
		await act.wait(260);
		tap = null;
		viewing = index;
		await act.wait(1500);
	}

	async function script(act: Act) {
		await act.wait(1000);
		await show(act, 1);
		viewing = null;
		await act.wait(700);
		await show(act, 2);
		viewing = null;
		await act.wait(1300);
	}
</script>

<Tile title={t('landing.card.phone.title')} body={t('landing.card.phone.body')}>
	<Stage width={320} height={240} {script} {reset} poster={() => (viewing = 1)}>
		{#snippet children()}
			<div class="ph-phone absolute top-2 left-1/2 h-[224px] w-[108px] -translate-x-1/2 rounded-[20px] bg-fc-inverse p-[5px]">
				<div class="relative flex h-full flex-col gap-2 overflow-hidden rounded-[15px] bg-fc-page px-1.5 pt-5">
					<span class="absolute top-1.5 left-1/2 h-[7px] w-8 -translate-x-1/2 rounded-full bg-fc-inverse"></span>
					<div class="flex items-center gap-1 px-0.5 text-[8px] font-bold text-fc-fg">
						<Icon icon={nuage.brand} size={11} />
						<span class="truncate">{t('landing.album.folder')}</span>
					</div>
					<div class="grid grid-cols-2 gap-1">
						{#each SCENES as scene, i (i)}
							<div class="relative rounded-[4px] bg-fc-component p-[3px] pb-1">
								<Photo {scene} class="rounded-[3px]" />
								<div class="mt-1 flex items-center justify-between text-fc-fg-muted">
									<span class="h-1 w-5 rounded-full bg-fc-fg/15"></span>
									<Icon icon={icons.download} size={7} />
								</div>
								{#if tap === i}
									<span class="ph-tap" transition:fade={{ duration: 160 }}></span>
								{/if}
							</div>
						{/each}
					</div>
					{#if viewing !== null}
						<div class="absolute inset-0 flex items-center bg-fc-inverse" transition:fade={{ duration: 220 }}>
							{#key viewing}
								<div class="w-full" in:scale={{ start: 0.7, duration: 320 }}>
									<Photo scene={SCENES[viewing]} />
								</div>
							{/key}
						</div>
					{/if}
				</div>
			</div>
		{/snippet}
	</Stage>
</Tile>

<style>
	.ph-phone {
		box-shadow: 0 14px 28px -12px rgb(0 0 0 / 0.35);
	}
	.ph-tap {
		position: absolute;
		top: 40%;
		left: 50%;
		width: 22px;
		height: 22px;
		border-radius: 999px;
		background: color-mix(in oklab, var(--color-fc-fg) 25%, transparent);
		translate: -50% -50%;
	}
</style>

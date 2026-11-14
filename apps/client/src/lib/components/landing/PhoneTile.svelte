<script lang="ts">
	import { Stage, Tile, type Act } from '@facile/mise-en-scene';
	import { Icon, icons } from '@facile/muse';
	import { nuage } from '$lib/icons';
	import Photo from './Photo.svelte';
	import { t } from '$lib/i18n/index.svelte';

	const SCENES = [1, 3, 0, 5, 2, 4];
	const SHOWN = [0, 3, 4];

	let tap = $state<number | null>(null);
	let open = $state(false);
	let index = $state(0);

	function reset() {
		tap = null;
		open = false;
		index = 0;
	}

	function poster() {
		open = true;
		index = 1;
	}

	async function script(act: Act) {
		await act.wait(900);
		tap = SHOWN[0];
		await act.wait(350);
		open = true;
		await act.wait(200);
		tap = null;
		for (let i = 1; i < SHOWN.length; i++) {
			await act.wait(1500);
			index = i;
		}
		await act.wait(1600);
		open = false;
		await act.wait(700);
		index = 0;
		await act.wait(600);
	}
</script>

<Tile title={t('landing.card.phone.title')} body={t('landing.card.phone.body')}>
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<div class="ph-phone absolute top-2 left-1/2 h-[224px] w-[108px] -translate-x-1/2 rounded-[20px] bg-fc-inverse p-[5px]">
				<div class="relative flex h-full flex-col gap-2 overflow-hidden rounded-[15px] bg-fc-page px-1.5 pt-5">
					<span class="absolute top-1.5 left-1/2 z-10 h-[7px] w-8 -translate-x-1/2 rounded-full bg-fc-inverse"></span>
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
								<span class="ph-tap" class:ph-on={tap === i}></span>
							</div>
						{/each}
					</div>
					<div class="ph-viewer" class:ph-open={open}>
						<div class="ph-strip" style:translate="{-index * 100}% 0">
							{#each SHOWN as shot (shot)}
								<div class="ph-slide"><Photo scene={SCENES[shot]} /></div>
							{/each}
						</div>
						<div class="ph-dots">
							{#each SHOWN as shot, i (shot)}
								<span class="ph-dot" class:ph-current={i === index}></span>
							{/each}
						</div>
					</div>
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
		opacity: 0;
		translate: -50% -50%;
		scale: 0.6;
		transition:
			opacity 0.2s ease,
			scale 0.3s ease-out;
	}
	.ph-on {
		opacity: 1;
		scale: 1;
	}
	.ph-viewer {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		justify-content: center;
		overflow: hidden;
		background: var(--color-fc-inverse);
		opacity: 0;
		scale: 0.94;
		transition:
			opacity 0.35s ease,
			scale 0.4s cubic-bezier(0.2, 0.8, 0.2, 1);
	}
	.ph-open {
		opacity: 1;
		scale: 1;
	}
	.ph-strip {
		display: flex;
		transition: translate 0.55s cubic-bezier(0.45, 0, 0.15, 1);
	}
	.ph-slide {
		flex: 0 0 100%;
	}
	.ph-dots {
		position: absolute;
		bottom: 14px;
		left: 50%;
		display: flex;
		gap: 4px;
		translate: -50% 0;
	}
	.ph-dot {
		width: 5px;
		height: 5px;
		border-radius: 999px;
		background: var(--color-fc-on-inverse);
		opacity: 0.35;
		scale: 0.8;
		transition:
			opacity 0.3s ease,
			scale 0.3s ease;
	}
	.ph-current {
		opacity: 1;
		scale: 1.2;
	}
	@media (prefers-reduced-motion: reduce) {
		.ph-tap,
		.ph-viewer,
		.ph-strip,
		.ph-dot {
			transition: none;
		}
	}
</style>

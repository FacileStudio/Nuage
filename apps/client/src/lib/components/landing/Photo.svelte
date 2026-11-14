<script lang="ts">
	interface Props {
		scene: number;
		class?: string;
	}

	let { scene, class: className = '' }: Props = $props();

	const mix = (token: string, other: string, pct: number) =>
		`color-mix(in oklch, var(--color-fc-${token}), ${other} ${pct}%)`;

	const SCENES = [
		{
			sky: `linear-gradient(${mix('chart-3', 'white', 55)}, ${mix('chart-3', 'white', 80)})`,
			sun: { x: 72, y: 22, r: 16, c: mix('chart-2', 'white', 35) },
			back: { c: mix('chart-3', 'black', 5), clip: 'polygon(0 58%, 100% 54%, 100% 100%, 0 100%)' },
			front: { c: mix('chart-2', 'white', 55), clip: 'polygon(0 78%, 40% 72%, 100% 80%, 100% 100%, 0 100%)' }
		},
		{
			sky: `linear-gradient(${mix('chart-6', 'white', 10)}, ${mix('chart-2', 'white', 20)})`,
			sun: { x: 50, y: 52, r: 22, c: mix('chart-2', 'white', 70) },
			back: { c: mix('chart-1', 'black', 30), clip: 'polygon(0 64%, 100% 62%, 100% 100%, 0 100%)' },
			front: { c: mix('chart-1', 'black', 50), clip: 'polygon(0 86%, 100% 84%, 100% 100%, 0 100%)' }
		},
		{
			sky: `linear-gradient(${mix('chart-3', 'white', 45)}, ${mix('chart-1', 'white', 75)})`,
			sun: { x: 22, y: 20, r: 10, c: mix('chart-2', 'white', 60) },
			back: { c: mix('chart-1', 'black', 10), clip: 'polygon(0 70%, 28% 30%, 46% 55%, 66% 24%, 100% 66%, 100% 100%, 0 100%)' },
			front: { c: mix('chart-5', 'black', 15), clip: 'polygon(0 80%, 50% 72%, 100% 82%, 100% 100%, 0 100%)' }
		},
		{
			sky: `linear-gradient(${mix('chart-2', 'white', 55)}, ${mix('chart-2', 'white', 85)})`,
			sun: { x: 78, y: 28, r: 14, c: mix('chart-2', 'white', 15) },
			back: { c: mix('chart-5', 'white', 15), clip: 'polygon(0 62%, 30% 52%, 62% 60%, 100% 50%, 100% 100%, 0 100%)' },
			front: { c: mix('chart-5', 'black', 20), clip: 'polygon(0 82%, 55% 70%, 100% 78%, 100% 100%, 0 100%)' }
		},
		{
			sky: `linear-gradient(${mix('chart-4', 'white', 25)}, ${mix('chart-2', 'white', 45)})`,
			sun: { x: 30, y: 46, r: 18, c: mix('chart-2', 'white', 75) },
			back: { c: mix('chart-3', 'black', 20), clip: 'polygon(0 60%, 100% 58%, 100% 100%, 0 100%)' },
			front: { c: mix('chart-4', 'black', 35), clip: 'polygon(55% 100%, 62% 40%, 66% 40%, 70% 100%)' }
		},
		{
			sky: `linear-gradient(${mix('chart-1', 'black', 50)}, ${mix('chart-1', 'black', 15)})`,
			sun: { x: 70, y: 24, r: 12, c: mix('chart-2', 'white', 85) },
			back: { c: mix('chart-1', 'black', 35), clip: 'polygon(0 72%, 35% 58%, 70% 70%, 100% 60%, 100% 100%, 0 100%)' },
			front: { c: mix('chart-1', 'black', 62), clip: 'polygon(0 88%, 100% 82%, 100% 100%, 0 100%)' }
		}
	];

	const look = $derived(SCENES[scene % SCENES.length]);
</script>

<span class="ph {className}" style:background={look.sky}>
	<span
		class="ph-sun"
		style:left="{look.sun.x}%"
		style:top="{look.sun.y}%"
		style:width="{look.sun.r}%"
		style:background={look.sun.c}
	></span>
	<span class="ph-layer" style:background={look.back.c} style:clip-path={look.back.clip}></span>
	<span class="ph-layer" style:background={look.front.c} style:clip-path={look.front.clip}></span>
</span>

<style>
	.ph {
		position: relative;
		display: block;
		overflow: hidden;
		aspect-ratio: 4 / 3;
	}
	.ph-sun {
		position: absolute;
		aspect-ratio: 1;
		border-radius: 999px;
		translate: -50% -50%;
	}
	.ph-layer {
		position: absolute;
		inset: 0;
	}
</style>

<script lang="ts">
	import { Caret, Stage, Terminal, Tile, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';

	type Line = { kind: 'cmd' | 'ok' | 'out'; text: string };

	const SCRIPT: Line[] = [
		{ kind: 'cmd', text: 'nuage start' },
		{ kind: 'out', text: 'daemon started, PID 4182' },
		{ kind: 'ok', text: '↑ synced 3 files' },
		{ kind: 'cmd', text: 'nuage share report.pdf' },
		{ kind: 'out', text: '→ nuage.facile.studio/s/k3f9a' }
	];

	let lines = $state<Line[]>([]);

	function reset() {
		lines = [];
	}

	function poster() {
		lines = [...SCRIPT];
	}

	async function typeLine(act: Act, text: string, interval: number) {
		lines.push({ kind: 'cmd', text: '' });
		const at = lines.length - 1;
		await act.type(text, (v) => (lines[at].text = v), interval);
	}

	async function script(act: Act) {
		await act.wait(600);
		for (const line of SCRIPT) {
			if (line.kind === 'cmd') {
				await typeLine(act, line.text, 70);
				await act.wait(450);
			} else {
				lines.push(line);
				await act.wait(800);
			}
		}
		await act.wait(1800);
		await typeLine(act, 'clear', 80);
		await act.wait(350);
		reset();
	}
</script>

<Tile title={t('landing.card.sync.title')} body={t('landing.card.sync.body')}>
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<Terminal style="top:16px;left:20px;right:20px;bottom:40px">
				<div class="mb-3 flex items-center gap-2">
					<span class="size-2 rounded-full bg-fc-success"></span>
					<span>~/Photos</span>
				</div>
				{#each lines as line, i (i)}
					<div class="whitespace-nowrap">
						{#if line.kind === 'cmd'}
							<span class="dim">$</span>
							{line.text}{#if i === lines.length - 1}<Caret />{/if}
						{:else}
							<span class:ok={line.kind === 'ok'} class:dim={line.kind === 'out'}>{line.text}</span>
						{/if}
					</div>
				{/each}
				{#if lines.at(-1)?.kind !== 'cmd'}
					<div class="whitespace-nowrap"><span class="dim">$</span> <Caret /></div>
				{/if}
			</Terminal>
		{/snippet}
	</Stage>
</Tile>

<style>
	.dim {
		color: var(--color-fc-on-inverse-muted);
	}
	.ok {
		color: color-mix(in oklab, var(--color-fc-success), var(--color-fc-on-inverse) 35%);
	}
</style>

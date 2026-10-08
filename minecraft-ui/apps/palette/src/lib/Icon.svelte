<script lang="ts">
	interface Props {
		name: 'trash' | 'plus' | 'shuffle' | 'x';
		class?: string;
	}

	let { name, class: className = 'size-4' }: Props = $props();

	const PATHS: Record<Props['name'], string> = {
		trash:
			'M3 6h18M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2m2 0v14a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V6M10 11v6M14 11v6',
		plus: 'M12 5v14M5 12h14',
		shuffle: 'M16 3h5v5M4 20 21 3M21 16v5h-5M15 15l6 6M4 4l5 5',
		x: 'M18 6 6 18M6 6l12 12'
	};
</script>

<svg
	viewBox="0 0 24 24"
	class={className}
	fill="none"
	stroke="currentColor"
	stroke-width={name === 'plus' ? 2.5 : 2}
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"><path d={PATHS[name]} /></svg
>

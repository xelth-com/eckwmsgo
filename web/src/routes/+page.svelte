<script>
	let count = $state(0);

	function increment() {
		count += 1;
	}
</script>

<main>
	<h1>ECKWMS</h1>
	<p>SvelteKit SPA + Go Backend</p>

	<button onclick={increment}>
		Count is {count}
	</button>
</main>

<style>
	main {
		text-align: center;
		padding: 2rem;
		font-family: system-ui, sans-serif;
	}

	h1 {
		color: #646cff;
	}

	button {
		padding: 0.5rem 1rem;
		font-size: 1rem;
		cursor: pointer;
		background: #646cff;
		color: white;
		border: none;
		border-radius: 4px;
	}

	button:hover {
		background: #535bf2;
	}
</style>

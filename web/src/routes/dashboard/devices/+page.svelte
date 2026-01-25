<script>
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { toastStore } from '$lib/stores/toastStore';
	import { base } from '$app/paths';

	let devices = [];
	let loading = true;
	let qrUrl = '';
	let showQr = false;

	async function loadDevices() {
		try {
			devices = await api.get('/api/admin/devices');
		} catch (e) {
			toastStore.add('Failed to load devices: ' + e.message, 'error');
		} finally {
			loading = false;
		}
	}

	async function updateStatus(deviceId, status) {
		try {
			await api.put(`/api/admin/devices/${deviceId}/status`, { status });
			toastStore.add(`Device ${status}`, 'success');
			await loadDevices();
		} catch (e) {
			toastStore.add(e.message, 'error');
		}
	}

	async function loadQr() {
		// Fetch the blob and create a local URL for security/auth handling
		try {
			const token = localStorage.getItem('auth_token');
			const res = await fetch(`${base}/api/internal/pairing-qr`, {
				headers: { Authorization: `Bearer ${token}` }
			});
			const blob = await res.blob();
			qrUrl = URL.createObjectURL(blob);
			showQr = true;
		} catch (e) {
			toastStore.add('Failed to load Pairing QR', 'error');
		}
	}

	onMount(() => {
		loadDevices();
	});
</script>

<div class="page">
	<header>
		<h1>Device Management</h1>
		<button class="btn primary" on:click={() => (showQr ? (showQr = false) : loadQr())}>
			{showQr ? 'Hide Pairing QR' : 'Show Pairing QR'}
		</button>
	</header>

	{#if showQr && qrUrl}
		<div class="qr-panel">
			<h3>Scan to Pair Device</h3>
			<img src={qrUrl} alt="Pairing QR" />
			<p class="hint">Use the ECK Scanner App to scan this code.</p>
		</div>
	{/if}

	<div class="device-list">
		{#if loading}
			<div class="loading">Loading devices...</div>
		{:else if devices.length === 0}
			<div class="empty">No devices registered. Scan the QR code to add one.</div>
		{:else}
			<table>
				<thead>
					<tr>
						<th>Status</th>
						<th>Device Name</th>
						<th>Device ID</th>
						<th>Last Seen</th>
						<th>Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each devices as device}
						<tr class={device.status}>
							<td>
								<span class="badge {device.status}">{device.status}</span>
							</td>
							<td>{device.name || 'Unknown'}</td>
							<td class="mono">{device.deviceId.substring(0, 8)}...</td>
							<td>{new Date(device.lastSeenAt).toLocaleString()}</td>
							<td class="actions">
								{#if device.status === 'pending' || device.status === 'blocked'}
									<button
										class="btn-icon approve"
										title="Approve"
										on:click={() => updateStatus(device.deviceId, 'active')}>✅</button
									>
								{/if}
								{#if device.status === 'active' || device.status === 'pending'}
									<button
										class="btn-icon block"
										title="Block"
										on:click={() => updateStatus(device.deviceId, 'blocked')}>⛔</button
									>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>

<style>
	header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 2rem;
	}
	h1 {
		color: #fff;
		margin: 0;
	}

	.qr-panel {
		background: #fff;
		padding: 2rem;
		border-radius: 12px;
		text-align: center;
		margin-bottom: 2rem;
		color: #000;
		max-width: 400px;
		margin-left: auto;
		margin-right: auto;
	}
	.qr-panel img {
		max-width: 100%;
		height: auto;
		display: block;
		margin: 0 auto;
	}

	table {
		width: 100%;
		border-collapse: collapse;
		background: #1e1e1e;
		border-radius: 8px;
		overflow: hidden;
	}
	th,
	td {
		padding: 1rem;
		text-align: left;
		border-bottom: 1px solid #333;
		color: #eee;
	}
	th {
		background: #252525;
		font-weight: 600;
		color: #888;
		text-transform: uppercase;
		font-size: 0.8rem;
	}

	.mono {
		font-family: monospace;
		color: #aaa;
	}

	.badge {
		padding: 4px 8px;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: bold;
		text-transform: uppercase;
	}
	.badge.active {
		background: rgba(40, 167, 69, 0.2);
		color: #28a745;
	}
	.badge.pending {
		background: rgba(255, 193, 7, 0.2);
		color: #ffc107;
	}
	.badge.blocked {
		background: rgba(220, 53, 69, 0.2);
		color: #dc3545;
	}

	.btn {
		padding: 0.6rem 1.2rem;
		border-radius: 6px;
		border: none;
		font-weight: 600;
		cursor: pointer;
	}
	.btn.primary {
		background: #4a69bd;
		color: white;
	}

	.btn-icon {
		background: none;
		border: none;
		font-size: 1.2rem;
		cursor: pointer;
		padding: 4px;
		transition: transform 0.2s;
	}
	.btn-icon:hover {
		transform: scale(1.2);
	}

	.empty,
	.loading {
		text-align: center;
		padding: 3rem;
		color: #666;
		font-style: italic;
	}
</style>

<script>
import { onMount } from 'svelte';
import { api } from '$lib/api';
import { toastStore } from '$lib/stores/toastStore.js';

let pickings = [];
let shipments = [];
let loading = true;
let error = null;
let activeTab = 'pickings'; // 'pickings' or 'shipments'
let processingPickings = new Set();
let isSyncing = false; // New state for OPAL sync
let expandedShipments = new Set(); // Track which shipments are expanded

onMount(async () => {
    await loadData();
});

async function loadData() {
    loading = true;
    error = null;
    try {
        // Load both pickings and shipments in parallel
        const [pickingsData, shipmentsData] = await Promise.all([
            api.get('/api/odoo/pickings?state=assigned'),
            api.get('/api/delivery/shipments')
        ]);
        pickings = pickingsData || [];
        shipments = shipmentsData || [];
    } catch (e) {
        console.error(e);
        error = e.message;
    } finally {
        loading = false;
    }
}

async function createShipment(pickingId) {
    if (processingPickings.has(pickingId)) return;

    processingPickings.add(pickingId);
    processingPickings = processingPickings; // Trigger reactivity

    try {
        await api.post('/api/delivery/shipments', {
            picking_id: pickingId,
            provider_code: 'opal'
        });

        // Reload data to show new shipment
        await loadData();

        // Switch to shipments tab to show the result
        activeTab = 'shipments';
    } catch (e) {
        alert('Failed to create shipment: ' + e.message);
    } finally {
        processingPickings.delete(pickingId);
        processingPickings = processingPickings;
    }
}

async function cancelShipment(pickingId) {
    if (!confirm('Are you sure you want to cancel this shipment?')) return;

    try {
        await api.post(`/api/delivery/shipments/${pickingId}/cancel`);
        await loadData();
    } catch (e) {
        alert('Failed to cancel shipment: ' + e.message);
    }
}

async function syncOpal() {
    isSyncing = true;
    toastStore.add('Syncing with OPAL...', 'info');
    try {
        await api.post('/api/delivery/import/opal', {});
        toastStore.add('Sync started. Refreshing data...', 'success');

        // Wait a bit before reloading to let the scraper start/finish
        setTimeout(async () => {
            await loadData();
            isSyncing = false;
        }, 4000);
    } catch (e) {
        toastStore.add('Sync failed: ' + e.message, 'error');
        isSyncing = false;
    }
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleDateString('de-DE', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function getStateColor(state) {
    const colors = {
        'draft': '#6c757d',
        'assigned': '#ffc107',
        'confirmed': '#17a2b8',
        'done': '#28a745',
        'cancel': '#dc3545'
    };
    return colors[state] || '#6c757d';
}

function getDeliveryStateColor(state) {
    const colors = {
        'pending': '#ffc107',
        'processing': '#17a2b8',
        'shipped': '#28a745',
        'delivered': '#28a745',
        'failed': '#dc3545',
        'cancelled': '#6c757d'
    };
    return colors[state] || '#6c757d';
}

function toggleShipmentDetails(id) {
    if (expandedShipments.has(id)) {
        expandedShipments.delete(id);
    } else {
        expandedShipments.add(id);
    }
    expandedShipments = expandedShipments; // Trigger reactivity
}

function parseRawResponse(rawResponse) {
    if (!rawResponse) return null;
    try {
        return JSON.parse(rawResponse);
    } catch {
        return null;
    }
}

function formatAddress(data, prefix) {
    if (!data) return '-';
    const parts = [
        data[`${prefix}_name`],
        data[`${prefix}_street`],
        [data[`${prefix}_zip`], data[`${prefix}_city`]].filter(Boolean).join(' ')
    ].filter(Boolean);
    return parts.join(', ') || '-';
}
</script>

<div class="shipping-page">
    <header>
        <h1>📦 Shipping & Delivery</h1>
        <div class="header-actions">
            <button class="action-btn secondary" on:click={syncOpal} disabled={isSyncing || loading}>
                {isSyncing ? '⏳ Syncing...' : '🔄 Sync OPAL'}
            </button>
            <button class="refresh-btn" on:click={loadData} disabled={loading}>
                {loading ? '↻ Loading...' : '↻ Refresh'}
            </button>
        </div>
    </header>

    <div class="tabs">
        <button
            class="tab"
            class:active={activeTab === 'pickings'}
            on:click={() => activeTab = 'pickings'}
        >
            📋 Ready to Ship ({pickings.length})
        </button>
        <button
            class="tab"
            class:active={activeTab === 'shipments'}
            on:click={() => activeTab = 'shipments'}
        >
            🚚 Shipments ({shipments.length})
        </button>
    </div>

    {#if loading && pickings.length === 0 && shipments.length === 0}
        <div class="loading">Loading shipping data...</div>
    {:else if error}
        <div class="error">Failed to load data: {error}</div>
    {:else}
        {#if activeTab === 'pickings'}
            <div class="pickings-section">
                <p class="section-desc">
                    These are Odoo Transfer Orders ready to be shipped. Click "Ship with OPAL" to create a delivery shipment.
                </p>

                {#if pickings.length === 0}
                    <div class="empty-state">
                        <p>✅ No pickings ready to ship</p>
                        <small>Pickings with status "assigned" will appear here</small>
                    </div>
                {:else}
                    <div class="table-container">
                        <table>
                            <thead>
                                <tr>
                                    <th>Picking #</th>
                                    <th>Origin</th>
                                    <th>Partner</th>
                                    <th>Location</th>
                                    <th>State</th>
                                    <th>Scheduled</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each pickings as picking}
                                    <tr>
                                        <td class="picking-name">{picking.name}</td>
                                        <td>{picking.origin || '-'}</td>
                                        <td>{picking.partner_id || '-'}</td>
                                        <td>
                                            <div class="location-cell">
                                                <span class="from">{picking.location_id || '-'}</span>
                                                <span class="arrow">→</span>
                                                <span class="to">{picking.location_dest_id || '-'}</span>
                                            </div>
                                        </td>
                                        <td>
                                            <span class="state-badge" style="background-color: {getStateColor(picking.state)}">
                                                {picking.state}
                                            </span>
                                        </td>
                                        <td>{formatDate(picking.scheduled_date)}</td>
                                        <td>
                                            <button
                                                class="action-btn ship-btn"
                                                on:click={() => createShipment(picking.id)}
                                                disabled={processingPickings.has(picking.id)}
                                            >
                                                {processingPickings.has(picking.id) ? '⏳ Processing...' : '🚚 Ship with OPAL'}
                                            </button>
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                {/if}
            </div>
        {:else}
            <div class="shipments-section">
                <p class="section-desc">
                    Active and past shipments created through the delivery system.
                </p>

                {#if shipments.length === 0}
                    <div class="empty-state">
                        <p>📭 No shipments yet</p>
                        <small>Create your first shipment from the "Ready to Ship" tab</small>
                    </div>
                {:else}
                    <div class="table-container">
                        <table>
                            <thead>
                                <tr>
                                    <th></th>
                                    <th>Tracking</th>
                                    <th>From → To</th>
                                    <th>Product</th>
                                    <th>Status</th>
                                    <th>Delivered</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each shipments as shipment}
                                    {@const details = parseRawResponse(shipment.raw_response)}
                                    <tr class="shipment-row" class:expanded={expandedShipments.has(shipment.id)} on:click={() => toggleShipmentDetails(shipment.id)}>
                                        <td class="expand-cell">
                                            <span class="expand-icon">{expandedShipments.has(shipment.id) ? '▼' : '▶'}</span>
                                        </td>
                                        <td class="tracking-cell">
                                            {#if shipment.tracking_number}
                                                <span class="tracking-number">{shipment.tracking_number}</span>
                                                {#if details?.hwb_number}
                                                    <span class="hwb-number">HWB: {details.hwb_number}</span>
                                                {/if}
                                            {:else}
                                                <span class="muted">Pending...</span>
                                            {/if}
                                        </td>
                                        <td class="route-cell">
                                            {#if details}
                                                <div class="route">
                                                    <span class="from">{details.pickup_name || details.pickup_city || '-'}</span>
                                                    <span class="arrow">→</span>
                                                    <span class="to">{details.delivery_name || details.delivery_city || '-'}</span>
                                                </div>
                                            {:else}
                                                <span class="muted">-</span>
                                            {/if}
                                        </td>
                                        <td>
                                            {#if details?.product_type}
                                                <span class="product-badge">{details.product_type}</span>
                                            {:else}
                                                <span class="muted">-</span>
                                            {/if}
                                        </td>
                                        <td>
                                            <span class="state-badge" style="background-color: {getDeliveryStateColor(shipment.status)}">
                                                {shipment.status}
                                            </span>
                                        </td>
                                        <td>
                                            {#if details?.status_date}
                                                <div class="delivery-info">
                                                    <span>{details.status_date} {details.status_time || ''}</span>
                                                    {#if details.receiver}
                                                        <span class="receiver">📝 {details.receiver}</span>
                                                    {/if}
                                                </div>
                                            {:else}
                                                <span class="muted">-</span>
                                            {/if}
                                        </td>
                                        <td on:click|stopPropagation>
                                            {#if shipment.status === 'pending' || shipment.status === 'processing'}
                                                <button
                                                    class="action-btn cancel-btn"
                                                    on:click={() => cancelShipment(shipment.picking_id)}
                                                >
                                                    ❌ Cancel
                                                </button>
                                            {:else}
                                                <span class="muted">-</span>
                                            {/if}
                                        </td>
                                    </tr>
                                    {#if expandedShipments.has(shipment.id) && details}
                                        <tr class="details-row">
                                            <td colspan="7">
                                                <div class="shipment-details">
                                                    <div class="details-grid">
                                                        <div class="detail-section">
                                                            <h4>📦 Pickup (Abholung)</h4>
                                                            <div class="detail-item">
                                                                <label>Company:</label>
                                                                <span>{details.pickup_name || '-'}</span>
                                                            </div>
                                                            {#if details.pickup_contact}
                                                            <div class="detail-item">
                                                                <label>Contact:</label>
                                                                <span>{details.pickup_contact}</span>
                                                            </div>
                                                            {/if}
                                                            <div class="detail-item">
                                                                <label>Address:</label>
                                                                <span>{details.pickup_street || '-'}, {details.pickup_zip} {details.pickup_city}</span>
                                                            </div>
                                                            {#if details.pickup_phone && details.pickup_phone !== '+49 ()'}
                                                            <div class="detail-item">
                                                                <label>Phone:</label>
                                                                <span>{details.pickup_phone}</span>
                                                            </div>
                                                            {/if}
                                                            <div class="detail-item">
                                                                <label>Date/Time:</label>
                                                                <span>{details.pickup_date || '-'} {details.pickup_time || ''}</span>
                                                            </div>
                                                            {#if details.pickup_note}
                                                            <div class="detail-item note">
                                                                <label>Note:</label>
                                                                <span>{details.pickup_note}</span>
                                                            </div>
                                                            {/if}
                                                        </div>

                                                        <div class="detail-section">
                                                            <h4>🚚 Delivery (Zustellung)</h4>
                                                            <div class="detail-item">
                                                                <label>Company:</label>
                                                                <span>{details.delivery_name || '-'}</span>
                                                            </div>
                                                            {#if details.delivery_contact}
                                                            <div class="detail-item">
                                                                <label>Contact:</label>
                                                                <span>{details.delivery_contact}</span>
                                                            </div>
                                                            {/if}
                                                            <div class="detail-item">
                                                                <label>Address:</label>
                                                                <span>{details.delivery_street || '-'}, {details.delivery_zip} {details.delivery_city}</span>
                                                            </div>
                                                            {#if details.delivery_phone && details.delivery_phone !== '+49 ()'}
                                                            <div class="detail-item">
                                                                <label>Phone:</label>
                                                                <span>{details.delivery_phone}</span>
                                                            </div>
                                                            {/if}
                                                            <div class="detail-item">
                                                                <label>Date/Time:</label>
                                                                <span>{details.delivery_date || '-'} {details.delivery_time || ''}</span>
                                                            </div>
                                                            {#if details.delivery_note}
                                                            <div class="detail-item note">
                                                                <label>Note:</label>
                                                                <span>{details.delivery_note}</span>
                                                            </div>
                                                            {/if}
                                                        </div>

                                                        <div class="detail-section">
                                                            <h4>📋 Package Info</h4>
                                                            {#if details.description}
                                                            <div class="detail-item">
                                                                <label>Contents:</label>
                                                                <span class="highlight">{details.description}</span>
                                                            </div>
                                                            {/if}
                                                            {#if details.package_count}
                                                            <div class="detail-item">
                                                                <label>Packages:</label>
                                                                <span>{details.package_count} pcs</span>
                                                            </div>
                                                            {/if}
                                                            {#if details.weight}
                                                            <div class="detail-item">
                                                                <label>Weight:</label>
                                                                <span>{details.weight} kg</span>
                                                            </div>
                                                            {/if}
                                                            {#if details.dimensions}
                                                            <div class="detail-item">
                                                                <label>Dimensions:</label>
                                                                <span>{details.dimensions} cm</span>
                                                            </div>
                                                            {/if}
                                                            {#if details.value}
                                                            <div class="detail-item">
                                                                <label>Value:</label>
                                                                <span class="value">{details.value.toLocaleString('de-DE')} EUR</span>
                                                            </div>
                                                            {/if}
                                                        </div>

                                                        <div class="detail-section">
                                                            <h4>📊 Status</h4>
                                                            <div class="detail-item">
                                                                <label>OPAL Status:</label>
                                                                <span class="status-value">{details.status || '-'}</span>
                                                            </div>
                                                            {#if details.status_date}
                                                            <div class="detail-item">
                                                                <label>Date/Time:</label>
                                                                <span>{details.status_date} {details.status_time || ''}</span>
                                                            </div>
                                                            {/if}
                                                            {#if details.receiver}
                                                            <div class="detail-item">
                                                                <label>Received by:</label>
                                                                <span class="highlight">{details.receiver}</span>
                                                            </div>
                                                            {/if}
                                                            {#if details.created_at}
                                                            <div class="detail-item">
                                                                <label>Created:</label>
                                                                <span>{details.created_at} by {details.created_by || '-'}</span>
                                                            </div>
                                                            {/if}
                                                        </div>
                                                    </div>
                                                </div>
                                            </td>
                                        </tr>
                                    {/if}
                                {/each}
                            </tbody>
                        </table>
                    </div>
                {/if}
            </div>
        {/if}
    {/if}
</div>

<style>
.shipping-page {
    padding: 0;
}

header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
}

h1 {
    font-size: 1.8rem;
    color: #fff;
    margin: 0;
}

.header-actions {
    display: flex;
    gap: 1rem;
}

.refresh-btn {
    padding: 0.6rem 1.2rem;
    border-radius: 4px;
    border: 1px solid #4a69bd;
    background: transparent;
    color: #4a69bd;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
}

.refresh-btn:hover:not(:disabled) {
    background: #4a69bd;
    color: white;
}

.refresh-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.action-btn.secondary {
    background: #333;
    color: #ccc;
    border: 1px solid #444;
}

.action-btn.secondary:hover:not(:disabled) {
    background: #444;
    color: #fff;
}

.tabs {
    display: flex;
    gap: 1rem;
    margin-bottom: 2rem;
    border-bottom: 2px solid #333;
}

.tab {
    padding: 0.8rem 1.5rem;
    border: none;
    background: transparent;
    color: #aaa;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    border-bottom: 3px solid transparent;
    transition: all 0.2s;
}

.tab:hover {
    color: #fff;
}

.tab.active {
    color: #4a69bd;
    border-bottom-color: #4a69bd;
}

.section-desc {
    color: #aaa;
    margin-bottom: 1.5rem;
    font-size: 0.95rem;
}

.loading, .error {
    text-align: center;
    padding: 3rem;
    color: #666;
    background: #1e1e1e;
    border-radius: 8px;
    border: 1px solid #333;
}

.error {
    color: #ff6b6b;
    border-color: #ff6b6b;
}

.empty-state {
    text-align: center;
    padding: 3rem;
    color: #666;
    background: #1e1e1e;
    border-radius: 8px;
    border: 1px dashed #333;
}

.empty-state p {
    font-size: 1.2rem;
    margin: 0 0 0.5rem 0;
}

.empty-state small {
    color: #555;
}

.table-container {
    background: #1e1e1e;
    border-radius: 8px;
    border: 1px solid #333;
    overflow-x: auto;
}

table {
    width: 100%;
    border-collapse: collapse;
}

thead {
    background: #252525;
}

th {
    padding: 1rem;
    text-align: left;
    font-weight: 600;
    color: #aaa;
    text-transform: uppercase;
    font-size: 0.75rem;
    letter-spacing: 0.5px;
    border-bottom: 2px solid #333;
}

td {
    padding: 1rem;
    border-bottom: 1px solid #2a2a2a;
    color: #e0e0e0;
}

tbody tr:hover {
    background: #252525;
}

.picking-name {
    font-family: monospace;
    color: #4a69bd;
    font-weight: 600;
}

.shipment-id {
    font-family: monospace;
    color: #4a69bd;
}

.location-cell {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.location-cell .arrow {
    color: #666;
}

.location-cell .from {
    color: #ffc107;
}

.location-cell .to {
    color: #28a745;
}

.state-badge {
    display: inline-block;
    padding: 0.3rem 0.8rem;
    border-radius: 12px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    color: white;
}

.provider-badge {
    display: inline-block;
    padding: 0.3rem 0.8rem;
    border-radius: 4px;
    background: #2a2a2a;
    font-family: monospace;
    font-size: 0.85rem;
    text-transform: uppercase;
    color: #4a69bd;
}

.tracking-link {
    color: #4a69bd;
    text-decoration: none;
    font-family: monospace;
}

.tracking-link:hover {
    text-decoration: underline;
}

.muted {
    color: #666;
    font-style: italic;
}

.action-btn {
    padding: 0.5rem 1rem;
    border-radius: 4px;
    border: none;
    font-weight: 600;
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 0.2s;
    white-space: nowrap;
}

.ship-btn {
    background: #28a745;
    color: white;
}

.ship-btn:hover:not(:disabled) {
    background: #218838;
}

.ship-btn:disabled {
    background: #555;
    cursor: not-allowed;
    opacity: 0.6;
}

.cancel-btn {
    background: transparent;
    border: 1px solid #dc3545;
    color: #dc3545;
}

.cancel-btn:hover {
    background: #dc3545;
    color: white;
}

/* Expandable rows */
.shipment-row {
    cursor: pointer;
    transition: background 0.2s;
}

.shipment-row:hover {
    background: #2a2a2a;
}

.shipment-row.expanded {
    background: #252525;
    border-bottom: none;
}

.expand-cell {
    width: 30px;
    text-align: center;
}

.expand-icon {
    color: #666;
    font-size: 0.8rem;
}

.tracking-cell {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
}

.tracking-number {
    font-family: monospace;
    color: #4a69bd;
    font-weight: 600;
}

.hwb-number {
    font-family: monospace;
    font-size: 0.75rem;
    color: #888;
}

.route-cell .route {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.route .from {
    color: #ffc107;
    font-weight: 500;
}

.route .arrow {
    color: #666;
}

.route .to {
    color: #28a745;
    font-weight: 500;
}

.product-badge {
    display: inline-block;
    padding: 0.2rem 0.6rem;
    border-radius: 4px;
    background: #2a2a2a;
    font-size: 0.8rem;
    color: #aaa;
}

.delivery-info {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
}

.receiver {
    font-size: 0.8rem;
    color: #28a745;
}

/* Details row */
.details-row {
    background: #1a1a1a;
}

.details-row td {
    padding: 0;
    border-bottom: 2px solid #333;
}

.shipment-details {
    padding: 1.5rem;
}

.details-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 1.5rem;
}

.detail-section {
    background: #252525;
    border-radius: 8px;
    padding: 1rem;
}

.detail-section h4 {
    margin: 0 0 1rem 0;
    color: #fff;
    font-size: 0.95rem;
    border-bottom: 1px solid #333;
    padding-bottom: 0.5rem;
}

.detail-item {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
    font-size: 0.9rem;
}

.detail-item label {
    color: #888;
    min-width: 80px;
    flex-shrink: 0;
}

.detail-item span {
    color: #e0e0e0;
}

.detail-item .highlight {
    color: #4a69bd;
    font-weight: 600;
}

.detail-item .value {
    color: #28a745;
    font-weight: 600;
}

.detail-item .status-value {
    color: #28a745;
    font-weight: 600;
    text-transform: uppercase;
}

.detail-item.note span {
    color: #ffc107;
    font-style: italic;
}
</style>

<script>
import { page } from '$app/stores';
import { onMount } from 'svelte';
import { api } from '$lib/api';
import { goto } from '$app/navigation';

let item = null;
let loading = true;
let error = null;

onMount(async () => {
    try {
        item = await api.get(`/api/items/${$page.params.id}`);
    } catch (e) {
        error = e.message;
    } finally {
        loading = false;
    }
});

function goBack() {
    goto('/dashboard/items');
}
</script>

<div class="detail-page">
<div class="header">
<button class="back-btn" on:click={goBack}>← Back</button>
{#if item}
<h1>{item.name}</h1>
{:else}
<h1>Item Details</h1>
{/if}
</div>

{#if loading}
    <div class="loading">Loading details...</div>
{:else if error}
    <div class="error">Error: {error}</div>
{:else if item}
    <div class="detail-grid">
        <!-- Main Info -->
        <div class="section main-info">
            <div class="field">
                <label>SKU</label>
                <div class="value code">{item.sku}</div>
            </div>
            <div class="field">
                <label>Barcode</label>
                <div class="value code">{item.barcode || '-'}</div>
            </div>
            <div class="field">
                <label>Category</label>
                <div class="value">{item.category || 'Uncategorized'}</div>
            </div>
            <div class="field full">
                <label>Description</label>
                <div class="value">{item.description || 'No description provided.'}</div>
            </div>
        </div>

        <!-- Stats -->
        <div class="section stats">
            <div class="stat-box">
                <span class="label">Quantity</span>
                <span class="val">{item.quantity}</span>
            </div>
            <div class="stat-box">
                <span class="label">Min Stock</span>
                <span class="val">{item.min_stock}</span>
            </div>
            <div class="stat-box">
                <span class="label">Price</span>
                <span class="val">${item.unit_price?.toFixed(2) || '0.00'}</span>
            </div>
        </div>

        <!-- Location Info -->
        <div class="section location">
            <h2>Location</h2>
            {#if item.place}
                <div class="location-card">
                    <div class="loc-icon">📍</div>
                    <div class="loc-details">
                        <span class="loc-name">{item.place.name}</span>
                        <span class="loc-coords">Row: {item.place.row}, Col: {item.place.column}</span>
                    </div>
                </div>
            {:else}
                <div class="empty-loc">Not assigned to a location</div>
            {/if}

            {#if item.box}
                <div class="location-card box-loc">
                    <div class="loc-icon">📦</div>
                    <div class="loc-details">
                        <span class="loc-name">Inside Box: {item.box.name || item.box.box_number}</span>
                    </div>
                </div>
            {/if}
        </div>
    </div>
{/if}
</div>

<style>
.detail-page { max-width: 800px; margin: 0 auto; }

.header { margin-bottom: 2rem; }
.back-btn { background: none; border: none; color: #888; cursor: pointer; font-size: 1rem; padding: 0; margin-bottom: 1rem; }
.back-btn:hover { color: #fff; }
h1 { color: #fff; font-size: 2rem; margin: 0; }

.detail-grid { display: grid; grid-template-columns: 2fr 1fr; gap: 2rem; }

.section { background: #1e1e1e; border: 1px solid #333; border-radius: 8px; padding: 1.5rem; }
.main-info { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; }
.field label { display: block; color: #666; font-size: 0.8rem; text-transform: uppercase; margin-bottom: 0.4rem; }
.field .value { color: #e0e0e0; font-size: 1.1rem; }
.field .code { font-family: monospace; background: #2a2a2a; padding: 2px 6px; border-radius: 4px; display: inline-block; }
.field.full { grid-column: 1 / -1; }

.stats { display: flex; flex-direction: column; gap: 1rem; }
.stat-box { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #333; padding-bottom: 0.5rem; }
.stat-box:last-child { border-bottom: none; padding-bottom: 0; }
.stat-box .label { color: #888; }
.stat-box .val { color: #fff; font-weight: 700; font-size: 1.2rem; }

.location { grid-column: 1 / -1; }
.location h2 { color: #ccc; font-size: 1.2rem; margin-top: 0; margin-bottom: 1rem; }
.location-card { display: flex; align-items: center; gap: 1rem; background: #252525; padding: 1rem; border-radius: 6px; margin-bottom: 0.5rem; }
.loc-icon { font-size: 1.5rem; }
.loc-details { display: flex; flex-direction: column; }
.loc-name { color: #fff; font-weight: 600; }
.loc-coords { color: #888; font-size: 0.9rem; }
.empty-loc { color: #666; font-style: italic; }

@media (max-width: 700px) {
.detail-grid { grid-template-columns: 1fr; }
.main-info { grid-template-columns: 1fr; }
}
</style>

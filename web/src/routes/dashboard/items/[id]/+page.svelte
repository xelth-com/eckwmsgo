<script>
    import { page } from '$app/stores';
    import { onMount } from 'svelte';
    import { api } from '$lib/api';
    import { goto } from '$app/navigation';
    import { toastStore } from '$lib/stores/toastStore';

    let item = null;
    let loading = true;
    let isEditing = false;
    let error = null;

    // Clone for editing
    let editForm = {};

    onMount(async () => {
        await loadItem();
    });

    async function loadItem() {
        try {
            item = await api.get(`/api/items/${$page.params.id}`);
            editForm = { ...item };
        } catch (e) {
            error = e.message;
        } finally {
            loading = false;
        }
    }

    function toggleEdit() {
        if (isEditing) {
            // Cancelled
            editForm = { ...item };
        }
        isEditing = !isEditing;
    }

    async function saveItem() {
        try {
            const updated = await api.put(`/api/items/${item.id}`, editForm);
            item = updated;
            editForm = { ...item };
            isEditing = false;
            toastStore.add('Item updated successfully', 'success');
        } catch (e) {
            toastStore.add(e.message, 'error');
        }
    }

    function goBack() {
        goto('/dashboard/items');
    }
</script>

<div class="detail-page">
    <div class="header">
        <button class="back-btn" on:click={goBack}>← Back</button>
        <div class="title-row">
            {#if item}
                <h1>{isEditing ? 'Editing: ' : ''}{item.name}</h1>
                <div class="actions">
                    {#if isEditing}
                        <button class="btn secondary" on:click={toggleEdit}>Cancel</button>
                        <button class="btn primary" on:click={saveItem}>Save</button>
                    {:else}
                        <button class="btn primary" on:click={toggleEdit}>Edit Item</button>
                    {/if}
                </div>
            {:else}
                <h1>Item Details</h1>
            {/if}
        </div>
    </div>

    {#if loading}
        <div class="loading">Loading details...</div>
    {:else if error}
        <div class="error">Error: {error}</div>
    {:else if item}
        <div class="detail-grid">
            <!-- Main Info -->
            <div class="section main-info">
                <h3>Basic Information</h3>
                <div class="field">
                    <label>SKU</label>
                    {#if isEditing}
                        <input type="text" bind:value={editForm.sku} class="code-input" />
                    {:else}
                        <div class="value code">{item.sku}</div>
                    {/if}
                </div>
                <div class="field">
                    <label>Name</label>
                    {#if isEditing}
                        <input type="text" bind:value={editForm.name} />
                    {:else}
                        <div class="value">{item.name}</div>
                    {/if}
                </div>
                <div class="field">
                    <label>Barcode</label>
                    {#if isEditing}
                        <input type="text" bind:value={editForm.barcode} class="code-input" />
                    {:else}
                        <div class="value code">{item.barcode || '-'}</div>
                    {/if}
                </div>
                <div class="field">
                    <label>Category</label>
                    {#if isEditing}
                        <input type="text" bind:value={editForm.category} />
                    {:else}
                        <div class="value">{item.category || 'Uncategorized'}</div>
                    {/if}
                </div>
                <div class="field full">
                    <label>Description</label>
                    {#if isEditing}
                        <textarea bind:value={editForm.description} rows="3"></textarea>
                    {:else}
                        <div class="value">{item.description || 'No description provided.'}</div>
                    {/if}
                </div>
            </div>

            <!-- Stats -->
            <div class="section stats">
                <h3>Inventory Status</h3>
                <div class="stat-box">
                    <span class="label">Quantity</span>
                    {#if isEditing}
                        <input type="number" bind:value={editForm.quantity} class="num-input" />
                    {:else}
                        <span class="val">{item.quantity}</span>
                    {/if}
                </div>
                <div class="stat-box">
                    <span class="label">Min Stock</span>
                    {#if isEditing}
                        <input type="number" bind:value={editForm.min_stock} class="num-input" />
                    {:else}
                        <span class="val">{item.min_stock}</span>
                    {/if}
                </div>
                <div class="stat-box">
                    <span class="label">Price</span>
                    {#if isEditing}
                        <input type="number" step="0.01" bind:value={editForm.unit_price} class="num-input" />
                    {:else}
                        <span class="val">${item.unit_price?.toFixed(2) || '0.00'}</span>
                    {/if}
                </div>
                <div class="stat-box">
                    <span class="label">Status</span>
                    {#if isEditing}
                        <select bind:value={editForm.is_active}>
                            <option value={true}>Active</option>
                            <option value={false}>Inactive</option>
                        </select>
                    {:else}
                        <span class="val {item.is_active ? 'active' : 'inactive'}">{item.is_active ? 'Active' : 'Inactive'}</span>
                    {/if}
                </div>
            </div>

            <!-- Location Info (Read-only for now) -->
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
    .detail-page { max-width: 900px; margin: 0 auto; }

    .header { margin-bottom: 2rem; }
    .back-btn { background: none; border: none; color: #888; cursor: pointer; font-size: 1rem; padding: 0; margin-bottom: 1rem; }
    .back-btn:hover { color: #fff; }

    .title-row { display: flex; justify-content: space-between; align-items: center; }
    h1 { color: #fff; font-size: 2rem; margin: 0; }

    .actions { display: flex; gap: 10px; }

    .btn {
        padding: 0.6rem 1.2rem;
        border-radius: 4px;
        border: none;
        font-weight: 600;
        cursor: pointer;
        font-size: 0.9rem;
    }
    .btn.primary { background: #4a69bd; color: white; }
    .btn.secondary { background: #444; color: #ccc; }
    .btn:hover { opacity: 0.9; }

    .detail-grid { display: grid; grid-template-columns: 2fr 1fr; gap: 2rem; }

    .section { background: #1e1e1e; border: 1px solid #333; border-radius: 8px; padding: 1.5rem; }
    .section h3 { margin-top: 0; color: #6bc5f0; font-size: 1.1rem; border-bottom: 1px solid #333; padding-bottom: 10px; margin-bottom: 15px; }

    .main-info { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; }
    .field label { display: block; color: #666; font-size: 0.8rem; text-transform: uppercase; margin-bottom: 0.4rem; }
    .field .value { color: #e0e0e0; font-size: 1.1rem; }
    .field .code { font-family: monospace; background: #2a2a2a; padding: 2px 6px; border-radius: 4px; display: inline-block; }
    .field.full { grid-column: 1 / -1; }

    /* Inputs */
    input, textarea, select {
        width: 100%;
        background: #121212;
        border: 1px solid #444;
        color: white;
        padding: 8px;
        border-radius: 4px;
        font-size: 1rem;
        box-sizing: border-box;
    }
    input:focus, textarea:focus { border-color: #4a69bd; outline: none; }
    .code-input { font-family: monospace; }
    .num-input { text-align: right; }

    .stats { display: flex; flex-direction: column; gap: 1rem; }
    .stat-box { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #333; padding-bottom: 0.5rem; }
    .stat-box:last-child { border-bottom: none; padding-bottom: 0; }
    .stat-box .label { color: #888; }
    .stat-box .val { color: #fff; font-weight: 700; font-size: 1.2rem; }
    .stat-box .active { color: #28a745; }
    .stat-box .inactive { color: #555; }

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

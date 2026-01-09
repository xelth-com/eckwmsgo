<script>
    import { onMount } from 'svelte';
    import { api } from '$lib/api';

    let items = [];
    let loading = true;
    let error = null;

    onMount(async () => {
        try {
            items = await api.get('/api/items');
        } catch (e) {
            console.error(e);
            error = e.message;
        } finally {
            loading = false;
        }
    });
</script>

<div class="dashboard-home">
    <header>
        <h1>Inventory Overview</h1>
        <div class="actions">
            <button class="action-btn primary">+ Add Item</button>
        </div>
    </header>

    {#if loading}
        <div class="loading">Loading inventory...</div>
    {:else if error}
        <div class="error">Failed to load items: {error}</div>
    {:else}
        <div class="grid-container">
            {#if items.length === 0}
                <div class="empty-state">No items found. Start by adding one.</div>
            {/if}

            {#each items as item}
                <div class="card item-card">
                    <div class="card-header">
                        <span class="sku">{item.sku}</span>
                        <span class="status {item.is_active ? 'active' : 'inactive'}">
                            {item.is_active ? 'Active' : 'Inactive'}
                        </span>
                    </div>
                    <div class="card-body">
                        <h3>{item.name}</h3>
                        <p class="desc">{item.description || 'No description'}</p>
                    </div>
                    <div class="card-footer">
                        <div class="stat">
                            <span class="label">Qty</span>
                            <span class="value">{item.quantity}</span>
                        </div>
                        <div class="stat">
                            <span class="label">Location</span>
                            <span class="value">{item.place?.name || '-'}</span>
                        </div>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 2rem;
    }

    h1 {
        font-size: 1.8rem;
        color: #fff;
        margin: 0;
    }

    .action-btn {
        padding: 0.6rem 1.2rem;
        border-radius: 4px;
        border: none;
        font-weight: 600;
        cursor: pointer;
    }

    .action-btn.primary {
        background: #4a69bd;
        color: white;
    }

    .grid-container {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 1.5rem;
    }

    .card {
        background: #1e1e1e;
        border: 1px solid #333;
        border-radius: 8px;
        padding: 1.2rem;
        display: flex;
        flex-direction: column;
        transition: transform 0.2s, border-color 0.2s;
    }

    .card:hover {
        transform: translateY(-2px);
        border-color: #555;
    }

    .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 0.8rem;
    }

    .sku {
        font-family: monospace;
        color: #888;
        background: #2a2a2a;
        padding: 2px 6px;
        border-radius: 4px;
        font-size: 0.8rem;
    }

    .status {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        display: inline-block;
    }

    .status.active { background: #28a745; box-shadow: 0 0 5px rgba(40,167,69,0.5); }
    .status.inactive { background: #555; }

    .card-body h3 {
        margin: 0 0 0.5rem 0;
        color: #e0e0e0;
        font-size: 1.1rem;
    }

    .desc {
        color: #888;
        font-size: 0.9rem;
        margin: 0;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
    }

    .card-footer {
        margin-top: auto;
        padding-top: 1rem;
        border-top: 1px solid #333;
        display: flex;
        gap: 1.5rem;
    }

    .stat {
        display: flex;
        flex-direction: column;
    }

    .stat .label {
        font-size: 0.7rem;
        color: #666;
        text-transform: uppercase;
    }

    .stat .value {
        font-size: 1rem;
        font-weight: 600;
        color: #fff;
    }

    .empty-state {
        grid-column: 1 / -1;
        text-align: center;
        padding: 3rem;
        color: #666;
        background: #1e1e1e;
        border-radius: 8px;
        border: 1px dashed #333;
    }
</style>

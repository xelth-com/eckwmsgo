<script>
    import { page } from '$app/stores';
    import { onMount } from 'svelte';
    import { api } from '$lib/api';
    import { goto } from '$app/navigation';
    import { toastStore } from '$lib/stores/toastStore';

    let whId = $page.params.id;
    let warehouse = null;
    let loading = true;
    let error = null;
    let racks = [];

    // Canvas settings
    const GRID_SIZE = 50; // px

    onMount(async () => {
        await loadWarehouse();
    });

    async function loadWarehouse() {
        try {
            loading = true;
            warehouse = await api.get(`/api/warehouse/${whId}`);
            racks = warehouse.racks || [];
        } catch (e) {
            error = e.message;
            toastStore.add('Error loading warehouse', 'error');
        } finally {
            loading = false;
        }
    }

    function goBack() {
        goto('/dashboard/warehouse');
    }

    function getRackStyle(rack) {
        // Use visual dims if available, else calc from cols/rows * grid
        const width = (rack.visual_width > 0 ? rack.visual_width : rack.columns * GRID_SIZE);
        const height = (rack.visual_height > 0 ? rack.visual_height : rack.rows * GRID_SIZE);

        return `
            left: ${rack.posX || 0}px;
            top: ${rack.posY || 0}px;
            width: ${width}px;
            height: ${height}px;
            transform: rotate(${rack.rotation || 0}deg);
        `;
    }
</script>

<div class="blueprint-page">
    <div class="header">
        <button class="back-btn" on:click={goBack}>← Back</button>
        {#if warehouse}
            <h1>{warehouse.name} <span class="blueprint-label">Blueprint</span></h1>
        {:else}
            <h1>Warehouse Blueprint</h1>
        {/if}
    </div>

    {#if loading}
        <div class="loading">Loading blueprint...</div>
    {:else if error}
        <div class="error">{error}</div>
    {:else}
        <div class="viewport">
            <div class="canvas">
                {#each racks as rack}
                    <!-- svelte-ignore a11y-click-events-have-key-events -->
                    <!-- svelte-ignore a11y-no-static-element-interactions -->
                    <div
                        class="rack"
                        style={getRackStyle(rack)}
                        title="{rack.name} ({rack.columns}x{rack.rows})"
                    >
                        <div class="rack-label" style="transform: rotate(-{rack.rotation || 0}deg);">
                            <span class="rack-name">{rack.name}</span>
                            <span class="rack-info">{rack.columns}×{rack.rows}</span>
                        </div>
                    </div>
                {/each}

                {#if racks.length === 0}
                    <div class="empty-canvas">
                        No racks defined. Use Admin Panel to setup layout.
                    </div>
                {/if}
            </div>
        </div>

        <div class="controls">
            <div class="legend">
                <div class="legend-item"><span class="box rack-box"></span> Rack</div>
                <div class="legend-item"><span class="box selected-box"></span> Selected</div>
            </div>
            <div class="info">
                {racks.length} racks total
            </div>
        </div>
    {/if}
</div>

<style>
    .blueprint-page {
        height: calc(100vh - 4rem); /* Adjust for dashboard padding */
        display: flex;
        flex-direction: column;
    }

    .header {
        margin-bottom: 1rem;
        flex-shrink: 0;
    }

    .back-btn { background: none; border: none; color: #888; cursor: pointer; font-size: 1rem; padding: 0; margin-bottom: 0.5rem; }
    .back-btn:hover { color: #fff; }

    h1 { color: #fff; font-size: 1.8rem; margin: 0; display: flex; align-items: center; gap: 10px; }
    .blueprint-label { font-size: 0.8rem; background: #333; padding: 2px 8px; border-radius: 4px; color: #aaa; text-transform: uppercase; letter-spacing: 1px; font-weight: normal; }

    .viewport {
        flex: 1;
        background-color: #1a1a1a;
        border: 1px solid #333;
        border-radius: 8px;
        overflow: auto;
        position: relative;
        background-image:
            linear-gradient(#222 1px, transparent 1px),
            linear-gradient(90deg, #222 1px, transparent 1px);
        background-size: 50px 50px;
    }

    .canvas {
        width: 3000px; /* Large canvas */
        height: 3000px;
        position: relative;
    }

    .rack {
        position: absolute;
        background-color: rgba(74, 105, 189, 0.2);
        border: 2px solid #4a69bd;
        border-radius: 4px;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: background-color 0.2s, box-shadow 0.2s;
        box-shadow: 0 4px 6px rgba(0,0,0,0.3);
    }

    .rack:hover {
        background-color: rgba(74, 105, 189, 0.4);
        box-shadow: 0 0 15px rgba(74, 105, 189, 0.5);
        z-index: 10;
    }

    .rack-label {
        text-align: center;
        pointer-events: none;
    }

    .rack-name {
        display: block;
        color: #fff;
        font-weight: 700;
        font-size: 0.9rem;
        text-shadow: 0 1px 2px rgba(0,0,0,0.8);
    }

    .rack-info {
        display: block;
        color: #fbbf24;
        font-size: 0.75rem;
        font-weight: 600;
    }

    .empty-canvas {
        position: absolute;
        top: 50px;
        left: 50px;
        color: #666;
        font-style: italic;
    }

    .controls {
        margin-top: 1rem;
        padding: 1rem;
        background: #1e1e1e;
        border: 1px solid #333;
        border-radius: 8px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-shrink: 0;
    }

    .legend { display: flex; gap: 1.5rem; }
    .legend-item { display: flex; align-items: center; gap: 0.5rem; color: #aaa; font-size: 0.9rem; }
    .box { width: 16px; height: 16px; border-radius: 3px; display: inline-block; }
    .rack-box { background: rgba(74, 105, 189, 0.2); border: 2px solid #4a69bd; }
    .selected-box { background: rgba(255, 255, 255, 0.1); border: 2px solid #fff; }

    .info { color: #666; font-size: 0.9rem; }
</style>

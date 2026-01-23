<script>
    import { onMount, onDestroy } from 'svelte';
    import { base } from '$app/paths';

    let meshNodes = [];
    let loading = true;
    let pollInterval;

    async function fetchMeshNodes() {
        try {
            const response = await fetch(`${base}/mesh/nodes`);
            if (response.ok) {
                meshNodes = await response.json();
                loading = false;
            }
        } catch (error) {
            console.error('Failed to fetch mesh nodes:', error);
            loading = false;
        }
    }

    onMount(() => {
        fetchMeshNodes();
        // Poll every 30 seconds
        pollInterval = setInterval(fetchMeshNodes, 30000);
    });

    onDestroy(() => {
        if (pollInterval) clearInterval(pollInterval);
    });

    function getNodeIcon(role) {
        switch (role) {
            case 'master': return '🌐';
            case 'peer': return '🖥️';
            case 'edge': return '📱';
            default: return '🔗';
        }
    }

    function getNodeLabel(node) {
        if (node.instance_id === 'global_prod_01') return 'Production';
        if (node.instance_id === 'local_windows_dev') return 'Local Dev';
        return node.instance_id;
    }
</script>

<div class="mesh-status">
    {#if loading}
        <div class="mesh-node loading">
            <span class="node-icon">⏳</span>
            <span class="node-label">Loading...</span>
        </div>
    {:else if meshNodes.length === 0}
        <div class="mesh-node offline">
            <span class="node-icon">⚠️</span>
            <span class="node-label">No nodes</span>
        </div>
    {:else}
        {#each meshNodes as node}
            <div class="mesh-node" class:online={node.is_online} class:offline={!node.is_online}>
                <span class="node-icon">{getNodeIcon(node.role)}</span>
                <span class="node-label">{getNodeLabel(node)}</span>
                <span class="node-status" class:online={node.is_online}></span>
            </div>
        {/each}
    {/if}
</div>

<style>
    .mesh-status {
        display: flex;
        flex-direction: column;
        gap: 4px;
        font-size: 0.7rem;
    }

    .mesh-node {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 8px;
        border-radius: 4px;
        background: #1a1a1a;
        border: 1px solid #333;
        transition: all 0.2s;
    }

    .mesh-node.online {
        background: rgba(40, 167, 69, 0.1);
        border-color: rgba(40, 167, 69, 0.3);
    }

    .mesh-node.offline {
        background: rgba(220, 53, 69, 0.1);
        border-color: rgba(220, 53, 69, 0.3);
        opacity: 0.7;
    }

    .mesh-node.loading {
        background: rgba(255, 193, 7, 0.1);
        border-color: rgba(255, 193, 7, 0.3);
    }

    .node-icon {
        font-size: 1rem;
        line-height: 1;
    }

    .node-label {
        flex: 1;
        font-weight: 600;
        color: #ccc;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .mesh-node.online .node-label {
        color: #28a745;
    }

    .mesh-node.offline .node-label {
        color: #dc3545;
    }

    .node-status {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: #666;
    }

    .node-status.online {
        background: #28a745;
        box-shadow: 0 0 6px rgba(40, 167, 69, 0.6);
    }
</style>

<script>
    import { authStore } from '$lib/stores/authStore';
    import { wsStore } from '$lib/stores/wsStore';
    import { toastStore } from '$lib/stores/toastStore';
    import ToastContainer from '$lib/components/ToastContainer.svelte';
    import { goto } from '$app/navigation';
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/stores';

    onMount(() => {
        // 1. Auth Guard
        const unsubscribeAuth = authStore.subscribe(state => {
            if (!state.isLoading && !state.isAuthenticated) {
                goto('/login');
            }
        });

        // 2. Init WebSocket
        wsStore.connect();

        return () => {
            unsubscribeAuth();
        };
    });

    onDestroy(() => {
        // Don't close WS on destroy of layout if navigating within dashboard,
        // but fine for now as +layout is persistent.
    });

    function handleLogout() {
        authStore.logout();
        wsStore.close();
        goto('/login');
    }

    // Reactive listener for WebSocket messages
    $: if ($wsStore.lastMessage) {
        handleWsMessage($wsStore.lastMessage);
    }

    function handleWsMessage(msg) {
        // Prevent processing if message is too old (basic check)
        if (Date.now() - (msg._receivedAt || 0) > 2000) return;

        // Handle Scan Events
        if (msg.barcode || (msg.data && msg.data.barcode)) {
             const barcode = msg.barcode || msg.data.barcode;
             processScan(barcode);
             return;
        }

        if (msg.success && msg.data) {
             toastStore.add(`Operation Success`, 'success');
        } else if (msg.type === 'ERROR' || msg.error) {
             toastStore.add(msg.text || msg.error || 'Error occurred', 'error');
        } else if (msg.text) {
             toastStore.add(msg.text, 'info');
        }
    }

    function processScan(barcode) {
        // Play sound (optional, browser policy might block)
        // const audio = new Audio('/beep.mp3'); audio.play().catch(e=>{});

        toastStore.add(`Scanned: ${barcode}`, 'info');

        // Logic routing based on barcode prefix
        if (barcode.startsWith('i')) {
            // Item Scan
            // Remove 'i' prefix logic? Usually ID is full string 'i7...'
            goto(`/dashboard/items/${barcode}`);
        } else if (barcode.startsWith('RMA')) {
            // RMA Scan - Find by RMA number
            // We might need an API lookup first, but for now assuming direct link or search
            // If backend supports finding by code, we can redirect.
            // For now, let's just go to list and toast, or search page if we had one.
            toastStore.add(`RMA Scanned: ${barcode}`, 'success');
            // goto('/dashboard/rma?search=' + barcode); // Future improvement
        } else if (barcode.startsWith('b')) {
            // Box Scan
            toastStore.add(`Box Scanned: ${barcode}`, 'info');
            // goto(`/dashboard/boxes/${barcode}`);
        } else {
            // Unknown
            toastStore.add(`Unknown barcode type: ${barcode}`, 'warning');
        }
    }
</script>

<div class="dashboard-layout">
    <aside class="sidebar">
        <div class="brand">
            <span class="brand-text">eckWMS</span>
            <div class="connection-status" class:connected={$wsStore.connected}>
                {$wsStore.connected ? 'ONLINE' : 'OFFLINE'}
            </div>
        </div>

        <nav>
            <a href="/dashboard" class:active={$page.url.pathname === '/dashboard'}>
                Dashboard
            </a>
            <a href="/dashboard/items" class:active={$page.url.pathname.includes('/items')}>
                Inventory
            </a>
            <a href="/dashboard/warehouse" class:active={$page.url.pathname.includes('/warehouse')}>
                Warehouse
            </a>
            <a href="/dashboard/rma" class:active={$page.url.pathname.includes('/rma')}>
                RMA Requests
            </a>
        </nav>

        <div class="user-panel">
            <div class="user-info">
                <span class="username">{$authStore.currentUser?.username || 'User'}</span>
                <span class="role">{$authStore.currentUser?.role || 'Operator'}</span>
            </div>
            <button on:click={handleLogout} class="logout-btn">Logout</button>
        </div>
    </aside>

    <main class="content">
        <slot />
    </main>

    <ToastContainer />
</div>

<style>
    .dashboard-layout {
        display: grid;
        grid-template-columns: 250px 1fr;
        height: 100vh;
        overflow: hidden;
    }

    .sidebar {
        background: #1e1e1e;
        border-right: 1px solid #333;
        display: flex;
        flex-direction: column;
        padding: 1rem;
    }

    .brand {
        padding: 1rem 0 2rem 0;
        text-align: center;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.5rem;
    }

    .brand-text {
        font-size: 1.5rem;
        font-weight: 800;
        color: #4a69bd;
        letter-spacing: 1px;
    }

    .connection-status {
        font-size: 0.7rem;
        font-weight: 700;
        padding: 2px 6px;
        border-radius: 4px;
        background: #333;
        color: #666;
    }

    .connection-status.connected {
        background: rgba(40, 167, 69, 0.2);
        color: #28a745;
    }

    nav {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    nav a {
        padding: 0.8rem 1rem;
        color: #aaa;
        text-decoration: none;
        border-radius: 6px;
        transition: all 0.2s;
        font-weight: 500;
    }

    nav a:hover {
        background: #2a2a2a;
        color: #fff;
    }

    nav a.active {
        background: #4a69bd;
        color: white;
    }

    .user-panel {
        border-top: 1px solid #333;
        padding-top: 1rem;
        margin-top: 1rem;
    }

    .user-info {
        display: flex;
        flex-direction: column;
        margin-bottom: 1rem;
    }

    .username {
        color: #fff;
        font-weight: 600;
    }

    .role {
        color: #666;
        font-size: 0.8rem;
        text-transform: uppercase;
    }

    .logout-btn {
        width: 100%;
        background: #2a2a2a;
        color: #ff6b6b;
        border: 1px solid #333;
        padding: 0.5rem;
        border-radius: 4px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .logout-btn:hover {
        background: #333;
        border-color: #ff6b6b;
    }

    .content {
        overflow-y: auto;
        padding: 2rem;
        background: #121212;
    }
</style>

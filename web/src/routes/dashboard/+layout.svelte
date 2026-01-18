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
                goto('login');
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
        goto('login');
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

    async function processScan(barcode) {
        // Play sound (optional, browser policy might block)
        // const audio = new Audio('/beep.mp3'); audio.play().catch(e=>{});

        toastStore.add('Scanning...', 'info', 1000);

        try {
            const res = await fetch('/api/scan', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${$authStore.token}`
                },
                body: JSON.stringify({ barcode })
            });

            if (!res.ok) {
                const err = await res.json();
                throw new Error(err.error || 'Scan failed');
            }

            const data = await res.json();

            // Show result
            toastStore.add(data.message, 'success');

            // Handle Navigation / Action based on type
            if (data.type === 'item' && data.data?.id) {
                // Navigate to item detail using internal ID
                goto(`items/${data.data.id}`);
            } else if (data.type === 'box' && data.data?.id) {
                // Box detail page pending - just show console log for now
                console.log('Box scanned:', data.data);
                toastStore.add(`Box ${data.data.name || data.data.id} scanned`, 'success');
            } else if (data.type === 'place' && data.data?.id) {
                goto(`warehouse/${data.data.id}`);
            } else if (data.type === 'product' && data.data?.id) {
                goto(`items/${data.data.id}`);
            } else if (data.type === 'label') {
                // Label codes contain action metadata - just log for now
                console.log('Label scanned:', data.data);
            }

        } catch (e) {
            console.error('Scan error:', e);
            toastStore.add(`Error: ${e.message}`, 'error');
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
            <a href="." class:active={$page.url.pathname === '/dashboard'}>
                Dashboard
            </a>
            <a href="items" class:active={$page.url.pathname.includes('/items')}>
                Inventory
            </a>
            <a href="warehouse" class:active={$page.url.pathname.includes('/warehouse')}>
                Warehouse
            </a>
            <a href="rma" class:active={$page.url.pathname.includes('/rma')}>
                RMA Requests
            </a>
            <a href="print" class:active={$page.url.pathname.includes('/print')}>
                Printing
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

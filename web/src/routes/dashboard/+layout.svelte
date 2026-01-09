<script>
    import { authStore } from '$lib/stores/authStore';
    import { goto } from '$app/navigation';
    import { onMount } from 'svelte';
    import { page } from '$app/stores';

    onMount(() => {
        const unsubscribe = authStore.subscribe(state => {
            if (!state.isLoading && !state.isAuthenticated) {
                goto('/login');
            }
        });
        return unsubscribe;
    });

    function handleLogout() {
        authStore.logout();
        goto('/login');
    }
</script>

<div class="dashboard-layout">
    <aside class="sidebar">
        <div class="brand">
            <span class="brand-text">eckWMS</span>
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
    }

    .brand-text {
        font-size: 1.5rem;
        font-weight: 800;
        color: #4a69bd;
        letter-spacing: 1px;
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

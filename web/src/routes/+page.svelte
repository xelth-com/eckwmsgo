<script>
    import { onMount } from 'svelte';
    import { goto } from '$app/navigation';
    import { authStore } from '$lib/stores/authStore';

    onMount(() => {
        const unsubscribe = authStore.subscribe(state => {
            if (!state.isLoading) {
                if (state.isAuthenticated) {
                    goto('/dashboard');
                } else {
                    goto('/login');
                }
            }
        });
        return unsubscribe;
    });
</script>

<div class="loading-screen">
    <div class="spinner"></div>
    <p>Loading eckWMS...</p>
</div>

<style>
    .loading-screen {
        height: 100vh;
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        background-color: var(--bg-color);
    }
    .spinner {
        width: 40px;
        height: 40px;
        border: 4px solid rgba(255,255,255,0.1);
        border-left-color: var(--accent-color);
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin-bottom: 1rem;
    }
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
</style>

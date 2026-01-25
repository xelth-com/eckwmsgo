import { get } from 'svelte/store';
import { authStore } from '$lib/stores/authStore';
import { base } from '$app/paths';

const BASE_URL = base || '/E'; // Use SvelteKit base path or fallback to /E

async function request(endpoint, options = {}) {
    const state = get(authStore);
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers
    };

    if (state.token) {
        headers['Authorization'] = `Bearer ${state.token}`;
    }

    const config = {
        ...options,
        headers
    };

    const response = await fetch(`${BASE_URL}${endpoint}`, config);

    if (response.status === 401) {
        authStore.logout();
        if (typeof window !== 'undefined') {
            // FIX: Robust base path handling
            // 1. Try to get base from current URL (anything before /dashboard)
            // 2. Fallback to /E if on root
            let basePath = '/E';
            if (window.location.pathname.includes('/dashboard')) {
                basePath = window.location.pathname.split('/dashboard')[0] || '/E';
            }
            window.location.href = `${basePath}/login`;
        }
        throw new Error('Unauthorized');
    }

    if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Request failed: ${response.status}`);
    }

    return response.json();
}

export const api = {
    get: (endpoint) => request(endpoint, { method: 'GET' }),
    post: (endpoint, body) => request(endpoint, { method: 'POST', body: JSON.stringify(body) }),
    put: (endpoint, body) => request(endpoint, { method: 'PUT', body: JSON.stringify(body) }),
    delete: (endpoint) => request(endpoint, { method: 'DELETE' })
};

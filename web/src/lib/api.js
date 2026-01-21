import { get } from 'svelte/store';
import { authStore } from '$lib/stores/authStore';

const BASE_URL = ''; // Relative path for proxy/embedded serving

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
            // Use base path from current location to construct proper login URL
            const basePath = window.location.pathname.split('/dashboard')[0] || '';
            window.location.href = basePath + '/login';
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

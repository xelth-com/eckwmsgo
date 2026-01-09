import { writable } from 'svelte/store';
import { browser } from '$app/environment';

// Simplified auth store based on the snapshot
const initialState = {
  isAuthenticated: false,
  currentUser: null,
  token: null,
  isLoading: true
};

function createAuthStore() {
  const { subscribe, set, update } = writable(initialState);

  return {
    subscribe,
    init: () => {
        if (!browser) return;
        const token = localStorage.getItem('auth_token');
        if (token) {
            update(s => ({ ...s, isAuthenticated: true, token, isLoading: false }));
        } else {
            update(s => ({ ...s, isLoading: false }));
        }
    },
    login: async (email, password) => {
        update(s => ({ ...s, isLoading: true }));
        try {
            const res = await fetch('/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, password })
            });

            if (!res.ok) throw new Error('Login failed');

            const data = await res.json();
            if (browser) {
                localStorage.setItem('auth_token', data.tokens.accessToken);
            }

            update(s => ({
                ...s,
                isAuthenticated: true,
                currentUser: data.user,
                token: data.tokens.accessToken,
                isLoading: false
            }));
            return { success: true };
        } catch (e) {
            update(s => ({ ...s, isLoading: false }));
            return { success: false, error: e.message };
        }
    },
    logout: () => {
        if (browser) localStorage.removeItem('auth_token');
        set({ ...initialState, isLoading: false });
    }
  };
}

export const authStore = createAuthStore();

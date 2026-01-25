<script>
    import { authStore } from '$lib/stores/authStore';
    import { goto } from '$app/navigation';
    import { base } from '$app/paths';

    let email = '';
    let password = '';
    let error = '';
    let isLoading = false;

    async function handleLogin() {
        if (!email || !password) {
            error = 'Please fill in all fields';
            return;
        }

        isLoading = true;
        error = '';

        const result = await authStore.login(email, password);

        if (result.success) {
            // FIX: Robust base path handling. If base is empty, default to '/E' for production consistency.
            const pathBase = base || '/E';
            goto(`${pathBase}/dashboard`);
        } else {
            error = result.error || 'Login failed';
        }
        isLoading = false;
    }
</script>

<div class="login-container">
    <div class="login-card">
        <div class="logo">
            <h1>eckWMS</h1>
            <span class="version">GO Edition</span>
        </div>

        <form on:submit|preventDefault={handleLogin}>
            <div class="form-group">
                <label for="email">Email</label>
                <input
                    type="text"
                    id="email"
                    bind:value={email}
                    placeholder="operator@eckwms.local"
                    disabled={isLoading}
                />
            </div>

            <div class="form-group">
                <label for="password">Password</label>
                <input
                    type="password"
                    id="password"
                    bind:value={password}
                    placeholder="••••••••"
                    disabled={isLoading}
                />
            </div>

            {#if error}
                <div class="error-msg">{error}</div>
            {/if}

            <button type="submit" disabled={isLoading}>
                {isLoading ? 'Authenticating...' : 'Login'}
            </button>
        </form>
    </div>
</div>

<style>
    .login-container {
        height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background-color: #1e1e1e;
        background-image: radial-gradient(#2a2a2a 1px, transparent 1px);
        background-size: 20px 20px;
    }

    .login-card {
        background: #2c2c2e;
        padding: 2.5rem;
        border-radius: 8px;
        width: 100%;
        max-width: 400px;
        box-shadow: 0 10px 25px rgba(0,0,0,0.5);
        border: 1px solid #444;
    }

    .logo {
        text-align: center;
        margin-bottom: 2rem;
    }

    .logo h1 {
        margin: 0;
        color: #4a69bd;
        font-size: 2rem;
        font-weight: 700;
        letter-spacing: -1px;
    }

    .version {
        font-size: 0.8rem;
        color: #666;
        text-transform: uppercase;
        letter-spacing: 2px;
    }

    .form-group {
        margin-bottom: 1.5rem;
    }

    label {
        display: block;
        margin-bottom: 0.5rem;
        color: #aaa;
        font-size: 0.9rem;
    }

    input {
        width: 100%;
        padding: 0.75rem;
        background: #1a1a1a;
        border: 1px solid #444;
        border-radius: 4px;
        color: #fff;
        font-size: 1rem;
        transition: border-color 0.2s;
        box-sizing: border-box;
    }

    input:focus {
        outline: none;
        border-color: #4a69bd;
    }

    button {
        width: 100%;
        padding: 0.8rem;
        background: #4a69bd;
        color: white;
        border: none;
        border-radius: 4px;
        font-size: 1rem;
        font-weight: 600;
        cursor: pointer;
        transition: background 0.2s;
    }

    button:hover:not(:disabled) {
        background: #3d5aa8;
    }

    button:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .error-msg {
        color: #ff6b6b;
        background: rgba(255, 107, 107, 0.1);
        padding: 0.75rem;
        border-radius: 4px;
        margin-bottom: 1.5rem;
        font-size: 0.9rem;
        text-align: center;
    }
</style>

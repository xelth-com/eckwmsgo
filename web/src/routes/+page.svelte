<script>
    import { authStore } from '$lib/stores/authStore';
    import { onMount } from 'svelte';

    // Мы больше не делаем авто-редирект, чтобы пользователи могли прочитать о системе.
    // Состояние авторизации используется только для переключения кнопки Login/Dashboard.
</script>

<div class="landing-page">
    <nav class="navbar">
        <div class="logo">eckWMS <span class="badge">GO</span></div>
        <div class="nav-links">
            <a href="https://github.com/xelth-com/eckwmsgo" target="_blank" rel="noreferrer" class="github-link">
                GitHub
            </a>
        </div>
    </nav>

    <main class="hero">
        <div class="hero-content">
            <h1>Warehouse Management <br><span class="accent">Reimagined</span></h1>

            <p class="description">
                Добро пожаловать в <strong>eckWMS</strong> — современную систему управления складом с открытым исходным кодом.
                Построена на микросервисной архитектуре с использованием <strong>Go</strong> и <strong>SvelteKit</strong>.
            </p>

            <div class="cta-group">
                {#if $authStore.isLoading}
                    <button class="btn primary loading">Загрузка...</button>
                {:else if $authStore.isAuthenticated}
                    <a href="/dashboard" class="btn primary">
                        Открыть Dashboard &rarr;
                    </a>
                {:else}
                    <a href="/login" class="btn primary">
                        Войти в систему
                    </a>
                {/if}
                <a href="https://github.com/xelth-com/eckwmsgo" target="_blank" rel="noreferrer" class="btn secondary">
                    Исходный код
                </a>
            </div>
        </div>

        <div class="features-grid">
            <div class="feature-card">
                <h3>🚀 High Performance</h3>
                <p>Бэкенд на Go обеспечивает высокую скорость обработки запросов и минимальное потребление ресурсов.</p>
            </div>
            <div class="feature-card">
                <h3>📱 Smart Codes</h3>
                <p>Поддержка умных штрихкодов (i/b/p/l) для офлайн-валидации и мгновенного сканирования.</p>
            </div>
            <div class="feature-card">
                <h3>🔄 Odoo Sync</h3>
                <p>Двусторонняя синхронизация с ERP Odoo 17. Полная интеграция складского учета.</p>
            </div>
            <div class="feature-card">
                <h3>🔒 Zero-Knowledge</h3>
                <p>Архитектура Relay позволяет синхронизировать данные через недоверенные сети с шифрованием.</p>
            </div>
        </div>
    </main>

    <footer>
        <p>&copy; {new Date().getFullYear()} xelth-com. Open Source Software.</p>
    </footer>
</div>

<style>
    .landing-page {
        min-height: 100vh;
        background-color: #121212;
        color: #e0e0e0;
        display: flex;
        flex-direction: column;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    }

    .navbar {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 1.5rem 2rem;
        background: rgba(30, 30, 30, 0.5);
        backdrop-filter: blur(10px);
        border-bottom: 1px solid #333;
    }

    .logo {
        font-size: 1.5rem;
        font-weight: 800;
        color: #fff;
        letter-spacing: -0.5px;
    }

    .badge {
        background: #4a69bd;
        font-size: 0.7rem;
        padding: 2px 6px;
        border-radius: 4px;
        vertical-align: middle;
        margin-left: 5px;
    }

    .github-link {
        color: #aaa;
        text-decoration: none;
        font-weight: 500;
        transition: color 0.2s;
    }
    .github-link:hover { color: #fff; }

    .hero {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 4rem 2rem;
        text-align: center;
        background-image: radial-gradient(#2a2a2a 1px, transparent 1px);
        background-size: 30px 30px;
    }

    h1 {
        font-size: 3.5rem;
        line-height: 1.1;
        margin-bottom: 1.5rem;
        background: linear-gradient(to right, #fff, #aaa);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .accent {
        color: #4a69bd;
        -webkit-text-fill-color: #4a69bd;
    }

    .description {
        max-width: 600px;
        font-size: 1.2rem;
        color: #888;
        line-height: 1.6;
        margin-bottom: 3rem;
    }

    .cta-group {
        display: flex;
        gap: 1rem;
        margin-bottom: 5rem;
        flex-wrap: wrap;
        justify-content: center;
    }

    .btn {
        padding: 1rem 2rem;
        border-radius: 8px;
        font-weight: 600;
        text-decoration: none;
        transition: transform 0.2s, opacity 0.2s;
        font-size: 1.1rem;
    }

    .btn:hover {
        transform: translateY(-2px);
    }

    .btn.primary {
        background: #4a69bd;
        color: white;
        box-shadow: 0 4px 15px rgba(74, 105, 189, 0.4);
    }

    .btn.secondary {
        background: #2a2a2a;
        color: #fff;
        border: 1px solid #444;
    }

    .features-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
        gap: 2rem;
        max-width: 1200px;
        width: 100%;
        text-align: left;
    }

    .feature-card {
        background: #1e1e1e;
        border: 1px solid #333;
        padding: 2rem;
        border-radius: 12px;
        transition: border-color 0.2s;
    }

    .feature-card:hover {
        border-color: #4a69bd;
    }

    .feature-card h3 {
        margin-top: 0;
        color: #e0e0e0;
        margin-bottom: 0.5rem;
    }

    .feature-card p {
        color: #888;
        font-size: 0.95rem;
        line-height: 1.5;
        margin: 0;
    }

    footer {
        padding: 2rem;
        text-align: center;
        color: #555;
        font-size: 0.9rem;
        border-top: 1px solid #222;
    }

    @media (max-width: 768px) {
        h1 { font-size: 2.5rem; }
        .hero { padding: 2rem 1rem; }
    }
</style>

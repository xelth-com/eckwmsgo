import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		// Изменяем папку с _app на internal, чтобы избежать проблем с Nginx/Go
		appDir: 'internal',

		adapter: adapter({
			fallback: 'index.html',
			strict: false
		}),
		paths: {
			// Базовый путь берется из ENV. Если не задан - корень.
			// Для продакшена вы будете собирать с BASE_PATH=/E
			base: process.env.BASE_PATH || ''
		}
	}
};

export default config;

import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		// Короткое имя для минимизации размера URL и QR кодов
		appDir: 'i',

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

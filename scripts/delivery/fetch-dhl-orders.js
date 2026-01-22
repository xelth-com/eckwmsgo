#!/usr/bin/env node

/**
 * DHL Geschäftskundenportal Order Fetcher
 *
 * Fetches shipment list from DHL business customer portal via CSV export.
 * Based on the implementation from service-center-server project.
 *
 * Usage:
 *   node scripts/delivery/fetch-dhl-orders.js [options]
 *
 * Options:
 *   --username EMAIL    DHL login email
 *   --password PASS     DHL login password
 *   --days N            Number of days to fetch (default: 14)
 *   --headless          Run browser in headless mode
 *   --output FORMAT     Output format: json (default) or csv
 *   --verbose           Enable verbose logging
 */

require('dotenv').config();
const { chromium } = require('playwright');
const fs = require('fs').promises;
const path = require('path');

// Parse command line arguments
const args = process.argv.slice(2);
const getArg = (name, defaultValue = '') => {
    const idx = args.indexOf(`--${name}`);
    if (idx === -1) return defaultValue;
    if (idx + 1 < args.length && !args[idx + 1].startsWith('--')) {
        return args[idx + 1];
    }
    return true;
};

const CONFIG = {
    username: getArg('username') || process.env.DHL_USERNAME,
    password: getArg('password') || process.env.DHL_PASSWORD,
    url: process.env.DHL_URL || 'https://geschaeftskunden.dhl.de',
    days: parseInt(getArg('days', '14')),
    headless: args.includes('--headless'),
    output: getArg('output', 'json'),
    verbose: args.includes('--verbose'),
    timeout: 300000 // 5 minutes
};

function log(message, level = 'info') {
    if (level === 'debug' && !CONFIG.verbose) return;
    const prefix = { info: '✓', warn: '⚠', error: '✗', debug: '•' }[level] || '•';
    console.error(`[${new Date().toISOString()}] ${prefix} ${message}`);
}

/**
 * Parse CSV content to array of shipment objects
 */
function parseCSV(csvContent) {
    // Handle Windows line endings
    const lines = csvContent.replace(/\r/g, '').split('\n').filter(line => line.trim());
    if (lines.length < 2) return [];

    // CSV is semicolon-separated
    const headers = lines[0].split(';').map(h => h.trim());

    // Map German headers to our field names
    const headerMap = {
        'Sendungsnummer': 'tracking_number',
        'Sendungsreferenz': 'reference',
        'internationale Sendungsnummer': 'international_number',
        'Abrechnungsnummer': 'billing_number',
        'Empfängername': 'recipient_name',
        'Empfängerstraße (inkl. Hausnummer)': 'recipient_street',
        'Empfänger-PLZ': 'recipient_zip',
        'Empfänger-Ort': 'recipient_city',
        'Empfänger-Land': 'recipient_country',
        'Status': 'status',
        'Datum Status': 'status_date',
        'Hinweis': 'note',
        'Zugestellt an - Name': 'delivered_to_name',
        'Zugestellt an - Straße (inkl. Hausnummer)': 'delivered_to_street',
        'Zugestellt an - PLZ': 'delivered_to_zip',
        'Zugestellt an - Ort': 'delivered_to_city',
        'Zugestellt an - Land': 'delivered_to_country',
        'Produkt': 'product',
        'Services': 'services'
    };

    const shipments = [];
    for (let i = 1; i < lines.length; i++) {
        const values = lines[i].split(';').map(v => v.trim());
        if (values.length < headers.length) continue;

        const shipment = {};
        headers.forEach((header, idx) => {
            const fieldName = headerMap[header] || header.toLowerCase().replace(/[^a-z0-9]/g, '_');
            shipment[fieldName] = values[idx] || '';
        });

        // Only include if has tracking number
        if (shipment.tracking_number) {
            shipments.push(shipment);
        }
    }

    return shipments;
}

/**
 * Main fetch function
 */
async function fetchDHLOrders() {
    if (!CONFIG.username || !CONFIG.password) {
        throw new Error('DHL_USERNAME and DHL_PASSWORD are required');
    }

    let browser, context, page;

    try {
        log('Starting DHL order fetch...', 'info');

        // Launch browser
        browser = await chromium.launch({
            headless: CONFIG.headless,
            args: ['--no-sandbox', '--disable-setuid-sandbox']
        });

        context = await browser.newContext({
            viewport: { width: 1920, height: 1080 },
            userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36',
            locale: 'de-DE',
            timezoneId: 'Europe/Berlin',
            acceptDownloads: true
        });

        page = await context.newPage();

        // Navigate to login page
        log('Navigating to DHL portal...', 'info');
        await page.goto(CONFIG.url, { waitUntil: 'load', timeout: 60000 });
        await page.waitForTimeout(3000);

        // Handle cookie consent banner (OneTrust)
        log('Checking for cookie consent banner...', 'debug');
        try {
            const cookieSelectors = [
                '#onetrust-accept-btn-handler',
                'button:has-text("Alle akzeptieren")',
                'button:has-text("Zustimmen")',
                'button:has-text("Accept all")',
                'button:has-text("Akzeptieren")',
                '.onetrust-accept-btn',
                'button[id*="accept"]',
                '#cookie-consent-accept'
            ];

            for (const selector of cookieSelectors) {
                try {
                    const btn = page.locator(selector).first();
                    if (await btn.count() > 0 && await btn.isVisible({ timeout: 2000 })) {
                        await btn.click();
                        log(`Cookie banner accepted using: ${selector}`, 'debug');
                        await page.waitForTimeout(1500);
                        break;
                    }
                } catch (e) {
                    // Try next selector
                }
            }
        } catch (e) {
            log('No cookie banner found or already accepted', 'debug');
        }

        // Wait for page to settle after cookie banner
        await page.waitForTimeout(2000);

        // Debug: take screenshot
        if (CONFIG.verbose) {
            const debugPath = path.join(__dirname, '../../data/dhl-debug-1.png');
            await page.screenshot({ path: debugPath, fullPage: true });
            log(`Debug screenshot saved: ${debugPath}`, 'debug');
        }

        // Click login button to open login form/popup
        log('Clicking login button to open login form...', 'info');

        // Wait for the Anmelden button to appear
        try {
            await page.waitForSelector('button:has-text("Anmelden")', { timeout: 10000 });
        } catch (e) {
            log('Waiting for Anmelden button timed out, trying anyway...', 'warn');
        }

        const loginBtnSelectors = [
            'button:has-text("Im Post & DHL Geschäftskundenportal anmelden")',
            'button:has-text("Anmelden")',
            'a:has-text("Anmelden")',
            'button[data-testid="noName"]',
            '.dhlBtn:has-text("Anmelden")',
            '.login-module-container button'
        ];

        let clicked = false;
        for (const selector of loginBtnSelectors) {
            try {
                const btn = page.locator(selector).first();
                const count = await btn.count();
                log(`Trying selector "${selector}": count=${count}`, 'debug');
                if (count > 0) {
                    await btn.click({ timeout: 5000 });
                    log(`Clicked login button: ${selector}`, 'info');
                    clicked = true;
                    break;
                }
            } catch (e) {
                log(`Selector "${selector}" failed: ${e.message}`, 'debug');
                continue;
            }
        }

        if (!clicked) {
            // Debug: list all buttons
            const buttons = await page.evaluate(() => {
                return Array.from(document.querySelectorAll('button')).map(b => ({
                    text: b.textContent?.trim().substring(0, 50),
                    id: b.id,
                    class: b.className
                }));
            });
            log(`Available buttons: ${JSON.stringify(buttons)}`, 'debug');
            throw new Error('No login button found on page');
        }

        // Wait for login form to appear
        log('Waiting for login form...', 'debug');
        await page.waitForTimeout(3000);

        // Check if we need to login (look for email/password fields)
        const emailField = page.locator('input[type="email"], input[name="email"], input[name="username"]').first();
        const passField = page.locator('input[type="password"]').first();

        if (await emailField.count() > 0 && await emailField.isVisible({ timeout: 5000 })) {
            // Fill login form
            log('Filling login credentials...', 'info');
            await emailField.fill(CONFIG.username);
            await page.waitForTimeout(500);

            if (await passField.count() > 0 && await passField.isVisible({ timeout: 3000 })) {
                await passField.fill(CONFIG.password);
            }

            // Submit login - look for submit button in the form
            log('Submitting login form...', 'debug');
            const submitBtnSelectors = [
                'button[type="submit"]',
                'button:has-text("Anmelden"):visible',
                'input[type="submit"]',
                '.login-button',
                'button.dhlBtn-primary'
            ];

            for (const selector of submitBtnSelectors) {
                try {
                    const btn = page.locator(selector).first();
                    if (await btn.count() > 0 && await btn.isVisible({ timeout: 2000 })) {
                        await btn.click();
                        log(`Clicked submit: ${selector}`, 'debug');
                        break;
                    }
                } catch (e) {
                    continue;
                }
            }

            // Wait for login to complete
            log('Waiting for login to complete...', 'info');
            await page.waitForTimeout(5000);

            // Verify login success - check if we're redirected or see dashboard elements
            const currentUrl = page.url();
            log(`Current URL after login: ${currentUrl}`, 'debug');
        } else {
            log('Login form not found - assuming already logged in', 'info');
        }

        // Navigate to Sendungsliste
        log('Navigating to shipment list...', 'info');
        await page.goto(`${CONFIG.url}/content/scc/shipmentlist`, { waitUntil: 'load', timeout: 60000 });
        await page.waitForTimeout(5000);

        // Find and switch to content iframe
        // The shipment list loads in an iframe with src="/scc/shipmentlist?lng=de"
        let contentFrame = page.mainFrame();
        log('Waiting for shipment list iframe to load...', 'debug');

        // Wait for the iframe to appear
        try {
            await page.waitForSelector('iframe[src*="shipmentlist"]', { timeout: 10000 });
            const iframeElement = await page.$('iframe[src*="shipmentlist"]');
            if (iframeElement) {
                contentFrame = await iframeElement.contentFrame();
                log(`Switched to content iframe`, 'info');
            }
        } catch (e) {
            log('No dedicated iframe found, trying frames...', 'debug');
            const frames = page.frames();
            log(`Found ${frames.length} frames`, 'debug');
            for (const frame of frames) {
                const url = frame.url();
                log(`Frame: ${url}`, 'debug');
                if (url.includes('shipmentlist') && url !== page.url()) {
                    contentFrame = frame;
                    log(`Switched to frame: ${url}`, 'info');
                    break;
                }
            }
        }

        // Wait for iframe to be ready
        await page.waitForTimeout(3000);

        // Click "Sendungsliste laden" button
        log('Loading shipment list...', 'info');
        const loadButton = contentFrame.locator('button:has-text("Sendungsliste laden")');
        if (await loadButton.count() > 0) {
            await loadButton.click();
            log('Clicked "Sendungsliste laden" button, waiting for data...', 'debug');
            await page.waitForTimeout(10000); // Wait for data to load
        } else {
            log('WARNING: "Sendungsliste laden" button not found', 'warn');
        }

        // Click CSV export button
        log('Exporting to CSV...', 'info');
        // Button text is "Sendungsliste als CSV exportieren" (active button with btn-primary class)
        const csvButton = contentFrame.locator('button:has-text("Sendungsliste als CSV exportieren"), button.btn-primary:has-text("CSV")').first();

        if (await csvButton.count() === 0) {
            throw new Error('CSV export button not found');
        }

        // Setup download handler
        const downloadPath = path.join(__dirname, '../../data/dhl-shipments.csv');
        const [download] = await Promise.all([
            page.waitForEvent('download', { timeout: 30000 }),
            csvButton.click()
        ]);

        // Save download
        await download.saveAs(downloadPath);
        log(`CSV saved to: ${downloadPath}`, 'info');

        // Read and parse CSV
        const csvContent = await fs.readFile(downloadPath, 'utf-8');
        const shipments = parseCSV(csvContent);

        log(`Parsed ${shipments.length} shipments`, 'info');

        return shipments;

    } catch (error) {
        log(`Fetch failed: ${error.message}`, 'error');
        throw error;
    } finally {
        if (browser) {
            await browser.close();
        }
    }
}

// Main execution
(async () => {
    try {
        const shipments = await fetchDHLOrders();

        // Output based on format
        if (CONFIG.output === 'json') {
            console.log(JSON.stringify(shipments, null, 2));
        } else {
            // CSV output
            if (shipments.length > 0) {
                const headers = Object.keys(shipments[0]);
                console.log(headers.join(';'));
                shipments.forEach(s => {
                    console.log(headers.map(h => s[h] || '').join(';'));
                });
            }
        }

        process.exit(0);
    } catch (error) {
        console.error(JSON.stringify({ error: error.message }));
        process.exit(1);
    }
})();

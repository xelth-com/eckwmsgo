#!/usr/bin/env node

/**
 * OPAL Kurier Order Fetcher (CLI-friendly version for eckwmsgo)
 *
 * This script uses Playwright to scrape the order list from
 * the OPAL Kurier web system and outputs JSON to stdout.
 *
 * Usage:
 *   node fetch-opal-orders.js [--json-output] [--verbose] [--headless]
 *
 * When --json-output is specified, outputs structured JSON to stdout:
 * {"success": true, "orders": [...]}
 * {"success": false, "error": "..."}
 */

// Load environment variables
require('dotenv').config();

const { chromium } = require('playwright');

const OPAL_URL = process.env.OPAL_URL || 'https://opal-kurier.de';
const OPAL_USERNAME = process.env.OPAL_USERNAME;
const OPAL_PASSWORD = process.env.OPAL_PASSWORD;
const TIMEOUT = 60000;
const VERBOSE = process.argv.includes('--verbose');
const HEADLESS = process.argv.includes('--headless');
const JSON_OUTPUT = process.argv.includes('--json-output');

/**
 * Logger utility
 */
function log(message, level = 'info') {
    if (JSON_OUTPUT && level !== 'error') return;

    const timestamp = new Date().toISOString();
    const prefix = {
        'info': 'i',
        'warn': '!',
        'error': 'x',
        'debug': '.'
    }[level] || '.';

    if (level === 'debug' && !VERBOSE) return;

    console.error(`[${timestamp}] ${prefix} ${message}`);
}

/**
 * Output JSON result
 */
function outputJSON(data) {
    console.log(JSON.stringify(data));
}

/**
 * Initialize browser
 */
async function initializeBrowser() {
    log('Launching browser...', 'debug');

    const browser = await chromium.launch({
        headless: HEADLESS,
        args: ['--no-sandbox', '--disable-setuid-sandbox']
    });

    const context = await browser.newContext({
        viewport: { width: 1920, height: 1080 },
        userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
    });

    const page = await context.newPage();

    log('Browser initialized', 'debug');

    return { browser, context, page };
}

/**
 * Perform login if needed
 */
async function performLogin(page) {
    log('Login required - performing automatic login...', 'info');

    if (!OPAL_USERNAME || !OPAL_PASSWORD) {
        throw new Error('OPAL_USERNAME and OPAL_PASSWORD must be set');
    }

    // Find and fill username field
    const usernameSelectors = [
        'input[name="username"]',
        'input[name="email"]',
        'input[type="email"]',
        'input[id*="user"]',
        'input[id*="email"]'
    ];

    let usernameFilled = false;
    for (const selector of usernameSelectors) {
        try {
            const field = await page.locator(selector).first();
            if (await field.count() > 0 && await field.isVisible()) {
                await field.fill(OPAL_USERNAME);
                log(`Username filled`, 'debug');
                usernameFilled = true;
                break;
            }
        } catch (e) {
            // Try next selector
        }
    }

    if (!usernameFilled) {
        const textInputs = await page.locator('input[type="text"], input[type="email"], input:not([type])').all();
        for (const input of textInputs) {
            if (await input.isVisible()) {
                await input.fill(OPAL_USERNAME);
                log('Username filled in first visible input', 'debug');
                usernameFilled = true;
                break;
            }
        }
    }

    if (!usernameFilled) {
        throw new Error('Could not find username field for login');
    }

    // Fill password
    const passwordField = await page.locator('input[type="password"]').first();
    if (await passwordField.count() === 0) {
        throw new Error('Could not find password field for login');
    }
    await passwordField.fill(OPAL_PASSWORD);
    log('Password filled', 'debug');

    // Submit form
    const buttonSelectors = [
        'button[type="submit"]',
        'input[type="submit"]',
        'button:has-text("Login")',
        'button:has-text("Anmelden")'
    ];

    let buttonClicked = false;
    for (const selector of buttonSelectors) {
        try {
            const button = await page.locator(selector).first();
            if (await button.count() > 0 && await button.isVisible()) {
                await button.click();
                log('Login submitted', 'debug');
                buttonClicked = true;
                break;
            }
        } catch (e) {
            // Try next selector
        }
    }

    if (!buttonClicked) {
        await passwordField.press('Enter');
        log('Login submitted via Enter key', 'debug');
    }

    // Wait for navigation
    await page.waitForLoadState('networkidle', { timeout: 30000 });
    await page.waitForTimeout(2000);

    log('Login completed', 'info');
}

/**
 * Navigate to OPAL and wait for frameset
 */
async function navigateToOpal(page) {
    log('Navigating to OPAL...', 'debug');

    await page.goto(OPAL_URL, { waitUntil: 'networkidle', timeout: TIMEOUT });

    // Wait for frameset to load
    try {
        await page.waitForSelector('frameset, frame[name="optop"]', { timeout: 30000 });
        log('OPAL frameset loaded', 'info');
    } catch (e) {
        // Check for login form
        const hasLoginForm = await page.locator('input[type="password"]').count();
        if (hasLoginForm > 0) {
            log('Login form detected - attempting automatic login...', 'info');
            await performLogin(page);

            // After login, wait for frameset again
            try {
                await page.waitForSelector('frameset, frame[name="optop"]', { timeout: 30000 });
                log('OPAL frameset loaded after login', 'info');
            } catch (e2) {
                throw new Error('Login succeeded but frameset still not found');
            }
        } else {
            throw new Error('Could not find OPAL frameset and no login form detected');
        }
    }

    return page;
}

/**
 * Navigate to "Auftragsliste" (Order List)
 */
async function navigateToAuftragsliste(page) {
    log('Navigating to Auftragsliste...', 'info');

    // Find header frame (optop)
    let headerFrame = null;
    for (let attempt = 0; attempt < 20; attempt++) {
        const frames = page.frames();
        for (const frame of frames) {
            const frameName = await frame.name();
            if (frameName === 'optop') {
                headerFrame = frame;
                log('Found header frame (optop)', 'debug');
                break;
            }
        }
        if (headerFrame) break;
        await page.waitForTimeout(500);
    }

    if (!headerFrame) {
        throw new Error('Could not find header frame (optop)');
    }

    // Wait for navigation links
    await headerFrame.waitForSelector('a', { timeout: 10000 });

    // Try to find and click "Auftragsliste" link
    const selectors = [
        'a:has-text("Auftragsliste")',
        'a:has-text("auftragsliste")',
        'a:has-text("Liste")',
        'a[href*="list"]',
        'a[href*="auftrag"]'
    ];

    let clicked = false;
    for (const selector of selectors) {
        try {
            await headerFrame.click(selector, { timeout: 5000 });
            log(`Clicked on Auftragsliste using selector: ${selector}`, 'info');
            clicked = true;
            break;
        } catch (e) {
            log(`Selector ${selector} not found, trying next...`, 'debug');
        }
    }

    if (!clicked) {
        throw new Error('Could not find Auftragsliste link');
    }

    // Wait for table to load
    await page.waitForTimeout(3000);

    log('Successfully navigated to Auftragsliste', 'info');
}

/**
 * Parse order table from the main frame
 */
async function parseOrderTable(page) {
    log('Parsing order table...', 'info');

    // Find main content frame (opmain)
    let mainFrame = null;
    for (let i = 0; i < 10; i++) {
        const frames = page.frames();
        for (const frame of frames) {
            const frameName = await frame.name();
            if (frameName === 'opmain') {
                mainFrame = frame;
                log('Found main content frame (opmain)', 'debug');
                break;
            }
        }
        if (mainFrame) break;
        await page.waitForTimeout(500);
    }

    if (!mainFrame) {
        throw new Error('Could not find main content frame (opmain)');
    }

    // Wait for table to be present
    try {
        await mainFrame.waitForSelector('table, #order-list-body, tr', { timeout: 30000 });
    } catch (e) {
        log('No table found in main frame', 'warn');
        return [];
    }

    // Parse orders from the page
    const orders = await mainFrame.evaluate(() => {
        const results = [];

        // Try to find table rows - adjust selectors based on actual OPAL structure
        const rows = Array.from(document.querySelectorAll('tr[onmouseover], tr[data-id], tr.order-row, table tr'));

        for (const row of rows) {
            const text = row.innerText || '';
            const rowText = text.trim();

            if (!rowText || rowText.length < 5) continue;

            // Skip header rows
            if (rowText.match(/^(Sendungsnummer|Auftrag|Datum|Status|Name|Empfänger)/i)) continue;

            // Extract tracking numbers using regex patterns
            // OCU pattern: OCU-XXX-XXXXXX
            const ocuMatch = rowText.match(/OCU[-\s]?\d{3}[-\s]?\d{6}/i);
            // HWB pattern: 0419XXXXXXXX (GO barcode)
            const hwbMatch = rowText.match(/0419\d{8}/);

            // Extract order/reference number
            const refMatch = rowText.match(/(?:Auftrag|Ref)[:\s]*([A-Z0-9-]+)/i);
            const orderMatch = refMatch ? refMatch[1] : '';

            // Try to extract date
            const dateMatch = rowText.match(/(\d{2}[\.\/]\d{2}[\.\/]\d{2,4})/);
            const date = dateMatch ? dateMatch[1] : '';

            // Try to extract status from styled elements
            let status = 'Unknown';
            const statusElement = row.querySelector('[style*="background"], [class*="status"]');
            if (statusElement) {
                status = statusElement.innerText.trim() || 'Unknown';
            }

            // Extract sender/recipient name if present
            const nameMatch = rowText.match(/^(.{1,30})\s{2,}/);
            const senderName = nameMatch ? nameMatch[1].trim() : '';

            // Skip if no valid identifiers found
            if (!ocuMatch && !hwbMatch && !orderMatch) continue;

            results.push({
                tracking_number: ocuMatch ? ocuMatch[0].replace(/[-\s]/g, '-') : '',
                hwb_number: hwbMatch ? hwbMatch[0] : '',
                order_number: orderMatch,
                status: status,
                date: date,
                sender: senderName,
                raw_text: rowText.substring(0, 200)
            });
        }

        return results;
    });

    log(`Found ${orders.length} orders`, 'info');

    return orders;
}

/**
 * Main function to fetch OPAL orders
 */
async function fetchOpalOrders() {
    let browser, context, page;

    try {
        // Initialize browser
        ({ browser, context, page } = await initializeBrowser());

        // Navigate to OPAL
        await navigateToOpal(page);

        // Navigate to order list
        await navigateToAuftragsliste(page);

        // Parse orders from table
        const orders = await parseOrderTable(page);

        log('Orders fetched successfully!', 'info');

        // Return success
        return {
            success: true,
            orders: orders,
            count: orders.length
        };

    } catch (error) {
        log(`Order fetching failed: ${error.message}`, 'error');
        return {
            success: false,
            error: error.message
        };

    } finally {
        if (browser) {
            await browser.close();
            log('Browser closed', 'debug');
        }
    }
}

// Run if executed directly
if (require.main === module) {
    if (!JSON_OUTPUT) {
        console.log('\n=== OPAL Kurier Order Fetcher ===\n');
    }

    fetchOpalOrders()
        .then(result => {
            if (JSON_OUTPUT) {
                outputJSON(result);
            } else {
                console.log(JSON.stringify(result, null, 2));
            }
            process.exit(result.success ? 0 : 1);
        })
        .catch(error => {
            if (JSON_OUTPUT) {
                outputJSON({ success: false, error: error.message });
            } else {
                console.error('\nOrder fetching failed:', error.message);
            }
            process.exit(1);
        });
}

module.exports = {
    fetchOpalOrders
};

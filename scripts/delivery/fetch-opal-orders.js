#!/usr/bin/env node

/**
 * OPAL Kurier Order Fetcher - Detail Page Parser
 *
 * This script uses Playwright to scrape orders from OPAL by:
 * 1. Opening the order list
 * 2. Clicking into each order's detail page
 * 3. Parsing complete order information
 * 4. Returning to list and processing next order
 *
 * Usage:
 *   node fetch-opal-orders.js [--json-output] [--verbose] [--headless] [--limit=N]
 */

require('dotenv').config();

const { chromium } = require('playwright');

const OPAL_URL = process.env.OPAL_URL || 'https://opal-kurier.de';
const OPAL_USERNAME = process.env.OPAL_USERNAME;
const OPAL_PASSWORD = process.env.OPAL_PASSWORD;
const TIMEOUT = 60000;
const VERBOSE = process.argv.includes('--verbose');
const HEADLESS = process.argv.includes('--headless');
const JSON_OUTPUT = process.argv.includes('--json-output');

// Parse --limit=N argument
const limitArg = process.argv.find(arg => arg.startsWith('--limit='));
const ORDER_LIMIT = limitArg ? parseInt(limitArg.split('=')[1]) : 50;

/**
 * Logger utility - logs to stderr to keep stdout clean for JSON
 */
function log(message, level = 'info') {
    const timestamp = new Date().toISOString();
    const prefix = { 'info': 'i', 'warn': '!', 'error': 'x', 'debug': '.' }[level] || '.';
    if (level === 'debug' && !VERBOSE) return;
    console.error(`[${timestamp}] ${prefix} ${message}`);
}

/**
 * Output JSON result to STDOUT
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
        userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
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

    // Wait for login form
    try {
        await page.waitForSelector('input[type="password"]', { timeout: 10000 });
    } catch (e) {
        log('Login form not found', 'debug');
    }

    // Fill username
    const usernameSelectors = [
        'input[name="username"]', 'input[name="email"]',
        'input[type="email"]', 'input[id*="user"]'
    ];

    let usernameFilled = false;
    for (const selector of usernameSelectors) {
        try {
            const field = await page.locator(selector).first();
            if (await field.count() > 0 && await field.isVisible()) {
                await field.fill(OPAL_USERNAME);
                usernameFilled = true;
                break;
            }
        } catch (e) { }
    }

    if (!usernameFilled) {
        const textInputs = await page.locator('input[type="text"], input:not([type])').all();
        for (const input of textInputs) {
            if (await input.isVisible()) {
                await input.fill(OPAL_USERNAME);
                usernameFilled = true;
                break;
            }
        }
    }

    if (!usernameFilled) throw new Error('Could not find username field');

    // Fill password
    const passwordField = await page.locator('input[type="password"]').first();
    await passwordField.fill(OPAL_PASSWORD);

    // Submit
    const buttonSelectors = [
        'button[type="submit"]', 'input[type="submit"]',
        'button:has-text("Login")', 'button:has-text("Anmelden")'
    ];

    for (const selector of buttonSelectors) {
        try {
            const button = await page.locator(selector).first();
            if (await button.count() > 0 && await button.isVisible()) {
                await button.click();
                break;
            }
        } catch (e) { }
    }

    await page.waitForLoadState('networkidle', { timeout: 30000 });
    await page.waitForTimeout(2000);
    log('Login completed', 'info');
}

/**
 * Navigate to OPAL and handle login
 */
async function navigateToOpal(page) {
    log('Navigating to OPAL...', 'debug');
    await page.goto(OPAL_URL, { waitUntil: 'networkidle', timeout: TIMEOUT });

    try {
        await page.waitForSelector('frameset, frame[name="optop"]', { timeout: 30000 });
        log('OPAL frameset loaded', 'info');
    } catch (e) {
        const hasLoginForm = await page.locator('input[type="password"]').count();
        if (hasLoginForm > 0) {
            await performLogin(page);
            await page.waitForSelector('frameset, frame[name="optop"]', { timeout: 30000 });
        } else {
            throw new Error('Could not find OPAL frameset');
        }
    }
    return page;
}

/**
 * Get the main content frame
 */
async function getMainFrame(page) {
    for (let i = 0; i < 20; i++) {
        const frames = page.frames();
        for (const frame of frames) {
            if (await frame.name() === 'opmain') {
                return frame;
            }
        }
        await page.waitForTimeout(500);
    }
    throw new Error('Could not find opmain frame');
}

/**
 * Navigate to Auftragsliste
 */
async function navigateToAuftragsliste(page) {
    log('Navigating to Auftragsliste...', 'info');

    // Find header frame
    let headerFrame = null;
    for (let i = 0; i < 20; i++) {
        const frames = page.frames();
        for (const frame of frames) {
            if (await frame.name() === 'optop') {
                headerFrame = frame;
                break;
            }
        }
        if (headerFrame) break;
        await page.waitForTimeout(500);
    }

    if (!headerFrame) throw new Error('Could not find optop frame');

    await headerFrame.waitForSelector('a', { timeout: 10000 });

    // Click Auftragsliste
    const selectors = [
        'a:has-text("Auftragsliste")',
        'a:has-text("Liste")',
        'a[href*="list"]'
    ];

    for (const selector of selectors) {
        try {
            await headerFrame.click(selector, { timeout: 5000 });
            log('Clicked on Auftragsliste', 'info');
            break;
        } catch (e) { }
    }

    await page.waitForTimeout(3000);
}

/**
 * Parse detail page content
 */
function parseDetailPage(text) {
    const order = {
        tracking_number: '',
        hwb_number: '',
        product_type: '',
        reference: '',
        created_at: '',
        created_by: '',

        pickup_name: '',
        pickup_name2: '',
        pickup_contact: '',
        pickup_phone: '',
        pickup_email: '',
        pickup_street: '',
        pickup_city: '',
        pickup_zip: '',
        pickup_country: 'DE',
        pickup_note: '',
        pickup_date: '',
        pickup_time_from: '',
        pickup_time_to: '',
        pickup_vehicle: '',

        delivery_name: '',
        delivery_name2: '',
        delivery_contact: '',
        delivery_phone: '',
        delivery_email: '',
        delivery_street: '',
        delivery_city: '',
        delivery_zip: '',
        delivery_country: 'DE',
        delivery_note: '',
        delivery_date: '',
        delivery_time_from: '',
        delivery_time_to: '',

        package_count: null,
        weight: null,
        value: null,
        description: '',
        dimensions: '',

        status: '',
        status_date: '',
        status_time: '',
        receiver: ''
    };

    const lines = text.split('\n').map(l => l.trim()).filter(l => l);

    // Parse SendungsNr and HWB
    for (const line of lines) {
        if (line.includes('SendungsNr')) {
            const match = line.match(/SendungsNr\s+(OCU[-\d]+)/);
            if (match) order.tracking_number = match[1];
        }
        if (line.includes('HWB') && !order.hwb_number) {
            // HWB can be 12 digits OR SendungsNr format (OCU-998-XXXXXX)
            const match = line.match(/HWB\s+(\d{12}|OCU-[\d-]+)/);
            if (match) order.hwb_number = match[1];
        }
        if (line.includes('Auftragsart')) {
            const match = line.match(/Auftragsart\s+(\S+)/);
            if (match) order.product_type = match[1];
        }
        if (line.includes('Referenz') && !line.includes('Ref/KST')) {
            const match = line.match(/Referenz\s+(\S+)/);
            if (match) order.reference = match[1];
        }
    }

    // Parse created info
    const createdMatch = text.match(/erfasst am\s+([\d\.\-\s:]+Uhr)/);
    if (createdMatch) order.created_at = createdMatch[1].trim();

    const createdByMatch = text.match(/erfasst durch\s+(\S+)/);
    if (createdByMatch) order.created_by = createdByMatch[1];

    // Parse Abholung section
    const abholungIdx = lines.findIndex(l => l === 'Abholung');
    if (abholungIdx >= 0) {
        for (let i = abholungIdx + 1; i < Math.min(abholungIdx + 15, lines.length); i++) {
            const line = lines[i];
            if (line.startsWith('Name1')) order.pickup_name = line.replace('Name1', '').trim();
            if (line.startsWith('Name2')) order.pickup_name2 = line.replace('Name2', '').trim();
            if (line.startsWith('Ansprechpartner')) order.pickup_contact = line.replace('Ansprechpartner', '').trim();
            if (line.startsWith('Telefon')) order.pickup_phone = line.replace('Telefon', '').trim();
            if (line.startsWith('Mail')) order.pickup_email = line.replace('Mail', '').trim();
            if (line.startsWith('Straße/Hs')) order.pickup_street = line.replace('Straße/Hs', '').trim();
            if (line.startsWith('LKZ-Land')) {
                const addr = line.replace('LKZ-Land', '').trim();
                // Support 4-5 digit ZIP codes (AT/CH have 4, DE has 5)
                const match = addr.match(/([A-Z]{2})-(\d{4,5})\s+(.+)/);
                if (match) {
                    order.pickup_country = match[1];
                    order.pickup_zip = match[2];
                    order.pickup_city = match[3];
                }
            }
            if (line.startsWith('Hinweis') && !order.pickup_note) order.pickup_note = line.replace('Hinweis', '').trim();
            if (line === 'Zustellung') break;
        }
    }

    // Parse Zustellung section
    const zustellungIdx = lines.findIndex(l => l === 'Zustellung');
    if (zustellungIdx >= 0) {
        for (let i = zustellungIdx + 1; i < Math.min(zustellungIdx + 15, lines.length); i++) {
            const line = lines[i];
            if (line.startsWith('Name1')) order.delivery_name = line.replace('Name1', '').trim();
            if (line.startsWith('Name2')) order.delivery_name2 = line.replace('Name2', '').trim();
            if (line.startsWith('Ansprechpartner')) order.delivery_contact = line.replace('Ansprechpartner', '').trim();
            if (line.startsWith('Telefon')) order.delivery_phone = line.replace('Telefon', '').trim();
            if (line.startsWith('Mail')) order.delivery_email = line.replace('Mail', '').trim();
            if (line.startsWith('Straße/Hs')) order.delivery_street = line.replace('Straße/Hs', '').trim();
            if (line.startsWith('LKZ-Land')) {
                const addr = line.replace('LKZ-Land', '').trim();
                // Support 4-5 digit ZIP codes (AT/CH have 4, DE has 5)
                const match = addr.match(/([A-Z]{2})-(\d{4,5})\s+(.+)/);
                if (match) {
                    order.delivery_country = match[1];
                    order.delivery_zip = match[2];
                    order.delivery_city = match[3];
                }
            }
            if (line.startsWith('Hinweis') && !order.delivery_note) order.delivery_note = line.replace('Hinweis', '').trim();
            if (line.includes('Abholtermin') || line.includes('Frühtermine')) break;
        }
    }

    // Parse pickup date/time
    const abholTerminIdx = lines.findIndex(l => l.includes('Abholtermin'));
    if (abholTerminIdx >= 0) {
        for (let i = abholTerminIdx + 1; i < Math.min(abholTerminIdx + 5, lines.length); i++) {
            const line = lines[i];
            const dateMatch = line.match(/(\d{2}\.\d{2}\.\d{4})/);
            if (dateMatch) order.pickup_date = dateMatch[1];

            const timeMatch = line.match(/Zeit\s+(\d{2}:\d{2})\s+-\s+(\d{2}:\d{2})/);
            if (timeMatch) {
                order.pickup_time_from = timeMatch[1];
                order.pickup_time_to = timeMatch[2];
            }

            if (line.includes('Fahrzeug')) {
                const vehicleMatch = line.match(/Fahrzeug\s+(\S+)/);
                if (vehicleMatch) order.pickup_vehicle = vehicleMatch[1];
            }
            if (line.includes('Zustelltermin')) break;
        }
    }

    // Parse delivery date/time
    const zustellTerminIdx = lines.findIndex(l => l.includes('Zustelltermin'));
    if (zustellTerminIdx >= 0) {
        for (let i = zustellTerminIdx + 1; i < Math.min(zustellTerminIdx + 3, lines.length); i++) {
            const line = lines[i];
            const dateMatch = line.match(/(\d{2}\.\d{2}\.\d{4})/);
            if (dateMatch) order.delivery_date = dateMatch[1];

            const timeMatch = line.match(/Zeit\s+(\d{2}:\d{2})\s+-\s+(\d{2}:\d{2})/);
            if (timeMatch) {
                order.delivery_time_from = timeMatch[1];
                order.delivery_time_to = timeMatch[2];
            }
            if (line.includes('Sendung & Pack')) break;
        }
    }

    // Parse package info
    const wertMatch = text.match(/Wert\s+([\d\.,]+)\s*EUR/);
    if (wertMatch) order.value = parseFloat(wertMatch[1].replace('.', '').replace(',', '.'));

    // Parse weight and description
    const weightMatch = text.match(/(\d+)\s+([\d,]+)\s+([A-Za-z_][\w\s]+?)(?:\s+VolG|$)/m);
    if (weightMatch) {
        order.package_count = parseInt(weightMatch[1]);
        order.weight = parseFloat(weightMatch[2].replace(',', '.'));
        order.description = weightMatch[3].trim();
    }

    // Parse dimensions
    const dimMatch = text.match(/L:\s*([\d,]+)\s*B:\s*([\d,]+)\s*H:\s*([\d,]+)/);
    if (dimMatch) {
        order.dimensions = `${dimMatch[1]}x${dimMatch[2]}x${dimMatch[3]}`;
    }

    // Parse status history
    // Status can be: Zugestellt, Abgeholt, Storniert, AKTIV, geliefert (older orders)
    // HWB can be 12 digits OR SendungsNr format (OCU-998-XXXXXX)
    const statusMatch = text.match(/(\d{12}|OCU-[\d-]+)\s+(\d{2}\.\d{2}\.\d{2})\s+(\d{2}:\d{2})\s+(Zugestellt|Abgeholt|Storniert|AKTIV|geliefert)\s*(\S*)/i);
    if (statusMatch) {
        order.status = statusMatch[4];
        order.status_date = statusMatch[2];
        order.status_time = statusMatch[3];
        order.receiver = statusMatch[5] || '';
    }

    return order;
}

/**
 * Parse orders by clicking into detail pages
 */
async function parseOrdersFromDetailPages(page, mainFrame, limit) {
    const orders = [];

    // Get total count
    const countText = await mainFrame.evaluate(() => {
        const text = document.body.innerText;
        const match = text.match(/(\d+)\s+von\s+(\d+)\s+Datensätzen/);
        return match ? { current: match[1], total: match[2] } : null;
    });

    if (countText) {
        log(`Found ${countText.total} total orders, processing up to ${limit}`, 'info');
    }

    let currentRow = 0;

    while (orders.length < limit) {
        // Get current row count
        const rowCount = await mainFrame.evaluate(() => {
            return document.querySelectorAll('tr[onmouseover]').length;
        });

        if (currentRow >= rowCount) {
            // Try to go to next page
            log('Trying to go to next page...', 'debug');
            const hasNextPage = await mainFrame.evaluate(() => {
                const links = document.querySelectorAll('a, td[onclick]');
                for (const link of links) {
                    if (link.textContent.trim() === '>') {
                        link.click();
                        return true;
                    }
                }
                return false;
            });

            if (!hasNextPage) {
                log('No more pages', 'info');
                break;
            }

            await page.waitForTimeout(2000);
            currentRow = 0;
            continue;
        }

        // Click on the order row to open detail page
        log(`Opening order ${orders.length + 1}/${limit} (row ${currentRow})...`, 'debug');

        const clicked = await mainFrame.evaluate((rowIndex) => {
            const rows = document.querySelectorAll('tr[onmouseover]');
            if (rows[rowIndex]) {
                const clickableTD = rows[rowIndex].querySelector('td[onclick]');
                if (clickableTD) {
                    clickableTD.click();
                    return true;
                }
            }
            return false;
        }, currentRow);

        if (!clicked) {
            log(`Could not click row ${currentRow}`, 'warn');
            currentRow++;
            continue;
        }

        // Wait for detail page to load - look for key indicators
        try {
            await mainFrame.waitForFunction(() => {
                const text = document.body.innerText;
                return text.includes('SendungsNr') && text.includes('zur Liste zurück');
            }, { timeout: 10000 });
        } catch (e) {
            log(`Detail page load timeout for row ${currentRow}`, 'warn');
            currentRow++;
            continue;
        }

        // Parse detail page
        const detailText = await mainFrame.evaluate(() => {
            return document.body.innerText;
        });

        // Check if we're on detail page (double-check after wait)
        if (!detailText.includes('zur Liste zurück')) {
            log('Not on detail page, retrying...', 'warn');
            currentRow++;
            continue;
        }

        const order = parseDetailPage(detailText);

        if (order.tracking_number || order.hwb_number) {
            orders.push(order);
            log(`Parsed: ${order.tracking_number || order.hwb_number} - ${order.delivery_name} - ${order.status}`, 'info');

            // Log warning if key fields are missing
            if (!order.delivery_name && !order.delivery_zip) {
                log(`WARNING: Missing delivery address data for ${order.tracking_number}`, 'warn');
                log(`Raw text preview: ${detailText.substring(0, 500)}...`, 'debug');
            }
            if (!order.pickup_name && !order.pickup_zip) {
                log(`WARNING: Missing pickup address data for ${order.tracking_number}`, 'warn');
            }
        } else {
            log(`Failed to parse order at row ${currentRow}. Raw text: ${detailText.substring(0, 300)}...`, 'warn');
        }

        // Go back to list
        const wentBack = await mainFrame.evaluate(() => {
            const links = document.querySelectorAll('a, button');
            for (const link of links) {
                if (link.textContent.includes('zur Liste')) {
                    link.click();
                    return true;
                }
            }
            return false;
        });

        if (!wentBack) {
            log('Could not go back to list', 'error');
            break;
        }

        // Wait for list to reload
        try {
            await mainFrame.waitForFunction(() => {
                const text = document.body.innerText;
                return text.includes('Datensätzen') && !text.includes('SendungsNr');
            }, { timeout: 10000 });
        } catch (e) {
            log('List reload timeout, using fallback wait', 'warn');
            await page.waitForTimeout(2000);
        }
        currentRow++;
    }

    return orders;
}

/**
 * Main function
 */
async function fetchOpalOrders() {
    let browser, context, page;

    try {
        ({ browser, context, page } = await initializeBrowser());
        await navigateToOpal(page);
        await navigateToAuftragsliste(page);

        const mainFrame = await getMainFrame(page);

        // Wait for order list to load
        await mainFrame.waitForSelector('tr[onmouseover]', { timeout: 30000 });

        const orders = await parseOrdersFromDetailPages(page, mainFrame, ORDER_LIMIT);

        log(`Successfully fetched ${orders.length} orders`, 'info');

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
        console.error('\n=== OPAL Kurier Order Fetcher (Detail Mode) ===\n');
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

module.exports = { fetchOpalOrders };

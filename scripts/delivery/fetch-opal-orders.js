#!/usr/bin/env node

/**
 * OPAL Kurier Order Fetcher (CLI-friendly version for eckwmsgo)
 *
 * This script uses Playwright to scrape the order list from
 * the OPAL Kurier web system and outputs JSON to stdout.
 *
 * Based on the working implementation from inBody service-center-server.
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
 * Logger utility - logs to stderr to keep stdout clean for JSON
 */
function log(message, level = 'info') {
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

    // Wait for the login form to be ready
    try {
        await page.waitForSelector('#opal-login-form, input[name="username"], input[type="password"]', { timeout: 10000 });
    } catch (e) {
        log('Login form selector timeout, trying generic approach', 'debug');
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
                log(`Username filled using ${selector}`, 'debug');
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

    // Wait for order list to load
    await page.waitForTimeout(3000);

    log('Successfully navigated to Auftragsliste', 'info');
}

/**
 * Parse order table from the main frame
 * Based on the working implementation from inBody service-center-server
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

    // Wait for order list body to be present
    try {
        await mainFrame.waitForSelector('#order-list-body, table', { timeout: 30000 });
    } catch (e) {
        log('Order list body not found', 'warn');
        return [];
    }

    // Parse orders from the page using the correct OPAL HTML structure
    const orders = await mainFrame.evaluate(() => {
        const orderRows = [];

        // Find the order list container
        const orderListBody = document.querySelector('#order-list-body');
        if (!orderListBody) {
            // Fallback: try to find any table
            const tables = document.querySelectorAll('table');
            if (tables.length === 0) return orderRows;
        }

        // Each order is a <tr> with onmouseover attribute containing a nested table
        // The nested table has 2 rows: pickup info (row 1) and delivery info (row 2)
        const rows = document.querySelectorAll('tr[onmouseover]');

        rows.forEach(row => {
            try {
                // Find the nested table inside this row
                const nestedTable = row.querySelector('table');
                if (!nestedTable) return;

                // Get the 2 data rows from nested table (pickup and delivery)
                const dataRows = nestedTable.querySelectorAll('tr');
                if (dataRows.length < 2) return;

                const firstRow = dataRows[0];  // Pickup info
                const secondRow = dataRows[1]; // Delivery info

                // Extract cells from first row (pickup info)
                const firstCells = firstRow.querySelectorAll('td');
                // Extract cells from second row (delivery info)
                const secondCells = secondRow.querySelectorAll('td');

                if (firstCells.length < 8) return;

                // Column mapping based on actual OPAL HTML structure:
                // First row (pickup):
                // [0] = checkbox (rowspan=2)
                // [1] = OCU tracking number (e.g., "OCU-998-511590")
                // [2] = pickup date
                // [3] = pickup time from
                // [4] = pickup time to
                // [5] = pickup company name
                // [6] = pickup city (e.g., "DE-65760 Eschborn")
                // [7] = pickup street
                // [8] = product type (e.g., "Overnight", "X-Change / Swap")
                // [9] = NN field
                // [10] = + Adr field
                // [11] = package count and weight (e.g., "1 Pks. 43,50 kg")
                //
                // Second row (delivery):
                // [0] = HWB number / GO barcode (e.g., "041940529157")
                // [1] = delivery date
                // [2] = delivery time from
                // [3] = delivery time to
                // [4] = delivery company name
                // [5] = delivery city
                // [6] = delivery street
                // [7] = Ref field
                // [8-10] = Status (e.g., "OK 15.01.26-07:56 BECKER")

                const trackingNumber = firstCells[1]?.textContent?.trim() || '';
                const hwbNumber = secondCells[0]?.textContent?.trim() || '';

                // Skip if no valid tracking numbers
                if (!trackingNumber && !hwbNumber) return;

                // Parse package info (e.g., "1 Pks. 43,50 kg")
                const packageInfo = firstCells[11]?.textContent?.trim() || '';
                let packageCount = null;
                let weight = null;

                if (packageInfo) {
                    const pkgMatch = packageInfo.match(/(\d+)\s*Pks/);
                    const weightMatch = packageInfo.match(/(\d+[,.]?\d*)\s*kg/);
                    if (pkgMatch) packageCount = parseInt(pkgMatch[1]);
                    if (weightMatch) weight = parseFloat(weightMatch[1].replace(',', '.'));
                }

                // Parse actual delivery status from cells [8-10]
                // Format: "OK 30.10.25-09:48 KAUFMANN" in a colored div
                let status = '';
                let actualDeliveryDate = '';
                let actualDeliveryReceiver = '';

                // Check cells 8, 9, 10 for status div
                for (let i = 8; i <= 10 && i < secondCells.length; i++) {
                    const statusDiv = secondCells[i]?.querySelector('div[style*="background-color"]');
                    if (statusDiv) {
                        const statusText = statusDiv.textContent.trim();
                        status = statusText;

                        // Parse status text: "OK 30.10.25-09:48 KAUFMANN"
                        const statusMatch = statusText.match(/^([A-Z]+)\s+(\d{2}\.\d{2}\.\d{2})-(\d{2}:\d{2})\s+(.+)$/);
                        if (statusMatch) {
                            status = statusMatch[1]; // OK, STORNO, AKTIV, etc.
                            actualDeliveryDate = `${statusMatch[2]} ${statusMatch[3]}`; // 30.10.25 09:48
                            actualDeliveryReceiver = statusMatch[4]; // KAUFMANN
                        }
                        break;
                    }
                }

                // If no status div found, check for text directly
                if (!status) {
                    for (let i = 8; i <= 10 && i < secondCells.length; i++) {
                        const cellText = secondCells[i]?.textContent?.trim() || '';
                        if (cellText && cellText.match(/^(OK|AKTIV|STORNO|OFFEN)/)) {
                            const statusMatch = cellText.match(/^([A-Z]+)\s+(\d{2}\.\d{2}\.\d{2})-?(\d{2}:\d{2})?\s*(.*)$/);
                            if (statusMatch) {
                                status = statusMatch[1];
                                if (statusMatch[2]) {
                                    actualDeliveryDate = statusMatch[2];
                                    if (statusMatch[3]) actualDeliveryDate += ' ' + statusMatch[3];
                                }
                                if (statusMatch[4]) actualDeliveryReceiver = statusMatch[4];
                            } else {
                                status = cellText;
                            }
                            break;
                        }
                    }
                }

                orderRows.push({
                    tracking_number: trackingNumber,
                    hwb_number: hwbNumber,
                    pickup_date: firstCells[2]?.textContent?.trim() || '',
                    pickup_time_from: firstCells[3]?.textContent?.trim() || '',
                    pickup_time_to: firstCells[4]?.textContent?.trim() || '',
                    pickup_name: firstCells[5]?.textContent?.trim() || '',
                    pickup_city: firstCells[6]?.textContent?.trim() || '',
                    pickup_street: firstCells[7]?.textContent?.trim() || '',
                    product_type: firstCells[8]?.textContent?.trim() || '',
                    package_count: packageCount,
                    weight: weight,
                    delivery_date: secondCells[1]?.textContent?.trim() || '',
                    delivery_time_from: secondCells[2]?.textContent?.trim() || '',
                    delivery_time_to: secondCells[3]?.textContent?.trim() || '',
                    delivery_name: secondCells[4]?.textContent?.trim() || '',
                    delivery_city: secondCells[5]?.textContent?.trim() || '',
                    delivery_street: secondCells[6]?.textContent?.trim() || '',
                    ref_number: secondCells[7]?.textContent?.trim() || '',
                    status: status,
                    actual_delivery_date: actualDeliveryDate,
                    actual_receiver: actualDeliveryReceiver
                });
            } catch (error) {
                console.error('Error parsing order row:', error);
            }
        });

        return orderRows;
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
    // Only output banner if NOT in JSON mode to keep stdout clean
    if (!JSON_OUTPUT) {
        console.error('\n=== OPAL Kurier Order Fetcher ===\n');
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

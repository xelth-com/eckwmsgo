#!/usr/bin/env node

/**
 * OPAL Kurier Order Creator (CLI-friendly version for eckwmsgo)
 *
 * This script uses Playwright to fill out the "Neuer Auftrag" form
 * in the OPAL Kurier web system and submits the order.
 *
 * Usage:
 *   node create-opal-order.js --data=order-data.json [--json-output] [--verbose] [--headless]
 *
 * When --json-output is specified, outputs structured JSON to stdout:
 * {"success": true, "trackingNumber": "...", "orderNumber": "...", "message": "..."}
 * {"success": false, "error": "..."}
 */

// Load environment variables
require('dotenv').config();

const { chromium } = require('playwright');
const fs = require('fs').promises;
const path = require('path');

// Configuration
const OPAL_URL = process.env.OPAL_URL || 'https://opal-kurier.de';
const OPAL_USERNAME = process.env.OPAL_USERNAME;
const OPAL_PASSWORD = process.env.OPAL_PASSWORD;
const TIMEOUT = 300000; // 5 minutes
const VERBOSE = process.argv.includes('--verbose');
const HEADLESS = process.argv.includes('--headless');
const JSON_OUTPUT = process.argv.includes('--json-output');

/**
 * Logger utility
 */
function log(message, level = 'info') {
    if (JSON_OUTPUT && level !== 'error') return; // Suppress logs in JSON mode except errors

    const timestamp = new Date().toISOString();
    const prefix = {
        'info': '✓',
        'warn': '⚠',
        'error': '✗',
        'debug': '•'
    }[level] || '•';

    if (level === 'debug' && !VERBOSE) return;

    console.error(`[${timestamp}] ${prefix} ${message}`); // Use stderr for logs
}

/**
 * Output JSON result
 */
function outputJSON(data) {
    console.log(JSON.stringify(data)); // stdout for JSON
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

    log('Browser initialized', 'info');

    return { browser, context, page };
}

/**
 * Perform login if needed
 */
async function performLogin(page) {
    log('Login required - performing automatic login...', 'info');

    if (!OPAL_USERNAME || !OPAL_PASSWORD) {
        throw new Error('OPAL_USERNAME and OPAL_PASSWORD must be set for automatic login');
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
                throw new Error('Login succeeded but frameset still not found. Page structure may have changed.');
            }
        } else {
            throw new Error('Could not find OPAL frameset and no login form detected');
        }
    }

    return page;
}

/**
 * Navigate to "Neuer Auftrag" (New Order) form
 */
async function navigateToNeuerAuftrag(page) {
    log('Navigating to Neuer Auftrag form...', 'info');

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

    // Try to find and click "Neuer Auftrag" link
    const selectors = [
        'a:has-text("Neuer Auftrag")',
        'a:has-text("neuer auftrag")',
        'a:has-text("Auftrag")',
        'a[href*="new"]',
        'a[href*="auftrag"]'
    ];

    let clicked = false;
    for (const selector of selectors) {
        try {
            await headerFrame.click(selector, { timeout: 5000 });
            log(`Clicked on Neuer Auftrag using selector: ${selector}`, 'info');
            clicked = true;
            break;
        } catch (e) {
            log(`Selector ${selector} not found, trying next...`, 'debug');
        }
    }

    if (!clicked) {
        throw new Error('Could not find Neuer Auftrag link');
    }

    // Wait for form to load
    await page.waitForTimeout(2000);

    log('Successfully navigated to Neuer Auftrag form', 'info');
}

/**
 * Find the main content frame (opmain)
 */
async function findMainFrame(page) {
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

    return mainFrame;
}

/**
 * Validate order data
 */
function validateOrderData(orderData) {
    const required = [
        'deliveryName1',
        'deliveryStreet',
        'deliveryZip',
        'deliveryCity'
    ];

    const missing = required.filter(field => !orderData[field]);

    if (missing.length > 0) {
        throw new Error(`Missing required delivery fields: ${missing.join(', ')}`);
    }

    log('Order data validation passed', 'debug');
}

/**
 * Fill pickup (sender) information
 */
async function fillPickupInfo(frame, orderData) {
    log('Filling pickup (sender) information...', 'info');

    const fieldMappings = [
        { key: 'pickupName1', selector: 'input[name="address_name1[]"]', index: 0 },
        { key: 'pickupName2', selector: 'input[name="address_name2[]"]', index: 0 },
        { key: 'pickupContact', selector: 'input[name="address_name3[]"]', index: 0 },
        { key: 'pickupStreet', selector: 'input[name="address_str[]"]', index: 0 },
        { key: 'pickupHouseNumber', selector: 'input[name="address_hsnr[]"]', index: 0 },
        { key: 'pickupCountry', selector: 'input[name="address_lkz[]"]', index: 0 },
        { key: 'pickupZip', selector: 'input[name="address_plz[]"]', index: 0 },
        { key: 'pickupCity', selector: 'input[name="address_ort[]"]', index: 0 },
        { key: 'pickupPhoneCountry', selector: 'input[name="address_telefonA[]"]', index: 0 },
        { key: 'pickupPhoneArea', selector: 'input[name="address_telefonB[]"]', index: 0 },
        { key: 'pickupPhoneNumber', selector: 'input[name="address_telefonC[]"]', index: 0 },
        { key: 'pickupEmail', selector: 'input[name="address_mail[]"]', index: 0 },
        { key: 'pickupHinweis', selector: 'textarea[name="address_hinweis[]"]', index: 0 }
    ];

    for (const mapping of fieldMappings) {
        const value = orderData[mapping.key];
        if (!value) continue;

        try {
            const elements = await frame.$$(mapping.selector);
            if (elements && elements[mapping.index]) {
                await elements[mapping.index].fill(value.toString());
                log(`  ✓ Filled ${mapping.key}: ${value}`, 'debug');
            }
        } catch (e) {
            log(`  ⚠ Error filling ${mapping.key}: ${e.message}`, 'warn');
        }
    }

    log('Pickup information filled successfully', 'info');
}

/**
 * Fill delivery (recipient) information
 */
async function fillDeliveryInfo(frame, orderData) {
    log('Filling delivery (recipient) information...', 'info');

    const fieldMappings = [
        { key: 'deliveryName1', selector: 'input[name="address_name1[]"]', index: 1 },
        { key: 'deliveryName2', selector: 'input[name="address_name2[]"]', index: 1 },
        { key: 'deliveryContact', selector: 'input[name="address_name3[]"]', index: 1 },
        { key: 'deliveryStreet', selector: 'input[name="address_str[]"]', index: 1 },
        { key: 'deliveryHouseNumber', selector: 'input[name="address_hsnr[]"]', index: 1 },
        { key: 'deliveryCountry', selector: 'input[name="address_lkz[]"]', index: 1 },
        { key: 'deliveryZip', selector: 'input[name="address_plz[]"]', index: 1 },
        { key: 'deliveryCity', selector: 'input[name="address_ort[]"]', index: 1 },
        { key: 'deliveryPhoneCountry', selector: 'input[name="address_telefonA[]"]', index: 1 },
        { key: 'deliveryPhoneArea', selector: 'input[name="address_telefonB[]"]', index: 1 },
        { key: 'deliveryPhoneNumber', selector: 'input[name="address_telefonC[]"]', index: 1 },
        { key: 'deliveryEmail', selector: 'input[name="address_mail[]"]', index: 1 },
        { key: 'deliveryHinweis', selector: 'textarea[name="address_hinweis[]"]', index: 1 }
    ];

    for (const mapping of fieldMappings) {
        const value = orderData[mapping.key];
        if (!value) continue;

        try {
            const elements = await frame.$$(mapping.selector);
            if (elements && elements[mapping.index]) {
                await elements[mapping.index].fill(value.toString());
                log(`  ✓ Filled ${mapping.key}: ${value}`, 'debug');
            }
        } catch (e) {
            log(`  ⚠ Error filling ${mapping.key}: ${e.message}`, 'warn');
        }
    }

    log('Delivery information filled successfully', 'info');
}

/**
 * Fill shipment details (dates, times, package info)
 */
async function fillShipmentDetails(frame, orderData) {
    log('Filling shipment details...', 'info');

    // Fill order type
    if (orderData.orderType) {
        try {
            await frame.selectOption('select#seordertype', orderData.orderType);
            log(`  ✓ Selected order type: ${orderData.orderType}`, 'debug');
        } catch (e) {
            log(`  ⚠ Could not select order type`, 'warn');
        }
    }

    // Fill vehicle type
    if (orderData.vehicleType) {
        try {
            await frame.selectOption('select#sefztype', orderData.vehicleType);
            log(`  ✓ Selected vehicle type: ${orderData.vehicleType}`, 'debug');
        } catch (e) {
            log(`  ⚠ Could not select vehicle type`, 'warn');
        }
    }

    // Fill dates and times
    const dateTimeFields = [
        { key: 'pickupDate', selector: 'input[name="address_date[]"]', index: 0 },
        { key: 'pickupTimeFrom', selector: 'input[name="address_time_von[]"]', index: 0 },
        { key: 'pickupTimeTo', selector: 'input[name="address_time_bis[]"]', index: 0 },
        { key: 'deliveryDate', selector: 'input[name="address_date[]"]', index: 1 },
        { key: 'deliveryTimeFrom', selector: 'input[name="address_time_von[]"]', index: 1 },
        { key: 'deliveryTimeTo', selector: 'input[name="address_time_bis[]"]', index: 1 }
    ];

    for (const field of dateTimeFields) {
        if (!orderData[field.key]) continue;
        try {
            const elements = await frame.$$(field.selector);
            if (elements && elements[field.index]) {
                await elements[field.index].fill(orderData[field.key]);
                log(`  ✓ Filled ${field.key}: ${orderData[field.key]}`, 'debug');
            }
        } catch (e) {
            log(`  ⚠ Error filling ${field.key}`, 'warn');
        }
    }

    // Fill package information
    const shipmentFields = [
        { key: 'packageCount', selector: 'input#sepksnr' },
        { key: 'packageWeight', selector: 'input#segewicht' },
        { key: 'packageDescription', selector: 'input#seinhalt' },
        { key: 'shipmentValue', selector: 'input#sewert' },
        { key: 'refNumber', selector: 'input#seclref' },
        { key: 'notes', selector: 'input#sehinweis' }
    ];

    for (const field of shipmentFields) {
        if (!orderData[field.key]) continue;
        try {
            await frame.fill(field.selector, orderData[field.key].toString());
            log(`  ✓ Filled ${field.key}: ${orderData[field.key]}`, 'debug');
        } catch (e) {
            log(`  ⚠ Could not fill ${field.key}`, 'warn');
        }
    }

    // Fill currency
    if (orderData.shipmentValueCurrency) {
        try {
            await frame.selectOption('select#sewertcu', orderData.shipmentValueCurrency);
            log(`  ✓ Selected currency: ${orderData.shipmentValueCurrency}`, 'debug');
        } catch (e) {
            log(`  ⚠ Could not select currency`, 'warn');
        }
    }

    log('Shipment details filled successfully', 'info');
}

/**
 * Submit the order form and extract tracking number
 */
async function submitOrder(frame) {
    log('Submitting order...', 'info');

    // Find and click submit button
    const submitSelectors = [
        'input[type="submit"]',
        'button[type="submit"]',
        'button:has-text("Auftrag senden")',
        'button:has-text("Senden")'
    ];

    let submitted = false;
    for (const selector of submitSelectors) {
        try {
            await frame.click(selector, { timeout: 5000 });
            log('Order submitted', 'info');
            submitted = true;
            break;
        } catch (e) {
            log(`Submit selector ${selector} not found, trying next...`, 'debug');
        }
    }

    if (!submitted) {
        throw new Error('Could not find submit button');
    }

    // Wait for response
    await frame.waitForTimeout(3000);

    // Try to extract tracking number or order confirmation
    // This is highly dependent on OPAL's response page structure
    // You may need to adjust these selectors based on the actual response
    let trackingNumber = '';
    let orderNumber = '';

    try {
        // Try common selectors for tracking/order numbers
        const textContent = await frame.textContent('body');

        // Look for tracking number patterns (adjust regex as needed)
        const trackingMatch = textContent.match(/Sendungsnummer[:\s]*([A-Z0-9-]+)/i);
        if (trackingMatch) {
            trackingNumber = trackingMatch[1];
        }

        const orderMatch = textContent.match(/Auftragsnummer[:\s]*([A-Z0-9-]+)/i);
        if (orderMatch) {
            orderNumber = orderMatch[1];
        }

        log(`Tracking number: ${trackingNumber || 'Not found'}`, 'debug');
        log(`Order number: ${orderNumber || 'Not found'}`, 'debug');

    } catch (e) {
        log('Could not extract tracking information', 'warn');
    }

    return { trackingNumber, orderNumber };
}

/**
 * Main function to create OPAL order
 */
async function createOpalOrder(orderData) {
    let browser, context, page;

    try {
        // Validate order data
        validateOrderData(orderData);

        // Initialize browser
        ({ browser, context, page } = await initializeBrowser());

        // Navigate to OPAL
        await navigateToOpal(page);

        // Navigate to "Neuer Auftrag" form
        await navigateToNeuerAuftrag(page);

        // Find main content frame
        const mainFrame = await findMainFrame(page);

        // Fill form sections
        await fillPickupInfo(mainFrame, orderData);
        await fillDeliveryInfo(mainFrame, orderData);
        await fillShipmentDetails(mainFrame, orderData);

        // Submit the order
        const { trackingNumber, orderNumber } = await submitOrder(mainFrame);

        log('Order created successfully!', 'info');

        // Return success
        return {
            success: true,
            trackingNumber: trackingNumber || 'UNKNOWN',
            orderNumber: orderNumber || 'UNKNOWN',
            message: 'Order created successfully'
        };

    } catch (error) {
        log(`Order creation failed: ${error.message}`, 'error');
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

/**
 * Load order data from JSON file
 */
async function loadOrderData(filePath) {
    try {
        const data = await fs.readFile(filePath, 'utf-8');
        return JSON.parse(data);
    } catch (error) {
        throw new Error(`Failed to load order data from ${filePath}: ${error.message}`);
    }
}

// Run if executed directly
if (require.main === module) {
    if (!JSON_OUTPUT) {
        console.log('\n=== OPAL Kurier Order Creator ===\n');
    }

    // Get data file path from command line
    const dataFileArg = process.argv.find(arg => arg.startsWith('--data='));

    if (!dataFileArg) {
        if (JSON_OUTPUT) {
            outputJSON({ success: false, error: 'Missing --data parameter' });
        } else {
            console.error('✗ Error: Please provide order data file with --data=path/to/file.json');
        }
        process.exit(1);
    }

    const dataFile = dataFileArg.split('=')[1];

    loadOrderData(dataFile)
        .then(orderData => {
            log(`Loaded order data from ${dataFile}`, 'info');
            return createOpalOrder(orderData);
        })
        .then(result => {
            if (JSON_OUTPUT) {
                outputJSON(result);
            } else {
                console.log('\n' + JSON.stringify(result, null, 2));
            }
            process.exit(result.success ? 0 : 1);
        })
        .catch(error => {
            if (JSON_OUTPUT) {
                outputJSON({ success: false, error: error.message });
            } else {
                console.error('\n✗ Order creation failed:', error.message);
            }
            process.exit(1);
        });
}

module.exports = {
    createOpalOrder,
    validateOrderData
};

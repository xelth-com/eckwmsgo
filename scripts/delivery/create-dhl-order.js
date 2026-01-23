#!/usr/bin/env node

/**
 * DHL Order Creator (Automation)
 *
 * Automates the creation of a shipment in the DHL Business Customer Portal.
 * Uses Playwright to fill the "ShipmentDetails" form.
 *
 * Usage:
 *   node create-dhl-order.js --data=order.json [--json-output] [--headless]
 */

require('dotenv').config();
const { chromium } = require('playwright');
const fs = require('fs').promises;
const path = require('path');

// Configuration
const CONFIG = {
    username: process.env.DHL_USERNAME,
    password: process.env.DHL_PASSWORD,
    url: process.env.DHL_URL || 'https://geschaeftskunden.dhl.de',
    headless: process.argv.includes('--headless'),
    verbose: process.argv.includes('--verbose'),
    jsonOutput: process.argv.includes('--json-output'),
    dryRun: process.argv.includes('--dry-run'),
    timeout: 300000 // 5 minutes
};

// Logger
function log(msg, type = 'info') {
    if (CONFIG.jsonOutput && type !== 'error') return;
    const ts = new Date().toISOString();
    console.error(`[${ts}] [${type.toUpperCase()}] ${msg}`);
}

// JSON Output
function output(data) {
    console.log(JSON.stringify(data));
}

// Load Data
async function loadData() {
    const dataArg = process.argv.find(a => a.startsWith('--data='));
    if (!dataArg) throw new Error('Missing --data argument');
    const filePath = dataArg.split('=')[1];
    const content = await fs.readFile(filePath, 'utf-8');
    return JSON.parse(content);
}

// Main Automation
async function run() {
    let browser;
    try {
        const orderData = await loadData();
        log('Loaded order data for: ' + orderData.delivery_name);

        browser = await chromium.launch({
            headless: CONFIG.headless,
            args: ['--no-sandbox', '--disable-setuid-sandbox']
        });

        const context = await browser.newContext({
            viewport: { width: 1920, height: 1080 },
            locale: 'de-DE',
            userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36',
            timezoneId: 'Europe/Berlin'
        });

        const page = await context.newPage();

        // 1. Navigate to DHL portal
        log('Navigating to DHL portal...');
        await page.goto(CONFIG.url, { waitUntil: 'load', timeout: 60000 });
        await page.waitForTimeout(3000);

        // Handle cookie consent banner (OneTrust)
        log('Checking for cookie consent banner...');
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
                        log('Cookie banner accepted using: ' + selector);
                        await page.waitForTimeout(1500);
                        break;
                    }
                } catch (e) {
                    // Try next selector
                }
            }
        } catch (e) {
            log('No cookie banner found or already accepted');
        }

        // Wait for page to settle after cookie banner
        await page.waitForTimeout(2000);

        // Debug: take screenshot if verbose mode
        if (CONFIG.verbose) {
            const debugPath = path.join(__dirname, '../../data/dhl-debug-1.png');
            await page.screenshot({ path: debugPath, fullPage: true });
            log('Debug screenshot saved: ' + debugPath);
        }

        // Click login button to open login form/popup
        log('Clicking login button to open login form...');

        // Wait for the Anmelden button to appear
        try {
            await page.waitForSelector('button:has-text("Anmelden")', { timeout: 10000 });
        } catch (e) {
            log('Waiting for Anmelden button timed out, trying anyway...');
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
                log('Trying selector "' + selector + '": count=' + count);
                if (count > 0) {
                    await btn.click({ timeout: 5000 });
                    log('Clicked login button: ' + selector);
                    clicked = true;
                    break;
                }
            } catch (e) {
                log('Selector "' + selector + '" failed: ' + e.message);
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
            log('Available buttons: ' + JSON.stringify(buttons));
            throw new Error('No login button found on page');
        }

        // Wait for login form to appear
        log('Waiting for login form...');
        await page.waitForTimeout(3000);

        // Check if we need to login (look for email/password fields)
        const emailField = page.locator('input[type="email"], input[name="email"], input[name="username"]').first();
        const passField = page.locator('input[type="password"]').first();

        if (await emailField.count() > 0 && await emailField.isVisible({ timeout: 5000 })) {
            // Fill login form
            log('Filling login credentials...');
            await emailField.fill(CONFIG.username);
            await page.waitForTimeout(500);

            if (await passField.count() > 0 && await passField.isVisible({ timeout: 3000 })) {
                await passField.fill(CONFIG.password);
            }

            // Submit login - look for submit button in the form
            log('Submitting login form...');
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
                        log('Clicked submit: ' + selector);
                        break;
                    }
                } catch (e) {
                    continue;
                }
            }

            // Wait for login to complete
            log('Waiting for login to complete...');
            await page.waitForTimeout(5000);

            // Verify login success - check if we're redirected or see dashboard elements
            const currentUrl = page.url();
            log('Current URL after login: ' + currentUrl);
        } else {
            log('Login form not found - assuming already logged in');
        }

        // 2. Navigate to Shipment Entry Form
        log('Navigating to Shipment Details...');
        await page.goto(`${CONFIG.url}/content/vls/vc/ShipmentDetails`, { waitUntil: 'domcontentloaded' });
        await page.waitForTimeout(3000);

        // Handle possible iframes
        let contentFrame = page.mainFrame();
        try {
            const frames = page.frames();
            log(`Found ${frames.length} frames`);
            for (const frame of frames) {
                const url = frame.url();
                if (url.includes('ShipmentDetails') && url !== page.url()) {
                    contentFrame = frame;
                    log('Switched to shipment details frame: ' + url);
                    break;
                }
            }
        } catch (e) {
            log('No iframe found, using main frame');
        }

        await page.waitForTimeout(2000);

        // DEBUG: List all input fields on the page
        if (CONFIG.verbose) {
            log('=== DEBUG: Listing all input fields ===');
            const inputs = await contentFrame.evaluate(() => {
                return Array.from(document.querySelectorAll('input, textarea, select')).map(el => ({
                    tag: el.tagName,
                    type: el.type,
                    id: el.id,
                    name: el.name,
                    placeholder: el.placeholder,
                    value: el.value,
                    className: el.className
                }));
            });
            inputs.forEach((inp, idx) => {
                log(`Input ${idx}: ${inp.tag} type="${inp.type}" id="${inp.id}" name="${inp.name}" placeholder="${inp.placeholder}" class="${inp.className}"`);
            });
            log('=== END DEBUG ===');
        }

        // 3. Fill Receiver Address
        log('Filling receiver address...');

        // Helper to find and fill input
        const fillField = async (locatorStrings, value, fieldName) => {
            for (const loc of locatorStrings) {
                try {
                    const input = contentFrame.locator(loc).first();
                    if (await input.count() > 0 && await input.isVisible({ timeout: 2000 })) {
                        await input.fill(value);
                        log(`✓ Filled ${fieldName}: ${loc}`);
                        return true;
                    }
                } catch (e) {}
            }
            log(`✗ Could not fill ${fieldName}`);
            return false;
        };

        // Name (Receiver Name 1) - ID with dot needs escaping
        await fillField([
            '#receiver\\.name1',
            'input[id="receiver.name1"]'
        ], orderData.delivery_name, 'Name');

        // Street
        await fillField([
            '#receiver\\.street',
            'input[id="receiver.street"]'
        ], orderData.delivery_street, 'Street');

        // House Number (Street Number)
        if (orderData.delivery_house_number) {
            await fillField([
                '#receiver\\.streetNumber',
                'input[id="receiver.streetNumber"]'
            ], orderData.delivery_house_number, 'House Number');
        }

        // Zip Code (PLZ)
        await fillField([
            '#receiver\\.plz',
            'input[id="receiver.plz"]'
        ], orderData.delivery_zip, 'Zip Code');

        // City
        await fillField([
            '#receiver\\.city',
            'input[id="receiver.city"]'
        ], orderData.delivery_city, 'City');

        // 4. Fill Shipment Data
        log('Filling shipment details...');

        // Weight
        await fillField([
            '#shipment-weight',
            'input[id="shipment-weight"]'
        ], orderData.weight.toString(), 'Weight');

        // 5. Submit Form
        log('Ready to submit shipment...');

        // Look for submit button
        const submitSelectors = [
            'button:has-text("Versenden")',
            'button:has-text("Drucken")',
            'button:has-text("Speichern")',
            'button[type="submit"]',
            'input[type="submit"]'
        ];

        // Find the submit button
        let submitBtn = null;
        for (const selector of submitSelectors) {
            try {
                const btn = contentFrame.locator(selector).first();
                if (await btn.count() > 0 && await btn.isVisible({ timeout: 2000 })) {
                    submitBtn = btn;
                    log('Found submit button: ' + selector);
                    break;
                }
            } catch (e) {}
        }

        if (!submitBtn) {
            throw new Error('Submit button not found');
        }

        // DRY RUN MODE: Stop here and keep browser open
        if (CONFIG.dryRun) {
            log('🟡 DRY RUN MODE ACTIVE 🟡', 'warn');
            log('Form has been filled but NOT submitted.', 'warn');
            log('The browser will stay open for manual inspection.', 'warn');
            log('You can:', 'info');
            log('  1. Verify the filled data is correct', 'info');
            log('  2. Manually click submit to complete the order', 'info');
            log('  3. Press Ctrl+C in terminal to cancel', 'info');
            log('Pausing execution... (Use Playwright Inspector or just inspect the page)', 'info');

            // Keep browser open indefinitely for manual inspection
            await page.waitForTimeout(300000); // Wait 5 minutes or until manually closed

            output({
                success: true,
                dryRun: true,
                message: 'Dry run completed - form filled but not submitted'
            });

            return;
        }

        // Normal mode: submit the form
        log('Submitting shipment...');
        await submitBtn.click();
        log('Form submitted');

        // 6. Wait for Result and Extract Tracking Number
        log('Waiting for confirmation...');
        await page.waitForLoadState('networkidle', { timeout: 30000 });
        await page.waitForTimeout(3000);

        // Try to find tracking number
        let trackingNumber = '';

        try {
            const bodyText = await page.textContent('body');

            // Try various patterns for tracking number
            const patterns = [
                /Sendungsnummer[:\s]+(\d{10,20})/i,
                /Tracking[:\s]+(\d{10,20})/i,
                /Paketnummer[:\s]+(\d{10,20})/i,
                /(\d{12,20})/  // Fallback: any long number
            ];

            for (const pattern of patterns) {
                const match = bodyText.match(pattern);
                if (match) {
                    trackingNumber = match[1];
                    log('Found tracking number via pattern: ' + pattern);
                    break;
                }
            }
        } catch (e) {
            log('Could not extract tracking number from body text: ' + e.message, 'warn');
        }

        if (!trackingNumber) {
            // Take screenshot for debugging
            const shotPath = path.join(__dirname, '../../data/dhl-create-error.png');
            await page.screenshot({ path: shotPath, fullPage: true });
            log('Failed to find tracking number. Screenshot saved to ' + shotPath, 'error');
            throw new Error('Tracking number not found in confirmation page');
        }

        output({
            success: true,
            trackingNumber: trackingNumber,
            orderNumber: orderData.reference || '',
            message: 'Shipment created successfully'
        });

    } catch (e) {
        log(e.message, 'error');
        output({
            success: false,
            error: e.message
        });
        process.exit(1);
    } finally {
        if (browser) await browser.close();
    }
}

run();

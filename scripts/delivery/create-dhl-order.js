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

        // 1. Navigate and Login
        log('Navigating to login...');
        await page.goto(CONFIG.url, { waitUntil: 'domcontentloaded' });
        await page.waitForTimeout(3000);

        // Handle Cookies
        try {
            const cookieSelectors = [
                '#onetrust-accept-btn-handler',
                'button:has-text("Alle akzeptieren")',
                'button:has-text("Zustimmen")',
                'button:has-text("Akzeptieren")'
            ];

            for (const selector of cookieSelectors) {
                try {
                    const btn = page.locator(selector).first();
                    if (await btn.count() > 0 && await btn.isVisible({ timeout: 2000 })) {
                        await btn.click();
                        log('Cookies accepted using: ' + selector);
                        await page.waitForTimeout(1500);
                        break;
                    }
                } catch (e) {}
            }
        } catch (e) {
            log('No cookie banner found');
        }

        await page.waitForTimeout(2000);

        // Click login button to open login form
        log('Opening login form...');
        const loginBtnSelectors = [
            'button:has-text("Im Post & DHL Geschäftskundenportal anmelden")',
            'button:has-text("Anmelden")',
            'a:has-text("Anmelden")'
        ];

        let clicked = false;
        for (const selector of loginBtnSelectors) {
            try {
                const btn = page.locator(selector).first();
                if (await btn.count() > 0) {
                    await btn.click({ timeout: 5000 });
                    log('Clicked login button: ' + selector);
                    clicked = true;
                    break;
                }
            } catch (e) {
                continue;
            }
        }

        if (!clicked) {
            throw new Error('No login button found on page');
        }

        // Wait for login form
        await page.waitForTimeout(3000);

        // Fill login credentials
        const emailField = page.locator('input[type="email"], input[name="email"], input[name="username"]').first();
        const passField = page.locator('input[type="password"]').first();

        if (await emailField.count() > 0 && await emailField.isVisible({ timeout: 5000 })) {
            log('Filling login credentials...');
            await emailField.fill(CONFIG.username);
            await page.waitForTimeout(500);

            if (await passField.count() > 0 && await passField.isVisible({ timeout: 3000 })) {
                await passField.fill(CONFIG.password);
            }

            // Submit login
            const submitBtnSelectors = [
                'button[type="submit"]',
                'button:has-text("Anmelden"):visible',
                'input[type="submit"]'
            ];

            for (const selector of submitBtnSelectors) {
                try {
                    const btn = page.locator(selector).first();
                    if (await btn.count() > 0 && await btn.isVisible({ timeout: 2000 })) {
                        await btn.click();
                        log('Login submitted');
                        break;
                    }
                } catch (e) {
                    continue;
                }
            }

            await page.waitForTimeout(5000);
        } else {
            log('Already logged in or login form not found');
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

        // 3. Fill Receiver Address
        log('Filling receiver address...');

        // Helper to find and fill input
        const fillField = async (locatorStrings, value) => {
            for (const loc of locatorStrings) {
                try {
                    const input = contentFrame.locator(loc).first();
                    if (await input.count() > 0 && await input.isVisible({ timeout: 2000 })) {
                        await input.fill(value);
                        log(`Filled field: ${loc}`);
                        return true;
                    }
                } catch (e) {}
            }
            return false;
        };

        // Name (Receiver Name 1)
        await fillField([
            'input[id*="receiver-name"]',
            'input[name*="ReceiverName1"]',
            'input[name*="receiverName1"]',
            'input[placeholder*="Name"]'
        ], orderData.delivery_name);

        // Street
        await fillField([
            'input[id*="street"]',
            'input[name*="Street"]',
            'input[name*="street"]',
            'input[placeholder*="Straße"]'
        ], orderData.delivery_street);

        // House Number (if exists and provided)
        if (orderData.delivery_house_number) {
            await fillField([
                'input[id*="house-number"]',
                'input[id*="houseNumber"]',
                'input[name*="HouseNumber"]',
                'input[name*="houseNumber"]',
                'input[placeholder*="Hausnummer"]'
            ], orderData.delivery_house_number);
        }

        // Zip Code
        await fillField([
            'input[id*="zip"]',
            'input[id*="postal"]',
            'input[name*="Zip"]',
            'input[name*="PostalCode"]',
            'input[placeholder*="PLZ"]'
        ], orderData.delivery_zip);

        // City
        await fillField([
            'input[id*="city"]',
            'input[name*="City"]',
            'input[placeholder*="Ort"]'
        ], orderData.delivery_city);

        // 4. Fill Shipment Data
        log('Filling shipment details...');

        // Weight
        await fillField([
            'input[id*="weight"]',
            'input[name*="Weight"]',
            'input[name*="weight"]',
            'input[placeholder*="Gewicht"]'
        ], orderData.weight.toString());

        // 5. Submit Form
        log('Submitting shipment...');

        // Look for submit button
        const submitSelectors = [
            'button:has-text("Versenden")',
            'button:has-text("Drucken")',
            'button:has-text("Speichern")',
            'button[type="submit"]',
            'input[type="submit"]'
        ];

        let submitted = false;
        for (const selector of submitSelectors) {
            try {
                const btn = contentFrame.locator(selector).first();
                if (await btn.count() > 0 && await btn.isVisible({ timeout: 2000 })) {
                    await btn.click();
                    log('Clicked submit: ' + selector);
                    submitted = true;
                    break;
                }
            } catch (e) {}
        }

        if (!submitted) {
            throw new Error('Submit button not found');
        }

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

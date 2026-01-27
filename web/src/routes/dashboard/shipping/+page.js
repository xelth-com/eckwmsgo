import { api } from "$lib/api";

/** @type {import('./$types').PageLoad} */
export async function load() {
    try {
        const [pickings, shipments, syncHistory, providersConfig] = await Promise.all([
            api.get("/api/odoo/pickings?state=assigned"),
            api.get("/api/delivery/shipments"),
            api.get("/api/delivery/sync/history"),
            api.get("/api/delivery/config")
        ]);

        return {
            pickings: pickings || [],
            shipments: shipments || [],
            syncHistory: syncHistory || [],
            providersConfig: providersConfig || { opal: false, dhl: false }
        };
    } catch (e) {
        console.error("Load error:", e);
        return {
            pickings: [],
            shipments: [],
            syncHistory: [],
            providersConfig: { opal: false, dhl: false },
            error: e.message
        };
    }
}

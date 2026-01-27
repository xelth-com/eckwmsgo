<script>
    import { onMount } from "svelte";
    import { api } from "$lib/api";
    import { toastStore } from "$lib/stores/toastStore.js";

    let pickings = [];
    let shipments = [];
    let syncHistory = [];
    let loading = true;
    let error = null;
    let activeTab = "pickings"; // 'pickings', 'shipments', or 'sync'
    let processingPickings = new Set();
    let isSyncingOpal = false; // State for OPAL sync
    let isSyncingDhl = false; // State for DHL sync
    let expandedShipments = new Set(); // Track which shipments are expanded
    let expandedSyncLogs = new Set(); // Track which sync logs are expanded
    let providersConfig = { opal: false, dhl: false }; // Provider availability

    onMount(async () => {
        await loadData();
    });

    async function loadData() {
        loading = true;
        error = null;
        try {
            // Load pickings, shipments, sync history, and provider config in parallel
            const [pickingsData, shipmentsData, syncHistoryData, configData] =
                await Promise.all([
                    api.get("/api/odoo/pickings?state=assigned"),
                    api.get("/api/delivery/shipments"),
                    api.get("/api/delivery/sync/history"),
                    api.get("/api/delivery/config"),
                ]);
            pickings = pickingsData || [];
            shipments = shipmentsData || [];
            syncHistory = syncHistoryData || [];
            syncHistory = syncHistoryData || [];
            if (configData) providersConfig = configData;
        } catch (e) {
            console.error(e);
            error = e.message;
        } finally {
            loading = false;
        }
    }

    async function createShipment(pickingId) {
        if (processingPickings.has(pickingId)) return;

        processingPickings.add(pickingId);
        processingPickings = processingPickings; // Trigger reactivity

        try {
            await api.post("/api/delivery/shipments", {
                pickingId: pickingId,
                providerCode: "opal",
            });

            // Reload data to show new shipment
            await loadData();

            // Switch to shipments tab to show the result
            activeTab = "shipments";
        } catch (e) {
            alert("Failed to create shipment: " + e.message);
        } finally {
            processingPickings.delete(pickingId);
            processingPickings = processingPickings;
        }
    }

    async function cancelShipment(pickingId) {
        if (!confirm("Are you sure you want to cancel this shipment?")) return;

        try {
            await api.post(`/api/delivery/shipments/${pickingId}/cancel`);
            await loadData();
        } catch (e) {
            alert("Failed to cancel shipment: " + e.message);
        }
    }

    async function syncOpal() {
        isSyncingOpal = true;
        toastStore.add("Syncing with OPAL...", "info");
        try {
            await api.post("/api/delivery/import/opal", {});
            toastStore.add("OPAL sync started. Refreshing data...", "success");

            // Wait a bit before reloading to let the scraper start/finish
            setTimeout(async () => {
                await loadData();
                isSyncingOpal = false;
            }, 4000);
        } catch (e) {
            toastStore.add("OPAL sync failed: " + e.message, "error");
            isSyncingOpal = false;
        }
    }

    async function syncDhl() {
        isSyncingDhl = true;
        toastStore.add("Syncing with DHL...", "info");
        try {
            await api.post("/api/delivery/import/dhl", {});
            toastStore.add("DHL sync started. Refreshing data...", "success");

            // Wait a bit before reloading to let the scraper start/finish
            setTimeout(async () => {
                await loadData();
                isSyncingDhl = false;
            }, 4000);
        } catch (e) {
            toastStore.add("DHL sync failed: " + e.message, "error");
            isSyncingDhl = false;
        }
    }

    function getProvider(details) {
        if (!details) return "unknown";
        if (details.provider === "dhl") return "dhl";
        if (details.ocu_number || details.hwb_number) return "opal";
        if (details.product?.includes("DHL")) return "dhl";
        return "opal";
    }

    function formatDate(dateStr) {
        if (!dateStr) return "-";
        return new Date(dateStr).toLocaleDateString("de-DE", {
            day: "2-digit",
            month: "2-digit",
            year: "numeric",
            hour: "2-digit",
            minute: "2-digit",
        });
    }

    function getStateColor(state) {
        const colors = {
            draft: "#6c757d",
            assigned: "#ffc107",
            confirmed: "#17a2b8",
            done: "#28a745",
            cancel: "#dc3545",
        };
        return colors[state] || "#6c757d";
    }

    function getDeliveryStateColor(state) {
        const colors = {
            pending: "#ffc107",
            processing: "#17a2b8",
            shipped: "#28a745",
            delivered: "#28a745",
            failed: "#dc3545",
            cancelled: "#6c757d",
        };
        return colors[state] || "#6c757d";
    }

    function toggleShipmentDetails(id) {
        if (expandedShipments.has(id)) {
            expandedShipments.delete(id);
        } else {
            expandedShipments.add(id);
        }
        expandedShipments = expandedShipments; // Trigger reactivity
    }

    function parseRawResponse(rawResponse) {
        if (!rawResponse) return null;
        try {
            return JSON.parse(rawResponse);
        } catch {
            return null;
        }
    }

    function formatAddress(data, prefix) {
        if (!data) return "-";
        const parts = [
            data[`${prefix}_name`],
            data[`${prefix}_street`],
            [data[`${prefix}_zip`], data[`${prefix}_city`]]
                .filter(Boolean)
                .join(" "),
        ].filter(Boolean);
        return parts.join(", ") || "-";
    }

    function toggleSyncDetails(id) {
        if (expandedSyncLogs.has(id)) {
            expandedSyncLogs.delete(id);
        } else {
            expandedSyncLogs.add(id);
        }
        expandedSyncLogs = expandedSyncLogs; // Trigger reactivity
    }

    async function copyDebugInfo(sync) {
        const debugText = `
# Sync Error Debug Info
Provider: ${sync.provider}
Time: ${formatDate(sync.startedAt)}
Status: ${sync.status}
Duration: ${sync.duration ? (sync.duration / 1000).toFixed(1) + "s" : "N/A"}

## Error Message
${sync.errorDetail || "No error detail"}

## Debug Information
${sync.debugInfo ? JSON.stringify(sync.debugInfo, null, 2) : "No debug info available"}

## Statistics
- Created: ${sync.created || 0}
- Updated: ${sync.updated || 0}
- Skipped: ${sync.skipped || 0}
- Errors: ${sync.errors || 0}

---
Copy this to ChatGPT/Claude for analysis
`.trim();

        try {
            await navigator.clipboard.writeText(debugText);
            toastStore.add("Debug info copied to clipboard!", "success");
        } catch (err) {
            toastStore.add("Failed to copy: " + err.message, "error");
        }
    }
</script>

<div class="shipping-page">
    <header>
        <h1>📦 Shipping & Delivery</h1>
        <div class="header-actions">
            {#if providersConfig && providersConfig.opal === true}
                <button
                    class="action-btn opal-btn"
                    on:click={syncOpal}
                    disabled={isSyncingOpal || loading}
                >
                    {isSyncingOpal ? "⏳ Syncing..." : "🟢 Sync OPAL"}
                </button>
            {/if}
            {#if providersConfig && providersConfig.dhl === true}
                <button
                    class="action-btn dhl-btn"
                    on:click={syncDhl}
                    disabled={isSyncingDhl || loading}
                >
                    {isSyncingDhl ? "⏳ Syncing..." : "🟡 Sync DHL"}
                </button>
            {/if}
            <button class="refresh-btn" on:click={loadData} disabled={loading}>
                {loading ? "↻ Loading..." : "↻ Refresh"}
            </button>
        </div>
    </header>

    <div class="tabs">
        <button
            class="tab"
            class:active={activeTab === "pickings"}
            on:click={() => (activeTab = "pickings")}
        >
            📋 Ready to Ship ({pickings.length})
        </button>
        <button
            class="tab"
            class:active={activeTab === "shipments"}
            on:click={() => (activeTab = "shipments")}
        >
            🚚 Shipments ({shipments.length})
        </button>
        <button
            class="tab"
            class:active={activeTab === "sync"}
            on:click={() => (activeTab = "sync")}
        >
            🔄 Sync History
        </button>
    </div>

    {#if loading && pickings.length === 0 && shipments.length === 0}
        <div class="loading">Loading shipping data...</div>
    {:else if error}
        <div class="error">Failed to load data: {error}</div>
    {:else if activeTab === "pickings"}
        <div class="pickings-section">
            <p class="section-desc">
                These are Odoo Transfer Orders ready to be shipped. Click "Ship
                with OPAL" to create a delivery shipment.
            </p>

            {#if pickings.length === 0}
                <div class="empty-state">
                    <p>✅ No pickings ready to ship</p>
                    <small
                        >Pickings with status "assigned" will appear here</small
                    >
                </div>
            {:else}
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>Picking #</th>
                                <th>Origin</th>
                                <th>Partner</th>
                                <th>Location</th>
                                <th>State</th>
                                <th>Scheduled</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each pickings as picking}
                                <tr>
                                    <td class="picking-name">{picking.name}</td>
                                    <td>{picking.origin || "-"}</td>
                                    <td>{picking.partner_id || "-"}</td>
                                    <td>
                                        <div class="location-cell">
                                            <span class="from"
                                                >{picking.location_id ||
                                                    "-"}</span
                                            >
                                            <span class="arrow">→</span>
                                            <span class="to"
                                                >{picking.location_dest_id ||
                                                    "-"}</span
                                            >
                                        </div>
                                    </td>
                                    <td>
                                        <span
                                            class="state-badge"
                                            style="background-color: {getStateColor(
                                                picking.state,
                                            )}"
                                        >
                                            {picking.state}
                                        </span>
                                    </td>
                                    <td>{formatDate(picking.scheduled_date)}</td
                                    >
                                    <td>
                                        <button
                                            class="action-btn ship-btn"
                                            on:click={() =>
                                                createShipment(picking.id)}
                                            disabled={processingPickings.has(
                                                picking.id,
                                            )}
                                        >
                                            {processingPickings.has(picking.id)
                                                ? "⏳ Processing..."
                                                : "🚚 Ship with OPAL"}
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    {:else if activeTab === "shipments"}
        <div class="shipments-section">
            <p class="section-desc">
                Active and past shipments created through the delivery system.
            </p>

            {#if shipments.length === 0}
                <div class="empty-state">
                    <p>📭 No shipments yet</p>
                    <small
                        >Create your first shipment from the "Ready to Ship" tab</small
                    >
                </div>
            {:else}
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th></th>
                                <th>Tracking</th>
                                <th>From → To</th>
                                <th>Product</th>
                                <th>Status</th>
                                <th>Delivered</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each shipments as shipment}
                                {@const details = parseRawResponse(
                                    shipment.rawResponse,
                                )}
                                {@const provider = getProvider(details)}
                                <tr
                                    class="shipment-row"
                                    class:expanded={expandedShipments.has(
                                        shipment.id,
                                    )}
                                    on:click={() =>
                                        toggleShipmentDetails(shipment.id)}
                                >
                                    <td class="expand-cell">
                                        <span class="expand-icon"
                                            >{expandedShipments.has(shipment.id)
                                                ? "▼"
                                                : "▶"}</span
                                        >
                                    </td>
                                    <td class="tracking-cell">
                                        <span
                                            class="provider-badge"
                                            class:opal={provider === "opal"}
                                            class:dhl={provider === "dhl"}
                                        >
                                            {provider.toUpperCase()}
                                        </span>
                                        {#if shipment.trackingNumber || details?.ocu_number || details?.tracking_number}
                                            <span class="tracking-number"
                                                >{shipment.trackingNumber ||
                                                    details?.ocu_number ||
                                                    details?.tracking_number}</span
                                            >
                                            {#if details?.hwb_number}
                                                <span class="hwb-number"
                                                    >HWB: {details.hwb_number}</span
                                                >
                                            {/if}
                                        {:else}
                                            <span class="muted">Pending...</span
                                            >
                                        {/if}
                                    </td>
                                    <td class="route-cell">
                                        {#if details}
                                            <div class="route">
                                                <!-- OPAL format: pickup_name -> delivery_name -->
                                                <!-- DHL format: InBody -> recipient_name -->
                                                <span class="from"
                                                    >{details.pickup_name ||
                                                        details.pickup_city ||
                                                        "InBody"}</span
                                                >
                                                <span class="arrow">→</span>
                                                <span class="to"
                                                    >{details.delivery_name ||
                                                        details.recipient_name ||
                                                        details.delivery_city ||
                                                        details.recipient_city ||
                                                        "-"}</span
                                                >
                                            </div>
                                        {:else}
                                            <span class="muted">-</span>
                                        {/if}
                                    </td>
                                    <td>
                                        {#if details?.product_type || details?.product}
                                            <span class="product-badge"
                                                >{details.product_type ||
                                                    details.product}</span
                                            >
                                        {:else}
                                            <span class="muted">-</span>
                                        {/if}
                                    </td>
                                    <td>
                                        <span
                                            class="state-badge"
                                            style="background-color: {getDeliveryStateColor(
                                                shipment.status,
                                            )}"
                                        >
                                            {shipment.status}
                                        </span>
                                    </td>
                                    <td>
                                        {#if details?.status_date}
                                            <div class="delivery-info">
                                                <span
                                                    >{details.status_date}
                                                    {details.status_time ||
                                                        ""}</span
                                                >
                                                {#if details.receiver}
                                                    <span class="receiver"
                                                        >📝 {details.receiver}</span
                                                    >
                                                {/if}
                                            </div>
                                        {:else}
                                            <span class="muted">-</span>
                                        {/if}
                                    </td>
                                    <td on:click|stopPropagation>
                                        {#if shipment.status === "pending" || shipment.status === "processing"}
                                            <button
                                                class="action-btn cancel-btn"
                                                on:click={() =>
                                                    cancelShipment(
                                                        shipment.picking_id,
                                                    )}
                                            >
                                                ❌ Cancel
                                            </button>
                                        {:else}
                                            <span class="muted">-</span>
                                        {/if}
                                    </td>
                                </tr>
                                {#if expandedShipments.has(shipment.id) && details}
                                    <tr class="details-row">
                                        <td colspan="7">
                                            <div class="shipment-details">
                                                <div class="details-grid">
                                                    <div class="detail-section">
                                                        <h4>
                                                            📦 Pickup (Abholung)
                                                        </h4>
                                                        <div
                                                            class="detail-item"
                                                        >
                                                            <label
                                                                >Company:</label
                                                            >
                                                            <span
                                                                >{details.pickup_name ||
                                                                    "-"}</span
                                                            >
                                                        </div>
                                                        {#if details.pickup_contact}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Contact:</label
                                                                >
                                                                <span
                                                                    >{details.pickup_contact}</span
                                                                >
                                                            </div>
                                                        {/if}
                                                        <div
                                                            class="detail-item"
                                                        >
                                                            <label
                                                                >Address:</label
                                                            >
                                                            <span
                                                                >{details.pickup_street ||
                                                                    "-"}, {details.pickup_zip}
                                                                {details.pickup_city}</span
                                                            >
                                                        </div>
                                                        {#if details.pickup_phone && details.pickup_phone !== "+49 ()"}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Phone:</label
                                                                >
                                                                <span
                                                                    >{details.pickup_phone}</span
                                                                >
                                                            </div>
                                                        {/if}
                                                        <div
                                                            class="detail-item"
                                                        >
                                                            <label
                                                                >Date/Time:</label
                                                            >
                                                            <span
                                                                >{details.pickup_date ||
                                                                    "-"}
                                                                {details.pickup_time ||
                                                                    ""}</span
                                                            >
                                                        </div>
                                                        {#if details.pickup_note}
                                                            <div
                                                                class="detail-item note"
                                                            >
                                                                <label
                                                                    >Note:</label
                                                                >
                                                                <span
                                                                    >{details.pickup_note}</span
                                                                >
                                                            </div>
                                                        {/if}
                                                    </div>

                                                    <div class="detail-section">
                                                        <h4>
                                                            🚚 Delivery
                                                            (Zustellung)
                                                        </h4>
                                                        <div
                                                            class="detail-item"
                                                        >
                                                            <label
                                                                >Company:</label
                                                            >
                                                            <span
                                                                >{details.delivery_name ||
                                                                    "-"}</span
                                                            >
                                                        </div>
                                                        {#if details.delivery_contact}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Contact:</label
                                                                >
                                                                <span
                                                                    >{details.delivery_contact}</span
                                                                >
                                                            </div>
                                                        {/if}
                                                        <div
                                                            class="detail-item"
                                                        >
                                                            <label
                                                                >Address:</label
                                                            >
                                                            <span
                                                                >{details.delivery_street ||
                                                                    "-"}, {details.delivery_zip}
                                                                {details.delivery_city}</span
                                                            >
                                                        </div>
                                                        {#if details.delivery_phone && details.delivery_phone !== "+49 ()"}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Phone:</label
                                                                >
                                                                <span
                                                                    >{details.delivery_phone}</span
                                                                >
                                                            </div>
                                                        {/if}
                                                        <div
                                                            class="detail-item"
                                                        >
                                                            <label
                                                                >Date/Time:</label
                                                            >
                                                            <span
                                                                >{details.delivery_date ||
                                                                    "-"}
                                                                {details.delivery_time ||
                                                                    ""}</span
                                                            >
                                                        </div>
                                                        {#if details.delivery_note}
                                                            <div
                                                                class="detail-item note"
                                                            >
                                                                <label
                                                                    >Note:</label
                                                                >
                                                                <span
                                                                    >{details.delivery_note}</span
                                                                >
                                                            </div>
                                                        {/if}
                                                    </div>

                                                    <div class="detail-section">
                                                        <h4>📋 Package Info</h4>
                                                        {#if details.description}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Contents:</label
                                                                >
                                                                <span
                                                                    class="highlight"
                                                                    >{details.description}</span
                                                                >
                                                            </div>
                                                        {/if}
                                                        {#if details.package_count}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Packages:</label
                                                                >
                                                                <span
                                                                    >{details.package_count}
                                                                    pcs</span
                                                                >
                                                            </div>
                                                        {/if}
                                                        {#if details.weight}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Weight:</label
                                                                >
                                                                <span
                                                                    >{details.weight}
                                                                    kg</span
                                                                >
                                                            </div>
                                                        {/if}
                                                        {#if details.dimensions}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Dimensions:</label
                                                                >
                                                                <span
                                                                    >{details.dimensions}
                                                                    cm</span
                                                                >
                                                            </div>
                                                        {/if}
                                                        {#if details.value}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Value:</label
                                                                >
                                                                <span
                                                                    class="value"
                                                                    >{details.value.toLocaleString(
                                                                        "de-DE",
                                                                    )} EUR</span
                                                                >
                                                            </div>
                                                        {/if}
                                                    </div>

                                                    <div class="detail-section">
                                                        <h4>📊 Status</h4>
                                                        <div
                                                            class="detail-item"
                                                        >
                                                            <label
                                                                >{provider.toUpperCase()}
                                                                Status:</label
                                                            >
                                                            <span
                                                                class="status-value"
                                                                >{details.status ||
                                                                    "-"}</span
                                                            >
                                                        </div>
                                                        {#if details.status_date}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Date/Time:</label
                                                                >
                                                                <span
                                                                    >{details.status_date}
                                                                    {details.status_time ||
                                                                        ""}</span
                                                                >
                                                            </div>
                                                        {/if}
                                                        {#if details.receiver}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Received
                                                                    by:</label
                                                                >
                                                                <span
                                                                    class="highlight"
                                                                    >{details.receiver}</span
                                                                >
                                                            </div>
                                                        {/if}
                                                        {#if details.created_at}
                                                            <div
                                                                class="detail-item"
                                                            >
                                                                <label
                                                                    >Created:</label
                                                                >
                                                                <span
                                                                    >{details.created_at}
                                                                    by {details.created_by ||
                                                                        "-"}</span
                                                                >
                                                            </div>
                                                        {/if}
                                                    </div>
                                                </div>
                                            </div>
                                        </td>
                                    </tr>
                                {/if}
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    {:else if activeTab === "sync"}
        <div class="sync-section">
            <p class="section-desc">
                Synchronization history with external services (OPAL, DHL,
                Odoo). OPAL syncs every hour (on the hour), DHL syncs at :30
                past the hour. Active 8 AM - 6 PM.
            </p>

            {#if syncHistory.length === 0}
                <div class="empty-state">
                    <p>📭 No sync history yet</p>
                    <small>Synchronizations will appear automatically</small>
                </div>
            {:else}
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th></th>
                                <th>Time</th>
                                <th>Provider</th>
                                <th>Status</th>
                                <th>Created</th>
                                <th>Updated</th>
                                <th>Skipped</th>
                                <th>Duration</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each syncHistory as sync}
                                <tr
                                    class="sync-row"
                                    class:expanded={expandedSyncLogs.has(
                                        sync.id,
                                    )}
                                    class:has-error={sync.status === "error"}
                                    on:click={() =>
                                        sync.status === "error"
                                            ? toggleSyncDetails(sync.id)
                                            : null}
                                >
                                    <td class="expand-cell">
                                        {#if sync.status === "error"}
                                            <span class="expand-icon"
                                                >{expandedSyncLogs.has(sync.id)
                                                    ? "▼"
                                                    : "▶"}</span
                                            >
                                        {:else}
                                            <span class="muted">-</span>
                                        {/if}
                                    </td>
                                    <td class="sync-time"
                                        >{formatDate(sync.startedAt)}</td
                                    >
                                    <td>
                                        <span
                                            class="provider-badge"
                                            class:opal={sync.provider ===
                                                "opal"}
                                            class:dhl={sync.provider === "dhl"}
                                        >
                                            {sync.provider.toUpperCase()}
                                        </span>
                                    </td>
                                    <td>
                                        <span
                                            class="sync-badge"
                                            class:success={sync.status ===
                                                "success"}
                                            class:error={sync.status ===
                                                "error"}
                                            class:running={sync.status ===
                                                "running"}
                                        >
                                            {sync.status === "success"
                                                ? "✅ Success"
                                                : sync.status === "error"
                                                  ? "❌ Error"
                                                  : "⏳ Running"}
                                        </span>
                                    </td>
                                    <td class="stat-cell"
                                        >{sync.created || 0}</td
                                    >
                                    <td class="stat-cell"
                                        >{sync.updated || 0}</td
                                    >
                                    <td class="stat-cell muted"
                                        >{sync.skipped || 0}</td
                                    >
                                    <td class="duration-cell"
                                        >{sync.duration
                                            ? (sync.duration / 1000).toFixed(
                                                  1,
                                              ) + "s"
                                            : "-"}</td
                                    >
                                    <td on:click|stopPropagation>
                                        {#if sync.status === "error" && (sync.errorDetail || sync.debugInfo)}
                                            <button
                                                class="action-btn copy-btn"
                                                on:click={() =>
                                                    copyDebugInfo(sync)}
                                                title="Copy debug info for AI"
                                            >
                                                🤖 Copy for AI
                                            </button>
                                        {:else}
                                            <span class="muted">-</span>
                                        {/if}
                                    </td>
                                </tr>
                                {#if expandedSyncLogs.has(sync.id) && sync.status === "error"}
                                    <tr class="debug-row">
                                        <td colspan="9">
                                            <div class="debug-details">
                                                <div class="debug-section">
                                                    <h4>⚠️ Error</h4>
                                                    <pre
                                                        class="error-message">{sync.errorDetail ||
                                                            "No error detail"}</pre>
                                                </div>

                                                {#if sync.debugInfo}
                                                    <div class="debug-section">
                                                        <h4>
                                                            🔍 Debug Information
                                                        </h4>
                                                        <div class="debug-grid">
                                                            {#if sync.debugInfo.error_category}
                                                                <div
                                                                    class="debug-item"
                                                                >
                                                                    <label
                                                                        >Category:</label
                                                                    >
                                                                    <span
                                                                        class="category-badge"
                                                                        class:playwright={sync
                                                                            .debugInfo
                                                                            .error_category ===
                                                                            "playwright_scraper"}
                                                                    >
                                                                        {sync
                                                                            .debugInfo
                                                                            .error_category}
                                                                    </span>
                                                                </div>
                                                            {/if}
                                                            {#if sync.debugInfo.likely_cause}
                                                                <div
                                                                    class="debug-item"
                                                                >
                                                                    <label
                                                                        >Likely
                                                                        Cause:</label
                                                                    >
                                                                    <span
                                                                        class="highlight"
                                                                        >{sync
                                                                            .debugInfo
                                                                            .likely_cause}</span
                                                                    >
                                                                </div>
                                                            {/if}
                                                            {#if sync.debugInfo.ai_analysis_hint}
                                                                <div
                                                                    class="debug-item"
                                                                >
                                                                    <label
                                                                        >💡 AI
                                                                        Hint:</label
                                                                    >
                                                                    <span
                                                                        class="ai-hint"
                                                                        >{sync
                                                                            .debugInfo
                                                                            .ai_analysis_hint}</span
                                                                    >
                                                                </div>
                                                            {/if}
                                                            {#if sync.debugInfo.step}
                                                                <div
                                                                    class="debug-item"
                                                                >
                                                                    <label
                                                                        >Step:</label
                                                                    >
                                                                    <span
                                                                        >{sync
                                                                            .debugInfo
                                                                            .step}</span
                                                                    >
                                                                </div>
                                                            {/if}
                                                        </div>

                                                        {#if sync.debugInfo.playwright_stderr}
                                                            <div
                                                                class="stderr-section"
                                                            >
                                                                <h5>
                                                                    📋
                                                                    Playwright
                                                                    Output
                                                                    (stderr):
                                                                </h5>
                                                                <pre
                                                                    class="stderr-output">{sync
                                                                        .debugInfo
                                                                        .playwright_stderr}</pre>
                                                            </div>
                                                        {/if}

                                                        <details
                                                            class="raw-json"
                                                        >
                                                            <summary
                                                                >🔧 Raw Debug
                                                                JSON</summary
                                                            >
                                                            <pre>{JSON.stringify(
                                                                    sync.debugInfo,
                                                                    null,
                                                                    2,
                                                                )}</pre>
                                                        </details>
                                                    </div>
                                                {/if}
                                            </div>
                                        </td>
                                    </tr>
                                {/if}
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .shipping-page {
        padding: 0;
    }

    header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 1.5rem;
    }

    h1 {
        font-size: 1.8rem;
        color: #fff;
        margin: 0;
    }

    .header-actions {
        display: flex;
        gap: 1rem;
    }

    .refresh-btn {
        padding: 0.6rem 1.2rem;
        border-radius: 4px;
        border: 1px solid #4a69bd;
        background: transparent;
        color: #4a69bd;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
    }

    .refresh-btn:hover:not(:disabled) {
        background: #4a69bd;
        color: white;
    }

    .refresh-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .action-btn.secondary {
        background: #333;
        color: #ccc;
        border: 1px solid #444;
    }

    .action-btn.secondary:hover:not(:disabled) {
        background: #444;
        color: #fff;
    }

    .action-btn.opal-btn {
        background: #1a472a;
        color: #4ade80;
        border: 1px solid #22c55e;
    }

    .action-btn.opal-btn:hover:not(:disabled) {
        background: #166534;
    }

    .action-btn.dhl-btn {
        background: #422006;
        color: #fbbf24;
        border: 1px solid #f59e0b;
    }

    .action-btn.dhl-btn:hover:not(:disabled) {
        background: #713f12;
    }

    .provider-badge {
        display: inline-block;
        padding: 0.15rem 0.4rem;
        border-radius: 3px;
        font-size: 0.65rem;
        font-weight: 700;
        text-transform: uppercase;
        margin-right: 0.5rem;
    }

    .provider-badge.opal {
        background: #166534;
        color: #4ade80;
    }

    .provider-badge.dhl {
        background: #713f12;
        color: #fbbf24;
    }

    .tabs {
        display: flex;
        gap: 1rem;
        margin-bottom: 2rem;
        border-bottom: 2px solid #333;
    }

    .tab {
        padding: 0.8rem 1.5rem;
        border: none;
        background: transparent;
        color: #aaa;
        font-size: 1rem;
        font-weight: 600;
        cursor: pointer;
        border-bottom: 3px solid transparent;
        transition: all 0.2s;
    }

    .tab:hover {
        color: #fff;
    }

    .tab.active {
        color: #4a69bd;
        border-bottom-color: #4a69bd;
    }

    .section-desc {
        color: #aaa;
        margin-bottom: 1.5rem;
        font-size: 0.95rem;
    }

    .loading,
    .error {
        text-align: center;
        padding: 3rem;
        color: #666;
        background: #1e1e1e;
        border-radius: 8px;
        border: 1px solid #333;
    }

    .error {
        color: #ff6b6b;
        border-color: #ff6b6b;
    }

    .empty-state {
        text-align: center;
        padding: 3rem;
        color: #666;
        background: #1e1e1e;
        border-radius: 8px;
        border: 1px dashed #333;
    }

    .empty-state p {
        font-size: 1.2rem;
        margin: 0 0 0.5rem 0;
    }

    .empty-state small {
        color: #555;
    }

    .table-container {
        background: #1e1e1e;
        border-radius: 8px;
        border: 1px solid #333;
        overflow-x: auto;
    }

    table {
        width: 100%;
        border-collapse: collapse;
    }

    thead {
        background: #252525;
    }

    th {
        padding: 1rem;
        text-align: left;
        font-weight: 600;
        color: #aaa;
        text-transform: uppercase;
        font-size: 0.75rem;
        letter-spacing: 0.5px;
        border-bottom: 2px solid #333;
    }

    td {
        padding: 1rem;
        border-bottom: 1px solid #2a2a2a;
        color: #e0e0e0;
    }

    tbody tr:hover {
        background: #252525;
    }

    .picking-name {
        font-family: monospace;
        color: #4a69bd;
        font-weight: 600;
    }

    .shipment-id {
        font-family: monospace;
        color: #4a69bd;
    }

    .location-cell {
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }

    .location-cell .arrow {
        color: #666;
    }

    .location-cell .from {
        color: #ffc107;
    }

    .location-cell .to {
        color: #28a745;
    }

    .state-badge {
        display: inline-block;
        padding: 0.3rem 0.8rem;
        border-radius: 12px;
        font-size: 0.75rem;
        font-weight: 600;
        text-transform: uppercase;
        color: white;
    }

    .provider-badge {
        display: inline-block;
        padding: 0.3rem 0.8rem;
        border-radius: 4px;
        background: #2a2a2a;
        font-family: monospace;
        font-size: 0.85rem;
        text-transform: uppercase;
        color: #4a69bd;
    }

    .tracking-link {
        color: #4a69bd;
        text-decoration: none;
        font-family: monospace;
    }

    .tracking-link:hover {
        text-decoration: underline;
    }

    .muted {
        color: #666;
        font-style: italic;
    }

    .action-btn {
        padding: 0.5rem 1rem;
        border-radius: 4px;
        border: none;
        font-weight: 600;
        font-size: 0.85rem;
        cursor: pointer;
        transition: all 0.2s;
        white-space: nowrap;
    }

    .ship-btn {
        background: #28a745;
        color: white;
    }

    .ship-btn:hover:not(:disabled) {
        background: #218838;
    }

    .ship-btn:disabled {
        background: #555;
        cursor: not-allowed;
        opacity: 0.6;
    }

    .cancel-btn {
        background: transparent;
        border: 1px solid #dc3545;
        color: #dc3545;
    }

    .cancel-btn:hover {
        background: #dc3545;
        color: white;
    }

    /* Expandable rows */
    .shipment-row {
        cursor: pointer;
        transition: background 0.2s;
    }

    .shipment-row:hover {
        background: #2a2a2a;
    }

    .shipment-row.expanded {
        background: #252525;
        border-bottom: none;
    }

    .expand-cell {
        width: 30px;
        text-align: center;
    }

    .expand-icon {
        color: #666;
        font-size: 0.8rem;
    }

    .tracking-cell {
        display: flex;
        flex-direction: column;
        gap: 0.2rem;
    }

    .tracking-number {
        font-family: monospace;
        color: #4a69bd;
        font-weight: 600;
    }

    .hwb-number {
        font-family: monospace;
        font-size: 0.75rem;
        color: #888;
    }

    .route-cell .route {
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }

    .route .from {
        color: #ffc107;
        font-weight: 500;
    }

    .route .arrow {
        color: #666;
    }

    .route .to {
        color: #28a745;
        font-weight: 500;
    }

    .product-badge {
        display: inline-block;
        padding: 0.2rem 0.6rem;
        border-radius: 4px;
        background: #2a2a2a;
        font-size: 0.8rem;
        color: #aaa;
    }

    .delivery-info {
        display: flex;
        flex-direction: column;
        gap: 0.2rem;
    }

    .receiver {
        font-size: 0.8rem;
        color: #28a745;
    }

    /* Details row */
    .details-row {
        background: #1a1a1a;
    }

    .details-row td {
        padding: 0;
        border-bottom: 2px solid #333;
    }

    .shipment-details {
        padding: 1.5rem;
    }

    .details-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
        gap: 1.5rem;
    }

    .detail-section {
        background: #252525;
        border-radius: 8px;
        padding: 1rem;
    }

    .detail-section h4 {
        margin: 0 0 1rem 0;
        color: #fff;
        font-size: 0.95rem;
        border-bottom: 1px solid #333;
        padding-bottom: 0.5rem;
    }

    .detail-item {
        display: flex;
        gap: 0.5rem;
        margin-bottom: 0.5rem;
        font-size: 0.9rem;
    }

    .detail-item label {
        color: #888;
        min-width: 80px;
        flex-shrink: 0;
    }

    .detail-item span {
        color: #e0e0e0;
    }

    .detail-item .highlight {
        color: #4a69bd;
        font-weight: 600;
    }

    .detail-item .value {
        color: #28a745;
        font-weight: 600;
    }

    .detail-item .status-value {
        color: #28a745;
        font-weight: 600;
        text-transform: uppercase;
    }

    .detail-item.note span {
        color: #ffc107;
        font-style: italic;
    }

    /* Sync history styles */
    .sync-section {
        padding: 0;
    }

    .sync-time {
        font-family: monospace;
        color: #aaa;
        font-size: 0.9rem;
    }

    .sync-badge {
        display: inline-block;
        padding: 0.3rem 0.8rem;
        border-radius: 12px;
        font-size: 0.75rem;
        font-weight: 600;
        color: white;
    }

    .sync-badge.success {
        background: #28a745;
    }

    .sync-badge.error {
        background: #dc3545;
    }

    .sync-badge.running {
        background: #17a2b8;
    }

    .stat-cell {
        font-family: monospace;
        text-align: center;
        color: #4a69bd;
        font-weight: 600;
    }

    .duration-cell {
        font-family: monospace;
        color: #888;
    }

    .error-detail {
        color: #ff6b6b;
        font-size: 0.85rem;
        cursor: help;
    }

    .sync-row {
        transition: background 0.2s;
    }

    .sync-row.has-error {
        cursor: pointer;
    }

    .sync-row.has-error:hover {
        background: #2a2a2a;
    }

    .sync-row.expanded {
        background: #252525;
        border-bottom: none;
    }

    .copy-btn {
        background: #1a472a;
        color: #4ade80;
        border: 1px solid #22c55e;
        padding: 0.4rem 0.8rem;
        font-size: 0.8rem;
    }

    .copy-btn:hover {
        background: #166534;
    }

    /* Debug details row */
    .debug-row {
        background: #1a1a1a;
    }

    .debug-row td {
        padding: 0;
        border-bottom: 2px solid #333;
    }

    .debug-details {
        padding: 1.5rem;
    }

    .debug-section {
        background: #252525;
        border-radius: 8px;
        padding: 1rem;
        margin-bottom: 1rem;
    }

    .debug-section h4 {
        margin: 0 0 1rem 0;
        color: #fff;
        font-size: 0.95rem;
        border-bottom: 1px solid #333;
        padding-bottom: 0.5rem;
    }

    .debug-section h5 {
        margin: 1rem 0 0.5rem 0;
        color: #aaa;
        font-size: 0.85rem;
    }

    .error-message {
        background: #2a1a1a;
        color: #ff6b6b;
        padding: 1rem;
        border-radius: 4px;
        border-left: 3px solid #dc3545;
        overflow-x: auto;
        font-size: 0.85rem;
        line-height: 1.4;
        white-space: pre-wrap;
        word-wrap: break-word;
    }

    .debug-grid {
        display: grid;
        gap: 0.75rem;
    }

    .debug-item {
        display: flex;
        gap: 0.5rem;
        font-size: 0.9rem;
    }

    .debug-item label {
        color: #888;
        min-width: 150px;
        flex-shrink: 0;
    }

    .debug-item span {
        color: #e0e0e0;
    }

    .debug-item .highlight {
        color: #ffc107;
        font-weight: 600;
    }

    .debug-item .ai-hint {
        color: #4ade80;
        font-style: italic;
    }

    .category-badge {
        display: inline-block;
        padding: 0.2rem 0.6rem;
        border-radius: 4px;
        background: #2a2a2a;
        font-size: 0.8rem;
        text-transform: uppercase;
        font-weight: 600;
    }

    .category-badge.playwright {
        background: #422006;
        color: #fbbf24;
    }

    .stderr-section {
        margin-top: 1rem;
    }

    .stderr-output {
        background: #1a1a1a;
        color: #aaa;
        padding: 1rem;
        border-radius: 4px;
        border: 1px solid #333;
        overflow-x: auto;
        font-size: 0.8rem;
        line-height: 1.4;
        max-height: 300px;
        overflow-y: auto;
    }

    .raw-json {
        margin-top: 1rem;
    }

    .raw-json summary {
        cursor: pointer;
        color: #888;
        font-size: 0.85rem;
        padding: 0.5rem;
        background: #2a2a2a;
        border-radius: 4px;
        user-select: none;
    }

    .raw-json summary:hover {
        color: #aaa;
        background: #333;
    }

    .raw-json pre {
        background: #1a1a1a;
        color: #4a69bd;
        padding: 1rem;
        border-radius: 4px;
        border: 1px solid #333;
        overflow-x: auto;
        font-size: 0.75rem;
        line-height: 1.4;
        margin-top: 0.5rem;
    }
</style>

<script>
    import { onMount } from 'svelte';
    import { toastStore } from '$lib/stores/toastStore';

    // Config state
    let selectedType = null;
    let isTightMode = true;

    // Layout config
    let cols = 3;
    let rows = 7;
    let pages = 1;
    let count = 21;
    let startNumber = 1;

    // Margins (mm)
    let marginTop = 4;
    let marginBottom = 4;
    let marginLeft = 4;
    let marginRight = 4;
    let gapX = 8;
    let gapY = 6;

    // Content styling
    let serialDigits = 6;
    let selectedElement = 'qr1';

    // Style config per element
    let styleCfg = {
        qr1: { scale: 1, x: 0, y: 0 },
        qr2: { scale: 0.3, x: 62, y: 22 },
        qr3: { scale: 0.3, x: 82, y: 22 },
        checksum: { scale: 0.55, x: 61, y: 60 },
        serial: { scale: 0.15, x: 62, y: 4 }
    };

    // Defaults per type
    const defaults = {
        'i': {
            '3x7': {
                qr1: { scale: 1, x: 0, y: 0 },
                qr2: { scale: 0.3, x: 62, y: 22 },
                qr3: { scale: 0.3, x: 82, y: 22 },
                checksum: { scale: 0.55, x: 61, y: 60 },
                serial: { scale: 0.15, x: 62, y: 4 },
                serialDigits: 6
            },
            '2x8': {
                qr1: { scale: 0.8, x: 5, y: 10 },
                qr2: { scale: 0.3, x: 75, y: 50 },
                qr3: { scale: 0.3, x: 75, y: 10 },
                checksum: { scale: 0.5, x: 40, y: 35 },
                serial: { scale: 0.2, x: 40, y: 10 },
                serialDigits: 6
            }
        },
        'b': {
            '3x7': {
                qr1: { scale: 1, x: 0, y: 0 },
                qr2: { scale: 0.3, x: 61, y: 0 },
                qr3: { scale: 0.3, x: 82, y: 0 },
                checksum: { scale: 0.55, x: 61, y: 39 },
                serial: { scale: 0.15, x: 61, y: 87 },
                serialDigits: 6
            }
        },
        'p': {
            '3x7': {
                qr1: { scale: 0.8, x: 0, y: 19 },
                qr2: { scale: 0.4, x: 50, y: 0 },
                qr3: { scale: 0.4, x: 77, y: 0 },
                checksum: { scale: 0.5, x: 50, y: 51 },
                serial: { scale: 0.15, x: 0, y: 0 },
                serialDigits: 8
            }
        },
        'l': {
            '3x7': {
                qr1: { scale: 1, x: 0, y: 0 },
                qr2: { scale: 0.3, x: 62, y: 50 },
                qr3: { scale: 0.3, x: 82, y: 50 },
                checksum: { scale: 0.6, x: 60, y: 0 },
                serial: { scale: 0.15, x: 62, y: 87 },
                serialDigits: 6
            }
        }
    };

    // Warehouse config for Places
    let warehouseConfig = { regals: [] };
    let registeredRacks = [];
    let selectedRack = '';
    let showPlanner = false;

    // Rack editor
    let editRackId = null;
    let rackName = '';
    let rackCols = '';
    let rackRows = '';
    let rackStart = '';

    let loading = false;
    let previewEl;

    // Computed
    $: layoutKey = `${cols}x${rows}`;
    $: count = pages * cols * rows;

    // Label dimensions calculation
    $: labelDims = calculateLabelDims();

    function calculateLabelDims() {
        const extraX = isTightMode ? 0 : (gapX / 2);
        const extraY = isTightMode ? 0 : (gapY / 2);

        const effMarginLeft = marginLeft + extraX;
        const effMarginRight = marginRight + extraX;
        const effMarginTop = marginTop + extraY;
        const effMarginBottom = marginBottom + extraY;

        const pageWidth = 210;
        const pageHeight = 297;

        const workingWidth = pageWidth - effMarginLeft - effMarginRight;
        const workingHeight = pageHeight - effMarginTop - effMarginBottom;

        const totalGapWidth = (cols - 1) * gapX;
        const totalGapHeight = (rows - 1) * gapY;

        const w = Math.max(0, (workingWidth - totalGapWidth) / cols);
        const h = Math.max(0, (workingHeight - totalGapHeight) / rows);

        return { w: w.toFixed(1), h: h.toFixed(1), aspect: h === 0 ? 1 : w / h };
    }

    function selectType(type) {
        // Save current config before switching
        if (selectedType) {
            saveConfig(selectedType, cols, rows);
        }

        selectedType = type;

        // Load config for new type
        loadConfig(type, cols, rows);

        // Load racks if Places selected
        if (type === 'p') {
            loadRacks();
        }
    }

    function saveConfig(type, c, r) {
        const key = `${c}x${r}`;
        const saved = JSON.parse(localStorage.getItem('eck_print_layouts') || '{}');
        if (!saved[type]) saved[type] = {};
        saved[type][key] = { ...JSON.parse(JSON.stringify(styleCfg)), serialDigits };
        localStorage.setItem('eck_print_layouts', JSON.stringify(saved));
    }

    function loadConfig(type, c, r) {
        const key = `${c}x${r}`;
        const saved = JSON.parse(localStorage.getItem('eck_print_layouts') || '{}');

        // 1. Check saved
        if (saved[type]?.[key]) {
            styleCfg = JSON.parse(JSON.stringify(saved[type][key]));
            serialDigits = saved[type][key].serialDigits || 6;
            return;
        }

        // 2. Check defaults
        if (defaults[type]?.[key]) {
            styleCfg = JSON.parse(JSON.stringify(defaults[type][key]));
            serialDigits = defaults[type][key].serialDigits || 6;
            return;
        }

        // 3. Fallback
        if (defaults[type]) {
            const firstKey = Object.keys(defaults[type])[0];
            if (firstKey) {
                styleCfg = JSON.parse(JSON.stringify(defaults[type][firstKey]));
                serialDigits = defaults[type][firstKey].serialDigits || 6;
            }
        }
    }

    function resetToDefault() {
        const type = selectedType || 'i';
        if (defaults[type]?.[layoutKey]) {
            styleCfg = JSON.parse(JSON.stringify(defaults[type][layoutKey]));
            serialDigits = defaults[type][layoutKey].serialDigits || 6;
        }
    }

    function onLayoutChange() {
        if (selectedType) {
            saveConfig(selectedType, cols, rows);
        }
        loadConfig(selectedType || 'i', cols, rows);
    }

    // Warehouse/Rack functions
    async function loadRacks() {
        try {
            const token = localStorage.getItem('auth_token');
            const res = await fetch('/api/warehouse', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
                const warehouses = await res.json();
                // Flatten racks from all warehouses
                registeredRacks = [];
                warehouses.forEach(wh => {
                    if (wh.racks) {
                        wh.racks.forEach((rack, idx) => {
                            registeredRacks.push({
                                ...rack,
                                warehouse_name: wh.name,
                                sort_order: idx + 1
                            });
                        });
                    }
                });
                buildWarehouseConfig();
            }
        } catch (e) {
            console.error('Failed to load racks', e);
        }
    }

    function buildWarehouseConfig() {
        warehouseConfig.regals = registeredRacks.map((rack, idx) => ({
            index: rack.sort_order || (idx + 1),
            columns: parseInt(rack.columns) || 10,
            rows: parseInt(rack.rows) || 5,
            start_index: parseInt(rack.start_index) || 0
        }));
    }

    function onRackSelect() {
        if (!selectedRack) return;
        const rack = registeredRacks.find(r => r.id == selectedRack);
        if (rack) {
            startNumber = rack.start_index || 0;
            count = (rack.columns || 10) * (rack.rows || 5);
        }
    }

    async function generatePDF() {
        if (!selectedType) {
            toastStore.add('Please select a label type first', 'error');
            return;
        }

        loading = true;
        try {
            const extraX = isTightMode ? 0 : (gapX / 2);
            const extraY = isTightMode ? 0 : (gapY / 2);

            const requestBody = {
                type: selectedType,
                startNumber: parseInt(startNumber),
                count: parseInt(count),
                cols: parseInt(cols),
                rows: parseInt(rows),
                marginTop: marginTop + extraY,
                marginBottom: marginBottom + extraY,
                marginLeft: marginLeft + extraX,
                marginRight: marginRight + extraX,
                gapX: gapX,
                gapY: gapY,
                isTightMode: isTightMode,
                serialDigits: serialDigits,
                contentConfig: styleCfg
            };

            if (selectedType === 'p') {
                requestBody.warehouseConfig = warehouseConfig;
            }

            const token = localStorage.getItem('auth_token');
            const response = await fetch('/api/print/labels', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify(requestBody)
            });

            if (!response.ok) {
                const err = await response.text();
                throw new Error(err || 'Failed to generate PDF');
            }

            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `labels_${selectedType}_${startNumber}.pdf`;
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);

            toastStore.add('Labels generated successfully!', 'success');
        } catch (e) {
            console.error(e);
            toastStore.add(e.message, 'error');
        } finally {
            loading = false;
        }
    }

    onMount(() => {
        // Load saved settings
        const saved = localStorage.getItem('print_settings');
        if (saved) {
            try {
                const s = JSON.parse(saved);
                if (s.marginTop) marginTop = parseFloat(s.marginTop);
                if (s.marginBottom) marginBottom = parseFloat(s.marginBottom);
                if (s.marginLeft) marginLeft = parseFloat(s.marginLeft);
                if (s.marginRight) marginRight = parseFloat(s.marginRight);
                if (s.gapX) gapX = parseFloat(s.gapX);
                if (s.gapY) gapY = parseFloat(s.gapY);
                if (s.cols) cols = parseInt(s.cols);
                if (s.rows) rows = parseInt(s.rows);
            } catch (e) {}
        }
    });

    // Auto-save settings
    $: if (typeof window !== 'undefined') {
        localStorage.setItem('print_settings', JSON.stringify({
            marginTop, marginBottom, marginLeft, marginRight, gapX, gapY, cols, rows
        }));
    }
</script>

<div class="print-page">
    <header>
        <h1>Printing Center</h1>
    </header>

    <!-- Page Layout Editor -->
    <div class="card">
        <h2>Page Configuration</h2>

        <div class="config-header">
            <div class="toggle-group">
                <span class="toggle-label">Overlap:</span>
                <button
                    class="toggle-switch"
                    class:active={isTightMode}
                    on:click={() => isTightMode = !isTightMode}
                >
                    <div class="toggle-slider"></div>
                </button>
            </div>

            <div class="inline-field">
                <label>Cols:</label>
                <input type="number" bind:value={cols} min="1" max="10" on:change={onLayoutChange} />
            </div>
            <div class="inline-field">
                <label>Rows:</label>
                <input type="number" bind:value={rows} min="1" max="20" on:change={onLayoutChange} />
            </div>
            <div class="inline-field">
                <label>Pages:</label>
                <input type="number" bind:value={pages} min="1" />
            </div>
            <div class="inline-field">
                <label>Total:</label>
                <input type="number" bind:value={count} />
            </div>
            <div class="inline-field">
                <label>Start #:</label>
                <input type="number" bind:value={startNumber} />
            </div>
        </div>

        <div class="visual-editor" class:is-safe={!isTightMode}>
            <div class="visual-page">
                <!-- Margin inputs -->
                <div class="margin-input top">
                    <input type="number" bind:value={marginTop} min="0" step="1" />
                    <span>Top</span>
                </div>
                <div class="margin-input left">
                    <input type="number" bind:value={marginLeft} min="0" step="1" />
                    <span>Left</span>
                </div>
                <div class="margin-input right">
                    <input type="number" bind:value={marginRight} min="0" step="1" />
                    <span>Right</span>
                </div>
                <div class="margin-input bottom">
                    <input type="number" bind:value={marginBottom} min="0" step="1" />
                    <span>Bottom</span>
                </div>

                <!-- Label grid preview -->
                <div class="label-grid">
                    <div class="label-box">
                        Label 1
                        <div class="gap-x">
                            <span>↔</span>
                            <input type="number" bind:value={gapX} min="0" title="Gap X (mm)" />
                        </div>
                        <div class="gap-y">
                            <span>↕</span>
                            <input type="number" bind:value={gapY} min="0" title="Gap Y (mm)" />
                        </div>
                    </div>
                    <div class="label-box">Label 2</div>
                    <div class="label-box">Label 3</div>
                    <div class="label-box">Label 4</div>
                </div>
            </div>
        </div>
    </div>

    <!-- Label Type Selection -->
    <div class="card">
        <h2>Select Label Type</h2>
        <div class="type-grid">
            <button
                class="type-card"
                class:active={selectedType === 'i'}
                on:click={() => selectType('i')}
            >
                <h3>Items</h3>
                <p>Individual product labels</p>
            </button>
            <button
                class="type-card"
                class:active={selectedType === 'b'}
                on:click={() => selectType('b')}
            >
                <h3>Boxes</h3>
                <p>Container labels</p>
            </button>
            <button
                class="type-card"
                class:active={selectedType === 'p'}
                on:click={() => selectType('p')}
            >
                <h3>Places</h3>
                <p>Location markers</p>
            </button>
            <button
                class="type-card"
                class:active={selectedType === 'l'}
                on:click={() => selectType('l')}
            >
                <h3>Labels</h3>
                <p>General purpose</p>
            </button>
        </div>
    </div>

    <!-- Warehouse Config (for Places) -->
    {#if selectedType === 'p'}
    <div class="card">
        <div class="card-header">
            <h2>Warehouse Locations</h2>
            <button class="btn-sm" on:click={() => showPlanner = !showPlanner}>
                {showPlanner ? 'Print Mode' : 'Planner'}
            </button>
        </div>

        {#if !showPlanner}
        <div class="rack-select">
            <label>Select Rack:</label>
            <select bind:value={selectedRack} on:change={onRackSelect}>
                <option value="">-- Manual Configuration --</option>
                {#each registeredRacks as rack}
                <option value={rack.id}>
                    {rack.name} ({rack.columns}x{rack.rows}, ID: {rack.start_index}+)
                </option>
                {/each}
            </select>
        </div>
        {:else}
        <div class="planner">
            <table class="rack-table">
                <thead>
                    <tr>
                        <th>#</th>
                        <th>Name</th>
                        <th>Size</th>
                        <th>ID Range</th>
                    </tr>
                </thead>
                <tbody>
                    {#each registeredRacks as rack, idx}
                    <tr>
                        <td><strong>{idx + 1}</strong></td>
                        <td>{rack.name}</td>
                        <td>{rack.columns} x {rack.rows}</td>
                        <td>{rack.start_index} - {rack.start_index + rack.columns * rack.rows - 1}</td>
                    </tr>
                    {/each}
                </tbody>
            </table>
        </div>
        {/if}
    </div>
    {/if}

    <!-- Label Content Styling -->
    <div class="card">
        <h2>Label Content Styling</h2>
        <div class="styling-layout">
            <!-- Preview -->
            <div class="preview-container">
                <h3>Live Preview ({layoutKey})</h3>
                <div
                    class="label-preview"
                    bind:this={previewEl}
                    style="aspect-ratio: {labelDims.aspect};"
                >
                    <div
                        class="pv-element pv-qr"
                        class:selected={selectedElement === 'qr1'}
                        style="left: {styleCfg.qr1.x}%; bottom: {styleCfg.qr1.y}%;
                               width: {styleCfg.qr1.scale * 60}px; height: {styleCfg.qr1.scale * 60}px;"
                        on:click={() => selectedElement = 'qr1'}
                    >QR1</div>
                    <div
                        class="pv-element pv-qr"
                        class:selected={selectedElement === 'qr2'}
                        style="left: {styleCfg.qr2.x}%; bottom: {styleCfg.qr2.y}%;
                               width: {styleCfg.qr2.scale * 60}px; height: {styleCfg.qr2.scale * 60}px;"
                        on:click={() => selectedElement = 'qr2'}
                    >QR2</div>
                    <div
                        class="pv-element pv-qr"
                        class:selected={selectedElement === 'qr3'}
                        style="left: {styleCfg.qr3.x}%; bottom: {styleCfg.qr3.y}%;
                               width: {styleCfg.qr3.scale * 60}px; height: {styleCfg.qr3.scale * 60}px;"
                        on:click={() => selectedElement = 'qr3'}
                    >QR3</div>
                    <div
                        class="pv-element pv-text"
                        class:selected={selectedElement === 'checksum'}
                        style="left: {styleCfg.checksum.x}%; bottom: {styleCfg.checksum.y}%;
                               font-size: {styleCfg.checksum.scale * 30}px;"
                        on:click={() => selectedElement = 'checksum'}
                    >XX</div>
                    <div
                        class="pv-element pv-text serial"
                        class:selected={selectedElement === 'serial'}
                        style="left: {styleCfg.serial.x}%; bottom: {styleCfg.serial.y}%;
                               font-size: {styleCfg.serial.scale * 30}px;"
                        on:click={() => selectedElement = 'serial'}
                    >123456</div>
                </div>
                <p class="preview-dims">Actual size: {labelDims.w} x {labelDims.h} mm</p>
            </div>

            <!-- Controls -->
            <div class="styling-controls">
                <div class="control-group">
                    <label>Element:</label>
                    <select bind:value={selectedElement}>
                        <option value="qr1">QR1 (Master)</option>
                        <option value="qr2">QR2 (Small)</option>
                        <option value="qr3">QR3 (Small)</option>
                        <option value="checksum">Checksum</option>
                        <option value="serial">Serial</option>
                    </select>
                </div>

                <div class="control-row">
                    <div class="control-item">
                        <label>X (%)</label>
                        <input type="number" bind:value={styleCfg[selectedElement].x} min="0" max="100" />
                    </div>
                    <div class="control-item">
                        <label>Y (%)</label>
                        <input type="number" bind:value={styleCfg[selectedElement].y} min="0" max="100" />
                    </div>
                </div>

                <div class="control-group">
                    <label>Scale (0.1 - 1.0)</label>
                    <input type="number" bind:value={styleCfg[selectedElement].scale} step="0.05" min="0.05" max="2" />
                </div>

                <div class="control-group">
                    <label>Serial Digits (0 = all)</label>
                    <input type="number" bind:value={serialDigits} min="0" max="18" />
                </div>

                <button class="btn-sm" on:click={resetToDefault}>Reset to Default</button>
            </div>
        </div>
    </div>

    <!-- Generate Button -->
    <div class="actions">
        <button class="btn primary large" on:click={generatePDF} disabled={loading || !selectedType}>
            {loading ? 'Generating...' : 'Generate and Download Labels'}
        </button>
    </div>
</div>

<style>
    .print-page { max-width: 1000px; margin: 0 auto; padding-bottom: 2rem; }
    header { margin-bottom: 1.5rem; }
    h1 { color: #fff; font-size: 1.8rem; margin: 0; }
    h2 { color: #5a7ba9; font-size: 1rem; text-transform: uppercase; letter-spacing: 1px; margin: 0 0 1rem 0; }
    h3 { margin: 0 0 0.5rem 0; }

    .card {
        background: #1e1e1e;
        border: 1px solid #333;
        border-radius: 8px;
        padding: 1.5rem;
        margin-bottom: 1rem;
    }

    .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 1rem;
    }
    .card-header h2 { margin: 0; }

    /* Config Header */
    .config-header {
        display: flex;
        flex-wrap: wrap;
        gap: 1rem;
        padding: 1rem;
        background: #252525;
        border-radius: 8px;
        margin-bottom: 1rem;
        align-items: center;
    }

    .toggle-group {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        padding-right: 1rem;
        border-right: 1px solid #444;
    }

    .toggle-label { color: #888; font-size: 0.85rem; }

    .toggle-switch {
        position: relative;
        width: 44px;
        height: 24px;
        background: #555;
        border-radius: 12px;
        cursor: pointer;
        border: none;
        transition: background 0.3s;
    }
    .toggle-switch.active { background: #5a7ba9; }
    .toggle-slider {
        position: absolute;
        top: 2px;
        left: 2px;
        width: 20px;
        height: 20px;
        background: white;
        border-radius: 50%;
        transition: transform 0.3s;
    }
    .toggle-switch.active .toggle-slider { transform: translateX(20px); }

    .inline-field {
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }
    .inline-field label { color: #888; font-size: 0.85rem; }
    .inline-field input {
        width: 60px;
        padding: 0.4rem;
        background: #111;
        border: 1px solid #444;
        border-radius: 4px;
        color: #fff;
        text-align: center;
    }

    /* Visual Editor */
    .visual-editor {
        display: flex;
        justify-content: center;
        padding: 1rem;
    }

    .visual-page {
        width: 320px;
        height: 320px;
        background: #d1d1d1;
        border: 1px solid #999;
        position: relative;
        border-radius: 4px;
        box-shadow: 0 8px 24px rgba(0,0,0,0.4);
    }

    .visual-page::before {
        content: '';
        position: absolute;
        top: 0;
        left: 25px;
        width: 0;
        height: 100%;
        border-left: 1.5px dashed #991b1b;
        opacity: 0.7;
    }
    .visual-page::after {
        content: '';
        position: absolute;
        top: 25px;
        left: 0;
        width: 100%;
        height: 0;
        border-top: 1.5px dashed #991b1b;
        opacity: 0.7;
    }

    .margin-input {
        position: absolute;
        display: flex;
        align-items: center;
        gap: 4px;
        background: rgba(255,255,255,0.9);
        padding: 2px 6px;
        border-radius: 4px;
        border: 1px solid rgba(153, 27, 27, 0.3);
        font-size: 10px;
        color: #991b1b;
        z-index: 10;
    }
    .margin-input input {
        width: 36px;
        padding: 2px;
        border: 1px solid #991b1b;
        border-radius: 3px;
        text-align: center;
        font-size: 0.8rem;
        color: #991b1b;
        background: white;
    }
    .margin-input.top { top: 4px; left: 50%; transform: translateX(-50%); }
    .margin-input.bottom { bottom: 4px; left: 50%; transform: translateX(-50%); }
    .margin-input.left { top: 50%; left: 4px; transform: translateY(-50%); flex-direction: column; }
    .margin-input.right { top: 50%; right: 4px; transform: translateY(-50%); flex-direction: column; }

    .label-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        grid-template-rows: 1fr 1fr;
        gap: 8px;
        padding: 24px;
        height: 100%;
    }

    .label-box {
        background: #5a7ba9;
        border-radius: 2px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
        font-size: 0.75rem;
        font-weight: bold;
        position: relative;
        box-shadow: 1px 1px 3px rgba(0,0,0,0.2);
        transition: transform 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
    }

    .label-box::after {
        content: '';
        position: absolute;
        top: -8px; left: -10px; right: -10px; bottom: -8px;
        background: rgba(90, 123, 169, 0.2);
        border: 1px dashed rgba(90, 123, 169, 0.4);
        border-radius: 4px;
        z-index: -1;
        pointer-events: none;
    }

    .visual-editor.is-safe .label-box {
        transform: translate(20px, 15px);
    }

    .gap-x, .gap-y {
        position: absolute;
        display: flex;
        align-items: center;
        gap: 2px;
        font-size: 0.7rem;
        z-index: 10;
    }
    .gap-x { right: -28px; top: 25%; }
    .gap-y { bottom: -22px; left: 50%; transform: translateX(-50%); }
    .gap-x input, .gap-y input {
        width: 32px;
        padding: 2px;
        text-align: center;
        border: 1px solid #b45309;
        background: rgba(200,200,200,0.9);
        border-radius: 3px;
        font-size: 0.75rem;
        color: #78350f;
    }

    /* Type Grid */
    .type-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
        gap: 1rem;
    }

    .type-card {
        background: #252525;
        border: 2px solid #333;
        border-radius: 8px;
        padding: 1.25rem;
        cursor: pointer;
        transition: all 0.3s;
        text-align: center;
        color: inherit;
    }
    .type-card:hover {
        border-color: #5a7ba9;
        transform: translateY(-2px);
    }
    .type-card.active {
        border-color: #5a7ba9;
        background: rgba(90, 123, 169, 0.1);
    }
    .type-card h3 { color: #5a7ba9; font-size: 1rem; }
    .type-card p { color: #888; font-size: 0.85rem; margin: 0; }

    /* Rack selector */
    .rack-select select {
        width: 100%;
        padding: 0.8rem;
        background: #252525;
        border: 1px solid #444;
        border-radius: 6px;
        color: white;
        font-size: 1rem;
    }

    .rack-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 0.85rem;
    }
    .rack-table th, .rack-table td {
        padding: 0.75rem;
        text-align: left;
        border-bottom: 1px solid #333;
    }
    .rack-table th { color: #888; font-weight: 600; text-transform: uppercase; font-size: 0.75rem; }

    /* Styling Layout */
    .styling-layout {
        display: grid;
        grid-template-columns: 1.2fr 0.8fr;
        gap: 1.5rem;
    }
    @media (max-width: 768px) {
        .styling-layout { grid-template-columns: 1fr; }
    }

    .preview-container h3 { color: #888; font-size: 0.9rem; margin-bottom: 0.75rem; }

    .label-preview {
        width: 100%;
        max-width: 300px;
        background: #d1d1d1;
        border: 1px solid #999;
        position: relative;
        border-radius: 2px;
        min-height: 120px;
    }

    .pv-element {
        position: absolute;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: all 0.2s;
        border: 1px dashed #5a7ba9;
        background: rgba(90, 123, 169, 0.1);
        color: #5a7ba9;
        font-size: 10px;
    }
    .pv-element:hover { background: rgba(90, 123, 169, 0.2); }
    .pv-element.selected {
        border: 2px solid #5a7ba9;
        box-shadow: 0 0 8px rgba(90, 123, 169, 0.4);
        z-index: 10;
    }
    .pv-qr {
        background: rgba(255,255,255,0.6);
        border-style: solid;
        font-weight: bold;
    }
    .pv-text { font-family: monospace; font-weight: bold; white-space: nowrap; }
    .pv-text.serial { color: #666; }

    .preview-dims { color: #666; font-size: 0.8rem; margin-top: 0.5rem; text-align: center; }

    .styling-controls {
        background: #252525;
        padding: 1rem;
        border-radius: 8px;
        border: 1px solid #444;
    }

    .control-group { margin-bottom: 1rem; }
    .control-group label { display: block; color: #888; font-size: 0.8rem; margin-bottom: 0.25rem; }
    .control-group select, .control-group input {
        width: 100%;
        padding: 0.6rem;
        background: #111;
        border: 1px solid #444;
        border-radius: 4px;
        color: #fff;
    }

    .control-row { display: flex; gap: 0.75rem; margin-bottom: 1rem; }
    .control-item { flex: 1; }
    .control-item label { display: block; color: #888; font-size: 0.8rem; margin-bottom: 0.25rem; }
    .control-item input {
        width: 100%;
        padding: 0.6rem;
        background: #111;
        border: 1px solid #444;
        border-radius: 4px;
        color: #fff;
    }

    /* Buttons */
    .btn-sm {
        padding: 0.5rem 1rem;
        font-size: 0.85rem;
        background: #333;
        color: #ddd;
        border: none;
        border-radius: 4px;
        cursor: pointer;
    }
    .btn-sm:hover { background: #444; }

    .actions {
        display: flex;
        justify-content: center;
        margin-top: 1rem;
    }

    .btn {
        background: #5a7ba9;
        color: white;
        border: none;
        padding: 1rem 2rem;
        border-radius: 6px;
        font-weight: 600;
        cursor: pointer;
        font-size: 1rem;
        transition: background 0.2s;
    }
    .btn.large { padding: 1rem 3rem; font-size: 1.1rem; }
    .btn:hover:not(:disabled) { background: #4a6b99; }
    .btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>

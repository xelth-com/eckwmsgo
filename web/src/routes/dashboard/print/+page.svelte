<script>
    import { toastStore } from '$lib/stores/toastStore';

    let config = {
        type: 'i',
        startNumber: 1,
        count: 21,
        cols: 3,
        rows: 7,
        marginTop: 10,
        marginLeft: 10,
        gapX: 5,
        gapY: 5
    };

    let loading = false;

    async function generatePDF() {
        loading = true;
        try {
            // Need to fetch as blob for PDF download
            const token = localStorage.getItem('auth_token');
            const response = await fetch('/api/print/labels', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify(config)
            });

            if (!response.ok) throw new Error('Failed to generate PDF');

            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `labels_${config.type}_${config.startNumber}.pdf`;
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);

            toastStore.add('Labels generated successfully', 'success');
        } catch (e) {
            console.error(e);
            toastStore.add(e.message, 'error');
        } finally {
            loading = false;
        }
    }
</script>

<div class="print-page">
    <header>
        <h1>Printing Center</h1>
    </header>

    <div class="card">
        <div class="form-grid">
            <!-- Label Type -->
            <div class="field">
                <label>Label Type</label>
                <select bind:value={config.type}>
                    <option value="i">Item (i)</option>
                    <option value="b">Box (b)</option>
                    <option value="p">Place (p)</option>
                    <option value="l">Label/Marker (l)</option>
                </select>
            </div>

            <!-- Numbers -->
            <div class="field">
                <label>Start Number</label>
                <input type="number" bind:value={config.startNumber} min="1" />
            </div>
            <div class="field">
                <label>Count</label>
                <input type="number" bind:value={config.count} min="1" />
            </div>

            <!-- Layout -->
            <div class="field">
                <label>Columns</label>
                <input type="number" bind:value={config.cols} min="1" />
            </div>
            <div class="field">
                <label>Rows</label>
                <input type="number" bind:value={config.rows} min="1" />
            </div>

             <!-- Margins -->
             <div class="field">
                <label>Margin Top (mm)</label>
                <input type="number" bind:value={config.marginTop} min="0" step="0.1" />
            </div>
            <div class="field">
                <label>Margin Left (mm)</label>
                <input type="number" bind:value={config.marginLeft} min="0" step="0.1" />
            </div>
             <div class="field">
                <label>Gap X (mm)</label>
                <input type="number" bind:value={config.gapX} min="0" step="0.1" />
            </div>
            <div class="field">
                <label>Gap Y (mm)</label>
                <input type="number" bind:value={config.gapY} min="0" step="0.1" />
            </div>
        </div>

        <div class="actions">
            <button class="btn primary" on:click={generatePDF} disabled={loading}>
                {loading ? 'Generating...' : 'Download PDF'}
            </button>
        </div>
    </div>
</div>

<style>
    .print-page { max-width: 800px; margin: 0 auto; }
    header { margin-bottom: 2rem; }
    h1 { color: #fff; font-size: 1.8rem; margin: 0; }

    .card {
        background: #1e1e1e;
        border: 1px solid #333;
        border-radius: 8px;
        padding: 2rem;
    }

    .form-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
        gap: 1.5rem;
        margin-bottom: 2rem;
    }

    .field { display: flex; flex-direction: column; gap: 0.5rem; }
    label { color: #888; font-size: 0.85rem; font-weight: 500; }

    input, select {
        background: #121212;
        border: 1px solid #444;
        border-radius: 4px;
        padding: 0.8rem;
        color: #fff;
        font-size: 1rem;
    }

    input:focus, select:focus {
        border-color: #4a69bd;
        outline: none;
    }

    .actions { display: flex; justify-content: flex-end; }

    .btn {
        background: #4a69bd;
        color: white;
        border: none;
        padding: 1rem 2rem;
        border-radius: 4px;
        font-weight: 600;
        cursor: pointer;
        font-size: 1rem;
    }
    .btn:hover:not(:disabled) { background: #3d5aa8; }
    .btn:disabled { opacity: 0.6; cursor: not-allowed; }
</style>

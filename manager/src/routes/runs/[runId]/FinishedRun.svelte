<script lang="ts">
    import { bytesToSize, formatDurationFromNanoseconds, formatPercentFromPermille, titleCase } from "$lib/helper";
    import { authorizedFetch } from "$lib/auth.svelte";
    import { config } from "$lib/state.svelte";
    import { groupResultFiles, resultPathParam, sortResultFiles, type PreviewResponse } from "$lib/results";
    import type { ClassifiedResultFile, ResultFile } from "$lib/types/ResultFile";
    import type { RunResultSummary, RunState } from "$lib/types/RunState";
    import moment from "moment";

    interface $$Props {
        run: RunState;
        files: ResultFile[];
        initialTab?: 'overview' | 'outputs' | 'logs' | 'raw';
    }

    let { run, files, initialTab = 'overview' }: $$Props = $props();
    let activeTab = $state(initialTab);
    let showInternal = $state(false);
    let selectedPreviewFile = $state<ClassifiedResultFile | null>(null);
    let selectedPreview = $state<PreviewResponse | null>(null);
    let previewError = $state('');
    let previewLoading = $state(false);
    let logsLoaded = $state(false);
    let logPreview = $state<Record<string, PreviewResponse | null>>({});
    let logErrors = $state<Record<string, string>>({});
    let loadingLogs = $state(false);

    const sortedFiles = $derived(sortResultFiles(files));
    const summary = $derived.by((): RunResultSummary => {
        if (run.result_summary) {
            return run.result_summary;
        }
        return sortedFiles.reduce((acc, file) => {
            if (file.kind === 'artifact') acc.artifact_count += 1;
            if (file.kind === 'log') acc.log_count += 1;
            if (file.kind === 'metadata') acc.metadata_count += 1;
            if (file.kind === 'internal') acc.internal_count += 1;
            acc.total_size += file.size;
            return acc;
        }, {
            artifact_count: 0,
            log_count: 0,
            metadata_count: 0,
            internal_count: 0,
            total_size: 0
        } as RunResultSummary);
    });
    const outputFiles = $derived(showInternal
        ? sortedFiles.filter(file => file.kind !== 'log')
        : sortedFiles.filter(file => file.kind === 'artifact')
    );
    const outputGroups = $derived(groupResultFiles(outputFiles));
    const logFiles = $derived(
        sortedFiles
            .filter(file => file.kind === 'log')
            .sort((left, right) => {
                if (run.status === 'errored') {
                    if (left.name === 'STDERR.log') return -1;
                    if (right.name === 'STDERR.log') return 1;
                }
                return left.displayName.localeCompare(right.displayName);
            })
    );
    const metadataEntries = $derived(getMetadataEntries(run.gotap_metadata));
    const metadataMessages = $derived(extractMetadataMessages(run.gotap_metadata));
    const compactMetadataJson = $derived(run.gotap_metadata ? JSON.stringify(run.gotap_metadata, null, 2) : '');

    function setActiveTab(tab: 'overview' | 'outputs' | 'logs' | 'raw') {
        activeTab = tab;
        if (tab === 'logs') {
            void ensureLogsLoaded();
        }
    }

    function summarizeValue(value: unknown): string {
        if (value === null || value === undefined) return 'n/a';
        if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') return String(value);
        if (Array.isArray(value)) return value.length === 0 ? '[]' : `${value.length} item${value.length === 1 ? '' : 's'}`;
        if (typeof value === 'object') return `${Object.keys(value as Record<string, unknown>).length} field${Object.keys(value as Record<string, unknown>).length === 1 ? '' : 's'}`;
        return String(value);
    }

    function formatMetadataLabel(key: string): string {
        const normalizedKey = key.toUpperCase();

        if (normalizedKey.endsWith('_PERMILLE')) {
            return `${titleCase(key.replace(/_permille$/i, '').replaceAll('_', ' '))} (%)`;
        }
        if (normalizedKey.endsWith('_BYTES')) {
            return `${titleCase(key.replace(/_bytes$/i, '').replaceAll('_', ' '))}`;
        }
        if (normalizedKey.endsWith('_TIME')) {
            return `${titleCase(key.replace(/_time$/i, '').replaceAll('_', ' '))} Time`;
        }
        return titleCase(key.replaceAll('_', ' '));
    }

    function formatMetadataValue(key: string, value: unknown): string {
        const normalizedKey = key.toUpperCase();
        const numericValue = typeof value === 'number'
            ? value
            : typeof value === 'string' && value.trim() !== '' && !Number.isNaN(Number(value))
                ? Number(value)
                : null;

        if (normalizedKey.endsWith('_PERMILLE') && numericValue !== null) {
            return formatPercentFromPermille(numericValue);
        }

        if (normalizedKey.endsWith('_BYTES') && numericValue !== null) {
            return bytesToSize(numericValue);
        }

        if (normalizedKey.endsWith('_TIME') && numericValue !== null) {
            return formatDurationFromNanoseconds(numericValue);
        }

        return summarizeValue(value);
    }

    function getMetadataEntries(metadata: unknown): Array<[string, unknown]> {
        if (!metadata || typeof metadata !== 'object' || Array.isArray(metadata)) {
            return [];
        }
        return Object.entries(metadata as Record<string, unknown>);
    }

    function extractMetadataMessages(metadata: unknown): string[] {
        if (!metadata || typeof metadata !== 'object') {
            return [];
        }

        const entries = Object.entries(metadata as Record<string, unknown>);
        const values: string[] = [];
        for (const [key, value] of entries) {
            const normalizedKey = key.toLowerCase();
            if (!normalizedKey.includes('warning') && !normalizedKey.includes('message') && !normalizedKey.includes('error')) {
                continue;
            }
            if (typeof value === 'string' && value.trim()) {
                values.push(value.trim());
            } else if (Array.isArray(value)) {
                for (const item of value) {
                    if (typeof item === 'string' && item.trim()) {
                        values.push(item.trim());
                    }
                }
            }
        }
        return values;
    }

    async function downloadResult(file: ClassifiedResultFile) {
        const response = await authorizedFetch(`${config.apiServer}/runs/${run.id}/results/${resultPathParam(file.relPath)}`);
        if (!response.ok) {
            throw new Error(await response.text());
        }
        const blob = await response.blob();
        const url = URL.createObjectURL(blob);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = file.displayName;
        document.body.appendChild(anchor);
        anchor.click();
        URL.revokeObjectURL(url);
        document.body.removeChild(anchor);
    }

    async function fetchPreview(file: ClassifiedResultFile): Promise<PreviewResponse> {
        const response = await authorizedFetch(`${config.apiServer}/runs/${run.id}/results/${resultPathParam(file.relPath)}/preview`);
        if (!response.ok) {
            throw new Error(await response.text());
        }
        return await response.json() as PreviewResponse;
    }

    async function openPreview(file: ClassifiedResultFile) {
        selectedPreviewFile = file;
        selectedPreview = null;
        previewError = '';
        previewLoading = true;

        try {
            const payload = await fetchPreview(file);
            if (file.name.endsWith('.json')) {
                try {
                    payload.content = JSON.stringify(JSON.parse(payload.content), null, 2);
                } catch (_error) {
                    // Keep the original content if it is not valid JSON.
                }
            }
            selectedPreview = payload;
        } catch (error) {
            previewError = error instanceof Error ? error.message : 'Failed to load preview';
        } finally {
            previewLoading = false;
        }
    }

    async function ensureLogsLoaded() {
        if (logsLoaded || loadingLogs) {
            return;
        }
        loadingLogs = true;
        try {
            for (const logFile of logFiles) {
                try {
                    const payload = await fetchPreview(logFile);
                    logPreview = { ...logPreview, [logFile.relPath]: payload };
                    logErrors = { ...logErrors, [logFile.relPath]: '' };
                } catch (error) {
                    logErrors = {
                        ...logErrors,
                        [logFile.relPath]: error instanceof Error ? error.message : 'Failed to load log'
                    };
                    logPreview = { ...logPreview, [logFile.relPath]: null };
                }
            }
            logsLoaded = true;
        } finally {
            loadingLogs = false;
        }
    }
</script>

<div class="w-full">
    <div class="border-b border-gray-200">
        <nav class="-mb-px flex flex-wrap gap-2" aria-label="Tabs">
            {#each [
                { id: 'overview', label: 'Overview' },
                { id: 'outputs', label: 'Outputs' },
                { id: 'logs', label: 'Logs' },
                { id: 'raw', label: 'Raw' }
            ] as tab}
                <button
                    class={`min-w-24 py-2 px-3 text-center border-b-2 font-medium text-sm ${
                        activeTab === tab.id
                            ? 'border-blue-500 text-blue-600'
                            : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                    }`}
                    onclick={() => setActiveTab(tab.id as 'overview' | 'outputs' | 'logs' | 'raw')}
                >
                    {tab.label}
                </button>
            {/each}
        </nav>
    </div>

    <div class="mt-4">
        {#if activeTab === 'overview'}
            <div class="space-y-4">
                <div class="grid gap-3 md:grid-cols-4">
                    <div class="rounded-lg border border-gray-200 bg-white p-4">
                        <div class="text-xs uppercase text-gray-500">Status</div>
                        <div class="mt-2 text-lg font-semibold text-gray-900">{run.status}</div>
                    </div>
                    <div class="rounded-lg border border-gray-200 bg-white p-4">
                        <div class="text-xs uppercase text-gray-500">Artifacts</div>
                        <div class="mt-2 text-lg font-semibold text-gray-900">{summary.artifact_count}</div>
                    </div>
                    <div class="rounded-lg border border-gray-200 bg-white p-4">
                        <div class="text-xs uppercase text-gray-500">Output size</div>
                        <div class="mt-2 text-lg font-semibold text-gray-900">{bytesToSize(summary.total_size)}</div>
                    </div>
                    <div class="rounded-lg border border-gray-200 bg-white p-4">
                        <div class="text-xs uppercase text-gray-500">Finished</div>
                        <div class="mt-2 text-lg font-semibold text-gray-900">
                            {run.finished_at ? moment(run.finished_at).fromNow() : 'Not finished'}
                        </div>
                    </div>
                </div>

                <div class="rounded-lg border border-gray-200 bg-white p-4">
                    <div class="flex items-center justify-between">
                        <div>
                            <h2 class="text-lg font-semibold text-gray-900">gotap summary</h2>
                            <p class="mt-1 text-sm text-gray-600">Structured metadata from `_metadata.json`, rendered defensively.</p>
                        </div>
                        {#if run.gotap_metadata}
                            <span class="rounded-full bg-blue-100 px-2.5 py-1 text-xs font-medium text-blue-700">Metadata available</span>
                        {:else}
                            <span class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600">No metadata</span>
                        {/if}
                    </div>

                    {#if metadataMessages.length > 0}
                        <div class="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
                            <div class="font-medium">Messages</div>
                            <ul class="mt-2 list-disc pl-5">
                                {#each metadataMessages as message}
                                    <li>{message}</li>
                                {/each}
                            </ul>
                        </div>
                    {/if}

                    {#if metadataEntries.length > 0}
                        <div class="mt-4 grid gap-3 md:grid-cols-2">
                            {#each metadataEntries as [key, value]}
                                <div class="rounded-lg border border-gray-200 bg-gray-50 p-3">
                                    <div class="text-xs text-gray-500">{formatMetadataLabel(key)}</div>
                                    <div class="mt-2 text-sm font-medium text-gray-900 break-words">{formatMetadataValue(key, value)}</div>
                                </div>
                            {/each}
                        </div>
                    {:else}
                        <div class="mt-4 rounded-lg border border-dashed border-gray-300 p-4 text-sm text-gray-600">
                            No structured gotap metadata was found for this run. Outputs and logs are still available below.
                        </div>
                    {/if}

                    {#if compactMetadataJson}
                        <div class="mt-4">
                            <div class="text-sm font-medium text-gray-900">Compact JSON fallback</div>
                            <pre class="mt-2 max-h-72 overflow-auto rounded-lg bg-gray-900 p-3 text-xs text-gray-100">{compactMetadataJson}</pre>
                        </div>
                    {/if}
                </div>
            </div>
        {:else if activeTab === 'outputs'}
            <div class="space-y-4">
                <div class="flex items-center justify-between rounded-lg border border-gray-200 bg-white p-4">
                    <div>
                        <h2 class="text-lg font-semibold text-gray-900">Generated outputs</h2>
                        <p class="mt-1 text-sm text-gray-600">Artifacts are shown by default. Internal helper files stay available behind a toggle.</p>
                    </div>
                    <label class="flex items-center gap-2 text-sm text-gray-700">
                        <input type="checkbox" bind:checked={showInternal} class="rounded border-gray-300" />
                        Show internal files
                    </label>
                </div>

                {#if outputGroups.length === 0}
                    <div class="rounded-lg border border-dashed border-gray-300 p-4 text-sm text-gray-600">
                        No output files matched the current filter.
                    </div>
                {:else}
                    {#each outputGroups as group}
                        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white">
                            <div class="border-b border-gray-200 bg-gray-50 px-4 py-3 text-sm font-medium text-gray-900">
                                {group.path === '.' ? 'root output folder' : group.path}
                            </div>
                            <div class="overflow-x-auto">
                                <table class="w-full text-sm text-left">
                                    <thead class="text-xs uppercase bg-gray-50 text-gray-500">
                                        <tr>
                                            <th scope="col" class="px-4 py-3">File</th>
                                            <th scope="col" class="px-4 py-3">Relative path</th>
                                            <th scope="col" class="px-4 py-3">Size</th>
                                            <th scope="col" class="px-4 py-3">Modified</th>
                                            <th scope="col" class="px-4 py-3">Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {#each group.files as file}
                                            <tr class="border-t border-gray-200">
                                                <td class="px-4 py-3">
                                                    <div class="font-medium text-gray-900">{file.displayName}</div>
                                                    <div class="mt-1 text-xs text-gray-500 uppercase">{file.kind}</div>
                                                </td>
                                                <td class="px-4 py-3 font-mono text-xs text-gray-600">{file.relPath}</td>
                                                <td class="px-4 py-3 text-gray-700">{bytesToSize(file.size)}</td>
                                                <td class="px-4 py-3 text-gray-700">
                                                    {file.lastModified ? moment(file.lastModified).fromNow() : 'Unknown'}
                                                </td>
                                                <td class="px-4 py-3">
                                                    <div class="flex gap-3">
                                                        {#if file.previewable}
                                                            <button class="text-blue-600 hover:text-blue-800 hover:cursor-pointer" onclick={() => openPreview(file)}>
                                                                Preview
                                                            </button>
                                                        {/if}
                                                        <button class="text-gray-700 hover:text-gray-900 hover:cursor-pointer" onclick={() => downloadResult(file)}>
                                                            Download
                                                        </button>
                                                    </div>
                                                </td>
                                            </tr>
                                        {/each}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    {/each}
                {/if}
            </div>
        {:else if activeTab === 'logs'}
            <div class="space-y-4">
                {#if logFiles.length === 0}
                    <div class="rounded-lg border border-dashed border-gray-300 p-4 text-sm text-gray-600">
                        No `STDOUT.log` or `STDERR.log` files were found for this run.
                    </div>
                {:else}
                    {#if loadingLogs}
                        <div class="rounded-lg border border-gray-200 bg-white p-4 text-sm text-gray-600">
                            Loading logs...
                        </div>
                    {/if}
                    {#each logFiles as file}
                        <div class="rounded-lg border border-gray-200 bg-white">
                            <div class="flex items-center justify-between border-b border-gray-200 px-4 py-3">
                                <div>
                                    <div class="text-sm font-semibold text-gray-900">{file.displayName}</div>
                                    <div class="text-xs text-gray-500">{file.relPath}</div>
                                </div>
                                <button class="text-sm text-gray-700 hover:text-gray-900 hover:cursor-pointer" onclick={() => downloadResult(file)}>
                                    Download
                                </button>
                            </div>
                            {#if logErrors[file.relPath]}
                                <div class="p-4 text-sm text-red-600">{logErrors[file.relPath]}</div>
                            {:else if logPreview[file.relPath]?.content}
                                <div>
                                    {#if logPreview[file.relPath]?.truncated}
                                        <div class="border-b border-amber-200 bg-amber-50 px-4 py-2 text-xs text-amber-900">
                                            Preview truncated to keep the UI responsive. Download the file for the full log.
                                        </div>
                                    {/if}
                                    <pre class="max-h-[28rem] overflow-auto bg-gray-950 p-4 text-xs text-gray-100">{logPreview[file.relPath]?.content}</pre>
                                </div>
                            {:else}
                                <div class="p-4 text-sm text-gray-600">Log file is empty.</div>
                            {/if}
                        </div>
                    {/each}
                {/if}
            </div>
        {:else if activeTab === 'raw'}
            <div class="rounded-lg border border-gray-200 bg-white p-4">
                <pre class="max-h-[32rem] overflow-auto rounded-lg bg-gray-950 p-4 text-xs text-gray-100">{JSON.stringify(run, null, 2)}</pre>
            </div>
        {/if}
    </div>

    {#if selectedPreviewFile}
        <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4">
            <div class="max-h-[90vh] w-full max-w-5xl overflow-hidden rounded-xl bg-white shadow-2xl">
                <div class="flex items-center justify-between border-b border-gray-200 px-5 py-4">
                    <div>
                        <div class="text-lg font-semibold text-gray-900">{selectedPreviewFile.displayName}</div>
                        <div class="text-xs text-gray-500">{selectedPreviewFile.relPath}</div>
                    </div>
                    <button class="text-sm text-gray-600 hover:text-gray-900 hover:cursor-pointer" onclick={() => {
                        selectedPreviewFile = null;
                        selectedPreview = null;
                        previewError = '';
                    }}>
                        Close
                    </button>
                </div>

                <div class="p-5">
                    {#if previewLoading}
                        <div class="text-sm text-gray-600">Loading preview...</div>
                    {:else if previewError}
                        <div class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{previewError}</div>
                    {:else if selectedPreview}
                        <div class="space-y-3">
                            <div class="flex items-center justify-between text-xs text-gray-500">
                                <div>{selectedPreview.mimeType} · {selectedPreview.encoding}</div>
                                <button class="text-sm text-gray-700 hover:text-gray-900 hover:cursor-pointer" onclick={() => downloadResult(selectedPreviewFile!)}>
                                    Download file
                                </button>
                            </div>
                            {#if selectedPreview.truncated}
                                <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
                                    Preview truncated to the first 64 KB. Download the file to inspect the full contents.
                                </div>
                            {/if}
                            <pre class="max-h-[60vh] overflow-auto rounded-lg bg-gray-950 p-4 text-xs text-gray-100">{selectedPreview.content}</pre>
                        </div>
                    {/if}
                </div>
            </div>
        </div>
    {/if}
</div>

<script lang="ts">
    import moment from 'moment';
    import type { RunState } from "$lib/types/RunState";
    import { page } from '$app/state';
    import { goto, invalidateAll } from '$app/navigation';
    import { config } from '$lib/state.svelte.js';
    import { authorizedFetch } from '$lib/auth.svelte';
    import { bytesToSize } from '$lib/helper';

    let {data} = $props();

    let runs: RunState[] = $state(data.runs); 
    $effect(() => {
        runs = data.runs
    });
    $inspect(runs);

    // refresh 
    async function refresh() {
        invalidateAll();
    }

    // Define available status options
    let status: RunState["status"] | "all" = $state(page.url.searchParams.get('status') || 'all') as RunState["status"] | "all"; 
    const statusOptions = ['all', 'pending', 'running', 'finished', 'errored'] as const; 
    $inspect(status)

    // Handle status change
    async function handleStatusChange(event: Event) {
        const target = event.target as HTMLSelectElement;
        status = target.value as typeof status;

        const params = new URLSearchParams(page.url.searchParams);
        params.set('status', status);
        await goto(`?${params.toString()}`);
    }

    async function onStart(runId: number) {
        const runUrl = `${config.apiServer}/runs/${runId}/start`;
        const res = await authorizedFetch(runUrl, { 
            method: 'POST'
        });
        const data = await res.json();
        $inspect(data);
        await refresh();
    }

    async function onDelete(runId: number) {
        const runUrl = `${config.apiServer}/runs/${runId}`;
        const res = await authorizedFetch(runUrl, { 
            method: 'DELETE'
        });
        const data = await res.json();
        console.log(data.message);

        await refresh();
    }
</script>

<button onclick={refresh} class="text-blue-600 hover:text-blue-800 hover:cursor-pointer" title="Refresh" aria-label="Refresh">
    Refresh
</button>
<div class="relative overflow-x-auto shadow-md sm:rounded-lg">
    <table class="w-full text-sm text-left">
        <thead class="text-xs uppercase bg-gray-100">
            <tr>
                <th scope="col" class="px-6 py-3 font-semibold">
                    Title
                </th>
                <th scope="col" class="px-6 py-3 font-semibold">
                    <select 
                        class="text-xs uppercase bg-gray-100 font-semibold outline-none cursor-pointer"
                        bind:value={status}
                        onchange={handleStatusChange}
                    >
                        {#each statusOptions as option}
                            <option value={option}>
                                Status: {option}
                            </option>
                        {/each}
                    </select>
                </th>
                <th scope="col" class="px-6 py-3 font-semibold">
                    Created
                </th>
                <th scope="col" class="px-6 py-3 font-semibold">
                    Finished
                </th>
                <th scope="col" class="px-6 py-3 font-semibold">
                    Outputs
                </th>
                <th scope="col" class="px-6 py-3 font-semibold">
                    Actions
                </th>
            </tr>
        </thead>
        <tbody>
            {#each runs as run}
                <tr class="bg-white border-b border-b-gray-300 hover:bg-gray-50">
                    <td class="px-6 py-4">
                        <a href="/manager/runs/{run.id}" class="font-medium text-blue-600 hover:underline">{run.title}</a>
                    </td>
                    <td class="px-6 py-4">
                        {run.status}
                    </td>
                    <td class="px-6 py-4">
                        {moment(run.created_at).fromNow()}
                    </td>
                    <td class="px-6 py-4">
                        {run.status === 'finished' ? moment(run.finished_at).fromNow() : null}
                        {run.status === 'errored' ? 'Errored' : null}
                    </td>
                    <td class="px-6 py-4">
                        {#if run.result_summary}
                            <div class="flex flex-wrap gap-2">
                                <span class="rounded-full bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700">
                                    {run.result_summary.artifact_count} artifacts
                                </span>
                                {#if run.result_summary.log_count > 0}
                                    <span class="rounded-full bg-amber-100 px-2 py-1 text-xs font-medium text-amber-700">
                                        {run.result_summary.log_count} logs
                                    </span>
                                {/if}
                                {#if run.gotap_metadata}
                                    <span class="rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-700">
                                        gotap
                                    </span>
                                {/if}
                                {#if run.result_summary.total_size > 0}
                                    <span class="rounded-full bg-emerald-100 px-2 py-1 text-xs font-medium text-emerald-700">
                                        {bytesToSize(run.result_summary.total_size)}
                                    </span>
                                {/if}
                            </div>
                        {:else}
                            <span class="text-xs text-gray-500">No output summary</span>
                        {/if}
                    </td>
                    <td class="px-6 py-4 flex gap-2">
                        {#if run.status === 'running'}
                            Running...
                        {/if}
                        {#if run.status === 'pending'}
                        <button 
                            onclick={() => onStart(run.id)}
                            class="text-green-600 hover:text-green-800 hover:cursor-pointer" 
                            title="Start" 
                            aria-label="Start Run"
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                        </button>
                        {/if}
                        {#if run.status === 'finished'}
                            <a
                                href={`/manager/runs/${run.id}?tab=outputs`}
                                class="text-blue-600 hover:text-blue-800"
                                title="Open results"
                                aria-label="Open Result"
                            >
                                Open results
                            </a>
                        {/if}
                        {#if run.status === 'errored' && (run.result_summary?.log_count ?? 0) > 0}
                            <a
                                href={`/manager/runs/${run.id}?tab=logs`}
                                class="text-amber-700 hover:text-amber-900"
                                title="View error logs"
                                aria-label="View error logs"
                            >
                                View error
                            </a>
                        {/if}
                        {#if run.status !== 'running'}
                        <button 
                            class="text-red-700  hover:cursor-pointer" 
                            title="Delete" 
                            aria-label="Delete Run"
                            onclick={() => onDelete(run.id)}
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                        </button>
                        {/if}
                    </td>
                </tr>
            {/each}
        </tbody>
    </table>
</div>

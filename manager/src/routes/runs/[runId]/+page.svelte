<script lang="ts">
    import { page } from "$app/state";
    import moment from "moment";
    import type { PageProps } from "./$types";
    import { bytesToSize } from "$lib/helper";
    import FinishedRun from "./FinishedRun.svelte";

    let { data }: PageProps = $props();
    let run = data.run;
    let files = data.files;
    let initialTab = $derived((page.url.searchParams.get('tab') ?? 'overview') as 'overview' | 'outputs' | 'logs' | 'raw');
    $inspect(data);

</script>

{#if run}
<div>
    <h1 class="text-2xl font-bold text-gray-900">{run.title}</h1>
    <p class="mt-2 text-gray-600">{run.description}</p>
    <div class="my-4 p-3 rounded-lg border border-gray-200">
        <table class="w-full text-sm">
            <tbody>
                <tr class="bg-gray-50">
                    <td class="p-2 font-semibold">Status</td>
                    <td class="p-2">{run.status}</td>
                </tr>
                <tr>
                    <td class="p-2 font-semibold">Has Error</td>
                    <td class="p-2">{run.has_errored ? 'Yes' : 'No'}</td>
                </tr>
                <tr class="bg-gray-50">
                    <td class="p-2 font-semibold">Run ID</td>
                    <td class="p-2">{run.id}</td>
                </tr>
            </tbody>
        </table>
    </div>

    {#if run.status === 'running'}
        <div class="mt-2 text-sm text-gray-600">
            Running since {moment(run.started_at).fromNow()}
        </div>
    {:else if run.status === 'finished'}
        <div class="flex flex-row justify-between mt-2 text-sm text-gray-600">
            <span>Finished {moment(run.finished_at).fromNow()}</span>
            <span>{files.length} results ({bytesToSize(files.map(f => f.size).reduce((a, b) => a + b, 0))})</span> 
        </div>
        <FinishedRun {run} {files} {initialTab} />
    {:else if run.status === 'errored'}
        <div class="flex flex-row justify-between mt-2 text-sm text-gray-600">
            <span>Errored {moment(run.finished_at).fromNow()}</span>
            <span>{files.length} captured files ({bytesToSize(files.map(f => f.size).reduce((a, b) => a + b, 0))})</span>
        </div>
        <p class="mt-2 text-sm text-red-500">{run.error_message}</p>
        <div class="mt-4">
            <FinishedRun {run} {files} {initialTab} />
        </div>
    {:else if run.status === 'pending'}
        <button 
            disabled
            class="w-full px-3 py-2 bg-green-500 text-white rounded-lg shadow-md hover:bg-green-600 transition-colors cursor-pointer" 
            onclick={() => console.log('start')}
        >
            Start
        </button>
    {/if}
</div>
{:else}
<div class="flex flex-col items-center justify-center">
    <div class="text-lg font-bold text-gray-900">No run found. Try to refresh the page, this should not happen.</div>
</div>
{/if}

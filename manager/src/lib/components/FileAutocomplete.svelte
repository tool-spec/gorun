<script lang="ts">
    import { authorizedFetch } from "$lib/auth.svelte";
    import { bytesToSize } from "$lib/helper";
import { config } from "$lib/state.svelte";
    import type { ResultFile } from "$lib/types/ResultFile";
    import type { RemoteFile } from "$lib/types/TempFile";
    import moment from "moment";

    interface $$Props {
        onselect: (file: RemoteFile) => void;
    }
    let { onselect }: $$Props = $props();


    let suggestions: ResultFile[] = $state([]);
    $inspect(suggestions);

    let pattern = $state('');
    let loading = $state(false); 
    let isOpen = $derived(suggestions.length > 0);
    let selectedExt = $state('.*');
    let q = $derived(`${pattern}*${selectedExt}`);
    
    // Add more extensions as needed
    const extensions = ['.*', '.txt', '.csv', '.json', '.nc', '.tif', '.html'];

    async function query() {
        if (pattern.length !== 0 && !loading) {
            loading = true;
            const res = await authorizedFetch(`${config.apiServer}/files?pattern=${q}&target=both`);
            const data = await res.json();

            // sort by last modified
            suggestions = (data.files as ResultFile[]).sort((prev, next) => prev.lastModified! > next.lastModified! ? -1 : 1);
            //suggestions = data.files;
            loading = false;
        }
    }

    function clear() {
        loading = false;
        pattern = '';
        suggestions = [];
    }

    function handleSelect(file: ResultFile) {
        onselect({
            path: file.absPath,
            name: file.name,
            size: file.size
        })
    }
</script>

<div class="relative w-full">
    <div class="relative flex">
        <input
            type="text"
            class="w-full px-4 py-2 border border-gray-300 rounded-l-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            bind:value={pattern}
            oninput={query}
            placeholder="Search for existing files..."
        />
        <select
            bind:value={selectedExt}
            onchange={query}
            class="px-3 py-2 bg-gray-50 border border-l-0 border-gray-300 rounded-r-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        >
            {#each extensions as ext}
                <option value={ext}>{ext}</option>
            {/each}
        </select>
    </div>
    
    {#if loading}
        <div class="absolute right-[4.5rem] top-3">
            <div class="animate-spin h-4 w-4 border-2 border-blue-500 border-t-transparent rounded-full"></div>
        </div>
    {:else }
    <button
        type="button"
        class="absolute right-[4.5rem] top-3 mr-2 text-gray-800 hover:text-gray-600 hover:cursor-pointer" 
        onclick={clear}
    >
        X
    </button>
    {/if}
    
    {#if isOpen}
        <div class="absolute z-10 w-full mt-1 bg-white border border-gray-300 rounded-lg shadow-lg max-h-60 overflow-y-auto">
            {#each suggestions as suggestion}
                <button
                    class="w-full text-left px-4 py-2 hover:bg-gray-100 focus:bg-gray-100 focus:outline-none"
                    onclick={() => handleSelect(suggestion)}
                >
                <p class="text-sm font-medium text-gray-500">{moment(suggestion.lastModified).fromNow()}</p>
                <h3 class="text-sm font-medium text-gray-900">{suggestion.name} ({bytesToSize(suggestion.size)})</h3>
                <p class="text-sm text-gray-500">{suggestion.absPath}</p>
                </button>
            {/each}
        </div>
    {/if}
</div>

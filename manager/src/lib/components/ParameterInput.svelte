<script lang="ts">
    import type { ParameterSpec } from "$lib/types/ToolSpec";
    import StructEditor from "./StructEditor.svelte";
    import ArrayInput from "./ArrayInput.svelte";

    interface ParameterProps {
        parameter: ParameterSpec,
        name: string,
        value?: string | number | boolean | Date | null | {[key: string]: any} | any[],
        oninput: (value: string | number | boolean | Date | null | {[key: string]: any} | any[]) => void,
    }
    let {parameter, name, value, oninput}: ParameterProps = $props()

//    let value: string | number | boolean | Date | null | {[key: string]: any} | any[] = $state(parameter.default ? parameter.default : null);
    let showDescription = $state(false);

    function getNumericValue(input: HTMLInputElement): number | null {
        if (input.value === '') {
            return null;
        }

        if (parameter.type === 'integer') {
            return Number.parseInt(input.value, 10);
        }

        return Number.parseFloat(input.value);
    }

</script>

<style>
    .animate-fadeIn {
        animation: fadeIn 0.2s ease-in-out;
    }
    
    @keyframes fadeIn {
        from { opacity: 0; transform: translateY(-5px); }
        to { opacity: 1; transform: translateY(0); }
    }
</style>

<div class="hover:border-b-blue-800">
    <div class="flex items-center gap-4 py-2 group">
        <div class="w-1/4">
            <span class="font-medium text-gray-700 flex items-center gap-2">
                {name}
                {#if parameter.description}
                    <button 
                        aria-label="Show Description"
                        class="text-gray-400 hover:text-gray-600 cursor-pointer"
                        onclick={() => showDescription = !showDescription}
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
                        </svg>
                    </button>
                {/if}
            </span>
        </div>

        <div class="w-3/4">
            {#if parameter.array}
                <ArrayInput 
                    {parameter} 
                    value={value as any}
                    oninput={v => oninput(v)} 
                />
            {:else if parameter.type === 'string'}
                <input 
                    type="text" 
                    bind:value 
                    oninput={e => oninput((e.target as HTMLInputElement)!.value)}
                    class="w-full px-3 py-1.5 border border-gray-200 rounded-md shadow-sm focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
                >
            {:else if parameter.type === 'integer' || parameter.type === 'float'}
                <input 
                    type="number" 
                    bind:value 
                    oninput={e => oninput(getNumericValue((e.target as HTMLInputElement)!))}
                    class="w-full px-3 py-1.5 border border-gray-200 rounded-md shadow-sm focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
                >
            {:else if parameter.type === 'boolean'}
                <label class="inline-flex items-center cursor-pointer">
                    <input 
                        type="checkbox" 
                        bind:checked={value as boolean}
                        oninput={e => oninput((e.target as HTMLInputElement)!.checked)}
                        class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-offset-0 focus:ring-indigo-200 focus:ring-opacity-50"
                    >
                </label>
            {:else if parameter.type === 'enum'}
                <select 
                    bind:value 
                    oninput={e => oninput((e.target as HTMLSelectElement)!.value)}
                    class="w-full px-3 py-1.5 border border-gray-200 rounded-md shadow-sm focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
                >
                    {#each parameter.values! as value}
                        <option value={value}>{value}</option>
                    {/each}
                </select>
            {:else if parameter.type === 'datetime'}
                <input 
                    bind:value 
                    type="datetime-local" 
                    oninput={e => oninput((e.target as HTMLInputElement)!.value)}
                    class="w-full px-3 py-1.5 border border-gray-200 rounded-md shadow-sm focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
                >
            {:else if parameter.type === 'struct'}
                <StructEditor 
                    value={value as {[key: string]: any} || {key: 'your value'}} 
                    oninput={v => oninput(v)}
                />
                {#if !parameter.optional && (value === null || value === undefined || value === '')}
                    <span class="text-red-500 text-sm ml-2">Required</span>
                {/if}
            {/if}
        </div>
    </div>

    {#if showDescription && parameter.description}
        <div class=" mt-1 text-sm text-gray-600 bg-gray-50 p-2 rounded-md animate-fadeIn">
            {parameter.description}
        </div>
    {/if}
</div>

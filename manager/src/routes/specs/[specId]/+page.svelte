<script lang="ts">
    import { authorizedFetch } from "$lib/auth.svelte";
    import DataInput from "$lib/components/DataInput.svelte";
    import ParameterInput from "$lib/components/ParameterInput.svelte";
    import type { RemoteFile } from "$lib/types/TempFile";
    import { config } from "$lib/state.svelte";
    import type { PageProps } from "./$types";
    import { goto } from "$app/navigation";
    import ClInfo from "./CLInfo.svelte";
    import type { ToolSpec } from "$lib/types/ToolSpec";
    import CitationInfo from "./CitationInfo.svelte";

    let { data }: PageProps = $props();
    let spec: ToolSpec = $state(data.spec!);

    let parameterValues: {[name: string]: any} = $state({});
    let dataValues: {[name: string]: RemoteFile} = $state({});
    $inspect(parameterValues);
    $inspect(dataValues);

    let currentTab: 'parameters' | 'cli' | 'citation' = $state('parameters');
    let submitError = $state('');
    let submitValidationErrors = $state<string[]>([]);
    let submitPending = $state(false);

    function updateParameterValues(name: string, value: any) {
        parameterValues = {...parameterValues, [name]: value};
    }

    let parameterAreValid = $derived(
        !spec.parameters ||
        Object.keys(spec.parameters)
        .map(name => parameterValues[name] !== null && parameterValues[name] !== undefined && parameterValues[name] !== '')
        .reduce((a, b) => a && b, true)
    );

    let dataAreValid = $derived(
        !spec.data ||
        Object.keys(spec.data)
        .map(name => dataValues[name] !== null && dataValues[name] !== undefined)
        .reduce((a, b) => a && b, true)
    );

    let allValid = $derived(parameterAreValid && dataAreValid);

    let dockerImage = $derived(spec.id.split('::')[0]);
    let toolName = $derived(spec.id.split('::')[1]);

    async function startRun() {
        submitError = '';
        submitValidationErrors = [];
        const [dockerImage, toolName, ...o] = spec.id.split('::');
        if (!dockerImage || !toolName || spec.name !== toolName || o.length > 0) {
            console.error(`Invalid tool slug: ${spec.id}`) 
            submitError = `Invalid tool slug: ${spec.id}`;
            return 
        }

        const payload = ({
            name: toolName,
            docker_image: dockerImage,
            parameters: {...parameterValues},
            data: Object.fromEntries(Object.entries(dataValues).map(([name, conf]) => ([name, conf.path])))
        })

        submitPending = true;
        try {
            const response = await authorizedFetch(`${config.apiServer}/runs`, {
                method: 'POST',
                headers: {
                    'Accept': 'application/json',
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            const responseText = await response.text();
            let responseBody: Record<string, unknown> | null = null;
            if (responseText) {
                try {
                    responseBody = JSON.parse(responseText) as Record<string, unknown>;
                } catch (_error) {
                    responseBody = null;
                }
            }

            if (!response.ok) {
                const message = typeof responseBody?.message === 'string'
                    ? responseBody.message
                    : `Failed to create run (${response.status})`;
                submitError = message;
                if (Array.isArray(responseBody?.errors)) {
                    submitValidationErrors = responseBody.errors
                        .map((entry) => typeof entry?.message === 'string' ? entry.message : JSON.stringify(entry))
                        .filter((entry): entry is string => Boolean(entry));
                }
                console.error("Failed to create run", response.status, responseBody ?? responseText);
                return;
            }

            const runId = responseBody?.id;
            if (typeof runId === 'number') {
                goto(`/manager/runs/${runId}`);
                return;
            }

            submitError = 'Run was created, but the server did not return a valid run id.';
            console.error("Unexpected create run response", responseBody ?? responseText);
        } catch (error) {
            submitError = error instanceof Error ? error.message : 'Failed to create run';
            console.error("Failed to create run", error);
        } finally {
            submitPending = false;
        }
    }
</script>


<div>
    <h1 class="text-2xl font-bold text-gray-900">{spec.title}</h1>
    <h4 class="text-md font-semibold text-gray-500">ID: {spec.id}</h4>
    
    <p class="mt-2 text-gray-600">{spec.description}</p>

    <div class="flex mt-4 ml-6">
        <button 
        class="px-4 pt-2 text-sm font-medium border-b-1 transition-colors duration-200" 
        onclick={() => currentTab = 'parameters'}
        class:border-indigo-500={currentTab === 'parameters'}
        class:text-indigo-600={currentTab === 'parameters'}
        class:border-transparent={currentTab !== 'parameters'}
        class:text-gray-500={currentTab !== 'parameters'}
        class:hover:text-gray-700={currentTab !== 'parameters'}
        class:hover:border-gray-300={currentTab !== 'parameters'}
        >
        Parameters
    </button>
    <button
        class="px-4 pt-2 text-sm font-medium border-b-1 transition-colors duration-200"
        onclick={() => currentTab = 'cli'}
        class:border-indigo-500={currentTab === 'cli'}
        class:text-indigo-600={currentTab === 'cli'}
        class:border-transparent={currentTab !== 'cli'}
        class:text-gray-500={currentTab !== 'cli'}
        class:hover:text-gray-700={currentTab !== 'cli'}
        class:hover:border-gray-300={currentTab !== 'cli'}
    >
        API Access</button>
    <button
        class="px-4 pt-2 text-sm font-medium border-b-1 transition-colors duration-200"
        onclick={() => currentTab = 'citation'}
        class:border-indigo-500={currentTab === 'citation'}
        class:text-indigo-600={currentTab === 'citation'}
        class:border-transparent={currentTab !== 'citation'}
        class:text-gray-500={currentTab !== 'citation'}
        class:hover:text-gray-700={currentTab !== 'citation'}
        class:hover:border-gray-300={currentTab !== 'citation'}
        disabled={!spec.citation}
    >Citation</button>
    </div>

    {#if currentTab === 'parameters'}
        <div class="p-3 rounded-lg border border-gray-200 shadow-md mb-6">
                <h2 class="text-lg font-semibold text-gray-900 mb-3">Parameters</h2>
                {#if spec.parameters}    
                    {#each Object.entries(spec.parameters) as [name, parameter]}
                    <ParameterInput {parameter} {name} value={parameterValues[name]} oninput={value => updateParameterValues(name, value)} />
                {/each}
            {:else}
                    <p class="mt-2 text-gray-600">Tool {spec.title} has no parameters defined.</p>
            {/if}
            {#if parameterAreValid}
                <p class="mt-2 text-green-500">All parameters are valid</p>
            {:else}
                <p class="mt-2 text-red-600">Some required parameters are not yet set.</p>
            {/if}
        </div>

        <div class="p-3 rounded-lg border border-gray-200 shadow-md my-6">
            <h2 class="text-lg font-semibold text-gray-900 mb-3">Data</h2>
            {#if spec.data}
                {#each Object.entries(spec.data) as [name, dataSpec]}
                    <DataInput {name} data={dataSpec} onupload={f => f ? dataValues[name] = {...f} : delete dataValues[name]} /> 
                {/each}
                {#if dataAreValid}
                        <p class="mt-2 text-green-500">All data is valid</p>
                {:else}
                        <p class="mt-2 text-red-600">Some required data is not yet set.</p>
                {/if}
            {:else}
                <p class="mt-2 text-gray-600">Tool {spec.title} does not require any data</p>
            {/if}
        </div>
    {:else if currentTab === 'cli'}
        <ClInfo id={spec.id} name={toolName} image={dockerImage} {parameterValues} {dataValues} />
    {:else if currentTab === 'citation' && spec.citation}
        <CitationInfo citation={spec.citation} />
    {/if}

</div>
{#if allValid}
{#if submitError}
<div class="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
    <div>{submitError}</div>
    {#if submitValidationErrors.length > 0}
    <ul class="mt-2 list-disc pl-5">
        {#each submitValidationErrors as validationError}
        <li>{validationError}</li>
        {/each}
    </ul>
    {/if}
</div>
{/if}
<button class="w-full px-3 py-2 bg-green-500 text-white rounded-lg shadow-md hover:bg-green-600 transition-colors cursor-pointer disabled:cursor-not-allowed disabled:bg-green-300" onclick={startRun} disabled={submitPending}>
    {submitPending ? 'Creating...' : 'Create'}
</button>
{/if}

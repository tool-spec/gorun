<script lang="ts">
    import { authorizedFetch } from "$lib/auth.svelte";
    import { config } from "$lib/state.svelte";
    import type { ResultFile } from "$lib/types/ResultFile";
    import type { RemoteFile, TempFile } from "$lib/types/TempFile";
    import type { DataSpec } from "$lib/types/ToolSpec";
    import FileAutocomplete from "./FileAutocomplete.svelte";

    interface $$Props {
        data: DataSpec;
        name: string;
        onupload: (path: RemoteFile | null) => void;
    }

    let { data, name, onupload }: $$Props = $props();
    let dragging = $state(false);
    let processing = $state(false);
    let hasError: string | null = $state(null);
    //let file: File | null = $state(null);
    let tempFile: RemoteFile| null = $state(null);
    $inspect(tempFile);

    function handleDrop(e: DragEvent) {
        e.preventDefault();
        dragging = false;

        let file: File | null = null;
        if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
            file = e.dataTransfer?.files[0];
        } else {
            return;
        }
        return handleUpload(file);
    }

    function handleFileInput(e: Event) {
        const files = (e.target as HTMLInputElement).files;
        if (files && files.length > 0) {
            return handleUpload(files[0]);
        }
    }

    function handleUpload(file: File) {
        hasError = null;
        if (file && data.extension.length > 0) {
            const allowed = data.extension.map(ext => file?.name.endsWith(ext)).reduce((a, b) => a || b, false) 
            if (!allowed) {
                hasError = `The input data ${name} must have a file extension of ${data.extension.join(', ')}`;
                return;
            }
        }

        processing = true;
        //  POST the file to the server using multipart formdata
        const formData = new FormData()
        formData.append('file', file)
        authorizedFetch(`${config.apiServer}/files`, {
            method: 'POST',
            body: formData
        })
        .then(res => res.json())
        .then(data => {
            processing = false;
            tempFile = data;
            if (!tempFile) return;
            onupload({
                path: tempFile.path,
                name: tempFile.name,
                size: tempFile.size
            });
        })
        .catch(error => {
            console.error(error);
            hasError = error.message;
            processing = false;
        })
    }

    function remove() {
        hasError = null;
        tempFile = null;
        onupload(null)
    }

    function handleAutocompleteSelect(file: RemoteFile) {
        tempFile = {
            path: file.path,
            name: file.name,
            size: file.size
            }
            onupload(tempFile);
    }
</script>


{#if tempFile} 
    <div class="mt-2 flex flex-row justify-between">
        <div class="text-sm text-gray-600">
            Selected file: {tempFile.name} ({tempFile.size} bytes)
        </div>
        <button type="button" class="text-sm text-gray-500 hover:cursor-pointer" onclick={remove}>X</button>
    </div>

{:else}
    <div
        class="w-full"
        role="button"
        tabindex="0"
        ondragenter={e => {e.preventDefault(); dragging = true}}
        ondragleave={e => {e.preventDefault(); dragging = false}}
        ondragover={e => e.preventDefault()}
        ondrop={handleDrop}
    >
        <div
            class={`
                flex flex-col items-center justify-center w-full 
                border-2 border-dashed rounded-lg
                transition-colors duration-200
                ${dragging ? 'border-indigo-500 bg-indigo-50' : 'border-gray-300'}
            `}
        >
            <div class="flex flex-col items-center justify-center pt-5 pb-6">
                <svg class="w-8 h-8 mb-4 text-gray-500" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 20 16">
                    <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 13h3a3 3 0 0 0 0-6h-.025A5.56 5.56 0 0 0 16 6.5 5.5 5.5 0 0 0 5.207 5.021C5.137 5.017 5.071 5 5 5a4 4 0 0 0 0 8h2.167M10 15V6m0 0L8 8m2-2 2 2"/>
                </svg>
                <div class="mb-2 flex flex-row items-center gap-2">
                    <div class="text-sm text-gray-500 font-bold">Dataset: {name}</div>
                    <div class="text-sm text-gray-500">drop your file here</div>
                </div>
                <div class="flex flex-row items-center gap-2 w-full mx-6">
                <label class="inline-flex items-center px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-md cursor-pointer">
                    Click to upload
                    <input 
                        type="file" 
                        class="hidden" 
                        accept={data.extension.join(', ')}
                        onchange={handleFileInput}
                    />
                </label>
                <div class="text-sm text-gray-500"> or </div>
                <FileAutocomplete onselect={handleAutocompleteSelect} />
                </div>
                <p class="mt-2 px-5 text-xs text-gray-500">
                    {data.description || 'Upload your file here'}
                </p>
            </div>
        </div>
        {#if hasError}
            <div class="mt-2 flex flex-row justify-between">
                <div class="text-sm text-red-500">{hasError}</div>
                <button type="button" class="text-sm text-red-500 hover:cursor-pointer" onclick={remove} >X</button>
            </div>
        {/if}
    </div> 
{/if}

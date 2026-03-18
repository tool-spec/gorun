<script lang="ts">
    import hljs from "highlight.js/lib/core";
    import json from "highlight.js/lib/languages/json";
    import bash from "highlight.js/lib/languages/bash";
    import python from "highlight.js/lib/languages/python";
    import "highlight.js/styles/github-dark.css";

    import { onMount } from "svelte";
    import { config } from "$lib/state.svelte";
    
    onMount(() => {
        hljs.registerLanguage("bash", bash);
        hljs.registerLanguage("json", json);
        hljs.registerLanguage("python", python);
        hljs.highlightAll();
    })

    interface CliProps {
        id: string,
        name: string,
        image: string,
        parameterValues: Record<string, any>,
        dataValues: Record<string, any>
    }

    let {id, name, image, parameterValues, dataValues}: CliProps = $props();

    let payloadData = $derived(
        Object.fromEntries(
            Object.entries(dataValues).map(([key, value]) => [key, value?.path])
        )
    );

    let payload = $derived({
        name: name,
        docker_image: image,
        parameters: parameterValues,
        data: payloadData,
    })

    let bashCode = $derived(`run_id=$(curl -X POST -H "Content-Type: application/json" -d '${JSON.stringify(payload).replace(/'/g, "'\\''")}' ${config.apiServer}/runs | jq -r '.id')
echo "Run ID: $run_id"
curl -X POST ${config.apiServer}/runs/$run_id/start`);

    let pythonCode = $derived(`import httpx
payload = {
    "name": "${name}",
    "docker_image": "${image}",
    "parameters": ${JSON.stringify(parameterValues, null, 2)},
    "data": ${JSON.stringify(dataValues, null, 2)}
}
response = httpx.post("${config.apiServer}/runs", json=payload).json()
run_id = response.get('id')
if run_id:
    run_data = httpx.post(f"{config.apiServer}/runs/{run_id}/start").json()
    print(run_data['status'])
else:
    print("Failed to create tool run")
`)
</script>

<div class="p-3 rounded-lg border border-gray-200 shadow-md mb-6">
    <h2 class="text-lg font-semibold text-gray-900 mb3">API Access</h2>
    <p class="mt-2 text-gray-600">
        You can access GoRun API from any programming language or build your own 
        Client application. The table below shows the connection details for the GoRun server you
        are currently connected to:
    </p>
    <table class="mt-2 w-full text-sm border-collapse border border-gray-200">
        <tbody>
            <tr class="bg-gray-50">
                <td class="p-1.5 font-bold">API Server</td>
                <td class="p-1.5">{config.apiServer}</td>
            </tr>
            <tr>
                <td class="p-1.5 font-bold">Tool name</td>
                <td class="p-1.5">{name}</td>
            </tr>
            <tr class="bg-gray-50">
                <td class="p-1.5 font-bold">Docker image</td>
                <td class="p-1.5">{image}</td>
            </tr>
            <tr>
                <td class="p-1.5 font-bold">Full invoke ID</td>
                <td class="p-1.5">{id}</td>
            </tr>
        </tbody>
    </table>

    <h4 class="mt-6 font-semibold text-gray-900">Invoke Tool Payload</h4>
    <p class="text text-gray-600">
        Below is the payload to run the {name} tool. If you invoke the GoRun API directly, 
        you can use this payload as the request body.
        <br>
        If you change the parameters, the payload will be updated automatically.
    </p>
    <div class="mt-2 p-2 shadow-md relative">
        <pre><code class="language-json">{JSON.stringify(payload, null, 2)}</code></pre>
        <button 
            aria-label="Copy to clipboard"
            class="absolute top-3 right-3 p-1 text-gray-500 hover:text-gray-400 transition-colors z-10 w-8 h-8 flex items-center justify-center border border-gray-400 hover:border-gray-300 hover:cursor-pointer rounded"
            onclick={() => navigator.clipboard.writeText(JSON.stringify(payload, null, 2))}
        >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
            </svg>
        </button>
    </div>

    <h4 class="mt-6 font-semibold text-gray-900">cURL</h4>
    <div class="mt-2 p-2 shadow-md relative">
        <pre class="text-wrap"><code class="language-bash">{bashCode}</code></pre>
        <button 
            aria-label="Copy to clipboard"
            class="absolute top-3 right-3 p-1 text-gray-500 hover:text-gray-400 transition-colors z-10 w-8 h-8 flex items-center justify-center border border-gray-400 hover:border-gray-300 hover:cursor-pointer rounded"
            onclick={() => navigator.clipboard.writeText(bashCode)}
        >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
            </svg>
        </button>
    </div>

    <h4 class="mt-4 font-semibold text-gray-900">Python</h4>
    <div class="mt-2 p-2 shadow-md relative">
        <pre class="text-wrap"><code>{pythonCode}</code></pre>
        <button 
            aria-label="Copy to clipboard"
            class="absolute top-3 right-3 p-1 text-gray-500 hover:text-gray-400 transition-colors z-10 w-8 h-8 flex items-center justify-center border border-gray-400 hover:border-gray-300 hover:cursor-pointer rounded"
            onclick={() => navigator.clipboard.writeText(pythonCode)}
        >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
            </svg>
        </button>
    </div>

</div>

import { config } from "$lib/state.svelte";
import type { ResultFile } from "$lib/types/ResultFile";
import type { RunState } from "$lib/types/RunState";
import type { PageLoad } from "./$types";
import { refreshToken } from "$lib/auth.svelte";

export const load: PageLoad = async ({ params, parent, fetch }): Promise<{run: RunState | undefined, files: ResultFile[]}> => {
    await parent();
    await refreshToken();

    const headers = new Headers();
    headers.set("Authorization", `Bearer ${config.auth.access_token}`);
    headers.set("Content-Type", "application/json");

    const runId = Number(params.runId);
    let run: RunState | undefined;
    try {
        const response = await fetch(`${config.apiServer}/runs/${runId}`, {
            method: 'GET',
            headers
        });
        if (response.ok) {
            run = await response.json() as RunState;
        }
    } catch (error) {
        console.log(`Failed to fetch run ${runId}`, error);
    }

    let files: ResultFile[] = [];
    if (run) {
        try {
            const response = await fetch(`${config.apiServer}/runs/${run.id}/results`, {
                method: 'GET',
                headers
            });
            if (response.ok) {
                const payload = await response.json();
                files = payload.files as ResultFile[];
            }
        } catch (error) {
            console.log(`Failed to fetch results for run ${run.id}`, error);
        }
    }

    return {
        run,
        files 
    };
} 

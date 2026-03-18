import { config } from "$lib/state.svelte";
import type { ResultFile } from "$lib/types/ResultFile";
import type { RunState } from "$lib/types/RunState";
import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";
import { authorizedFetch } from "$lib/auth.svelte";

export const load: PageLoad = async ({ params, parent, fetch }): Promise<{run: RunState | undefined, files: ResultFile[]}> => {
    await parent();

    const runId = Number(params.runId);
    let run: RunState | undefined;
    try {
        const response = await authorizedFetch(`${config.apiServer}/runs/${runId}`, {
            method: 'GET',
            headers: {
                "Content-Type": "application/json"
            }
        });
        if (response.status === 401) {
            throw redirect(307, '/manager/login');
        }
        if (response.ok) {
            run = await response.json() as RunState;
        }
    } catch (error) {
        console.log(`Failed to fetch run ${runId}`, error);
    }

    let files: ResultFile[] = [];
    if (run) {
        try {
            const response = await authorizedFetch(`${config.apiServer}/runs/${run.id}/results`, {
                method: 'GET',
                headers: {
                    "Content-Type": "application/json"
                }
            });
            if (response.status === 401) {
                throw redirect(307, '/manager/login');
            }
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

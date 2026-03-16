export interface RunResultSummary {
    artifact_count: number;
    log_count: number;
    internal_count: number;
    metadata_count: number;
    total_size: number;
}

export interface RunState {
    id: number;
    name: string,
    title: string,
    description: string,
    image: string,
    mounts?: {
        [containerPath: string]: string
    },
    parameters?: {
        [name: string]: string | number | boolean | Date | Object
    },
    data?: {
        [name: string]: string
    },
    status: "pending" | "running" | "finished" | "errored",
    created_at: Date,
    started_at?: Date,
    finished_at?: Date,
    has_errored: boolean,
    error_message?: string,
    gotap_metadata?: Record<string, unknown>,
    result_summary?: RunResultSummary
}

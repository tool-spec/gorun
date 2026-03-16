import type { ClassifiedResultFile, ResultFile, ResultFileKind } from "./types/ResultFile";

const previewableExtensions = new Set(['.json', '.txt', '.log', '.md', '.csv']);
const kindOrder: Record<ResultFileKind, number> = {
    artifact: 0,
    log: 1,
    metadata: 2,
    internal: 3
};

export interface PreviewResponse {
    filename: string;
    mimeType: string;
    encoding: string;
    truncated: boolean;
    content: string;
}

export function classifyResultFile(file: ResultFile): ClassifiedResultFile {
    let kind: ResultFileKind = 'artifact';
    if (file.name === '_metadata.json') {
        kind = 'metadata';
    } else if (file.name === 'STDOUT.log' || file.name === 'STDERR.log') {
        kind = 'log';
    } else if (file.name.startsWith('.') || file.name.startsWith('_')) {
        kind = 'internal';
    }

    const extension = file.name.includes('.') ? `.${file.name.split('.').pop()!.toLowerCase()}` : '';
    const previewable = previewableExtensions.has(extension) || kind === 'log';
    const relParts = file.relPath.split('/').filter(Boolean);
    const groupPath = relParts.length > 1 ? relParts.slice(0, -1).join('/') : '.';

    return {
        ...file,
        kind,
        previewable,
        displayName: file.name,
        groupPath
    };
}

export function sortResultFiles(files: ResultFile[]): ClassifiedResultFile[] {
    return files
        .map(classifyResultFile)
        .sort((left, right) => {
            const kindCmp = kindOrder[left.kind] - kindOrder[right.kind];
            if (kindCmp !== 0) {
                return kindCmp;
            }
            const groupCmp = left.groupPath.localeCompare(right.groupPath);
            if (groupCmp !== 0) {
                return groupCmp;
            }
            return left.displayName.localeCompare(right.displayName);
        });
}

export function groupResultFiles(files: ClassifiedResultFile[]): Array<{ path: string; files: ClassifiedResultFile[] }> {
    const groups = new Map<string, ClassifiedResultFile[]>();
    for (const file of files) {
        const bucket = groups.get(file.groupPath) ?? [];
        bucket.push(file);
        groups.set(file.groupPath, bucket);
    }

    return Array.from(groups.entries())
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([path, groupedFiles]) => ({ path, files: groupedFiles }));
}

export function resultPathParam(relPath: string): string {
    return encodeURIComponent(relPath);
}

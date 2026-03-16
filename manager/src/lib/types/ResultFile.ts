export type ResultFileKind = 'artifact' | 'log' | 'metadata' | 'internal';

export interface ResultFile {
    name: string;
    relPath: string;
    absPath: string;
    size: number;
    lastModified?: Date;
}

export interface ClassifiedResultFile extends ResultFile {
    kind: ResultFileKind;
    previewable: boolean;
    displayName: string;
    groupPath: string;
}

export const bytesToSize = (bytes: number): string => {
    let short = bytes;
    let suffix = 'B';
    if (short > 1024) {
        short = short / 1024;
        suffix = 'KB';
    }
    if (short > 1024) {
        short = short / 1024;
        suffix = 'MB';
    }
    if (short > 1024) {
        short = short / 1024;
        suffix = 'GB';
    }

    return `${short.toFixed(1)} ${suffix}`
}

export const formatPercentFromPermille = (permille: number): string => {
    return `${(permille / 10).toFixed(1)}%`;
}

export const titleCase = (value: string): string => {
    return value
        .toLowerCase()
        .split(' ')
        .filter(Boolean)
        .map(word => word.charAt(0).toUpperCase() + word.slice(1))
        .join(' ');
}

export const formatDurationFromNanoseconds = (nanoseconds: number): string => {
    if (nanoseconds < 1_000) {
        return `${nanoseconds} ns`;
    }

    if (nanoseconds < 1_000_000) {
        return `${(nanoseconds / 1_000).toFixed(1)} us`;
    }

    if (nanoseconds < 1_000_000_000) {
        return `${(nanoseconds / 1_000_000).toFixed(1)} ms`;
    }

    return `${(nanoseconds / 1_000_000_000).toFixed(2)} s`;
}

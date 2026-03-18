import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params, parent }) => {
    return parent().then(({ specs }) => {
        const specId = decodeURIComponent(params.specId);
        const spec = specs.find(spec => spec.id === specId);
        return { spec };
    });
};

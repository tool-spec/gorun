<script lang="ts">
    import type { PageProps } from "./$types";
    import type { ToolSpec } from "$lib/types/ToolSpec";
    import type { CitationFile, Author } from "$lib/types/CitationFile";
    import { goto } from "$app/navigation";

    let { data }: PageProps = $props();
    let specs: ToolSpec[] = $state(data.specs);
    $inspect(specs);

    function getAuthorName(author: Author): string {
        if (author.IsPerson && author.Person) {
            return author.Person.Family;
        } else if (author.IsEntity && author.Entity) {
            return author.Entity.Name;
        }
        return '';
    }

    function getRepositoryUrl(citation: CitationFile): string {
        if (!citation.RepositoryCode) return '#';
        const { Scheme, Host, Path } = citation.RepositoryCode;
        return `${Scheme}://${Host}${Path}`;
    }

    function formatCitation(citation: CitationFile): string {
        if (!citation.Authors || citation.Authors.length === 0) {
            return 'link to repository';
        }

        const repo = citation.RepositoryCode?.Host || 'NaN';
        const authors = citation.Authors.map(getAuthorName).filter(Boolean);

        if (authors.length === 0) {
            return 'link to repository';
        } else if (authors.length === 1) {
            return `${authors[0]} (${repo})`;
        } else if (authors.length === 2) {
            return `${authors[0]} & ${authors[1]} (${repo})`;
        } else {
            return `${authors[0]} et al. (${repo})`;
        }
    }

    function getSpecHref(specId: string): string {
        return `/manager/specs/${encodeURIComponent(specId)}`;
    }
</script>

<div class="p-4">
    <div class="flex justify-between items-center mb-6">
        <h1 class="text-2xl font-bold text-gray-900">Specs</h1>
    </div>

    <div class="space-y-4">
        {#each specs as spec}
            <div 
                class="block cursor-pointer" 
                role="button"
                tabindex="0"
                onclick={() => goto(getSpecHref(spec.id))}
                onkeydown={(e: KeyboardEvent) => {
                    if (e.key === 'Enter') {
                        goto(getSpecHref(spec.id));
                    }
                }}
            >
                <div class="p-4 bg-white rounded-lg shadow hover:shadow-md transition-shadow border border-gray-200">
                    <h2 class="text-lg font-semibold text-gray-900">{spec.title}</h2>
                    <h4 class="text-md font-semibold text-gray-500">ID: {spec.id}</h4>
                    {#if spec.citation}
                        <div class="mt-2">
                            {#if spec.citation.Authors}
                                <a 
                                    href={getRepositoryUrl(spec.citation)}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 hover:bg-blue-200"
                                >
                                    {formatCitation(spec.citation)}
                                </a>
                            {/if}
                        </div>
                    {/if}
                    <p class="mt-2 text-gray-600">{spec.description}</p>
                    <div class="mt-2 flex items-center text-sm text-blue-500">
                        View details →
                    </div>
                </div>
            </div>
        {/each}
    </div>
</div>

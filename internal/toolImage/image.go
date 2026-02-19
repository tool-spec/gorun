package toolImage

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/alexander-lindner/go-cff"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hydrocode-de/gorun/internal/cache"
	toolspec "github.com/hydrocode-de/tool-spec-go"
)

func ReadAllTools(ctx context.Context, cache *cache.Cache, verbose bool) ([]string, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer c.Close()

	summary, err := c.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Filter images with tags
	var imagesWithTags []string
	for _, img := range summary {
		if len(img.RepoTags) > 0 {
			imagesWithTags = append(imagesWithTags, img.RepoTags[0])
		}
	}

	// Use a channel to collect results from goroutines
	type result struct {
		tools []string
		err   error
	}
	resultChan := make(chan result, len(imagesWithTags))

	// Process each image in its own goroutine
	for _, imgTag := range imagesWithTags {
		go func(tag string) {
			var tools []string

			// Check if already cached
			image, ok := cache.GetImageSpec(tag)
			if !ok {
				// Create a new Docker client for this goroutine
				client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
				if err != nil {
					resultChan <- result{nil, err}
					return
				}
				defer client.Close()

				spec, err := readToolSpec(ctx, client, tag)
				if err != nil {
					if verbose {
						log.Printf("image %s does not contain a tool-spec", tag)
					}
					resultChan <- result{tools, nil}
					return
				}
				citation, citationErr := readToolCitation(ctx, client, tag)
				if citationErr != nil && verbose {
					log.Printf("image %s does not contain a CITATION.cff", tag)
				}

				cache.SetImageSpec(tag, spec)
				for name, tool := range spec.Tools {
					slug := fmt.Sprintf("%s::%s", tag, name)
					tool.ID = slug
					if citationErr == nil {
						tool.Citation = citation
					}
					cache.SetToolSpec(slug, &tool)
					tools = append(tools, slug)
				}
			} else {
				// Image already cached, ensure individual tools are cached
				for name, tool := range image.Tools {
					slug := fmt.Sprintf("%s::%s", tag, name)
					tool.ID = slug
					cache.SetToolSpec(slug, &tool)
					tools = append(tools, slug)
				}
			}

			resultChan <- result{tools, nil}
		}(imgTag)
	}

	// Collect results
	var allTools []string
	for i := 0; i < len(imagesWithTags); i++ {
		select {
		case res := <-resultChan:
			if res.err != nil {
				return nil, res.err
			}
			allTools = append(allTools, res.tools...)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	cache.Initialised = true
	return allTools, nil
}

func LoadToolSpec(ctx context.Context, c *client.Client, toolSlug string, cache *cache.Cache) (toolspec.ToolSpec, error) {
	chunks := strings.Split(toolSlug, "::")
	if len(chunks) == 1 {
		spec, ok := cache.GetToolSpec(toolSlug)
		if !ok {
			return toolspec.ToolSpec{}, fmt.Errorf("the tool %s was not found in the cache. Try to call like <image-name>::<tool-name>", toolSlug)
		}
		return *spec, nil
	}

	if len(chunks) == 2 {
		imageName := chunks[0]
		toolName := chunks[1]
		spec, ok := cache.GetImageSpec(imageName)
		if !ok {
			specFile, err := readToolSpec(ctx, c, imageName)
			if err != nil {
				return toolspec.ToolSpec{}, err
			}
			citation, citationErr := readToolCitation(ctx, c, imageName)
			if citationErr != nil {
				log.Printf("image %s does not contain a CITATION.cff", imageName)
			}
			cache.SetImageSpec(imageName, specFile)
			for name, tool := range specFile.Tools {
				cache.SetToolSpec(name, &tool)
			}
			tool, ok := specFile.Tools[toolName]
			if !ok {
				return toolspec.ToolSpec{}, fmt.Errorf("the tool %s was not found in the image %s", toolName, imageName)
			}
			tool.ID = toolSlug
			if citationErr == nil {
				tool.Citation = citation
			}
			return tool, nil
		} else {
			tool, ok := spec.Tools[toolName]
			if !ok {
				return toolspec.ToolSpec{}, fmt.Errorf("the tool %s was not found in the image %s", toolName, imageName)
			}
			tool.ID = toolSlug
			return tool, nil
		}
	}
	return toolspec.ToolSpec{}, fmt.Errorf("invalid tool slug: %s", toolSlug)
}

func ReadToolSpec(ctx context.Context, imageName string) (toolspec.SpecFile, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return toolspec.SpecFile{}, err
	}
	defer c.Close()

	return readToolSpec(ctx, c, imageName)
}

func readToolSpec(ctx context.Context, c *client.Client, imageName string) (toolspec.SpecFile, error) {
	gotapPath, gotapFound, err := ProbeGotap(ctx, c, imageName)
	if err != nil {
		return toolspec.SpecFile{}, err
	}

	if gotapFound {
		commands := [][]string{
			{gotapPath, "metadata", "--spec-file", "/src/tool.yml"},
			{gotapPath, "parse", "--spec-file", "/src/tool.yml"},
			{gotapPath, "parse", "/src/tool.yml"},
		}
		for _, cmd := range commands {
			stdout, _, exitCode, cmdErr := runContainerCommand(ctx, c, imageName, []string{cmd[0]}, cmd[1:])
			if cmdErr != nil || exitCode != 0 || strings.TrimSpace(stdout) == "" {
				continue
			}
			spec, parseErr := toolspec.LoadToolSpec([]byte(stdout))
			if parseErr == nil {
				return spec, nil
			}
		}
	}

	stdout, stderr, exitCode, err := runContainerCommand(ctx, c, imageName, []string{"cat"}, []string{"/src/tool.yml"})
	if err != nil {
		return toolspec.SpecFile{}, err
	}
	if exitCode != 0 {
		return toolspec.SpecFile{}, fmt.Errorf("the container errored while identifying the tool spec: %v", strings.TrimSpace(stderr))
	}
	if strings.TrimSpace(stdout) == "" {
		return toolspec.SpecFile{}, fmt.Errorf("the container did not respond")
	}

	spec, err := toolspec.LoadToolSpec([]byte(stdout))
	if err != nil {
		return toolspec.SpecFile{}, fmt.Errorf("the container %s did not contain a valid tool-spec at /src/tool.yml: %v", imageName, err)
	}

	return spec, nil
}

func readToolCitation(ctx context.Context, c *client.Client, imageName string) (cff.Cff, error) {
	cont, err := c.ContainerCreate(ctx, &container.Config{
		Image:      imageName,
		Entrypoint: []string{"cat"},
		Cmd:        []string{"/src/CITATION.cff"},
	}, &container.HostConfig{}, nil, nil, "")
	if err != nil {
		return cff.Cff{}, err
	}
	defer c.ContainerRemove(ctx, cont.ID, container.RemoveOptions{})

	if err = c.ContainerStart(ctx, cont.ID, container.StartOptions{}); err != nil {
		return cff.Cff{}, err
	}

	statusCh, errCh := c.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		return cff.Cff{}, err
	case <-statusCh:
	}

	logReader, err := c.ContainerLogs(ctx, cont.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return cff.Cff{}, err
	}
	defer logReader.Close()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stdcopy.StdCopy(stdout, stderr, logReader)

	if stderr.Len() != 0 {
		return cff.Cff{}, fmt.Errorf("Error while reading CITATION.cff: %v", stderr.String())
	}
	if stdout.Len() == 0 {
		return cff.Cff{}, fmt.Errorf("No CITATION.cff found in the container %s", imageName)
	}

	citation, err := cff.Parse(stdout.String())
	if err != nil {
		return cff.Cff{}, fmt.Errorf("Error while parsing CITATION.cff: %v", err)
	}
	return citation, nil
}

package toolImage

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func ProbeGotap(ctx context.Context, c *client.Client, imageName string) (string, bool, error) {
	stdout, _, exitCode, err := runContainerCommand(ctx, c, imageName, []string{"sh", "-lc"}, []string{"command -v gotap || which gotap"})
	if err != nil {
		return "", false, err
	}
	if exitCode != 0 {
		return "", false, nil
	}
	gotapPath := strings.TrimSpace(stdout)
	if gotapPath == "" {
		return "", false, nil
	}
	return gotapPath, true, nil
}

func runContainerCommand(ctx context.Context, c *client.Client, imageName string, entrypoint []string, cmd []string) (string, string, int64, error) {
	cont, err := c.ContainerCreate(ctx, &container.Config{
		Image:        imageName,
		Entrypoint:   entrypoint,
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}, &container.HostConfig{}, nil, nil, "")
	if err != nil {
		return "", "", 0, err
	}
	defer c.ContainerRemove(ctx, cont.ID, container.RemoveOptions{})

	if err = c.ContainerStart(ctx, cont.ID, container.StartOptions{}); err != nil {
		return "", "", 0, err
	}

	statusCh, errCh := c.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case err := <-errCh:
		if err != nil {
			return "", "", 0, err
		}
	case waitRes := <-statusCh:
		if waitRes.Error != nil {
			return "", "", waitRes.StatusCode, fmt.Errorf("container wait error: %s", waitRes.Error.Message)
		}
		exitCode = waitRes.StatusCode
	}

	logReader, err := c.ContainerLogs(ctx, cont.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", "", exitCode, err
	}
	defer logReader.Close()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if _, err := stdcopy.StdCopy(stdout, stderr, logReader); err != nil {
		return "", "", exitCode, err
	}

	return stdout.String(), stderr.String(), exitCode, nil
}

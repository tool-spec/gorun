package tool

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hydrocode-de/gorun/internal/db"
	"github.com/hydrocode-de/gorun/internal/toolImage"
)

type RunToolOptions struct {
	DB     *db.Queries
	Tool   Tool
	Env    []string
	Cmd    []string
	UserId string
}

func RunTool(ctx context.Context, opt RunToolOptions) error {
	// create a function to update the database
	updateDB := func(status string, origError error) {
		switch status {
		case "started":
			_, err := opt.DB.StartRun(ctx, db.StartRunParams{
				ID:     opt.Tool.ID,
				UserID: opt.UserId,
			})
			if err != nil {
				log.Fatal(err)
			}
		case "finished":
			_, err := opt.DB.FinishRun(ctx, opt.Tool.ID)
			if err != nil {
				log.Fatal(err)
			}
		case "errored":
			_, err := opt.DB.RunErrored(ctx, db.RunErroredParams{
				ID: opt.Tool.ID,
				ErrorMessage: sql.NullString{
					String: fmt.Sprintf("the execution of the tool (%v) container (%v) errored unexpectedly: %v", opt.Tool.Name, opt.Tool.Image, origError),
					Valid:  true,
				},
			})
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer c.Close()
	tool := &opt.Tool
	mounts := make([]mount.Mount, 0, len(tool.Mounts))
	for containerPath, hostPath := range tool.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: hostPath,
			Target: containerPath,
		})
	}
	//fmt.Println(mounts)

	config := container.Config{
		Image:        tool.Image,
		Tty:          false,
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
	}

	runMode := "default"
	if len(opt.Cmd) != 0 {
		fmt.Printf("Custom CMD: %v\n", opt.Cmd)
		config.Cmd = opt.Cmd
	} else {
		gotapPath, gotapFound, probeErr := toolImage.ProbeGotap(ctx, c, tool.Image)
		if probeErr != nil {
			updateDB("errored", probeErr)
			return probeErr
		}
		if gotapFound {
			config.Entrypoint = []string{gotapPath}
			config.Cmd = []string{"run", tool.Name, "--input-file", "/in/inputs.json", "--spec-file", "/src/tool.yml", "--output-folder", "/out"}
			runMode = "gotap"
			fmt.Printf("detected gotap shim at %s\n", gotapPath)
		}
	}
	fmt.Printf("running tool %v with image: %v\n", tool.Name, tool.Image)
	cont, err := c.ContainerCreate(ctx, &config, &container.HostConfig{
		Mounts: mounts,
	}, nil, nil, "")
	if err != nil {
		updateDB("errored", err)
		return err
	}
	defer c.ContainerRemove(ctx, cont.ID, container.RemoveOptions{})
	fmt.Printf("container created: %v\n", cont)

	if err = c.ContainerStart(ctx, cont.ID, container.StartOptions{}); err != nil {
		updateDB("errored", err)
		fmt.Println("starting container failed")
		return err
	}
	updateDB("started", nil)

	statusCh, errCh := c.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case err := <-errCh:
		if err != nil {
			fmt.Println(err)
			updateDB("errored", err)
			return err
		}
	case status := <-statusCh:
		if status.Error != nil {
			err := errors.New(status.Error.Message)
			updateDB("errored", err)
			return err
		}
		exitCode = status.StatusCode
		fmt.Println("container finished")
	}

	logReader, err := c.ContainerLogs(ctx, cont.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return err
	}
	defer logReader.Close()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if _, err := stdcopy.StdCopy(stdout, stderr, logReader); err != nil {
		updateDB("errored", err)
		return err
	}

	// create log files in the mounted out volume
	var outDir string
	for _, mount := range mounts {
		if mount.Target == "/out" {
			outDir = mount.Source
			os.WriteFile(path.Join(mount.Source, "STDOUT.log"), stdout.Bytes(), 0644)
			os.WriteFile(path.Join(mount.Source, "STDERR.log"), stderr.Bytes(), 0644)
			break
		}
	}

	if outDir != "" {
		metadataPath := path.Join(outDir, "_metadata.json")
		metadataBytes, err := os.ReadFile(metadataPath)
		if err == nil {
			if json.Valid(metadataBytes) {
				_, dbErr := opt.DB.SetRunGotapMetadata(ctx, db.SetRunGotapMetadataParams{
					GotapMetadata: sql.NullString{
						String: strings.TrimSpace(string(metadataBytes)),
						Valid:  true,
					},
					ID: opt.Tool.ID,
				})
				if dbErr != nil {
					log.Printf("failed to persist gotap metadata for run %d: %v", opt.Tool.ID, dbErr)
				}
			} else {
				log.Printf("invalid gotap metadata JSON at %s", metadataPath)
			}
		} else if !os.IsNotExist(err) {
			log.Printf("failed reading gotap metadata at %s: %v", metadataPath, err)
		}
	}

	if exitCode != 0 {
		runErr := fmt.Errorf("%s execution exited with status %d", runMode, exitCode)
		updateDB("errored", runErr)
		return runErr
	}
	updateDB("finished", nil)
	return nil
}

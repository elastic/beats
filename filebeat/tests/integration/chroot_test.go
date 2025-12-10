//go:build integration && linux

package integration

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/elastic/elastic-agent-libs/testing/fs"
)

func TestJournaldChroot(t *testing.T) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("Failed to create Docker client: %s", err)
	}
	defer cli.Close()

	tempDir := fs.TempDir(t, filepath.Join("..", "..", "build"))
	logFile := fs.NewLogFile(t, tempDir, "container-logs-*.log")

	dockerfileDir := "testdata/journald_chroot"
	dockerfileName := "Dockerfile"
	imageName := "filebeat-chroot-test"
	filebeatBinaryPath := "filebeat"

	buildFilebeatBinary(t)

	buildDockerImage(ctx, t, cli, dockerfileDir, dockerfileName, imageName, filebeatBinaryPath)

	containerName := "filebeat-chroot-container"
	startDockerContainer(ctx, t, cli, imageName, containerName, logFile)

	assertJournalctlStarted(t, logFile)
}

func buildDockerImage(ctx context.Context, t *testing.T, cli *client.Client, dockerfileDir, dockerfileName, imageName, filebeatBinaryPath string) {
	buildContext, err := archiveTarWithFilebeat(dockerfileDir, filebeatBinaryPath)
	if err != nil {
		t.Fatalf("Failed to create build context: %s", err)
	}

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: dockerfileName,
	}

	resp, err := cli.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		t.Fatalf("Failed to build Docker image: %s", err)
	}
	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)
}

func archiveTarWithFilebeat(dir, filebeatBinaryPath string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Add the directory contents to the tar archive
	err := filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		header.Name, _ = filepath.Rel(dir, file)
		if fi.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.IsDir() {
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Add the Filebeat binary to the tar archive
	filebeatFile, err := os.Open(filebeatBinaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Filebeat binary: %w", err)
	}
	defer filebeatFile.Close()

	fileInfo, err := filebeatFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat Filebeat binary: %w", err)
	}

	header, err := tar.FileInfoHeader(fileInfo, filebeatBinaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create tar header for Filebeat binary: %w", err)
	}
	header.Name = "filebeat" // Place the binary in the root of the build context

	if err := tw.WriteHeader(header); err != nil {
		return nil, fmt.Errorf("failed to write tar header for Filebeat binary: %w", err)
	}

	if _, err := io.Copy(tw, filebeatFile); err != nil {
		return nil, fmt.Errorf("failed to copy Filebeat binary to tar archive: %w", err)
	}

	return buf, nil
}

func buildFilebeatBinary(t *testing.T) {
	cmd := exec.Command("go", "build", "-ldflags", "-extldflags \"-static\" -s", "-tags", "timetzdata", "../../")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build Filebeat binary: %s\nOutput: %s", err, output)
	}
}

func startDockerContainer(ctx context.Context, t *testing.T, cli *client.Client, imageName, containerName string, logFile *fs.LogFile) string {
	resp, err := cli.ContainerCreate(
		ctx, &container.Config{
			Image: imageName,
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: "/",
					Target: "/host",
				},
			},
			CapAdd:     []string{"CAP_SYS_CHROOT"}, // Required for chroot
			AutoRemove: true,
		},
		nil, nil, "")
	if err != nil {
		t.Fatalf("Failed to create Docker container: %s", err)
	}

	// Attach to the container's logs
	attachResp, err := cli.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		t.Fatalf("Failed to attach to Docker container logs: %s", err)
	}

	// Stream logs to the log file
	go func() {
		defer attachResp.Close()
		if _, err := io.Copy(logFile, attachResp.Reader); err != nil {
			t.Errorf("Error streaming container logs: %s", err)
		}
	}()

	// Start the container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		t.Fatalf("Failed to start Docker container: %s", err)
	}

	// Allow some time for the container to start
	return resp.ID
}

func assertJournalctlStarted(t *testing.T, logFile *fs.LogFile) {
	// Wait for the log message "journalctl started with PID XX"
	logFile.WaitLogsContains(t, "journalctl started with PID", 30*time.Second, "journalctl did not start")
}

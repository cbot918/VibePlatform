package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	// Legacy ubuntu image (kept for existing container handler)
	ImageUbuntu = "ubuntu:22.04"

	// code-server image and its internal port
	ImageCodeServer     = "vibeplatform-code-server:latest"
	CodeServerPort      = "8080/tcp"
	CodeServerPortInner = "8080"
)

type Client struct {
	cli *client.Client
}

func New() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

// EnsureCodeServer 建立並啟動使用者的 code-server container（若已存在則直接回傳）。
// Container 命名為 vibe-{userID}；各 project 為 /home/coder/{projectName} 資料夾。
// 回傳 containerID 和對應的 host port。
func (c *Client) EnsureCodeServer(ctx context.Context, userID, apiKey string) (containerID, hostPort string, err error) {
	// Only pull if the image is not already present locally.
	if _, _, inspectErr := c.cli.ImageInspectWithRaw(ctx, ImageCodeServer); inspectErr != nil {
		rc, pullErr := c.cli.ImagePull(ctx, ImageCodeServer, image.PullOptions{})
		if pullErr != nil {
			return "", "", fmt.Errorf("pull image: %w", pullErr)
		}
		io.Copy(os.Stderr, rc)
		rc.Close()
	}

	containerName := fmt.Sprintf("vibe-cs-%s", userID)

	// If container already exists, reuse it (start if stopped).
	if info, err := c.cli.ContainerInspect(ctx, containerName); err == nil {
		if !info.State.Running {
			if err := c.cli.ContainerStart(ctx, info.ID, container.StartOptions{}); err != nil {
				return "", "", fmt.Errorf("start existing container: %w", err)
			}
		}
		port, err := c.hostPortFor(ctx, info.ID, CodeServerPort)
		if err != nil {
			return info.ID, "", err
		}
		return info.ID, port, nil
	}

	env := []string{
		"ANTHROPIC_API_KEY=" + apiKey,
	}

	exposed := nat.PortSet{nat.Port(CodeServerPort): struct{}{}}
	portBindings := nat.PortMap{
		nat.Port(CodeServerPort): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: ""}},
	}

	resp, err := c.cli.ContainerCreate(ctx,
		&container.Config{
			Image:        ImageCodeServer,
			Env:          env,
			ExposedPorts: exposed,
			Labels: map[string]string{
				"vibeplatform.user_id": userID,
			},
			// code-server: 不需要密碼、開啟 /home/coder；各 project 透過 ?folder= 切換
			Cmd: []string{
				"--auth", "none",
				"--bind-addr", "0.0.0.0:" + CodeServerPortInner,
				"/home/coder",
			},
		},
		&container.HostConfig{
			PortBindings: portBindings,
		},
		nil, nil,
		containerName,
	)
	if err != nil {
		return "", "", fmt.Errorf("create container: %w", err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", "", fmt.Errorf("start container: %w", err)
	}

	port, err := c.hostPortFor(ctx, resp.ID, CodeServerPort)
	if err != nil {
		return resp.ID, "", err
	}

	return resp.ID, port, nil
}

// MkdirProject 在 container 內建立 /home/coder/{projectName} 資料夾。
func (c *Client) MkdirProject(ctx context.Context, containerID, projectName string) error {
	return c.execIn(ctx, containerID, []string{"mkdir", "-p", "/home/coder/" + projectName})
}

// execIn 在 container 內執行一個指令（fire-and-forget，不讀 stdout/stderr）。
func (c *Client) execIn(ctx context.Context, containerID string, cmd []string) error {
	execResp, err := c.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{Cmd: cmd})
	if err != nil {
		return fmt.Errorf("exec create %v: %w", cmd, err)
	}
	return c.cli.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{})
}

// execWithStdin 在 container 內執行指令，並透過 stdin 傳入 input。
func (c *Client) execWithStdin(ctx context.Context, containerID string, cmd []string, input string) error {
	execResp, err := c.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:         cmd,
		AttachStdin: true,
	})
	if err != nil {
		return fmt.Errorf("exec create %v: %w", cmd, err)
	}
	hijack, err := c.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return fmt.Errorf("exec attach %v: %w", cmd, err)
	}
	defer hijack.Close()
	if _, err := fmt.Fprint(hijack.Conn, input); err != nil {
		return fmt.Errorf("write stdin %v: %w", cmd, err)
	}
	return hijack.CloseWrite()
}

// ConfigureGit 在 running container 內設定 git 使用者資訊並透過 gh CLI 認證 GitHub。
// 每次建立 project 時呼叫，確保憑證始終是最新的（即便 container 是被 reuse 的）。
func (c *Client) ConfigureGit(ctx context.Context, containerID, gitUser, gitEmail, gitToken string) error {
	// git global config
	for _, args := range [][]string{
		{"git", "config", "--global", "user.name", gitUser},
		{"git", "config", "--global", "user.email", gitEmail},
		{"git", "config", "--global", "credential.helper", "store"},
	} {
		if err := c.execIn(ctx, containerID, args); err != nil {
			return err
		}
	}

	// Write ~/.git-credentials for HTTPS push
	cred := fmt.Sprintf("https://%s:%s@github.com\n", gitUser, gitToken)
	if err := c.execWithStdin(ctx, containerID,
		[]string{"sh", "-c", "cat > /home/coder/.git-credentials"}, cred); err != nil {
		return fmt.Errorf("write git-credentials: %w", err)
	}

	// Authenticate gh CLI
	if err := c.execWithStdin(ctx, containerID,
		[]string{"gh", "auth", "login", "--with-token", "--hostname", "github.com"},
		gitToken+"\n"); err != nil {
		return fmt.Errorf("gh auth login: %w", err)
	}

	return nil
}

// Stop 停止並移除 container
func (c *Client) Stop(ctx context.Context, containerID string) error {
	if err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	return c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{})
}

// Status 回傳 container 狀態（running / exited / ...）
func (c *Client) Status(ctx context.Context, containerID string) (string, error) {
	info, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}
	return info.State.Status, nil
}

// Start 建立並啟動 ubuntu container（舊有功能保留）
func (c *Client) Start(ctx context.Context, userID string) (containerID, hostPort string, err error) {
	rc, err := c.cli.ImagePull(ctx, ImageUbuntu, image.PullOptions{})
	if err != nil {
		return "", "", fmt.Errorf("pull image: %w", err)
	}
	io.Copy(os.Stderr, rc)
	rc.Close()

	resp, err := c.cli.ContainerCreate(ctx,
		&container.Config{
			Image: ImageUbuntu,
			Cmd:   []string{"sleep", "infinity"},
			Labels: map[string]string{
				"vibeplatform.user_id": userID,
			},
		},
		&container.HostConfig{
			PublishAllPorts: true,
		},
		nil, nil,
		fmt.Sprintf("vibe-ubuntu-%s", userID),
	)
	if err != nil {
		return "", "", fmt.Errorf("create container: %w", err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", "", fmt.Errorf("start container: %w", err)
	}

	port, err := c.hostPortFor(ctx, resp.ID, "22/tcp")
	if err != nil {
		return resp.ID, "", err
	}
	return resp.ID, port, nil
}

func (c *Client) hostPortFor(ctx context.Context, containerID, portProto string) (string, error) {
	info, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}
	bindings := info.NetworkSettings.Ports[nat.Port(portProto)]
	if len(bindings) == 0 {
		return "", nil
	}
	return bindings[0].HostPort, nil
}

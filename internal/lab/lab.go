package lab

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"platform-lab/internal/ansible"
	"platform-lab/internal/inventory"
)

const (
	baseImage     = "platform-lab-base:local"
	appListenPort = 8080
)

type Options struct {
	Root   string
	Env    string
	Stdout io.Writer
	Stderr io.Writer
}

func (o Options) writers() (io.Writer, io.Writer) {
	stdout, stderr := o.Stdout, o.Stderr
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	return stdout, stderr
}

func (o Options) envDir() string {
	return filepath.Join(o.Root, "infra", "envs", o.Env)
}

func (o Options) localDir() string {
	return filepath.Join(o.Root, ".local", o.Env)
}

func (o Options) step(format string, args ...any) {
	stdout, _ := o.writers()
	fmt.Fprintf(stdout, "==> "+format+"\n", args...)
}

func (o Options) Up(ctx context.Context) error {
	if err := o.ensureDocker(ctx); err != nil {
		return err
	}
	if err := o.buildImage(ctx); err != nil {
		return err
	}
	if err := o.terraform(ctx, "init", "-input=false"); err != nil {
		return err
	}
	if err := o.terraform(ctx, "apply", "-auto-approve", "-input=false"); err != nil {
		return err
	}
	_, err := o.writeInventory(ctx)
	return err
}

func (o Options) Converge(ctx context.Context) error {
	if err := o.ensureDocker(ctx); err != nil {
		return err
	}
	if err := o.buildServer(ctx); err != nil {
		return err
	}
	if _, err := o.writeInventory(ctx); err != nil {
		return err
	}
	if err := o.installCollections(ctx); err != nil {
		return err
	}
	_, err := o.playbook(ctx)
	return err
}

func (o Options) Smoke(ctx context.Context) error {
	if err := o.ensureDocker(ctx); err != nil {
		return err
	}
	if err := o.buildImage(ctx); err != nil {
		return err
	}
	if err := o.buildServer(ctx); err != nil {
		return err
	}
	if err := o.terraform(ctx, "init", "-input=false"); err != nil {
		return err
	}
	if err := o.terraform(ctx, "apply", "-auto-approve", "-input=false"); err != nil {
		return err
	}
	o.step("confirm terraform plan is clean")
	if err := o.terraform(ctx, "plan", "-detailed-exitcode", "-input=false"); err != nil {
		return fmt.Errorf("terraform plan is not idempotent: %w", err)
	}
	o.step("apply terraform again")
	if err := o.terraformApplyNoChanges(ctx); err != nil {
		return err
	}
	vars, err := o.writeInventory(ctx)
	if err != nil {
		return err
	}
	if err := o.installCollections(ctx); err != nil {
		return err
	}
	if _, err := o.playbook(ctx); err != nil {
		return err
	}
	o.step("run playbook again and expect no changes")
	statsOut, err := o.playbook(ctx)
	if err != nil {
		return err
	}
	stats, err := ansible.ParseRecap(statsOut)
	if err != nil {
		return err
	}
	if err := ansible.AssertClean(stats); err != nil {
		return err
	}
	base := fmt.Sprintf("http://127.0.0.1:%d", vars.ProxyHTTPPort)
	o.step("check %s/health", base)
	if err := waitHTTP(ctx, base+"/health", "ok\n"); err != nil {
		return err
	}
	index := fmt.Sprintf("env=%s\ndb_host=%s\n", vars.EnvName, vars.DBHost)
	if err := waitHTTP(ctx, base+"/", index); err != nil {
		return err
	}
	o.step("smoke passed for %s at %s", o.Env, base+"/health")
	return nil
}

func (o Options) Down(ctx context.Context) error {
	if err := o.ensureDocker(ctx); err != nil {
		return err
	}
	// Destroy plans the image id data source, so the local tag has to exist.
	if err := o.buildImage(ctx); err != nil {
		return err
	}
	if err := o.terraform(ctx, "init", "-input=false"); err != nil {
		return err
	}
	return o.terraform(ctx, "destroy", "-auto-approve", "-input=false")
}

func (o Options) ensureDocker(ctx context.Context) error {
	home, _ := os.UserHomeDir()
	candidates := []string{}
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		candidates = append(candidates, host)
	}
	candidates = append(candidates,
		"",
		"unix:///var/run/docker.sock",
		"unix://"+filepath.Join(home, ".orbstack/run/docker.sock"),
		"unix://"+filepath.Join(home, ".docker/run/docker.sock"),
	)
	seen := map[string]bool{}
	var last error
	for _, host := range candidates {
		if seen[host] {
			continue
		}
		seen[host] = true
		probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		cmd := exec.CommandContext(probeCtx, "docker", "info", "--format", "{{.ServerVersion}}")
		if host != "" {
			cmd.Env = append(os.Environ(), "DOCKER_HOST="+host)
		}
		out, err := cmd.Output()
		cancel()
		if err != nil || strings.TrimSpace(string(out)) == "" {
			last = err
			continue
		}
		if host != "" && os.Getenv("DOCKER_HOST") != host {
			if err := os.Setenv("DOCKER_HOST", host); err != nil {
				return err
			}
			o.step("using Docker at %s", host)
		}
		return nil
	}
	if last == nil {
		last = errors.New("no version reported")
	}
	return fmt.Errorf("docker daemon is not reachable: %w", last)
}

func (o Options) buildImage(ctx context.Context) error {
	o.step("build base image %s", baseImage)
	return o.run(ctx, o.Root, nil, "docker", "build", "-t", baseImage, filepath.Join(o.Root, "images", "base"))
}

func (o Options) buildServer(ctx context.Context) error {
	arch, err := o.dockerArch(ctx)
	if err != nil {
		return err
	}
	o.step("build linux/%s server", arch)
	out := filepath.Join(o.Root, "dist", "server")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, "./cmd/server")
	cmd.Dir = o.Root
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
	stdout, stderr := o.writers()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	return nil
}

func (o Options) dockerArch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Arch}}")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker version: %w", err)
	}
	switch strings.TrimSpace(string(out)) {
	case "arm64", "aarch64":
		return "arm64", nil
	case "amd64", "x86_64":
		return "amd64", nil
	default:
		return "", fmt.Errorf("unsupported docker architecture %q", strings.TrimSpace(string(out)))
	}
}

func (o Options) writeInventory(ctx context.Context) (inventory.Vars, error) {
	o.step("render ansible inventory for %s", o.Env)
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = o.envDir()
	cmd.Stdout = &buf
	_, stderr := o.writers()
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return inventory.Vars{}, fmt.Errorf("terraform output: %w", err)
	}
	inv, vars, err := inventory.ParseTerraformOutput(buf.Bytes())
	if err != nil {
		return inventory.Vars{}, err
	}
	vars.ServerBinary = filepath.Join(o.Root, "dist", "server")
	vars.AppListenPort = appListenPort
	ini, err := inventory.RenderINI(inv, vars.SSHPrivateKeyPath)
	if err != nil {
		return inventory.Vars{}, err
	}
	extra, err := inventory.RenderExtraVars(vars)
	if err != nil {
		return inventory.Vars{}, err
	}
	dir := o.localDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return inventory.Vars{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "inventory.ini"), []byte(ini), 0o600); err != nil {
		return inventory.Vars{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "extra-vars.json"), extra, 0o600); err != nil {
		return inventory.Vars{}, err
	}
	return vars, nil
}

func (o Options) installCollections(ctx context.Context) error {
	o.step("install ansible collections")
	return o.run(ctx, o.Root, o.ansibleEnv(), "ansible-galaxy", "collection", "install", "-r",
		filepath.Join(o.Root, "ansible", "requirements.yml"),
		"-p", filepath.Join(o.Root, "ansible", "collections"))
}

func (o Options) playbook(ctx context.Context) ([]byte, error) {
	o.step("ansible-playbook %s", o.Env)
	inv := filepath.Join(o.localDir(), "inventory.ini")
	extra := filepath.Join(o.localDir(), "extra-vars.json")
	args := []string{"-i", inv, filepath.Join(o.Root, "ansible", "playbooks", "site.yml"), "-e", "@" + extra}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "ansible-playbook", args...)
	cmd.Dir = o.Root
	cmd.Env = o.ansibleEnv()
	stdout, stderr := o.writers()
	cmd.Stdout = io.MultiWriter(stdout, &buf)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return buf.Bytes(), fmt.Errorf("ansible-playbook: %w", err)
	}
	return buf.Bytes(), nil
}

func (o Options) ansibleEnv() []string {
	return append(os.Environ(),
		"ANSIBLE_CONFIG="+filepath.Join(o.Root, "ansible", "ansible.cfg"),
		"ANSIBLE_ROLES_PATH="+filepath.Join(o.Root, "ansible", "roles"),
		"ANSIBLE_COLLECTIONS_PATH="+filepath.Join(o.Root, "ansible", "collections"),
		"ANSIBLE_COLLECTIONS_PATHS="+filepath.Join(o.Root, "ansible", "collections"),
		"ANSIBLE_HOST_KEY_CHECKING=False",
	)
}

func (o Options) terraform(ctx context.Context, args ...string) error {
	o.step("terraform %s", strings.Join(args, " "))
	return o.run(ctx, o.envDir(), nil, "terraform", args...)
}

func (o Options) terraformApplyNoChanges(ctx context.Context) error {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "terraform", "apply", "-auto-approve", "-input=false")
	cmd.Dir = o.envDir()
	stdout, stderr := o.writers()
	cmd.Stdout = io.MultiWriter(stdout, &buf)
	cmd.Stderr = io.MultiWriter(stderr, &buf)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform apply: %w", err)
	}
	text := buf.String()
	if !strings.Contains(text, "0 added, 0 changed, 0 destroyed") {
		return fmt.Errorf("second terraform apply changed infrastructure")
	}
	return nil
}

func (o Options) run(ctx context.Context, dir string, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	stdout, stderr := o.writers()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && name == "terraform" && exit.ExitCode() == 2 {
			return fmt.Errorf("%s exited 2", name)
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func waitHTTP(ctx context.Context, url, want string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(60 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		resp, err := client.Get(url)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && string(body) == want {
				return nil
			}
			last = fmt.Errorf("status %d body %q", resp.StatusCode, bytes.TrimSpace(body))
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("GET %s: %w", url, last)
}

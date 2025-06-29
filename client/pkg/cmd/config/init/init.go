package initcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"strings"
	"time"
)

func NewConfigInitCmd(svc api.Pinger) *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Set sarabi configuration",
		Long:    "Set sarabi configuration",
		Example: "sarabi config init --path <config path>",
		Run: func(cmd *cobra.Command, args []string) {
			_, err := os.Stat(path)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()

			if err := runInitSarabi(ctx, path, svc); err != nil {
				cmdutil.PrintE(err.Error())
			}

		},
	}
	cmd.Flags().StringVarP(&path, "path", "p", "", "sarabi configuration path")
	return cmd
}

func toURL(u *url.URL) string {
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return u.String()
}

func runInitSarabi(ctx context.Context, path string, svc api.Pinger) error {
	cfg, err := parseConfig(path)
	if err != nil {
		return configErr(err)
	}

	g, ctx := errgroup.WithContext(ctx)
	// download sarabi installation script from github
	raw, err := http.Get("https://raw.githubusercontent.com/adxgun/sarabi/refs/heads/master/install.sh")
	if err != nil {
		return configErr(err)
	}

	body, err := io.ReadAll(raw.Body)
	if raw.StatusCode < 200 || raw.StatusCode >= 300 {
		return configErr(fmt.Errorf("failed to download sarabi installation script: code=%d body=%s", raw.StatusCode, string(body)))
	}

	script := string(body)
	for _, vm := range cfg.Servers {
		g.Go(func() error {
			return sshInstallSarabi(vm, script)
		})
	}

	if err := g.Wait(); err != nil {
		return configErr(err)
	}

	ctx = context.Background()
	for _, vm := range cfg.Servers {
		addr := fmt.Sprintf("http://%s:3646/", vm.IP)
		err = svc.Ping(ctx, addr, "ping")
		if err != nil {
			cmdutil.PrintE(fmt.Errorf("ping %s failed: %v", addr, err).Error())
		}
	}

	return nil
}

func runInit(ctx context.Context, configPath string) error {
	cfg, err := parseConfig(configPath)
	if err != nil {
		return configErr(err)
	}

	// install docker
	g, ctx := errgroup.WithContext(ctx)
	for _, vm := range cfg.Servers {
		g.Go(func() error {
			return sshInstallDocker(vm)
		})
	}

	if err := g.Wait(); err != nil {
		return configErr(err)
	}

	// install Portworx
	for i, vm := range cfg.Servers {
		g.Go(func() error {
			clusterID := fmt.Sprintf("sarabi-%s-%d", vm.Mode.String(), i)
			kvStoreEndpoint := "kvdb:///var/lib/osd/kvdb"
			return sshInstallPortworx(clusterID, kvStoreEndpoint, vm)
		})
	}

	if err := g.Wait(); err != nil {
		return configErr(err)
	}

	var manager *Server
	for _, vm := range cfg.Servers {
		if vm.Mode == Manager {
			manager = &vm
			break
		}
	}

	// assign a random manager if none is specified.
	if manager == nil {
		manager = &cfg.Servers[0]
		manager.Mode = Manager
	}

	// setup manager.
	err = sshSetupDockerSwarm(ctx, *manager, Server{})
	if err != nil {
		return configErr(err)
	}

	// setup workers
	for _, vm := range cfg.Servers {
		g.Go(func() error {
			return sshSetupDockerSwarm(ctx, *manager, vm)
		})
	}

	if err := g.Wait(); err != nil {
		return configErr(err)
	}

	// download sarabi installation script from github
	raw, err := http.Get("https://raw.githubusercontent.com/adxgun/sarabi/refs/heads/master/install.sh")
	if err != nil {
		return configErr(err)
	}

	body, err := io.ReadAll(raw.Body)
	if raw.StatusCode < 200 || raw.StatusCode >= 300 {
		return configErr(fmt.Errorf("failed to download sarabi installation script: code=%d body=%s", raw.StatusCode, string(body)))
	}

	script := string(body)
	for _, vm := range cfg.Servers {
		g.Go(func() error {
			return sshInstallSarabi(vm, script)
		})
	}

	if err := g.Wait(); err != nil {
		return configErr(err)
	}

	return nil
}

func sshInstallSarabi(vm Server, script string) error {
	sout, serr, err := executeSSHCommand(vm, script, "Installing sarabi...")
	if err != nil {
		return configErr(err)
	}

	if len(strings.TrimSpace(sout)) == 0 {
		return configErr(fmt.Errorf("failed to install sarabi installation: %s", serr))
	}

	cmdutil.Printf("[%s] sarabi server installation completed: %s", vm.IP, sout)
	return nil
}

func sshInstallPortworx(clusterID, kvStoreEndpoint string, vm Server) error {
	cmdutil.Printf("--- Starting Portworx installation on %s (%s) ---\n", vm.IP, vm.Mode.String())
	if isPortworxInstalled(vm) {
		cmdutil.Printf("[%s] Portworx is already installed and operational", vm.IP)
		return nil
	}

	// 1. Install NTP (Network Time Protocol) for time synchronization
	cleanPackageManager := `#!/bin/bash

set -e

echo "🔄 Cleaning apt cache..."
sudo apt clean

echo "📦 Updating package info..."
sudo apt update

echo "🛠 Attempting to fix broken dependencies..."
sudo apt install -f -y || true

echo "🔍 Checking for held packages..."
HELD_PACKAGES=$(dpkg --get-selections | grep hold | awk '{print $1}')
if [ -n "$HELD_PACKAGES" ]; then
  echo "🚫 Found held packages: $HELD_PACKAGES"
  for pkg in $HELD_PACKAGES; do
    echo "❎ Unholding $pkg..."
    sudo apt-mark unhold "$pkg"
  done
else
  echo "✅ No held packages found."
fi

echo "⬆️ Attempting to upgrade packages..."
sudo apt upgrade -y || true

echo "📦 Installing aptitude (smarter resolver)..."
sudo apt install aptitude -y

echo "🧠 Running aptitude to fix unresolved issues..."
sudo aptitude safe-upgrade -y || true

echo "🧹 Cleaning up unused packages..."
sudo apt autoremove -y

echo "✅ Done! Your system packages should be fixed and up to date."
`
	cmdutil.Printf("[%s] Installing NTP...\n", vm.IP)
	_, stderr, err := executeSSHCommand(vm, cleanPackageManager, "Cleaning package manager")
	if err != nil {
		return configErr(err)
	}

	_, stderr, err = executeSSHCommand(vm, "sudo apt clean && sudo apt update && sudo apt install linux-headers-$(uname -r) && sudo apt install -y ntp", "Installing NTP")
	if err != nil {
		return configErr(fmt.Errorf("failed to install NTP on %s: %v, Stderr: %s", vm.IP, err, stderr))
	}
	cmdutil.Printf("[%s] NTP installed.\n", vm.IP)

	// 2. Install necessary Docker modules (if not already enabled)
	cmdutil.Printf("[%s] Enabling Docker kernel modules...\n", vm.IP)
	_, stderr, err = executeSSHCommand(vm, `
        sudo modprobe overlay
        sudo modprobe br_netfilter
        echo "net.bridge.bridge-nf-call-ip6tables = 1" | sudo tee -a /etc/sysctl.conf
        echo "net.bridge.bridge-nf-call-iptables = 1" | sudo tee -a /etc/sysctl.conf
        echo "net.ipv4.ip_forward = 1" | sudo tee -a /etc/sysctl.conf
        sudo sysctl --system
    `, "Enabling Docker kernel modules")
	if err != nil {
		return configErr(fmt.Errorf("failed to enable kernel modules on %s: %v, Stderr: %s", vm.IP, err, stderr))
	}
	cmdutil.Printf("[%s] Docker kernel modules enabled.\n", vm.IP)

	// 3. Install Portworx OCI bundle (px-runc)
	cmdutil.Printf("[%s] Installing Portworx OCI bundle...\n", vm.IP)

	// Get the latest stable release link from Portworx install page (adjust REL if needed)
	getBundleCmd := `
        REL="/2.13"
        LATEST_STABLE=$(curl -fsSL "https://install.portworx.com$REL/?type=dock&stork=false&aut=false" | awk '/image: / {print $2}' | head -1)
        sudo docker run --rm --entrypoint /runc-entry-point.sh \
            --rm -i --privileged=true \
            -v /opt/pwx:/opt/pwx -v /etc/pwx:/etc/pwx \
            $LATEST_STABLE \
			--upgrade
    `
	_, stderr, err = executeSSHCommand(vm, getBundleCmd, "Downloading Portworx bundle")
	if err != nil {
		return configErr(fmt.Errorf("failed to install Portworx OCI bundle on %s: %v, Stderr: %s", vm.IP, err, stderr))
	}
	cmdutil.Printf("[%s] Portworx OCI bundle installed.\n", vm.IP)

	// 4. Run the Portworx installer
	cmdutil.Printf("[%s] Running Portworx installer...\n", vm.IP)
	installCmd := fmt.Sprintf(`
        sudo /opt/pwx/bin/px-runc install \
            -c "%s" \
            -k "%s" \
            -s "%s" \
			-b \
  			-kvdb_cluster_size 3
    `, clusterID, kvStoreEndpoint, vm.BlockDevice)

	_, stderr, err = executeSSHCommand(vm, installCmd, "Installing Portworx")
	if err != nil {
		return configErr(fmt.Errorf("failed to run Portworx installer on %s: %v, Stderr: %s", vm.IP, err, stderr))
	}
	cmdutil.Printf("[%s] Portworx installer started.\n", vm.IP)

	// 5. Verify Portworx status (wait for it to become operational)
	cmdutil.Printf("[%s] Waiting for Portworx to become operational...\n", vm.IP)
	checkStatusCmd := "sudo /opt/pwx/bin/pxctl status | grep 'PX is operational'"
	for i := 0; i < 10; i++ { // Try for 100 seconds (10 * 10s sleep)
		stdout, _, err := executeSSHCommand(vm, checkStatusCmd, "Checking Portworx status")
		if err == nil && strings.Contains(stdout, "PX is operational") {
			cmdutil.Printf("[%s] Portworx is operational.\n", vm.IP)
			return nil
		}
		cmdutil.Printf("[%s] Portworx not yet operational, retrying... (Attempt %d)\n", vm.IP, i+1)
		time.Sleep(10 * time.Second)
	}

	return configErr(fmt.Errorf("Portworx did not become operational on %s within timeout", vm.IP))
}

func isPortworxInstalled(vm Server) bool {
	checkStatusCmd := "sudo /opt/pwx/bin/pxctl status | grep 'PX is operational'"
	stdout, _, err := executeSSHCommand(vm, checkStatusCmd, "Checking Portworx status")
	return err == nil && strings.Contains(stdout, "PX is operational")
}

func sshInstallDocker(vm Server) error {
	cmdDockerInstall := `
	if ! command -v docker &> /dev/null; then
    echo "Docker not found. Installing Docker..."
    sudo apt update -y
    sudo apt install -y docker.io
    sudo systemctl enable docker
    sudo systemctl start docker
    echo "Docker installed and started successfully."
  else
    echo "Docker is already installed."
    sudo systemctl enable docker &> /dev/null || true
    sudo systemctl start docker &> /dev/null || true
    echo "Ensured Docker service is enabled and running."
  fi
`
	sout, serr, err := executeSSHCommand(vm, cmdDockerInstall, "Installing Docker")
	if err != nil {
		return configErr(err)
	}

	if len(strings.TrimSpace(serr)) != 0 {
		return configErr(errors.New(serr))
	}

	cmdutil.Print(fmt.Sprintf("[%s] Docker installed: %s", vm.IP, sout))
	return nil
}

func sshSetupDockerSwarm(ctx context.Context, manager, vm Server) error {
	dockercli, err := createDockerClientOverSSH(vm)
	if err != nil {
		return err
	}

	info, err := dockercli.Info(ctx)
	if err != nil {
		return err
	}

	if info.Swarm.LocalNodeState == "active" || info.Swarm.LocalNodeState == "inactive" {
		if vm.Mode == Manager {
			cmdutil.Printf("[%s] Swarm is already initialised.\n", vm.IP)
		} else {
			cmdutil.Printf("[%s] Already part of a swarm.\n", vm.IP)
		}
		return nil
	}

	if vm.Mode == Manager {
		return createManagerNode(ctx, vm, dockercli)
	} else {
		return createWorkNode(ctx, manager, vm, dockercli)
	}
}

func createWorkNode(ctx context.Context, manager, vm Server, dockercli *client.Client) error {
	if manager.IP == vm.IP {
		return nil
	}

	info, err := dockercli.SwarmInspect(ctx)
	if err != nil {
		return err
	}

	joinToken := info.JoinTokens.Worker
	managerAddr := manager.SwarmListenAddr()
	joinReq := swarm.JoinRequest{
		ListenAddr:    vm.SwarmListenAddr(),
		AdvertiseAddr: vm.AdsAddr(),
		RemoteAddrs:   []string{managerAddr},
		JoinToken:     joinToken,
	}
	if err = dockercli.SwarmJoin(ctx, joinReq); err != nil {
		return err
	}

	cmdutil.Print(fmt.Sprintf("[%s] Swarm joined successfully.", vm.IP))
	return nil
}

func createManagerNode(ctx context.Context, vm Server, dockercli *client.Client) error {
	req := swarm.InitRequest{
		ListenAddr:    vm.SwarmListenAddr(),
		AdvertiseAddr: vm.AdsAddr(),
	}
	id, err := dockercli.SwarmInit(ctx, req)
	if err != nil {
		cmdutil.Printf("[%s] Failed to initialize Swarm docker service.\n", vm.IP)
		return err
	}

	cmdutil.Printf("[%s] Swarm docker service initialized. ID=%s\n ", vm.IP, id)
	return nil
}

func createDockerClientOverSSH(vm Server) (*client.Client, error) {
	sshConfig, err := prepareDockerSSHConfig(vm)
	if err != nil {
		return nil, err
	}

	sshClient, err := ssh.Dial("tcp", vm.SSHConnectionString(), &sshConfig)
	if err != nil {
		return nil, err
	}

	dial := func(network, addr string) (net.Conn, error) {
		return sshClient.Dial("unix", "/var/run/docker.sock")
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
				return dial(network, addr)
			},
		},
	}

	return client.NewClientWithOpts(
		client.WithHTTPClient(httpClient),
		client.WithHost("http://docker"), // dummy, just to initialize
		client.WithAPIVersionNegotiation(),
	)
}

func configErr(err error) error {
	return err
}

func parseConfig(path string) (Config, error) {
	fi, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}

	content, err := io.ReadAll(fi)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func executeSSHCommand(vm Server, cmd, title string) (string, string, error) {
	var authMethods []ssh.AuthMethod
	if vm.Auth.Type == "ssh" {
		key, err := os.ReadFile(vm.Auth.Path)
		if err != nil {
			return "", "", fmt.Errorf("unable to read private key: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return "", "", fmt.Errorf("unable to parse private key: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else if vm.Auth.Type == "password" {
		authMethods = append(authMethods, ssh.Password(vm.Auth.Password))
	} else {
		return "", "", fmt.Errorf("no authentication method provided for %s", vm.IP)
	}

	sshConfig := &ssh.ClientConfig{
		User:            vm.Auth.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	cmdutil.Printf("[%s] Connecting: %s\n", vm.IP, vm.SSHConnectionString())
	cli, err := ssh.Dial("tcp", vm.SSHConnectionString(), sshConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to dial %s: %v", vm.IP, err)
	}

	defer func() {
		_ = cli.Close()
	}()

	session, err := cli.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create session: %v", err)
	}

	defer func() {
		_ = session.Close()
	}()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	cmdutil.Print(fmt.Sprintf("[%s] %s \n", vm.IP, title))
	err = session.Run(cmd)
	if err != nil {
		return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("command failed on %s: %s, Error: %v", vm.IP, stderrBuf.String(), err)
	}

	return stdoutBuf.String(), stderrBuf.String(), nil
}

func prepareDockerSSHConfig(vm Server) (ssh.ClientConfig, error) {
	authMethods := make([]ssh.AuthMethod, 0)
	if vm.Auth.Type == "password" {
		authMethods = append(authMethods, ssh.Password(vm.Auth.Password))
	} else if vm.Auth.Type == "key" {
		callback := ssh.PublicKeysCallback(func() (signers []ssh.Signer, err error) {
			key, err := os.ReadFile(vm.Auth.Path)
			if err != nil {
				return nil, fmt.Errorf("unable to read private key: %v", err)
			}

			pKeys, err := ssh.ParsePrivateKey(key)
			if err != nil {
				return nil, fmt.Errorf("unable to parse private key: %v", err)
			}
			return []ssh.Signer{pKeys}, nil
		})
		authMethods = append(authMethods, callback)
	} else {
		return ssh.ClientConfig{}, fmt.Errorf("no authentication method provided for %s", vm.IP)
	}

	return ssh.ClientConfig{
		User:            vm.Auth.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}

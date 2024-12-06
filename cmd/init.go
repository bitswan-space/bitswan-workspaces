package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bitswan-space/bitswan-gitops/internal/dockercompose"
	"github.com/bitswan-space/bitswan-gitops/internal/dockerhub"
	"github.com/spf13/cobra"
)

type initOptions struct {
	remoteRepo string
	onPrem     bool
	domain    string
	certsDir  string
}

func defaultInitOptions() *initOptions {
	return &initOptions{}
}

func newInitCmd() *cobra.Command {
	o := defaultInitOptions()

	cmd := &cobra.Command{
		Use:   "init [flags] <gitops-name>",
		Short: "Initializes a new GitOps, Caddy and Bitswan editor",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  o.run,
	}

	cmd.Flags().StringVar(&o.remoteRepo, "remote", "", "The remote repository to clone")
	cmd.Flags().StringVar(&o.domain, "domain", "", "The domain to use for the Caddyfile")
	cmd.Flags().StringVar(&o.certsDir, "certs-dir", "", "The directory where the certificates are located")

	return cmd
}

func (o *initOptions) run(cmd *cobra.Command, args []string) error {
	bitswanConfig := os.Getenv("HOME") + "/.config/bitswan/"

	if _, err := os.Stat(bitswanConfig); os.IsNotExist(err) {
		if err := os.MkdirAll(bitswanConfig, 0755); err != nil {
			return fmt.Errorf("failed to create BitSwan config directory: %w", err)
		}
	}
	
	gitopsName := "gitops"
	if len(args) == 1 {
		gitopsName = args[0]
	}

	gitopsConfig := bitswanConfig + gitopsName


	if _, err := os.Stat(gitopsConfig); !os.IsNotExist(err) {
		return fmt.Errorf("GitOps with this name was already initialized: %s", gitopsName)
	}

	if err := os.MkdirAll(gitopsConfig, 0755); err != nil {
		return fmt.Errorf("failed to create GitOps directory: %w", err)
	}



	// Initialize Bitswan workspace
	gitopsWorkspace := gitopsConfig + "/workspace"
	if o.remoteRepo != "" {
		com := exec.Command("git", "clone", o.remoteRepo, gitopsWorkspace)

		fmt.Println("Cloning remote repository...")
		if err := com.Run(); err != nil {
			return fmt.Errorf("Failed to clone remote repository: %w", err)
		}
		fmt.Println("Remote repository cloned!")
	} else {

		os.Mkdir(gitopsWorkspace, 0755)
		com := exec.Command("git", "init")
		com.Dir = gitopsWorkspace

		fmt.Println("Initializing git in workspace...")
		if err := com.Run(); err != nil {
			return fmt.Errorf("Failed to init git in workspace: %w", err)
		}
		fmt.Println("Git initialized in workspace!")
	}

	// Add GitOps worktree
	gitopsWorktree := gitopsConfig + "/gitops"
	worktreeAddCom := exec.Command("git", "worktree", "add", "--orphan", "-b", gitopsName, gitopsWorktree)
	worktreeAddCom.Dir = gitopsWorkspace

	fmt.Println("Setting up GitOps worktree...")
	if err := worktreeAddCom.Run(); err != nil {
		return fmt.Errorf("failed to create GitOps worktree: %w", err)
	}

	if o.remoteRepo != "" {
		// Create empty commit
		emptyCommitCom := exec.Command("git", "commit", "--allow-empty", "-m", "Initial commit")
		emptyCommitCom.Dir = gitopsWorktree
		if err := emptyCommitCom.Run(); err != nil {
			return fmt.Errorf("failed to create empty commit: %w", err)
		}

		// Push to remote
		setUpstreamCom := exec.Command("git", "push", "-u", "origin", gitopsName)
		setUpstreamCom.Dir = gitopsWorktree
		if err := setUpstreamCom.Run(); err != nil {
			return fmt.Errorf("failed to set upstream: %w", err)
		}
	}	

	fmt.Println("GitOps worktree set up successfully!")


	gitopsLatestVersion, err := dockerhub.GetLatestBitswanGitopsVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest BitSwan GitOps version: %w", err)
	}

	bitswanEditorLatestVersion, err := dockerhub.GetLatestBitswanEditorVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest BitSwan Editor version: %w", err)
	}

	
	createDockerNetworkCom := exec.Command("docker", "network", "create", "bitswan_network")

	fmt.Println("Creating BitSwan Docker network...")
	createDockerNetworkCom.Run()
	fmt.Println("BitSwan Docker network created!")


	fmt.Println("Setting up GitOps deployment...")
	gitopsDeployment := gitopsConfig + "/deployment"
	if err := os.MkdirAll(gitopsDeployment, 0755); err != nil {
		return fmt.Errorf("Failed to create deployment directory: %w", err)
	}

	var caddyfile string
	
	if o.certsDir != "" {
		caddyfile = fmt.Sprintf(`
		gitops.%s {
			reverse_proxy gitops:8079
			tls /tls/full-chain.pem /tls/private-key.pem
		}
		editor.%s {
			reverse_proxy bitswan-editor:8080
			tls /tls/full-chain.pem /tls/private-key.pem
		}`, o.domain, o.domain)
	} else {
		caddyfile = fmt.Sprintf(`
		gitops.%s {
			reverse_proxy gitops:8079
		}
		editor.%s {
			reverse_proxy bitswan-editor:8080
		}`, o.domain, o.domain)
		}
	
	if err := os.MkdirAll(gitopsDeployment + "/caddy", 0755); err != nil {
		return fmt.Errorf("Failed to create Caddy data directory: %w", err)
	}
	caddyfilePath := gitopsDeployment + "/caddy/Caddyfile"
	if err := os.WriteFile(caddyfilePath, []byte(caddyfile), 0644); err != nil {
		return fmt.Errorf("Failed to write Caddyfile: %w", err)
	}

	compose, err := dockercompose.CreateDockerComposeFile(
		gitopsConfig,
		gitopsName,
		gitopsLatestVersion,
		bitswanEditorLatestVersion,
		o.certsDir,
	)
	if err != nil {
		return fmt.Errorf("Failed to create docker-compose file: %w", err)
	}

	dockerComposePath := gitopsDeployment + "/docker-compose.yml"
	if err := os.WriteFile(dockerComposePath, []byte(compose), 0644); err != nil {
		return fmt.Errorf("Failed to write docker-compose file: %w", err)
	}

	err = os.Chdir(gitopsDeployment)
	if err != nil {
		return fmt.Errorf("Failed to change directory to GitOps deployment: %w", err)
	}

	fmt.Println("GitOps deployment set up successfully!")

	projectName := gitopsName + "-site"
	dockerComposeCom := exec.Command("docker-compose", "-p", projectName, "up", "-d")

	fmt.Println("Starting BitSwan GitOps...")
	if err := dockerComposeCom.Run(); err != nil {
		return fmt.Errorf("failed to start docker-compose: %w", err)
	}

	fmt.Println("BitSwan GitOps initialized successfully!")
	return nil
}
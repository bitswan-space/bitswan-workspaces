package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"io"
	"sync"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/bitswan-space/bitswan-workspaces/internal/caddyapi"
	"github.com/bitswan-space/bitswan-workspaces/internal/dockercompose"
	"github.com/bitswan-space/bitswan-workspaces/internal/dockerhub"
	"github.com/spf13/cobra"
)

type initOptions struct {
	remoteRepo  string
	domain      string
	certsDir    string
	verbose     bool
	mkCerts     bool
	noIde       bool
	setHosts    bool
	local       bool
	gitopsImage string
	editorImage string
}

type DockerNetwork struct {
	Name      string `json:"Name"`
	ID        string `json:"ID"`
	CreatedAt string `json:"CreatedAt"`
	Driver    string `json:"Driver"`
	IPv6      string `json:"IPv6"`
	Internal  string `json:"Internal"`
	Labels    string `json:"Labels"`
	Scope     string `json:"Scope"`
}

func defaultInitOptions() *initOptions {
	return &initOptions{}
}

func newInitCmd() *cobra.Command {
	o := defaultInitOptions()

	cmd := &cobra.Command{
		Use:   "init [flags] <workspace-name>",
		Short: "Initializes a new GitOps, Caddy and Bitswan editor",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  o.run,
	}

	cmd.Flags().StringVar(&o.remoteRepo, "remote", "", "The remote repository to clone")
	cmd.Flags().StringVar(&o.domain, "domain", "", "The domain to use for the Caddyfile")
	cmd.Flags().StringVar(&o.certsDir, "certs-dir", "", "The directory where the certificates are located")
	cmd.Flags().BoolVar(&o.noIde, "no-ide", false, "Do not start Bitswan Editor")
	cmd.Flags().BoolVarP(&o.verbose, "verbose", "v", false, "Verbose output")
	cmd.Flags().BoolVar(&o.mkCerts, "mkcerts", false, "Automatically generate local certificates using the mkcerts utility")
	cmd.Flags().BoolVar(&o.setHosts, "set-hosts", false, "Automatically set hosts to /etc/hosts file")
	cmd.Flags().BoolVar(&o.local, "local", false, "Automatically use flag --set-hosts and --mkcerts")
	cmd.Flags().StringVar(&o.gitopsImage, "gitops-image", "", "Custom image for the gitops")
	cmd.Flags().StringVar(&o.editorImage, "editor-image", "", "Custom image for the editor")

	return cmd
}


func checkNetworkExists(networkName string) (bool, error) {
	// Run docker network ls command with JSON format
	cmd := exec.Command("docker", "network", "ls", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("error running docker command: %v", err)
	}

	// Split output into lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Process each line
	for _, line := range lines {
		var network DockerNetwork
		if err := json.Unmarshal([]byte(line), &network); err != nil {
			return false, fmt.Errorf("error parsing JSON: %v", err)
		}

		if network.Name == networkName {
			return true, nil
		}
	}

	return false, nil
}

func runCommandVerbose(cmd *exec.Cmd, verbose bool) error {
    var stdoutBuf, stderrBuf bytes.Buffer

    if verbose {
        // Set up pipes for real-time streaming
        stdoutPipe, err := cmd.StdoutPipe()
        if err != nil {
            return fmt.Errorf("failed to create stdout pipe: %w", err)
        }

        stderrPipe, err := cmd.StderrPipe()
        if err != nil {
            return fmt.Errorf("failed to create stderr pipe: %w", err)
        }

        // Create multi-writers to both stream and capture output
        stdoutWriter := io.MultiWriter(os.Stdout, &stdoutBuf)
        stderrWriter := io.MultiWriter(os.Stderr, &stderrBuf)

        // Start the command
        if err := cmd.Start(); err != nil {
            return fmt.Errorf("failed to start command: %w", err)
        }

        // Copy stdout and stderr in separate goroutines
        var wg sync.WaitGroup
        wg.Add(2)

        go func() {
            defer wg.Done()
            io.Copy(stdoutWriter, stdoutPipe)
        }()

        go func() {
            defer wg.Done()
            io.Copy(stderrWriter, stderrPipe)
        }()

        // Wait for all output to be processed
        wg.Wait()

        // Wait for command to complete
        err = cmd.Wait()
        return err
    } else {
        // Not verbose, just capture output for potential error reporting
        cmd.Stdout = &stdoutBuf
        cmd.Stderr = &stderrBuf

        err := cmd.Run()

        // If command failed, print the captured output
        if err != nil {
            if stdoutBuf.Len() > 0 {
                fmt.Println("Command stdout:")
                fmt.Println(stdoutBuf.String())
            }

            if stderrBuf.Len() > 0 {
                fmt.Println("Command stderr:")
                fmt.Println(stderrBuf.String())
            }
        }

        return err
    }
}


// EnsureExamples clones the BitSwan repository if it doesn't exist,
// or updates it if it already exists
func EnsureExamples(bitswanConfig string, verbose bool) error {
	repoURL := "https://github.com/bitswan-space/BitSwan.git"
	targetDir := filepath.Join(bitswanConfig, "bitswan-src")

	// Check if the directory exists and contains a git repository
	if _, err := os.Stat(filepath.Join(targetDir, ".git")); os.IsNotExist(err) {
		// Directory doesn't exist or is not a git repo, clone it
		if verbose {
			fmt.Printf("Cloning BitSwan repository to %s\n", targetDir)
		}

		// Create parent directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		cmd := exec.Command("git", "clone", repoURL, targetDir)
		if err := runCommandVerbose(cmd, verbose); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}

		if verbose {
			fmt.Println("Repository cloned successfully")
		}
	} else {
		// Directory exists and is a git repo, update it
		if err := UpdateExamples(bitswanConfig, verbose); err != nil {
			return err
		}
	}

	return nil
}

// UpdateExamples performs a git pull on the repository
func UpdateExamples(bitswanConfig string, verbose bool) error {

	repoPath := filepath.Join(bitswanConfig, "bitswan-src")
	if verbose {
		fmt.Printf("Updating BitSwan repository at %s\n", repoPath)
	}

	cmd := exec.Command("git", "pull")
	cmd.Dir = repoPath

	if err := runCommandVerbose(cmd, verbose); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	if verbose {
		fmt.Println("Repository updated successfully")
	}
	return nil
}

func generateWildcardCerts(domain string) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "certs-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Store current working directory
	originalDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		return "", fmt.Errorf("failed to change to temp directory: %w", err)
	}

	// Ensure we change back to original directory when function returns
	defer os.Chdir(originalDir)

	// Generate wildcard certificate
	wildcardDomain := "*." + domain
	cmd := exec.Command("mkcert", wildcardDomain)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to generate certificate: %w", err)
	}

	// Generate file names
	keyFile := fmt.Sprintf("_wildcard.%s-key.pem", domain)
	certFile := fmt.Sprintf("_wildcard.%s.pem", domain)

	// Rename files
	if err := os.Rename(keyFile, "private-key.pem"); err != nil {
		return "", fmt.Errorf("failed to rename key file: %w", err)
	}
	if err := os.Rename(certFile, "full-chain.pem"); err != nil {
		return "", fmt.Errorf("failed to rename cert file: %w", err)
	}

	return tempDir, nil
}

func setHosts(workspaceName string, o *initOptions) error {
	fmt.Println("Checking if the user has permission to write to /etc/hosts...")
	fileInfo, err := os.Stat("/etc/hosts")
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	// Check if the current user can write to the file
	if fileInfo.Mode().Perm()&0200 == 0 {
		return fmt.Errorf("user does not have permission to write to /etc/hosts")
	}
	fmt.Println("File /etc/hosts is writable")

	hostsEntries := []string{
		"127.0.0.1 " + workspaceName + "-gitops.bitswan.local",
	}

	if !o.noIde {
		hostsEntries = append(hostsEntries, "127.0.0.1 "+workspaceName+"-editor.bitswan.local")
	}

	// Check if the entries already exist in /etc/hosts
	for _, entry := range hostsEntries {
		if exec.Command("grep", "-wq", entry, "/etc/hosts").Run() == nil {
			return fmt.Errorf("hosts already set in /etc/hosts")
		}
	}

	fmt.Println("Adding record to /etc/hosts...")
	for _, entry := range hostsEntries {
		cmdStr := "echo '" + entry + "' | sudo tee -a /etc/hosts"
		addHostsCom := exec.Command("sh", "-c", cmdStr)
		if err := runCommandVerbose(addHostsCom, o.verbose); err != nil {
			return fmt.Errorf("unable to write into '/etc/hosts'. \n Please add the records manually")
		}
	}

	fmt.Println("Records added to /etc/hosts successfully!")
	return nil
}

// After displaying the information, save it to metadata.yaml
func saveMetadata(gitopsConfig, workspaceName, token, domain string, noIde bool) error {
	// Create metadata structure
	type Metadata struct {
		Domain       string `yaml:"domain"`
		EditorURL    string `yaml:"editor-url,omitempty"`
		GitopsURL    string `yaml:"gitops-url"`
		GitopsSecret string `yaml:"gitops-secret"`
	}

	metadata := Metadata{
		Domain:       domain,
		GitopsURL:    fmt.Sprintf("https://%s-gitops.%s", workspaceName, domain),
		GitopsSecret: token,
	}

	// Add editor URL if IDE is enabled
	if !noIde {
		metadata.EditorURL = fmt.Sprintf("https://%s-editor.%s", workspaceName, domain)
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to file
	metadataPath := filepath.Join(gitopsConfig, "metadata.yaml")
	if err := os.WriteFile(metadataPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

func (o *initOptions) run(cmd *cobra.Command, args []string) error {
	// The first argument is the workspace name
	workspaceName := args[0]
	bitswanConfig := os.Getenv("HOME") + "/.config/bitswan/"

	if _, err := os.Stat(bitswanConfig); os.IsNotExist(err) {
		if err := os.MkdirAll(bitswanConfig, 0755); err != nil {
			return fmt.Errorf("failed to create BitSwan config directory: %w", err)
		}
	}

	// Init bitswan network
	networkName := "bitswan_network"
	exists, err := checkNetworkExists(networkName)
	if err != nil {
		panic(fmt.Errorf("Error checking network: %v\n", err))
	}

	if exists {
		fmt.Printf("Network '%s' exists\n", networkName)
	} else {
		createDockerNetworkCom := exec.Command("docker", "network", "create", "bitswan_network")
		fmt.Println("Creating BitSwan Docker network...")
		if err := runCommandVerbose(createDockerNetworkCom, o.verbose); err != nil {
			if err.Error() == "exit status 1" {
				fmt.Println("BitSwan Docker network already exists!")
			} else {
				fmt.Printf("Failed to create BitSwan Docker network: %s\n", err.Error())
			}
		} else {
			fmt.Println("BitSwan Docker network created!")
		}
	}

	// Init shared Caddy if not exists
	caddyConfig := bitswanConfig + "caddy"
	caddyCertsDir := caddyConfig + "/certs"

	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			fmt.Println("Failed to start Caddy.")
		}
	}()

	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get("http://localhost:2019")
	caddy_running := true
	if err != nil {
		caddy_running = false
	} else {
		defer resp.Body.Close()
	}

	if !caddy_running {
		fmt.Println("Setting up Caddy...")
		if err := os.MkdirAll(caddyConfig, 0755); err != nil {
			return fmt.Errorf("failed to create Caddy config directory: %w", err)
		}

		// Create Caddyfile with email and modify admin listener
		caddyfile := `
		{
			email info@bitswan.space
			admin 0.0.0.0:2019
		}`

		caddyfilePath := caddyConfig + "/Caddyfile"
		if err := os.WriteFile(caddyfilePath, []byte(caddyfile), 0755); err != nil {
			panic(fmt.Errorf("Failed to write Caddyfile: %w", err))
		}

		caddyDockerCompose, err := dockercompose.CreateCaddyDockerComposeFile(caddyConfig, o.domain)
		if err != nil {
			panic(fmt.Errorf("Failed to create Caddy docker-compose file: %w", err))
		}

		caddyDockerComposePath := caddyConfig + "/docker-compose.yml"
		if err := os.WriteFile(caddyDockerComposePath, []byte(caddyDockerCompose), 0755); err != nil {
			panic(fmt.Errorf("Failed to write Caddy docker-compose file: %w", err))
		}

		err = os.Chdir(caddyConfig)
		if err != nil {
			panic(fmt.Errorf("Failed to change directory to Caddy config: %w", err))
		}

		caddyProjectName := "bitswan-caddy"
		caddyDockerComposeCom := exec.Command("docker", "compose", "-p", caddyProjectName, "up", "-d")

		// Capture both stdout and stderr
		var stdout, stderr bytes.Buffer
		caddyDockerComposeCom.Stdout = &stdout
		caddyDockerComposeCom.Stderr = &stderr

		// Create certs directory if it doesn't exist
		if _, err := os.Stat(caddyCertsDir); os.IsNotExist(err) {
			if err := os.MkdirAll(caddyCertsDir, 0740); err != nil {
				return fmt.Errorf("failed to create Caddy certs directory: %w", err)
			}
		}

		fmt.Println("Starting Caddy...")
		if err := runCommandVerbose(caddyDockerComposeCom, o.verbose); err != nil {
			// Combine stdout and stderr for complete output
			fullOutput := stdout.String() + stderr.String()
			return fmt.Errorf("Failed to start Caddy:\nError: %v\nOutput:\n%s", err, fullOutput)
		}

		// wait 5s to make sure Caddy is up
		time.Sleep(5 * time.Second)
		err = caddyapi.InitCaddy()
		if err != nil {
			panic(fmt.Errorf("Failed to init Caddy: %w", err))
		}

		fmt.Println("Caddy started successfully!")
	} else {
		fmt.Println("A running instance of Caddy with admin found")
	}

	// Secure that --local flag is not used with --set-hosts or --mkcerts
	if o.local && (o.setHosts || o.mkCerts) {
		panic(fmt.Errorf("Cannot use --local flag with --set-hosts or --mkcerts"))
	}

	if o.local {
		o.setHosts = true
		o.mkCerts = true
	}

	inputCertsDir := o.certsDir

	if o.mkCerts {
		certDir, err := generateWildcardCerts(o.domain)
		if err != nil {
			return fmt.Errorf("Error generating certificates: %v\n", err)
		}
		inputCertsDir = certDir
	}

	if inputCertsDir != "" {
		fmt.Println("Installing certs from", inputCertsDir)
		caddyCertsDir := caddyConfig + "/certs"
		if _, err := os.Stat(caddyCertsDir); os.IsNotExist(err) {
			if err := os.MkdirAll(caddyCertsDir, 0755); err != nil {
				return fmt.Errorf("failed to create Caddy certs directory: %w", err)
			}
		}

		certsDir := caddyCertsDir + "/" + o.domain
		if _, err := os.Stat(certsDir); os.IsNotExist(err) {
			if err := os.MkdirAll(certsDir, 0755); err != nil {
				return fmt.Errorf("failed to create certs directory: %w", err)
			}
		}

		certs, err := os.ReadDir(inputCertsDir)
		if err != nil {
			panic(fmt.Errorf("Failed to read certs directory: %w", err))
		}

		for _, cert := range certs {
			if cert.IsDir() {
				continue
			}

			certPath := inputCertsDir + "/" + cert.Name()
			newCertPath := certsDir + "/" + cert.Name()

			bytes, err := os.ReadFile(certPath)
			if err != nil {
				panic(fmt.Errorf("Failed to read cert file: %w", err))
			}

			if err := os.WriteFile(newCertPath, bytes, 0755); err != nil {
				panic(fmt.Errorf("Failed to copy cert file: %w", err))
			}
		}

		fmt.Println("Certs copied successfully!")
	}

	gitopsConfig := bitswanConfig + "workspaces/" + workspaceName

	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			fmt.Println("Failed to initialize GitOps.")
		}
	}()

	if _, err := os.Stat(gitopsConfig); !os.IsNotExist(err) {
		return fmt.Errorf("GitOps with this name was already initialized: %s", workspaceName)
	}

	if err := os.MkdirAll(gitopsConfig, 0755); err != nil {
		return fmt.Errorf("failed to create GitOps directory: %w", err)
	}

	// Initialize Bitswan workspace
	gitopsWorkspace := gitopsConfig + "/workspace"
	if o.remoteRepo != "" {
		com := exec.Command("git", "clone", o.remoteRepo, gitopsWorkspace) //nolint:gosec

		fmt.Println("Cloning remote repository...")
		if err := runCommandVerbose(com, o.verbose); err != nil {
			panic(fmt.Errorf("Failed to clone remote repository: %w", err))
		}
		fmt.Println("Remote repository cloned!")
	} else {
		if err := os.Mkdir(gitopsWorkspace, 0755); err != nil {
			return fmt.Errorf("failed to create GitOps workspace directory %s: %w", gitopsWorkspace, err)
		}
		com := exec.Command("git", "init")
		com.Dir = gitopsWorkspace

		fmt.Println("Initializing git in workspace...")

		if err := runCommandVerbose(com, o.verbose); err != nil {
			panic(fmt.Errorf("Failed to init git in workspace: %w", err))
		}

		fmt.Println("Git initialized in workspace!")
	}

	// Add GitOps worktree
	gitopsWorktree := gitopsConfig + "/gitops"
	worktreeAddCom := exec.Command("git", "worktree", "add", "--orphan", "-b", workspaceName, gitopsWorktree)
	worktreeAddCom.Dir = gitopsWorkspace

	fmt.Println("Setting up GitOps worktree...")
	if err := runCommandVerbose(worktreeAddCom, o.verbose); err != nil {
		panic(fmt.Errorf("Failed to create GitOps worktree: exit code %w.", err))
	}

	// Add repo as safe directory
	safeDirCom := exec.Command("git", "config", "--global", "--add", "safe.directory", gitopsWorktree)
	if err := runCommandVerbose(safeDirCom, o.verbose); err != nil {
		panic(fmt.Errorf("Failed to add safe directory: %w", err))
	}

	if o.remoteRepo != "" {
		// Create empty commit
		emptyCommitCom := exec.Command("git", "commit", "--allow-empty", "-m", "Initial commit")
		emptyCommitCom.Dir = gitopsWorktree
		if err := runCommandVerbose(emptyCommitCom, o.verbose); err != nil {
			panic(fmt.Errorf("Failed to create empty commit: %w", err))
		}

		// Push to remote
		setUpstreamCom := exec.Command("git", "push", "-u", "origin", workspaceName)
		setUpstreamCom.Dir = gitopsWorktree
		if err := runCommandVerbose(setUpstreamCom, o.verbose); err != nil {
			panic(fmt.Errorf("Failed to set upstream: %w", err))
		}
	}

	fmt.Println("GitOps worktree set up successfully!")

	// Create secrets directory
	secretsDir := gitopsConfig + "/secrets"
	if err := os.MkdirAll(secretsDir, 0700); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}

	hostOsTmp := runtime.GOOS

	if !o.noIde {
		// Create codeserver config directory
		codeserverConfigDir := gitopsConfig + "/codeserver-config"
		if err := os.MkdirAll(codeserverConfigDir, 0700); err != nil {
			return fmt.Errorf("failed to create codeserver config directory: %w", err)
		}

		if hostOsTmp == "linux" {
			chownCom := exec.Command("sudo", "chown", "-R", "1000:1000", secretsDir)
			if err := runCommandVerbose(chownCom, o.verbose); err != nil {
				return fmt.Errorf("failed to change ownership of secrets folder: %w", err)
			}
			chownCom = exec.Command("sudo", "chown", "-R", "1000:1000", codeserverConfigDir)
			if err := runCommandVerbose(chownCom, o.verbose); err != nil {
				return fmt.Errorf("failed to change ownership of codeserver config folder: %w", err)
			}
			chownCom = exec.Command("sudo", "chown", "-R", "1000:1000", gitopsWorkspace)
			if err := runCommandVerbose(chownCom, o.verbose); err != nil {
				return fmt.Errorf("failed to change ownership of workspace folder: %w", err)
			}
		}
	}

	// Set hosts to /etc/hosts file
	if o.setHosts {
		err := setHosts(workspaceName, o)
		if err != nil {
			fmt.Printf("\033[33m%s\033[0m\n", err)
		}
	}

	gitopsImage := o.gitopsImage
	if gitopsImage == "" {
		gitopsLatestVersion, err := dockerhub.GetLatestDockerHubVersion("https://hub.docker.com/v2/repositories/bitswan/gitops/tags/")
		if err != nil {
			panic(fmt.Errorf("Failed to get latest BitSwan GitOps version: %w", err))
		}
		gitopsImage = "bitswan/gitops:" + gitopsLatestVersion
	}

	bitswanEditorImage := o.editorImage
	if bitswanEditorImage == "" {
		bitswanEditorLatestVersion, err := dockerhub.GetLatestDockerHubVersion("https://hub.docker.com/v2/repositories/bitswan/bitswan-editor/tags/")
		if err != nil {
			panic(fmt.Errorf("Failed to get latest BitSwan Editor version: %w", err))
		}
		bitswanEditorImage = "bitswan/bitswan-editor:" + bitswanEditorLatestVersion
	}

	fmt.Println("Setting up GitOps deployment...")
	gitopsDeployment := gitopsConfig + "/deployment"
	if err := os.MkdirAll(gitopsDeployment, 0755); err != nil {
		return fmt.Errorf("Failed to create deployment directory: %w", err)
	}

	err = caddyapi.AddCaddyRecords(workspaceName, o.domain, inputCertsDir != "", o.noIde)
	if err != nil {
		panic(fmt.Errorf("Failed to add Caddy records: %w", err))
	}

	err = EnsureExamples(bitswanConfig, o.verbose)
	if err != nil {
		panic(fmt.Errorf("Failed to download examples: %w", err))
	}

	compose, token, err := dockercompose.CreateDockerComposeFile(

		gitopsConfig,
		workspaceName,
		gitopsImage,
		bitswanEditorImage,
		o.domain,
		o.noIde,
	)
	if err != nil {
		panic(fmt.Errorf("Failed to create docker-compose file: %w", err))
	}

	dockerComposePath := gitopsDeployment + "/docker-compose.yml"
	if err := os.WriteFile(dockerComposePath, []byte(compose), 0755); err != nil {
		panic(fmt.Errorf("Failed to write docker-compose file: %w", err))
	}

	err = os.Chdir(gitopsDeployment)
	if err != nil {
		panic(fmt.Errorf("Failed to change directory to GitOps deployment: %w", err))
	}

	fmt.Println("GitOps deployment set up successfully!")

	projectName := workspaceName + "-site"
	dockerComposeCom := exec.Command("docker", "compose", "-p", projectName, "up", "-d")

	fmt.Println("Launching BitSwan Workspace services...")
	if err := runCommandVerbose(dockerComposeCom, true); err != nil {
		panic(fmt.Errorf("failed to start docker-compose: %w", err))
	}

	fmt.Println("BitSwan GitOps initialized successfully!")

	// Save metadata to file
	if err := saveMetadata(gitopsConfig, workspaceName, token, o.domain, o.noIde); err != nil {
		fmt.Printf("Warning: Failed to save metadata: %v\n", err)
	}

	// Get Bitswan Editor password from container
	if !o.noIde {
		// First, wait for the editor service to be ready by streaming logs
		if err := dockercompose.WaitForEditorReady(workspaceName); err != nil {
			panic(fmt.Errorf("failed to wait for editor to be ready: %w", err))
		}
		editorPassword, err := dockercompose.GetEditorPassword(workspaceName)
		if err != nil {
			panic(fmt.Errorf("Failed to get Bitswan Editor password: %w", err))
		}
		fmt.Println("------------BITSWAN EDITOR INFO------------")
		fmt.Printf("Bitswan Editor URL: https://%s-editor.%s\n", workspaceName, o.domain)
		fmt.Printf("Bitswan Editor Password: %s\n", editorPassword)
	}

	fmt.Println("------------GITOPS INFO------------")
	fmt.Printf("GitOps ID: %s\n", workspaceName)
	fmt.Printf("GitOps URL: https://%s-gitops.%s\n", workspaceName, o.domain)
	fmt.Printf("GitOps Secret: %s\n", token)

	return nil
}

package dockercompose

import (
	"bytes"
	"fmt"
	"os"
	"runtime"

	"github.com/dchest/uniuri"
	"gopkg.in/yaml.v3"
)

type OS int

const (
	WindowsMac OS = iota
	Linux
)

func CreateDockerComposeFile(gitopsPath, gitopsName, latestGitopsVersion, latestBitswanEditorVersion, certsPath string) (string, error) {
	sshDir := os.Getenv("HOME") + "/.ssh"
	gitConfig := os.Getenv("HOME") + "/.gitconfig"

	hostOsTmp := runtime.GOOS

	var hostOs OS
	switch hostOsTmp {
	case "windows", "darwin":
		hostOs = WindowsMac
	case "linux":
		hostOs = Linux
	default:
		return "", fmt.Errorf("unsupported host OS: %s", hostOsTmp)
	}

	// generate a random secret token
	gitopsSecretToken := uniuri.NewLen(64)

	gitopsService := map[string]interface{}{
		"image": "bitswan/gitops:" + latestGitopsVersion,
		"restart": "always",
		"networks": []string{"bitswan_network"},
		"volumes": []string{
			"/etc/bitswan-secrets/:/etc/bitswan-secrets/", // TODO: change this to gitops secrets
			gitopsPath + "/gitops" + ":/repo/",
			sshDir + ":/root/.ssh",
			"/var/run/docker.sock:/var/run/docker.sock",
		},
		"environment": []string{
			"BS_BITSWAN_DIR=/repo",
			"BS_HOST_DIR=" + gitopsPath + "/gitops",
			"BS_GITOPS_ID=" + gitopsName,
			"BS_GITOPS_SECRET=" + gitopsSecretToken,
		},
	}

	if hostOs == WindowsMac {
		gitopsVolumes := []string{
			gitConfig + ":/root/.gitconfig",
			gitopsPath + "/workspace/.git:/workspace-repo/.git",
		}

		gitopsService["volumes"] = append(gitopsService["volumes"].([]string), gitopsVolumes...)

		// Rewrite .git in worktree because it's calling git command inside the container (only for Windows and Mac)
		gitdir := "gitdir: /workspace-repo/.git/worktrees/gitops"
		if err := os.WriteFile(gitopsPath + "/gitops/.git", []byte(gitdir), 0644); err != nil {
			return "", fmt.Errorf("failed to rewrite gitops worktree .git file: %w", err)
		}
	} else if hostOs == Linux {
		gitopsService["privileged"] = true
		gitopsService["pid"] = "host"
		
		gitopsEnvVars := []string{
			"HOST_PATH=$PATH",
			"HOST_HOME=$HOME",
			"HOST_USER=$USER",
		}
		gitopsService["environment"] = append(gitopsService["environment"].([]string), gitopsEnvVars...)
	}

	caddyVolumes := []string{
		gitopsPath + "/deployment/caddy/Caddyfile:/etc/caddy/Caddyfile",
		gitopsPath + "/deployment/caddy/data:/data",
		gitopsPath + "/deployment/caddy/config:/config",
	}
	if certsPath != "" {
		caddyVolumes = append(caddyVolumes, certsPath + ":/tls")
	}

	// Construct the docker-compose data structure
	dockerCompose := map[string]interface{}{
		"version": "3.8",
		"services": map[string]interface{}{
			"gitops": gitopsService,
			"bitswan-editor": map[string]interface{}{
				"image": "bitswan/bitswan-editor:" + latestBitswanEditorVersion,
				"restart": "always",
				"networks": []string{"bitswan_network"},
				"environment": []string{
					"BITSWAN_DEPLOY_URL=http://gitops:8079",
					"BITSWAN_DEPLOY_SECRET=" + gitopsSecretToken, // TODO: change this for secret which is used in gitops
					"BITSWAN_GITOPS_DIR=/home/coder/workspace",
				},
				"volumes": []string{
					gitopsPath + "/workspace:/home/coder/workspace/workspace",
					gitopsPath + "/secrets:/home/coder/workspace/secrets",
					"bitswan-editor-data:/home/coder",
				},
			},
			"caddy": map[string]interface{}{
				"image": "caddy:2.9",
				"restart": "always",
				"ports": []string{"80:80", "443:443"},
				"networks": []string{"bitswan_network"},
				"volumes": caddyVolumes,
			},
		},
		"volumes": map[string]interface{}{
			"bitswan-editor-data": nil,
		},
		"networks": map[string]interface{}{
			"bitswan_network": map[string]interface{}{
				"external": true,
			},
		},
	}

	var buf bytes.Buffer

	// Serialize the docker-compose data structure to YAML and write it to the file
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2) // Optional: Set indentation
	if err := encoder.Encode(dockerCompose); err != nil {
		return "", fmt.Errorf("failed to encode docker-compose data structure: %w", err) 
	}

	return buf.String(), nil
}

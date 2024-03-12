/*
clone
------

- `bitswan-gitops clone --bitswan-dir=<bitswan-dir> <repo> <dest>(optional)`

Clone with create a directory named `dest` and then clone the git repo into a subdirectory named `prod`. It will also copy `prod` to `dev`. The prod directory is the directory from which gitops will launch services. The dev directory is a directory that launched IDEs can use to edit the source of services.

The data flow (sourcecode flow) is similar to this diagram.

```

	               -----------------
	-------------->| github/gitlab | ---------------------
	|              -----------------                     |
	|                                                    |
	| -------------------------------------------------- |
	| |        CRE (Virtual Machine or K8S cluster)    | |
	| |                                                | |
	| |     --------------         ---------------     | |
	--------| Development|         | Production  |<-------
	  |     |  git repo  |         | git repo    |     |
	  |     --------------         ---------------     |
	  |           ↑                      ↓             |
	  |     --------------         ---------------     |
	  |     | JupyterLab |         | Pipelines   |     |
	  |     |   Web IDE  |         |             |     |
	  |     --------------         ---------------     |
	  |                                                |
	  --------------------------------------------------

```
*/
package cmd

import (
	"fmt"
	"os"
	exec "os/exec"

	"github.com/bitswan-space/bitswan-gitops/internal/dockercompose"
	"github.com/bitswan-space/bitswan-gitops/internal/dockerhub"
	"github.com/dchest/uniuri"
	cp "github.com/otiai10/copy"
	"github.com/spf13/cobra"
)

type cloneOptions struct {
	creDir string
}

func defaultCloneOptions() *cloneOptions {
	return &cloneOptions{}
}

func newCloneCmd() *cobra.Command {
	o := defaultCloneOptions()

	cmd := &cobra.Command{
		Use:   "clone [flags] <repo> <dest>",
		Short: "Clone an existing bitswan-gitops repository and deploy the pipelines in it",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  o.run,
	}

	cmd.Flags().StringVar(&o.creDir, "cre-dir", "", "The directory where this cre's pipelines are found")

	return cmd
}

func (o *cloneOptions) run(cmd *cobra.Command, args []string) error {
	// Promp the user to either enter their bitswan.space gitops key or to enter "no-cloud" for standalone mode
	bitswanSpaceKey := ""
	for len(bitswanSpaceKey) < 32 {
		fmt.Println("Enter your cloud account gitops key or enter 'no-cloud' for standalone mode")
		fmt.Print("Enter cloud key [key/no-cloud/q]: ")
		fmt.Scanln(&bitswanSpaceKey)
		if bitswanSpaceKey == "no-cloud" {
			bitswanSpaceKey = ""
			break
		}
		if bitswanSpaceKey == "q" {
			return nil
		}
	}
	noCloud := bitswanSpaceKey == ""


	repoUrl := args[0]
	// create the destination directory from args[1]
	dest := "bitswan-gitops"
	if len(args) == 2 {
		dest = args[1]
	}
	// If the dest directory already exists, complain and exit
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		return fmt.Errorf("destination directory already exists: %s", dest)
	}
	os.Mkdir(dest, 0755)
	// Build path of prod subdir
	prod := dest + "/prod"
	// clone into the prod subdir of the dest directory
	com := exec.Command("git", "clone", repoUrl, prod)
	com.Stdout = os.Stdout
	com.Stderr = os.Stderr
	// Execute the command
	if err := com.Run(); err != nil {
		return fmt.Errorf("error cloning repo: %w", err)
	}

	// copy the prod directory to dev
	dev := dest + "/dev"
	// copy the prod directory to dev
	err := cp.Copy(prod, dev)
	if err != nil {
		return fmt.Errorf("error copying prod to dev: %w", err)
	}

	// Set up the gitops env file
	// Generate a secret key for the webhook
	key := uniuri.NewLen(64)
	key = "bs-secrete-webhook-key-" + key

	// Start by creating dict of env vars
	env := map[string]string{
		"BS_WEBHOOK_SECRET": key,
		"BS_CLOUD_KEY":      bitswanSpaceKey,
	}

	// Write the env vars to a file
	f, err := os.Create(dest + "/.env")
	if err != nil {
		return fmt.Errorf("error creating .env file: %w", err)
	}
	defer f.Close()
	for k, v := range env {
		_, err = f.WriteString(k + "=" + v + "\n")
		if err != nil {
			return fmt.Errorf("error writing to .env file: %w", err)
		}
	}

	// Create docker-compose.yml
	latestVersion, err := dockerhub.GetLatestBitswanGitopsVersion()
	if err != nil {
		return fmt.Errorf("error getting latest bitswan-gitops version: %w", err)
	}
	dockercompose.CreateDockerComposeFile(
		dest,
		latestVersion,
		o.creDir,
		noCloud,
	)

	// set the cwd to the dest directory and launch docker-compose
	err = os.Chdir(dest)
	if err != nil {
		return fmt.Errorf("error changing directory to dest: %w", err)
	}
	// Launch docker-compose
	com = exec.Command("docker-compose", "up", "-d")
	com.Stdout = os.Stdout
	com.Stderr = os.Stderr
	// Execute the command
	if err := com.Run(); err != nil {
		return fmt.Errorf("error launching docker-compose: %w", err)
	}

	return nil
}

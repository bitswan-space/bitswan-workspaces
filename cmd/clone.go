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
	"os"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/go-git/go-git/v5" // Import the go-git package
	"github.com/go-git/go-git/v5/plumbing"
)

type cloneOptions struct {
	bitswanDir string
}

func defaultCloneOptions() *cloneOptions {
	return &cloneOptions{}
}

func newCloneCmd() *cobra.Command {
	o := defaultCloneOptions()

	cmd := &cobra.Command{
		Use:          "clone [flags] <repo> <dest>",
		Short:        "Clone an existing bitswan-gitops repository and deploy the pipelines in it",
		Args:         cobra.RangeArgs(1, 2),
		RunE:         o.run,
	}

	cmd.Flags().StringVar(&o.bitswanDir, "bitswan-dir", "", "The directory to clone the repository into")

	return cmd
}

func (o *cloneOptions) run(cmd *cobra.Command, args []string) error {
	repoUrl := args[0]
	// create the destination directory from args[1]
	dest := "bitswan-gitops"
	if len(args) == 2 {
		dest := args[1]
	}
	os.Mkdir(dest, 0755)
	// Build path of prod subdir
	prod := dest + "/prod"
	// clone into the prod subdir of the dest directory
	_, err := git.PlainClone(prod, false, &git.CloneOptions{
		URL: repoUrl,
		Progress: os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("error cloning repository: %w", err)
	}

	// copy the prod directory to dev
	dev := dest + "/dev"
	// copy the prod directory to dev

	return nil
}

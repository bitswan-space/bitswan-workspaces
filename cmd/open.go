package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type MetadataOpen struct {
	EditorURL *string `yaml:"editor-url,omitempty"`
}

func newOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "open",
		Short:        "Open the editor for a workspace",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			gitopsPath := path.Join(os.Getenv("HOME"), "/.config/bitswan/workspaces", args[0])

			metadataPath := gitopsPath + "/metadata.yaml"

			data, err := os.ReadFile(metadataPath)
			if err != nil {
				panic(err)
			}

			var metadata MetadataOpen
			err = yaml.Unmarshal(data, &metadata)
			if err != nil {
				panic(err)
			}

			if metadata.EditorURL == nil {
				fmt.Println("No editor URL found in metadata.yaml")
				return nil
			}

			fmt.Printf("Opening editor at: %s\n", *metadata.EditorURL)
			err = exec.Command("xdg-open", *metadata.EditorURL).Start()
			if err != nil {
				panic(err)
			}

			return nil
		},
	}

	return cmd
}

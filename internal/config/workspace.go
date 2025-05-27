package config

import (
	"gopkg.in/yaml.v2"
	"os"
)

type Metadata struct {
	Domain       string `yaml:"domain"`
	EditorURL    string `yaml:"editor-url"`
	GitOpsURL    string `yaml:"gitops-url"`
	GitOpsSecret string `yaml:"gitops-secret"`
}

func GetWorkspaceMetadata(workspaceName string) Metadata {
	metadataPath := os.Getenv("HOME") + "/.config/bitswan/" + "workspaces/" + workspaceName + "/metadata.yaml"

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		panic(err)
	}

	var metadata Metadata
	err = yaml.Unmarshal(data, &metadata)
	if err != nil {
		panic(err)
	}

	return metadata
}

package dockercompose

import (
	"gopkg.in/yaml.v3"
	"os"
)

func CreateDockerComposeFile(dest, latestVersion, creDir string, noCloud bool) error {
	destFullPath := os.Getenv("PWD") + "/" + dest
	sshDir := os.Getenv("HOME") + "/.ssh"
	if creDir == "" {
		creDir = "cre-01"
	}

	// Construct the docker-compose data structure
	dockerCompose := map[string]interface{}{
		"version": "3.8",
		"services": map[string]interface{}{
			"bitswan_gitops": map[string]interface{}{
				"image": "bitswan/pipeline-runtime-environment:" + latestVersion,
				"volumes": []string{
					"/etc/bitswan-secrets/:/etc/bitswan-secrets/",
					destFullPath + "/prod:/repo/",
					sshDir + ":/root/.ssh",
					"/var/run/docker.sock:/var/run/docker.sock",
				},
				"environment": map[string]string{
					"BS_WEBHOOK_PORT": "8000",
					"BS_CRE_DIR":      "/repo/" + creDir,
					"BS_BITSWAN_DIR":  "/repo/" + creDir,
				},
				"env_file": []string{destFullPath + "/.env"},
			},
		},
	}

	if noCloud {
		addMosquitoToDockercompose(dockerCompose, destFullPath)
	}

	// Open the docker-compose.yml file for writing
	dcf, err := os.Create(dest + "/docker-compose.yml")
	if err != nil {
		return err
	}
	defer dcf.Close()

	// Serialize the docker-compose data structure to YAML and write it to the file
	encoder := yaml.NewEncoder(dcf)
	encoder.SetIndent(2) // Optional: Set indentation
	err = encoder.Encode(dockerCompose)
	if err != nil {
		return err
	}

	return nil
}

func addMosquitoToDockercompose(composeMap map[string]interface{}, dest string) {
	composeMap["services"].(map[string]interface{})["mosquitto"] = map[string]interface{}{
		"image": "eclipse-mosquitto",
		"ports": []string{"1883:1883"},
		"restart": "always",
		"volumes": []string{"mosquitto:/mosquitto", dest+"/mosquitto.conf:/mosquitto/config/mosquitto.conf"},
	}
	if _, ok := composeMap["volumes"]; !ok {
		composeMap["volumes"] = map[string]interface{}{}
	}
	composeMap["volumes"].(map[string]interface{})["mosquitto"] = map[string]interface{}{}
	mosquitoConf := `persistence true
persistence_location /mosquitto/data/
log_dest file /mosquitto/log/mosquitto.log
allow_anonymous true

# MQTT listener
listener 1883
protocol mqtt
`
	mosquittoConfFile, err := os.Create(dest + "/mosquitto.conf")
	if err != nil {
		panic(err)
	}
	defer mosquittoConfFile.Close()
	_, err = mosquittoConfFile.WriteString(mosquitoConf)
}

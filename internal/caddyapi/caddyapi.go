package caddyapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Route struct {
	ID       string   `json:"@id,omitempty"`
	Match    []Match  `json:"match"`
	Handle   []Handle `json:"handle"`
	Terminal bool     `json:"terminal"`
}

type Match struct {
	Host []string `json:"host"`
}

type Handle struct {
	Handler   string     `json:"handler"`
	Routes    []Route    `json:"routes,omitempty"`
	Upstreams []Upstream `json:"upstreams,omitempty"`
}

type Upstream struct {
	Dial string `json:"dial"`
}

type TLSPolicy struct {
	ID                   string                  `json:"@id,omitempty"`
	Match                TLSMatch                `json:"match"`
	CertificateSelection TLSCertificateSelection `json:"certificate_selection"`
}

type TLSMatch struct {
	SNI []string `json:"sni"`
}

type TLSCertificateSelection struct {
	AnyTag []string `json:"any_tag"`
}

type TLSFileLoad struct {
	ID          string   `json:"@id,omitempty"`
	Certificate string   `json:"certificate"`
	Key         string   `json:"key"`
	Tags        []string `json:"tags"`
}

func RegisterServiceWithCaddy(serviceName, workspaceName, domain, upstream string) error {
	caddyAPIRoutesBaseUrl := "http://localhost:2019/config/apps/http/servers/srv0/routes/..."

	// Create the route for the service
	route := Route{
		ID: fmt.Sprintf("%s_%s", workspaceName, serviceName),
		Match: []Match{
			{
				Host: []string{fmt.Sprintf("%s-%s.%s", workspaceName, serviceName, domain)},
			},
		},
		Handle: []Handle{
			{
				Handler: "subroute",
				Routes: []Route{
					{
						Handle: []Handle{
							{
								Handler: "reverse_proxy",
								Upstreams: []Upstream{
									{
										Dial: upstream,
									},
								},
							},
						},
					},
				},
			},
		},
		Terminal: true,
	}

	// Marshal the route into JSON
	jsonPayload, err := json.Marshal([]Route{route})
	if err != nil {
		return fmt.Errorf("failed to marshal route payload: %w", err)
	}

	// Send the payload to the Caddy API
	_, err = sendRequest("POST", caddyAPIRoutesBaseUrl, jsonPayload)
	if err != nil {
		return fmt.Errorf("failed to add %s route to Caddy: %w", serviceName, err)
	}

	return nil
}

func UnregisterCaddyService(serviceName, workspaceName string) error {
	// Construct the URL for the specific service
	url := fmt.Sprintf("http://localhost:2019/id/%s_%s", workspaceName, serviceName)

	// Send a DELETE request to the Caddy API
	if _, err := sendRequest("DELETE", url, nil); err != nil {
		return fmt.Errorf("failed to unregister Caddy service '%s': %w", serviceName, err)
	}

	fmt.Printf("Successfully unregistered Caddy service: %s\n", serviceName)
	return nil
}

func InstallTLSCerts(workspaceName, domain string) error {
	caddyAPITLSBaseUrl := "http://localhost:2019/config/apps/tls/certificates/load_files/..."
	caddyAPITLSPoliciesBaseUrl := "http://localhost:2019/config/apps/http/servers/srv0/tls_connection_policies/..."

	// Define TLS policies and certificates
	tlsPolicy := []TLSPolicy{
		{
			ID: fmt.Sprintf("%s_tlspolicy", workspaceName),
			Match: TLSMatch{
				SNI: []string{
					fmt.Sprintf("*.%s", domain),
				},
			},
			CertificateSelection: TLSCertificateSelection{
				AnyTag: []string{workspaceName},
			},
		},
	}

	tlsLoad := []TLSFileLoad{
		{
			ID:          fmt.Sprintf("%s_tlscerts", workspaceName),
			Certificate: fmt.Sprintf("/tls/%s/full-chain.pem", domain),
			Key:         fmt.Sprintf("/tls/%s/private-key.pem", domain),
			Tags:        []string{workspaceName},
		},
	}

	// Send TLS certificates to Caddy
	jsonPayload, err := json.Marshal(tlsLoad)
	if err != nil {
		return fmt.Errorf("failed to marshal TLS certificates payload: %w", err)
	}

	_, err = sendRequest("POST", caddyAPITLSBaseUrl, jsonPayload)
	if err != nil {
		return fmt.Errorf("failed to add TLS certificates to Caddy: %w", err)
	}

	// Send TLS policies to Caddy
	jsonPayload, err = json.Marshal(tlsPolicy)
	if err != nil {
		return fmt.Errorf("failed to marshal TLS policies payload: %w", err)
	}

	_, err = sendRequest("POST", caddyAPITLSPoliciesBaseUrl, jsonPayload)
	if err != nil {
		return fmt.Errorf("failed to add TLS policies to Caddy: %w", err)
	}

	fmt.Println("TLS certificates and policies installed successfully!")
	return nil
}

func InitCaddy() error {
	urls := []string{
		"http://localhost:2019/config/apps/http/servers/srv0/routes",
		"http://localhost:2019/config/apps/http/servers/srv0/listen",
		"http://localhost:2019/config/apps/tls/certificates/load_files",
		"http://localhost:2019/config/apps/http/servers/srv0/tls_connection_policies",
	}

	for idx, url := range urls {
		var payload []byte
		if idx == 1 {
			payload = []byte(`[":80", ":443"]`)
		} else {
			payload = []byte(`[]`)
		}

		if _, err := sendRequest("PUT", url, payload); err != nil {
			return fmt.Errorf("failed to initialize Caddy: %w", err)
		}
	}

	fmt.Println("Caddy initialized successfully!")
	return nil
}

func DeleteCaddyRecords(workspaceName string) error {
	services := []string{"gitops", "editor", "tlspolicy", "tlscerts"}

	for _, service := range services {
		if err := UnregisterCaddyService(service, workspaceName); err != nil {
			return fmt.Errorf("failed to delete Caddy records for service '%s': %w", service, err)
		}
	}

	return nil
}

func sendRequest(method, url string, payload []byte) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to call Caddy API: %w", err)
	}
	defer resp.Body.Close()

	if method == http.MethodDelete && (resp.StatusCode < 200 || resp.StatusCode >= 300) && resp.StatusCode != 404 {
		return nil, fmt.Errorf("Caddy API returned status code %d for DELETE request", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Caddy API returned status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

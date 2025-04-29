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

// TODO: we should think about how to handle the case when use would like to deploy GitOps on server where Caddy is already running
func AddCaddyRecords(workspaceName, domain string, certs, noIde bool) error {
	caddyAPIRoutesBaseUrl := "http://localhost:2019/config/apps/http/servers/srv0/routes/..."
	caddyAPITLSBaseUrl := "http://localhost:2019/config/apps/tls/certificates/load_files/..."
	caddyAPITLSPoliciesBaseUrl := "http://localhost:2019/config/apps/http/servers/srv0/tls_connection_policies/..."

	routes := []Route{}

	// GitOps route
	routes = append(routes, Route{
		ID:    fmt.Sprintf("%s_gitops", workspaceName),
		Match: []Match{{Host: []string{fmt.Sprintf("%s-gitops.%s", workspaceName, domain)}}},
		Handle: []Handle{{Handler: "subroute", Routes: []Route{
			{
				Handle: []Handle{{Handler: "reverse_proxy", Upstreams: []Upstream{
					{Dial: fmt.Sprintf("%s-gitops:8079", workspaceName)},
				}}},
			},
		}}},
		Terminal: true,
	})

	// Bitswan editor route
	if !noIde {
		routes = append(routes, Route{
			ID: fmt.Sprintf("%s_editor", workspaceName),
			Match: []Match{
				{
					Host: []string{fmt.Sprintf("%s-editor.%s", workspaceName, domain)},
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
											Dial: fmt.Sprintf("%s-editor:9999", workspaceName),
										},
									},
								},
							},
						},
					},
				},
			},
			Terminal: true,
		})
	}

	if certs {
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

		// send tls policy and tls to caddy api
		jsonPayload, err := json.Marshal(tlsLoad)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON payload: %w", err)
		}

		// Send the payload to the Caddy API
		_, err = sendRequest("POST", caddyAPITLSBaseUrl, jsonPayload)
		if err != nil {
			return fmt.Errorf("Failed to add TLS to Caddy: %w", err)
		}

		jsonPayload, err = json.Marshal(tlsPolicy)
		if err != nil {
			return fmt.Errorf("Failed to marshal TLS Policy JSON: %w", err)
		}

		// Send the payload to the Caddy API
		_, err = sendRequest("POST", caddyAPITLSPoliciesBaseUrl, jsonPayload)
		if err != nil {
			return fmt.Errorf("Failed to add TLS policies to Caddy: %w", err)
		}
	}

	jsonPayload, err := json.Marshal(routes)
	if err != nil {
		return fmt.Errorf("Failed to marshal routes payload: %w", err)
	}

	// Send the payload to the Caddy API
	_, err = sendRequest("POST", caddyAPIRoutesBaseUrl, jsonPayload)
	if err != nil {
		return fmt.Errorf("Failed to add routes to Caddy: %w", err)
	}

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

func DeleteCaddyRecords(gitopsName string, noIde bool) error {
	urls := []string{
		fmt.Sprintf("http://localhost:2019/id/%s_gitops", gitopsName),
		fmt.Sprintf("http://localhost:2019/id/%s_tlspolicy", gitopsName),
		fmt.Sprintf("http://localhost:2019/id/%s_tlscerts", gitopsName),
	}

	if !noIde {
		urls = append(urls, fmt.Sprintf("http://localhost:2019/id/%s_editor", gitopsName))
	}

	for _, url := range urls {
		if _, err := sendRequest("DELETE", url, nil); err != nil {
			return fmt.Errorf("failed to delete Caddy records: %w", err)
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

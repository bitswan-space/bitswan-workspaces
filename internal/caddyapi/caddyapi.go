package caddyapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"slices"
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
	Certificate string   `json:"certificate"`
	Key         string   `json:"key"`
	Tags        []string `json:"tags"`
}

// TODO: we should think about how to handle the case when use would like to deploy GitOps on server where Caddy is already running
// func AddCaddyRecords(gitopsName, domain string, certs, noIde bool) error {
// 	caddyAPIRoutesBaseUrl := "http://localhost:2019/config/apps/http/servers/srv0/routes/"
// 	caddyAPITLSBaseUrl := "http://localhost:2019/config/apps/tls/certificates/load_files/..."
// 	caddyAPITLSPoliciesBaseUrl := "http://localhost:2019/config/apps/http/servers/srv0/tls_connection_policies/..."

// 	routes := []Route{}

// 	// GitOps route
// 	routes = append(routes, Route{
// 		Match: []Match{{Host: []string{fmt.Sprintf("%s-gitops.%s", gitopsName, domain)}}},
// 		Handle: []Handle{{Handler: "subroute", Routes: []Route{
// 			{
// 				Handle: []Handle{{Handler: "reverse_proxy", Upstreams: []Upstream{
// 					{Dial: fmt.Sprintf("%s:8079", gitopsName)},
// 				}}},
// 			},
// 		}}},
// 		Terminal: true,
// 	})

// 	// Bitswan editor route
// 	if !noIde {
// 		routes = append(routes, Route{
// 			Match: []Match{
// 				{
// 					Host: []string{fmt.Sprintf("%s-editor.%s", gitopsName, domain)},
// 				},
// 			},
// 			Handle: []Handle{
// 				{
// 					Handler: "subroute",
// 					Routes: []Route{
// 						{
// 							Handle: []Handle{
// 								{
// 									Handler: "reverse_proxy",
// 									Upstreams: []Upstream{
// 										{
// 											Dial: fmt.Sprintf("bitswan-editor-%s:9999", gitopsName),
// 										},
// 									},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 			Terminal: true,
// 		})
// 	}

// 	if certs {
// 		tlsPolicy := []TLSPolicy{
// 			{
// 				Match: TLSMatch{
// 					SNI: []string{
// 						fmt.Sprintf("*.%s", domain),
// 					},
// 				},
// 				CertificateSelection: TLSCertificateSelection{
// 					AnyTag: []string{gitopsName},
// 				},
// 			},
// 		}

// 		tlsLoad := []TLSFileLoad{
// 			{
// 				Certificate: fmt.Sprintf("/tls/%s/full-chain.pem", domain),
// 				Key:         fmt.Sprintf("/tls/%s/private-key.pem", domain),
// 				Tags:        []string{gitopsName},
// 			},
// 		}

// 		// send tls policy and tls to caddy api
// 		jsonPayload, err := json.Marshal(tlsLoad)
// 		if err != nil {
// 			return fmt.Errorf("failed to marshal JSON payload: %w", err)
// 		}

// 		// Send the payload to the Caddy API
// 		_, err = sendRequest("POST", caddyAPITLSBaseUrl, jsonPayload)
// 		if err != nil {
// 			return fmt.Errorf("Failed to add TLS to Caddy: %w", err)
// 		}

// 		jsonPayload, err = json.Marshal(tlsPolicy)
// 		if err != nil {
// 			return fmt.Errorf("Failed to marshal TLS Policy JSON: %w", err)
// 		}

// 		// Send the payload to the Caddy API
// 		_, err = sendRequest("POST", caddyAPITLSPoliciesBaseUrl, jsonPayload)
// 		if err != nil {
// 			return fmt.Errorf("Failed to add TLS policies to Caddy: %w", err)
// 		}
// 	}

// 	jsonPayload, err := json.Marshal(routes)
// 	if err != nil {
// 		return fmt.Errorf("Failed to marshal routes payload: %w", err)
// 	}

// 	// Send the payload to the Caddy API
// 	_, err = sendRequest("PUT", caddyAPIRoutesBaseUrl+gitopsName+"_gitops", jsonPayload)
// 	if err != nil {
// 		return fmt.Errorf("Failed to add routes to Caddy: %w", err)
// 	}

// 	return nil
// }

func AddCaddyRecords(gitopsName, domain string, certs, noIde bool) error {
	caddyAPIRoutesBaseUrl := "http://localhost:2019/config/apps/http/servers/srv0/routes/"

	// Define the route
	routes := Route{
		ID:    fmt.Sprintf("%s_gitops", gitopsName),
		Match: []Match{{Host: []string{fmt.Sprintf("%s-gitops.%s", gitopsName, domain)}}},
		Handle: []Handle{{Handler: "subroute", Routes: []Route{
			{
				Handle: []Handle{{Handler: "reverse_proxy", Upstreams: []Upstream{
					{Dial: fmt.Sprintf("%s:8079", gitopsName)},
				}}},
			},
		}}},
		Terminal: true,
	}

	jsonPayload, err := json.Marshal(routes)
	if err != nil {
		return fmt.Errorf("Failed to marshal routes payload: %w", err)
	}

	// Send a PUT request to create or update the route by ID
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Caddy API returned status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

func fetchRoutes(url string) ([]Route, error) {
	body, err := sendRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var routes []Route
	if err := json.Unmarshal(body, &routes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal routes: %w", err)
	}

	return routes, nil
}

func fetchTLSFiles(url string) ([]TLSFileLoad, error) {
	body, err := sendRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var tlsFiles []TLSFileLoad
	if err := json.Unmarshal(body, &tlsFiles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TLS files: %w", err)
	}

	return tlsFiles, nil
}

func fetchTLSPoliciesFiles(url string) ([]TLSPolicy, error) {
	body, err := sendRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var tlsPolicies []TLSPolicy
	if err := json.Unmarshal(body, &tlsPolicies); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TLS policies: %w", err)
	}

	return tlsPolicies, nil
}

func RemoveCaddyRecord(recordName string) error {
	var routeIndex, tlsFileIndex, tlsPolicyIndex = -1, -1, -1
	routesUrl := "http://localhost:2019/config/apps/http/servers/srv0/routes"
	tlsFilesUrl := "http://localhost:2019/config/apps/tls/certificates/load_files"
	tlsPoliciesUrl := "http://localhost:2019/config/apps/http/servers/srv0/tls_connection_policies"

	// Fetch all routes
	routes, err := fetchRoutes(routesUrl)
	if err != nil {
		return fmt.Errorf("failed to fetch routes: %w", err)
	}

	// Find the route with the desired host
	for index, route := range routes {
		for _, match := range route.Match {
			if slices.Contains(match.Host, recordName+"-gitops.bitswan.local") {
				routeIndex = index
				break
			}
		}
	}

	tlsFiles, err := fetchTLSFiles(tlsFilesUrl)
	if err != nil {
		return fmt.Errorf("failed to fetch TLS files: %w", err)
	}

	// Find the TLS file with the desired tag
	for index, tlsFile := range tlsFiles {
		if slices.Contains(tlsFile.Tags, recordName) {
			tlsFileIndex = index
			break
		}
	}

	tlsPolicies, err := fetchTLSPoliciesFiles(tlsPoliciesUrl)
	if err != nil {
		return fmt.Errorf("failed to fetch TLS policies: %w", err)
	}

	// Find the TLS policy with the desired tag
	for index, tlsPolicy := range tlsPolicies {
		if slices.Contains(tlsPolicy.CertificateSelection.AnyTag, recordName) {
			tlsPolicyIndex = index
			break
		}
	}

	if routeIndex != -1 {
		if _, err := sendRequest("DELETE", fmt.Sprintf("%s/%d", routesUrl, routeIndex), nil); err != nil {
			return fmt.Errorf("failed to remove Caddy record: %w", err)
		}
	}

	if tlsFileIndex != -1 {
		if _, err := sendRequest("DELETE", fmt.Sprintf("%s/%d", tlsFilesUrl, tlsFileIndex), nil); err != nil {
			return fmt.Errorf("failed to remove Caddy record: %w", err)
		}
	}

	if tlsPolicyIndex != -1 {
		if _, err := sendRequest("DELETE", fmt.Sprintf("%s/%d", tlsPoliciesUrl, tlsPolicyIndex), nil); err != nil {
			return fmt.Errorf("failed to remove Caddy record: %w", err)
		}
	}

	return nil
}

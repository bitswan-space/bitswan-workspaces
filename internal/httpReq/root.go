package httpReq
/*
   This package patches the golang DNS resolver to resolve *.localhost to localhost just as many other linux resolvers work.
 */

import (
	  "fmt"
	  "io"
    "context"
    "net"
    "net/http"
    "strings"
    "time"
)

// NewRequestWithLocalhostResolution creates an HTTP request that automatically
// resolves .localhost domains to 127.0.0.1
func NewRequest(method, url string, body io.Reader) (*http.Request, error) {
    // Create the standard request
    req, err := http.NewRequest(method, url, body)
    if err != nil {
        return nil, err
    }

    // No need to store the client in context - this function just returns the request
    return req, nil
}

// ExecuteRequestWithLocalhostResolution executes a request with .localhost resolution
func ExecuteRequestWithLocalhostResolution(req *http.Request) (*http.Response, error) {
    // Create a transport with custom dialing for .localhost domains
    transport := &http.Transport{
        DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
            host, port, err := net.SplitHostPort(addr)
            if err != nil {
                return nil, err
            }
            fmt.Printf("Resolving host %s\n", host)

            if strings.HasSuffix(host, ".localhost") {
                fmt.Printf("Using localhost resolution for %s\n", host)
                // Force localhost resolution for .localhost domains
                return net.Dial(network, net.JoinHostPort("127.0.0.1", port))
            }

            // Use normal resolution for other domains
            return (&net.Dialer{
                Timeout:   30 * time.Second,
                KeepAlive: 30 * time.Second,
            }).DialContext(ctx, network, addr)
        },
    }

    // Create a client with our custom transport
    client := &http.Client{Transport: transport}

    // Execute the request with our custom client
    return client.Do(req)
}


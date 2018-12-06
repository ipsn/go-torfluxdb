// go-torfluxdb - Anonymous metrics from Go
// Copyright (c) 2018 Péter Szilágyi. All rights reserved.

// Package proxy implements a Tor onion service in front of an InfluxDB instance.
package proxy

import (
	"context"
	"crypto"
	"io"
	"log"
	"net/http"

	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
)

// Proxy is a Tor onion service in front of an InfluxDB instance.
type Proxy struct {
	gateway *tor.Tor
	service *tor.OnionService
	backend string
}

// New creates a Tor onion service for a backing InfluxDB.
func New(ctx context.Context, key crypto.PrivateKey, backend string) (*Proxy, error) {
	// Start tor with some defaults + elevated verbosity
	gateway, err := tor.Start(nil, &tor.StartConf{ProcessCreator: libtor.Creator})
	if err != nil {
		return nil, err
	}
	// Create an onion service to listen on (show as 8086)
	service, err := gateway.Listen(ctx, &tor.ListenConf{RemotePorts: []int{8086}, Version3: true, Key: key})
	if err != nil {
		gateway.Close()
		return nil, err
	}
	return &Proxy{
		gateway: gateway,
		service: service,
		backend: backend,
	}, nil
}

// Close terminates the onion service and Tor gateway.
func (p *Proxy) Close() error {
	p.service.Close()
	p.gateway.Close()

	return nil
}

// URL retrieves the onion URL of the proxy.
func (p *Proxy) URL() string {
	return "http://" + p.service.ID + ".onion:8086"
}

// Serve accepts incoming connections, creating a new service goroutine for each.
// The service goroutines read requests and forward them to the configured Influx
// instance.
//
// Serve always returns a non-nil error and closes its Tor gateway.
func (p *Proxy) Serve() error {
	defer p.gateway.Close()
	defer p.service.Close()

	return http.Serve(p.service, p)
}

// ServeHTTP proxies an inbound HTTP request to the backing InfluxDB instance,
// forwarding the reply back to the client.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Swap out the host in the destination URL to InfluxDB
	r.URL.Scheme, r.URL.Host = "http", p.backend

	// Clone the inbound request into an outbound one
	rr := http.Request{
		Method:        r.Method,
		URL:           r.URL,
		Header:        r.Header,
		Body:          r.Body,
		ContentLength: r.ContentLength,
		Close:         r.Close,
	}
	// Proxy the request to InfluxDB
	res, err := http.DefaultTransport.RoundTrip(&rr)
	if err != nil {
		log.Printf("Failed to forward metrics request: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	log.Printf("Metrics request forwarded successfully: %s", res.Status)
	defer res.Body.Close()

	// Set all the relevant headers, and forward the reply
	hh := w.Header()
	for key, value := range res.Header {
		hh[key] = value
	}
	w.WriteHeader(res.StatusCode)

	io.Copy(w, res.Body)
}

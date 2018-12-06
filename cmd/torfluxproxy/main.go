// go-torfluxdb - Anonymous metrics from Go
// Copyright (c) 2018 Péter Szilágyi. All rights reserved.

package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cretz/bine/torutil"
	"github.com/cretz/bine/torutil/ed25519"
	"github.com/ipsn/go-torfluxdb/proxy"
)

var (
	// printkeyFlag instructs the proxy to print it's private key on startup.
	printkeyFlag = flag.Bool("printkey", false, "Print the Tor onion private key to stdout")

	// timeoutFlag is the time allowance to join the Tor network
	timeoutFlag = flag.Duration("timeout", 3*time.Minute, "Time allowance to join the Tor network")

	// influxdbFlag is the address of the backing InfluxDB instance.
	influxdbFlag = flag.String("influxdb", "localhost:8086", "Address of the backing InfluxDB instance")
)

func main() {
	flag.Parse()

	// Retrieve any configured private key or generate a new one
	var key crypto.PrivateKey
	if hexstr := os.Getenv("TORFLUXDB_ONIONKEY"); hexstr == "" {
		// No onion key specified, generate a new one
		log.Printf("No pre-configured private key (TORFLUXDB_ONIONKEY), generating a new one...")
		newkey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			log.Printf("Failed to generate onion private key: %v", err)
			os.Exit(-1)
		}
		key = newkey.PrivateKey()
	} else {
		// Onion key specified, decode and validate it
		blob, err := hex.DecodeString(hexstr)
		if err != nil {
			log.Printf("Failed to decode private key (TORFLUXDB_ONIONKEY): %v", err)
			os.Exit(-1)
		}
		if len(blob) != ed25519.PrivateKeySize {
			log.Printf("Invalid private key length (TORFLUXDB_ONIONKEY): have %d, want %d", len(blob), ed25519.PrivateKeySize)
			os.Exit(-1)
		}
		key = ed25519.PrivateKey(blob)
	}
	// If the user requested printing the private key, do so now
	if *printkeyFlag {
		log.Printf("--------------------%s\n", strings.Repeat("-", 128))
		log.Printf("Your private key is %x\n", key)
		log.Printf("This key permits you to restart your service with the same Tor onion URL by setting the TORFLUXDB_ONIONKEY environmental variable.\n")
		log.Printf("WARNING: Anyone getting hold of your key will also be able to impersonate your server, so protect it like your house keys!\n")
		log.Printf("--------------------%s\n", strings.Repeat("-", 128))
	}
	// Create a Tor InfluxDB proxy with some sane timeout to avoid indefinite hangs
	ctx, cancel := context.WithTimeout(context.Background(), *timeoutFlag)
	defer cancel()

	log.Print("Starting Tor proxy, this might take a minute or so...")
	log.Printf("Endpoint will be published at http://%s.onion", torutil.OnionServiceIDFromPrivateKey(key))

	proxy, err := proxy.New(ctx, key, *influxdbFlag)
	if err != nil {
		log.Printf("Failed to start Tor proxy: %v", err)
		os.Exit(-1)
	}
	defer proxy.Close()

	// Report the service opened and start serving requests
	log.Printf("InfluxDB Tor proxy online at http://%s.onion", proxy.ID())
	fmt.Println(proxy.Serve())
}

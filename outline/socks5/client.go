// Copyright 2022 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This package provides support of socks5 client and the configuration
// that can be used by Outline Client.
//
// All data structures and functions will also be exposed as libraries that
// non-golang callers can use (for example, C/Java/Objective-C).
package socks5

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Jigsaw-Code/outline-go-tun2socks/outline"
	"github.com/Jigsaw-Code/outline-go-tun2socks/outline/connectivity"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
)

// A client object that can be used to connect to a remote socks5 proxy.
type Client outline.Client

func NewSocks5Client(host string, port int) (*Client, error) {
	// TODO: consider using net.LookupIP to get a list of IPs, and add logic for optimal selection.
	proxyIP, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve proxy address: %w", err)
	}
	proxyAddress := net.JoinHostPort(proxyIP.String(), fmt.Sprint(port))

	endpoint := &transport.TCPEndpoint{Address: proxyAddress}
	streamDialer, err := socks5.NewClient(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create StreamDialer: %w", err)
	}

	// Create a second SOCKS5 client for UDP using the same TCP endpoint
	packetListener, err := socks5.NewClient(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create PacketListener: %w", err)
	}
	// Enable UDP support on the SOCKS5 client
	packetListener.EnablePacket(&transport.UDPDialer{})

	return &Client{StreamDialer: streamDialer, PacketListener: packetListener}, nil
}

// Error number constants exported through gomobile
const (
	NoError                     = 0
	Unexpected                  = 1
	NoVPNPermissions            = 2 // Unused
	AuthenticationFailure       = 3
	UDPConnectivity             = 4
	Unreachable                 = 5
	VpnStartFailure             = 6  // Unused
	IllegalConfiguration        = 7  // Electron only
	socks5StartFailure          = 8  // Unused
	ConfigureSystemProxyFailure = 9  // Unused
	NoAdminPermissions          = 10 // Unused
	UnsupportedRoutingTable     = 11 // Unused
	SystemMisconfigured         = 12 // Electron only
)

const reachabilityTimeout = 10 * time.Second

// CheckConnectivity determines whether the socks5 proxy can relay TCP and UDP traffic under
// the current network. Parallelizes the execution of TCP and UDP checks, selects the appropriate
// error code to return accounting for transient network failures.
// Returns an error if an unexpected error ocurrs.
func CheckConnectivity(client *Client) (int, error) {
	errCode, err := connectivity.CheckConnectivity((*outline.Client)(client))
	return errCode.Number(), err
}

// CheckServerReachable determines whether the server at `host:port` is reachable over TCP.
// Returns an error if the server is unreachable.
func CheckServerReachable(host string, port int) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), reachabilityTimeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

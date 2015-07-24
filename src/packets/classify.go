// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package packets

// This file does basic classification of IP addresses.

import (
	"net"
)

var (
	LocalNetblock *net.IPNet
	stdLocalNets  = []*net.IPNet{
		MustParseCIDR("10.0.0.0/8"), // RFC1918 IPv4 private addresses
		MustParseCIDR("172.16.0.0/12"),
		MustParseCIDR("192.168.0.0/16"),
		MustParseCIDR("fd00::/8"),           // RFC4193 IPv6 private addresses
		MustParseCIDR("169.254.0.0/16"),     // RFC3917 IPv4 link-local addresses
		MustParseCIDR("fe80::/10"),          // RFC4862 IPv6 link-local/autoconfig addresses
		MustParseCIDR("0.0.0.0/32"),         // Broadcast source
		MustParseCIDR("255.255.255.255/32"), // Broadcast destination
	}
)

// MustParseCIDR attempts to net.ParseCIDR, and panics if it errors. This is
// useful for defining static netblocks in code.
func MustParseCIDR(s string) *net.IPNet {
	_, cidr, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return cidr
}

// IsLocal returns true if the IP is a private or link-local address. It also
// considers the LocalNetblock passed in (from a flag), useful in case NAT is
// not in use.
func IsLocal(ip net.IP) bool {
	if LocalNetblock != nil && LocalNetblock.Contains(ip) {
		return true
	}
	for _, cidr := range stdLocalNets {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// local returns the "most local" of two IP addresses.
// If both are local, it will return the first. If neither, it will return the second.
func local(ip1, ip2 net.IP) net.IP {
	if IsLocal(ip1) {
		return ip1
	}
	return ip2
}

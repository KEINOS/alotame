/*
Simple DNS client to check if domains are blocked by querying A records.

Usage:

	client <dns-server:53> <domain1> <domain2> ...

Example:

	$ go run . 1.1.1.1:53 yahoo.com dns.example.com
	[
		{
			"domain": "yahoo.com",
			"status": "ALLOWED",
			"detail": "98.xxx.xxx.xxx"
		},
		{
			"domain": "dns.example.com",
			"status": "BLOCKED",
			"detail": "no answer"
		}
	]
*/
package main

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"time"

	"github.com/miekg/dns"
)

const (
	requestTimeout = 3 * time.Second
	maxRetries     = 3
	retryDelay     = 3 * time.Second
)

type Result struct {
	Domain string `json:"domain"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: %s <dns-server:53> <domain>...", os.Args[0])
	}

	server := os.Args[1]
	results := make([]Result, 0, len(os.Args)-2)

	for _, domain := range os.Args[2:] {
		results = append(results, queryA(server, domain))
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(results); err != nil {
		log.Fatalf("failed to encode results: %v", err)
	}
}

func queryA(server, domain string) Result {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	m.RecursionDesired = true

	c := new(dns.Client)
	c.Timeout = requestTimeout

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		r, _, err := c.Exchange(m, server)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() && attempt < maxRetries {
				lastErr = err
				time.Sleep(retryDelay)

				continue
			}

			return Result{domain, "ERROR", err.Error()}
		}

		if r.Rcode == dns.RcodeNameError {
			return Result{domain, "BLOCKED", "NXDOMAIN"}
		}

		if len(r.Answer) == 0 {
			return Result{domain, "BLOCKED", "no answer"}
		}

		for _, ans := range r.Answer {
			if a, ok := ans.(*dns.A); ok {
				if a.A.String() != "0.0.0.0" {
					return Result{domain, "ALLOWED", a.A.String()}
				}
			}
		}

		return Result{domain, "BLOCKED", "only null IPs"}
	}

	return Result{domain, "ERROR", lastErr.Error()}
}

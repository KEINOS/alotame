/*
Simple DNS client to check if domains are blocked by querying A records.

NOTE:

	This tool assumes the following DNS semantics:

		- NXDOMAIN or empty answer => BLOCKED
		- A record != 0.0.0.0 => ALLOWED

	These assumptions are environment-dependent.

Usage:

	client <dns-server:port> [options] [domain1] [domain2] ...

Options:

	--require_allow <domains>  Comma-separated list of domains that must be ALLOWED
	--require_deny <domains>   Comma-separated list of domains that must be BLOCKED

Exit Codes:

	0  All test requirements met (or no requirements specified)
	1  One or more test requirements failed

Examples:

	# Simple query mode
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

	# E2E test mode (exits 0 if all conditions are met)
	$ go run . 1.1.1.1:53 --require_allow github.com,yahoo.com --require_deny unknowndomain.com
	[
	  {
	    "domain": "github.com",
	    "status": "ALLOWED",
	    "detail": "20.27.177.113",
	    "test_result": "PASS"
	  },
	  {
	    "domain": "yahoo.com",
	    "status": "ALLOWED",
	    "detail": "74.6.231.21",
	    "test_result": "PASS"
	  },
	  {
	    "domain": "unknowndomain.com",
	    "status": "BLOCKED",
	    "detail": "NXDOMAIN",
	    "test_result": "PASS"
	  }
	]

	# Mixed mode (with extra domains not in requirements)
	$ go run . 1.1.1.1:53 --require_allow github.com extra-query.com
	[
	  {
	    "domain": "github.com",
	    "status": "ALLOWED",
	    "detail": "20.27.177.113",
	    "test_result": "PASS"
	  },
	  {
	    "domain": "extra-query.com",
	    "status": "BLOCKED",
	    "detail": "NXDOMAIN",
	    "test_result": "UNDETERMINED"
	  }
	]

Test Result Values:

	PASS          Domain status matches the requirement
	FAIL          Domain status does not match the requirement
	UNDETERMINED  Domain was queried but not specified in any requirement
*/
package main

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// Timeout and retry settings. These are variables (not constants) to allow
// tests to override them for faster CI/CD execution.
var (
	requestTimeout = 3 * time.Second
	maxRetries     = 3
	retryDelay     = 3 * time.Second
)

const (
	statusAllowed = "ALLOWED"
	statusBlocked = "BLOCKED"
	statusError   = "ERROR"

	testResultPass         = "PASS"
	testResultFail         = "FAIL"
	testResultUndetermined = "UNDETERMINED"
)

type Result struct {
	Domain     string `json:"domain"`
	Status     string `json:"status"`
	Detail     string `json:"detail"`
	TestResult string `json:"test_result,omitempty"`
}

type TestConfig struct {
	RequireAllow []string
	RequireDeny  []string
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <dns-server:53> [--require_allow domains] [--require_deny domains] [domain...]", os.Args[0])
	}

	server := os.Args[1]
	config, domains := parseArgs(os.Args[2:])

	// Collect all domains to query
	allDomains := make([]string, 0)
	allDomains = append(allDomains, config.RequireAllow...)
	allDomains = append(allDomains, config.RequireDeny...)
	allDomains = append(allDomains, domains...)

	if len(allDomains) == 0 {
		log.Fatalf("no domains specified")
	}

	// Query all domains
	results := make([]Result, 0, len(allDomains))
	for _, domain := range allDomains {
		results = append(results, queryA(server, domain))
	}

	// Determine exit code and set test_result fields if in test mode
	isTestMode := len(config.RequireAllow) > 0 || len(config.RequireDeny) > 0
	exitCode := 0

	if isTestMode {
		exitCode = validateResults(results, config)
	}

	// Output results as JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(results); err != nil {
		log.Fatalf("failed to encode results: %v", err)
	}

	if isTestMode {
		os.Exit(exitCode)
	}
}

// parseArgs parses command line arguments and returns test configuration and remaining domains.
func parseArgs(args []string) (TestConfig, []string) {
	config := TestConfig{}
	domains := make([]string, 0)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--require_allow":
			if i+1 < len(args) {
				config.RequireAllow = splitDomains(args[i+1])
				i++
			}
		case "--require_deny":
			if i+1 < len(args) {
				config.RequireDeny = splitDomains(args[i+1])
				i++
			}
		default:
			domains = append(domains, args[i])
		}
	}

	return config, domains
}

// splitDomains splits a comma-separated list of domains.
func splitDomains(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// validateResults checks if query results match the expected allow/deny requirements.
// It sets the TestResult field for each result and returns 0 if all requirements are met, 1 otherwise.
func validateResults(results []Result, config TestConfig) int {
	// Build sets for quick lookup
	allowSet := make(map[string]struct{})
	for _, d := range config.RequireAllow {
		allowSet[d] = struct{}{}
	}

	denySet := make(map[string]struct{})
	for _, d := range config.RequireDeny {
		denySet[d] = struct{}{}
	}

	hasError := false

	for i := range results {
		domain := results[i].Domain
		status := results[i].Status

		if _, ok := allowSet[domain]; ok {
			// Domain must be ALLOWED
			if status == statusAllowed {
				results[i].TestResult = testResultPass
			} else {
				results[i].TestResult = testResultFail
				hasError = true
			}
		} else if _, ok := denySet[domain]; ok {
			// Domain must be BLOCKED
			if status == statusBlocked {
				results[i].TestResult = testResultPass
			} else {
				results[i].TestResult = testResultFail
				hasError = true
			}
		} else {
			// Domain not in any test requirement
			results[i].TestResult = testResultUndetermined
		}
	}

	if hasError {
		return 1
	}

	return 0
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

			return Result{Domain: domain, Status: statusError, Detail: err.Error()}
		}

		if r.Rcode == dns.RcodeNameError {
			return Result{Domain: domain, Status: statusBlocked, Detail: "NXDOMAIN"}
		}

		if len(r.Answer) == 0 {
			return Result{Domain: domain, Status: statusBlocked, Detail: "no answer"}
		}

		for _, ans := range r.Answer {
			if a, ok := ans.(*dns.A); ok {
				if a.A.String() != "0.0.0.0" {
					return Result{Domain: domain, Status: statusAllowed, Detail: a.A.String()}
				}
			}
		}

		return Result{Domain: domain, Status: statusBlocked, Detail: "only null IPs"}
	}

	return Result{Domain: domain, Status: statusError, Detail: lastErr.Error()}
}

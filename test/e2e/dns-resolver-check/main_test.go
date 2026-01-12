/*
NOTE: Some integration tests in this file query external DNS servers (1.1.1.1).
They require network connectivity and are NOT suitable for CI/CD pipelines
or continuous testing due to:
  - External network dependency
  - Potential rate limiting
  - Flaky results due to network conditions
  - DNS response changes over time

Usage:

	# Run all tests (requires network connectivity)
	go test -v ./...

	# Run only unit tests without network (suitable for CI/CD)
	go test -v -short ./...

The -short flag skips all integration tests that require network access.

IMPORTANT: If an integration test fails, do NOT assume it's a bug immediately.
External DNS behavior can change due to:
  - CDN migrations (e.g., GitHub changing providers)
  - IPv6-only responses in certain regions
  - DNS provider policy changes
  - Regional routing differences
  - Rate limiting or temporary outages

Before debugging:
 1. Verify the test expectations still match real-world DNS behavior
 2. Run `dig <domain>` or `nslookup <domain>` manually to confirm
 3. Check if the DNS provider made announcements about changes
 4. Consider whether the test needs updating, not the code
*/

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Override timeout settings for faster test execution in CI/CD.
	// Default production values: requestTimeout=3s, maxRetries=3, retryDelay=3s
	// Worst case with defaults: (3s + 3s) × 3 + 3s = ~21s per domain
	// With these overrides: (200ms + 0) × 1 + 200ms = ~400ms per domain
	requestTimeout = 200 * time.Millisecond
	maxRetries = 1
	retryDelay = 0
}

// ----------------------------------------------------------------------------
//  Unit Tests (no network required)
// ----------------------------------------------------------------------------

func TestSplitDomains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single domain",
			input:    "example.com",
			expected: []string{"example.com"},
		},
		{
			name:     "multiple domains",
			input:    "example.com,test.com,foo.com",
			expected: []string{"example.com", "test.com", "foo.com"},
		},
		{
			name:     "domains with spaces",
			input:    " example.com , test.com , foo.com ",
			expected: []string{"example.com", "test.com", "foo.com"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := splitDomains(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestParseArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		args            []string
		expectedConfig  TestConfig
		expectedDomains []string
	}{
		{
			name:            "no options",
			args:            []string{"domain1.com", "domain2.com"},
			expectedConfig:  TestConfig{},
			expectedDomains: []string{"domain1.com", "domain2.com"},
		},
		{
			name: "require_allow only",
			args: []string{"--require_allow", "allow1.com,allow2.com"},
			expectedConfig: TestConfig{
				RequireAllow: []string{"allow1.com", "allow2.com"},
			},
			expectedDomains: []string{},
		},
		{
			name: "require_deny only",
			args: []string{"--require_deny", "deny1.com,deny2.com"},
			expectedConfig: TestConfig{
				RequireDeny: []string{"deny1.com", "deny2.com"},
			},
			expectedDomains: []string{},
		},
		{
			name: "both options with extra domains",
			args: []string{"--require_allow", "allow.com", "--require_deny", "deny.com", "extra.com"},
			expectedConfig: TestConfig{
				RequireAllow: []string{"allow.com"},
				RequireDeny:  []string{"deny.com"},
			},
			expectedDomains: []string{"extra.com"},
		},
		{
			name:            "empty args",
			args:            []string{},
			expectedConfig:  TestConfig{},
			expectedDomains: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			config, domains := parseArgs(test.args)
			assert.Equal(t, test.expectedConfig, config)
			assert.Equal(t, test.expectedDomains, domains)
		})
	}
}

func TestValidateResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		results          []Result
		config           TestConfig
		expectedExitCode int
		expectedResults  []string
	}{
		{
			name: "all pass - allow check",
			results: []Result{
				{Domain: "allowed.com", Status: statusAllowed, Detail: "1.2.3.4"},
			},
			config:           TestConfig{RequireAllow: []string{"allowed.com"}},
			expectedExitCode: 0,
			expectedResults:  []string{testResultPass},
		},
		{
			name: "all pass - deny check",
			results: []Result{
				{Domain: "blocked.com", Status: statusBlocked, Detail: "NXDOMAIN"},
			},
			config:           TestConfig{RequireDeny: []string{"blocked.com"}},
			expectedExitCode: 0,
			expectedResults:  []string{testResultPass},
		},
		{
			name: "fail - expected allow but got blocked",
			results: []Result{
				{Domain: "shouldallow.com", Status: statusBlocked, Detail: "NXDOMAIN"},
			},
			config:           TestConfig{RequireAllow: []string{"shouldallow.com"}},
			expectedExitCode: 1,
			expectedResults:  []string{testResultFail},
		},
		{
			name: "fail - expected deny but got allowed",
			results: []Result{
				{Domain: "shouldblock.com", Status: statusAllowed, Detail: "1.2.3.4"},
			},
			config:           TestConfig{RequireDeny: []string{"shouldblock.com"}},
			expectedExitCode: 1,
			expectedResults:  []string{testResultFail},
		},
		{
			name: "mixed results with undetermined",
			results: []Result{
				{Domain: "allow.com", Status: statusAllowed, Detail: "1.2.3.4"},
				{Domain: "deny.com", Status: statusBlocked, Detail: "NXDOMAIN"},
				{Domain: "extra.com", Status: statusAllowed, Detail: "5.6.7.8"},
			},
			config: TestConfig{
				RequireAllow: []string{"allow.com"},
				RequireDeny:  []string{"deny.com"},
			},
			expectedExitCode: 0,
			expectedResults:  []string{testResultPass, testResultPass, testResultUndetermined},
		},
		{
			name: "partial failure",
			results: []Result{
				{Domain: "good.com", Status: statusAllowed, Detail: "1.2.3.4"},
				{Domain: "bad.com", Status: statusBlocked, Detail: "NXDOMAIN"},
			},
			config:           TestConfig{RequireAllow: []string{"good.com", "bad.com"}},
			expectedExitCode: 1,
			expectedResults:  []string{testResultPass, testResultFail},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			results := make([]Result, len(test.results))
			copy(results, test.results)

			exitCode := validateResults(results, test.config)

			assert.Equal(t, test.expectedExitCode, exitCode)

			for i, expectedResult := range test.expectedResults {
				assert.Equal(t, expectedResult, results[i].TestResult,
					"TestResult mismatch for domain %s", results[i].Domain)
			}
		})
	}
}

// ----------------------------------------------------------------------------
//  Integration Tests (requires network - NOT for CI/CD)
// ----------------------------------------------------------------------------

func TestQueryA_Integration(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const dnsServer = "1.1.1.1:53"

	t.Run("query allowed domain", func(t *testing.T) {
		t.Parallel()

		result := queryA(dnsServer, "github.com")

		require.Equal(t, "github.com", result.Domain)
		assert.Equal(t, statusAllowed, result.Status)
		assert.NotEmpty(t, result.Detail)
		assert.NotEqual(t, "0.0.0.0", result.Detail)
	})

	t.Run("query non-existent domain", func(t *testing.T) {
		t.Parallel()

		result := queryA(dnsServer, "this-domain-definitely-does-not-exist-xyz123.com")

		assert.Equal(t, statusBlocked, result.Status)
		assert.Equal(t, "NXDOMAIN", result.Detail)
	})
}

func TestE2ETestMode_Integration(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const dnsServer = "1.1.1.1:53"

	t.Run("all conditions met - exit 0", func(t *testing.T) {
		t.Parallel()

		config := TestConfig{
			RequireAllow: []string{"github.com", "yahoo.com"},
			RequireDeny:  []string{"this-domain-definitely-does-not-exist-xyz123.com"},
		}

		allDomains := append(config.RequireAllow, config.RequireDeny...)

		results := make([]Result, 0, len(allDomains))
		for _, domain := range allDomains {
			results = append(results, queryA(dnsServer, domain))
		}

		exitCode := validateResults(results, config)

		assert.Equal(t, 0, exitCode, "expected exit code 0 when all conditions are met")

		for _, r := range results {
			assert.Equal(t, testResultPass, r.TestResult,
				"expected PASS for domain %s", r.Domain)
		}
	})

	t.Run("condition not met - exit 1", func(t *testing.T) {
		t.Parallel()

		config := TestConfig{
			RequireAllow: []string{"this-domain-definitely-does-not-exist-xyz123.com"},
		}

		results := []Result{queryA(dnsServer, config.RequireAllow[0])}
		exitCode := validateResults(results, config)

		assert.Equal(t, 1, exitCode, "expected exit code 1 when condition is not met")
		assert.Equal(t, testResultFail, results[0].TestResult)
	})

	t.Run("undetermined domains do not affect exit code", func(t *testing.T) {
		t.Parallel()

		config := TestConfig{
			RequireAllow: []string{"github.com"},
		}

		allDomains := []string{"github.com", "yahoo.com"}

		results := make([]Result, 0, len(allDomains))
		for _, domain := range allDomains {
			results = append(results, queryA(dnsServer, domain))
		}

		exitCode := validateResults(results, config)

		assert.Equal(t, 0, exitCode)
		assert.Equal(t, testResultPass, results[0].TestResult)
		assert.Equal(t, testResultUndetermined, results[1].TestResult)
	})
}

func TestLegacyMode_NoTestResult(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const dnsServer = "1.1.1.1:53"

	result := queryA(dnsServer, "github.com")

	assert.Empty(t, result.TestResult,
		"TestResult should be empty in legacy mode (before validateResults is called)")
}

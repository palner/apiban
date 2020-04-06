package apiban

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// use this counter to count the number of requests
var counter int

// setup our mock http server
var mockServer = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// for whatever reason the application doesn't let you NOT pass an ID (it defaults it to 100 when empty)
	ID := 100

	// initialize all of our test permutations to a readable form
	// paths for Banned
	testPathBanned := fmt.Sprintf("/%s/banned/%d", "testKey", ID)
	testPathBannedID := fmt.Sprintf("/%s/banned/%d", "testKey", 1234567890)
	testPathBannedReturnNothing := fmt.Sprintf("/%s/banned/%d", "returnNothing", ID)
	testPathBannedReturnNoID := fmt.Sprintf("/%s/banned/%d", "returnNoID", ID)
	testPathBannedReturn400 := fmt.Sprintf("/%s/banned/%s", "testKey", "badInput")
	testPathBannedReturn500 := fmt.Sprintf("/%s/banned/%d", "return500", ID)
	testPathBannedBadAuth := fmt.Sprintf("/%s/banned/%d", "badAuth", ID)
	testPathBannedNothingNew := fmt.Sprintf("/%s/banned/%d", "testKey", 12345678901)

	// paths for Check
	testPathCheck := fmt.Sprintf("/%s/check/%s", "testKey", "1.2.3.251")
	testPathCheckNotBlocked := fmt.Sprintf("/%s/check/%s", "testKey", "1.2.3.254")
	testPathCheckBadIPv4 := fmt.Sprintf("/%s/check/%s", "testKey", "10.0.0.257")
	testPathCheckBadIPv6 := fmt.Sprintf("/%s/check/%s", "testKey", "1000:0000:0000:0000:0000:0000:0000:000g")
	testPathCheckDNS := fmt.Sprintf("/%s/check/%s", "testKey", "foo.bar")
	testPathCheckRateLimit := fmt.Sprintf("/%s/check/%s", "testRateLimit", "1.2.3.251")
	testPathCheckUnknown := fmt.Sprintf("/%s/check/%s", "testUnknown", "1.2.3.251")

	// add special unit testing path to return unexpected data
	switch r.URL.EscapedPath() {
	case testPathBannedReturnNothing:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(""))
		return
	case testPathBannedReturnNoID:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
		return
	case testPathBannedID:
		ID = 1234567890
	case testPathCheckBadIPv4:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{}"))
		return
	case testPathCheckBadIPv6:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{}"))
		return
	case testPathCheckDNS:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{}"))
		return
	case testPathBannedReturn400:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{}"))
		return
	case testPathBannedReturn500:
		w.WriteHeader(http.StatusBadRequest)
		return
	case testPathBannedBadAuth:
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("{\"ID: \"unauthorized\"}"))
		return
	case testPathCheckRateLimit:
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("{\"ipaddress: \"rate limit exceeded\"}"))
		return
	case testPathCheckUnknown:
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("{\"ipaddress: \"unknown\"}"))
		return
	case testPathBannedNothingNew:
		w.WriteHeader(http.StatusRequestTimeout)
		_, _ = w.Write([]byte("{\"ipaddress\":[\"no new bans\"], \"ID\":\"none\"}"))
		return
	}

	// paths for normal api for calls - Banned
	if r.URL.EscapedPath() == testPathBanned || r.URL.EscapedPath() == testPathBannedID {
		if r.Method == "GET" {
			// increment the counter
			counter += 1
			w.WriteHeader(http.StatusOK)
			if counter < 2 {
				// if the counter is below 5, return data
				_, _ = w.Write([]byte(fmt.Sprintf("{\"ipaddress\": [\"1.2.3.251\", \"1.2.3.252\"], \"ID\": \"%d\"}", ID)))
			} else {
				// reset counter, don't return anything
				_, _ = w.Write([]byte("{\"ID\": \"none\"}"))
				counter = 0
			}
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	} else if r.URL.EscapedPath() == testPathCheck || r.URL.EscapedPath() == testPathCheckNotBlocked {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			if r.URL.EscapedPath() == testPathCheck {
				_, _ = w.Write([]byte("{\"ipaddress\":[\"1.2.3.251\"], \"ID\":\"987654321\"}"))
			} else {
				_, _ = w.Write([]byte("{\"ipaddress\":[\"not blocked\"], \"ID\":\"none\"}"))
			}
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusNotFound)
})

func TestBanned(t *testing.T) {
	// initialize our test server
	testServer := httptest.NewServer(mockServer)
	defer testServer.Close()

	type mockInput struct {
		key         string
		startFrom   string
		badEndpoint bool
	}

	type mockOutput struct {
		data *Entry
		err  error
	}

	testCases := map[string]struct {
		input    mockInput
		expected mockOutput
	}{
		"succesful lookup": {
			input: mockInput{
				key: "testKey",
			},
			expected: mockOutput{
				data: &Entry{
					ID:  "100",
					IPs: []string{"1.2.3.251", "1.2.3.252"},
				},
			},
		},
		"succesful with ID": {
			input: mockInput{
				key:       "testKey",
				startFrom: "1234567890",
			},
			expected: mockOutput{
				data: &Entry{
					ID:  "1234567890",
					IPs: []string{"1.2.3.251", "1.2.3.252"},
				},
			},
		},
		"succesful nothing new": {
			input: mockInput{
				key:       "testKey",
				startFrom: "12345678901",
			},
			expected: mockOutput{
				err: fmt.Errorf("client error (408) from apiban.org: 408 Request Timeout from \"%s/testKey/banned/12345678901\"", testServer.URL),
			},
		},
		"no key": {
			expected: mockOutput{
				err: fmt.Errorf("API Key is required"),
			},
		},
		"unknown key": {
			input: mockInput{
				key: "badKey",
			},
			expected: mockOutput{
				err: fmt.Errorf("client error (404) from apiban.org: 404 Not Found from \"%s/badKey/banned/100\"", testServer.URL),
			},
		},
		"unreachable destination": {
			input: mockInput{
				key:         "testKey",
				badEndpoint: true,
			},
			expected: mockOutput{
				err: fmt.Errorf("Query Error: Get \"http://127.0.0.1:80/testKey/banned/100\": dial tcp 127.0.0.1:80: connectex: No connection could be made because the target machine actively refused it."),
			},
		},
		"nothing returned": {
			input: mockInput{
				key: "returnNothing",
			},
			expected: mockOutput{
				err: fmt.Errorf("failed to decode server response: EOF"),
			},
		},
		"no id returned": {
			input: mockInput{
				key: "returnNoID",
			},
			expected: mockOutput{
				err: fmt.Errorf("empty ID received"),
			},
		},
		"bad input": {
			input: mockInput{
				key:       "testKey",
				startFrom: "badInput",
			},
			expected: mockOutput{
				err: fmt.Errorf("Bad Request"),
			},
		},
		"Simulate unknown server error": {
			input: mockInput{
				key: "return500",
			},
			expected: mockOutput{
				err: fmt.Errorf("failed to decode Bad Request response: EOF"),
			},
		},
		"Simulate bad auth": {
			input: mockInput{
				key: "badAuth",
			},
			expected: mockOutput{
				err: fmt.Errorf("client error (401) from apiban.org: 401 Unauthorized from \"%s/badAuth/banned/100\"", testServer.URL),
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			RootURL = fmt.Sprintf("%s/", testServer.URL)
			if tc.input.badEndpoint {
				RootURL = "http://127.0.0.1:80/"
			}

			// Act
			result, err := Banned(tc.input.key, tc.input.startFrom)

			// Assert
			assert.Equal(t, tc.expected.data, result)
			assert.Equal(t, tc.expected.err, err)
		})
	}
}

func TestCheck(t *testing.T) {
	// initialize our test server
	testServer := httptest.NewServer(mockServer)
	defer testServer.Close()

	type mockInput struct {
		key         string
		ip          string
		badEndpoint bool
	}

	type mockOutput struct {
		data bool
		err  error
	}

	testCases := map[string]struct {
		input    mockInput
		expected mockOutput
	}{
		"succesful lookup - Blocked": {
			input: mockInput{
				key: "testKey",
				ip:  "1.2.3.251",
			},
			expected: mockOutput{
				data: true,
			},
		},
		"succesful lookup - Not Blocked": {
			input: mockInput{
				key: "testKey",
				ip:  "1.2.3.254",
			},
			expected: mockOutput{
				data: false,
			},
		},
		"no key": {
			input: mockInput{
				ip: "1.2.3.251",
			},
			expected: mockOutput{
				data: false,
				err:  fmt.Errorf("API Key is required"),
			},
		},
		"no IP": {
			input: mockInput{
				key: "testKey",
			},
			expected: mockOutput{
				data: false,
				err:  fmt.Errorf("IP address is required"),
			},
		},
		"bad IP v4": {
			input: mockInput{
				key: "testKey",
				ip:  "10.0.0.257",
			},
			expected: mockOutput{
				data: false,
			},
		},
		"bad IP v6": {
			input: mockInput{
				key: "testKey",
				ip:  "1000:0000:0000:0000:0000:0000:0000:000g",
			},
			expected: mockOutput{
				data: false,
			},
		},
		"DNS Name": {
			input: mockInput{
				key: "testKey",
				ip:  "foo.bar",
			},
			expected: mockOutput{
				data: false,
			},
		},
		"unknown key": {
			input: mockInput{
				key: "badKey",
				ip:  "1.2.3.251",
			},
			expected: mockOutput{
				err: fmt.Errorf("client error (404) from apiban.org: 404 Not Found from \"%s/badKey/check/1.2.3.251\"", testServer.URL),
			},
		},
		"simulate rate limiter": {
			input: mockInput{
				key: "testRateLimit",
				ip:  "1.2.3.251",
			},
			expected: mockOutput{
				err: fmt.Errorf("client error (429) from apiban.org: 429 Too Many Requests from \"%s/testRateLimit/check/1.2.3.251\"", testServer.URL),
			},
		},
		"simulate unknown": {
			input: mockInput{
				key: "testUnknown",
				ip:  "1.2.3.251",
			},
			expected: mockOutput{
				err: fmt.Errorf("client error (429) from apiban.org: 429 Too Many Requests from \"%s/testUnknown/check/1.2.3.251\"", testServer.URL),
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			RootURL = fmt.Sprintf("%s/", testServer.URL)
			if tc.input.badEndpoint {
				RootURL = "http://127.0.0.1:80/"
			}

			// Act
			result, err := Check(tc.input.key, tc.input.ip)

			// Assert
			assert.Equal(t, tc.expected.data, result)
			assert.Equal(t, tc.expected.err, err)
		})
	}
}

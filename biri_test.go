package biri

import (
	"strings"
	"testing"
	"time"
)

func TestGetClient(t *testing.T) {

	// Add a cached proxy
	reAddedProxies = append(reAddedProxies, Proxy{})

	// Create a channel to signal the timeout
	timeout := time.After(5 * time.Second) // Adjust the timeout duration as needed

	// Create a channel to receive the result from the test function
	done := make(chan struct{})

	// Execute the test function in a goroutine
	go func() {
		client := GetClient()

		t.Log(client)
		if client == nil {
			t.Error("Go no client")
		}
		// Call the test function here
		// For example: myResult := myFunction()
		// (Replace the above line with the actual test function call)

		// Signal that the test function is done
		close(done)
	}()

	// Use a select statement to wait for either the test to finish or the timeout
	select {
	case <-done:
		// Test completed successfully
		// Add any required assertions or checks here
	case <-timeout:
		// Test took too long to complete
		t.Error("Test timeout")
	}
}

func TestReadd(t *testing.T) {
	reAddedProxies = reAddedProxies[:0]

	p := &Proxy{}
	p.Readd()

	if len(reAddedProxies) != 1 {
		t.Error("We added only one proxy")
	}
}

func TestBanShouldCleanCache(t *testing.T) {
	dummyProxy := Proxy{Info: "dummy"}
	reAddedProxies = append(reAddedProxies, dummyProxy)
	go dummyProxy.Ban()
	banned := <-banProxy
	if banned != dummyProxy.Info {
		t.Error("Banned a other proxy")
	}

	for _, proxy := range reAddedProxies {
		if proxy.Info == dummyProxy.Info {
			t.Error(dummyProxy.Info, "should be banned")
		}
	}
}

func TestGetProxy(t *testing.T) {
	Config.PingServer = "https://github.com"
	go getProxy()

	first := <-availableProxies
	if strings.Count(first.Info, ".") != 3 {
		t.Errorf("Error in ip %v", first)
	}

	if !strings.Contains(first.Info, ":") {
		t.Errorf("Error in port %v", first)
	}
}

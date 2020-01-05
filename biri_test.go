package biri

import (
	"strings"
	"testing"
)

func TestGetProxy(t *testing.T) {
	go getProxy()

	first := <-availableProxies
	if strings.Count(first.Info, ".") != 3 {
		t.Errorf("Error in ip %v", first)
	}

	if !strings.Contains(first.Info, ":") {
		t.Errorf("Error in port %v", first)
	}
}

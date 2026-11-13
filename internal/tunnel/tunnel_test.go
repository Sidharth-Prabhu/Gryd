package tunnel

import (
	"testing"
)

func TestDetectProvider(t *testing.T) {
	// Should at least find ngrok on the current environment if installed
	provider, err := DetectProvider("")
	if err != nil {
		t.Logf("No provider in test env (expected if not installed): %v", err)
	} else {
		t.Logf("Found default provider: %s", provider)
	}

	// Test regex matchers
	ngrokLine := "t=2026-11-13T14:40:00+0530 lvl=info msg=\"started tunnel\" obj=tunnels name=command_line addr=http://localhost:8080 url=https://abc-123.ngrok-free.app"
	match := reNgrokURL.FindString(ngrokLine)
	if match != "https://abc-123.ngrok-free.app" {
		t.Errorf("expected ngrok regex to match URL, got %s", match)
	}

	cfLine := "2026-11-13T09:10:00Z INF +--------------------------------------------------------------------------------------------+\n2026-11-13T09:10:00Z INF |  Your quick Tunnel has been created! Visit it at (it may take some time to be reachable):  |\n2026-11-13T09:10:00Z INF |  https://temporary-subdomain.trycloudflare.com                                             |\n2026-11-13T09:10:00Z INF +--------------------------------------------------------------------------------------------+"
	cfMatch := reCloudflareURL.FindString(cfLine)
	if cfMatch != "https://temporary-subdomain.trycloudflare.com" {
		t.Errorf("expected trycloudflare regex to match URL, got %s", cfMatch)
	}
}

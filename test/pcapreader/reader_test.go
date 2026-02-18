package pcapreader_test

import (
	"os"
	"testing"

	"github.com/samaelod/nabu/pcapreader"
)

func TestReadPCAP(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantErr     bool
		wantMsgsMin int
	}{
		{"pcap_file", "../examples/test.pcap", false, 100},
		{"pcapng_file", "../examples/test001.pcapng", false, 0},
		{"empty_pcapng", "../examples/test002.pcapng", true, 0}, // Empty file returns EOF
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check file exists first
			if _, err := os.Stat(tt.filename); os.IsNotExist(err) {
				t.Skipf("file not found: %s", tt.filename)
				return
			}

			cfg, err := pcapreader.ReadPCAP(tt.filename)
			if tt.wantErr {
				// Expected error (e.g., empty file)
				if err != nil {
					t.Logf("ReadPCAP(%s) expected error: %v", tt.filename, err)
					return
				}
				// If wantErr is true but no error, check if config is nil
				if cfg == nil {
					t.Logf("ReadPCAP(%s) returned nil config as expected for empty file", tt.filename)
					return
				}
			}
			if err != nil {
				t.Errorf("ReadPCAP(%s) unexpected error: %v", tt.filename, err)
				return
			}
			if cfg == nil {
				t.Errorf("ReadPCAP(%s) returned nil config", tt.filename)
				return
			}

			t.Logf("Config: %d endpoints, %d messages", len(cfg.Endpoints), len(cfg.Messages))

			if len(cfg.Messages) < tt.wantMsgsMin {
				t.Errorf("ReadPCAP(%s) = %d messages, want >= %d", tt.filename, len(cfg.Messages), tt.wantMsgsMin)
			}

			// Log some details
			for i, ep := range cfg.Endpoints {
				if i < 3 {
					t.Logf("Endpoint %d: %s:%s:%d", i, ep.Kind, ep.Address, ep.Port)
				}
			}
			for i, msg := range cfg.Messages {
				if i < 5 {
					t.Logf("Message %d: %d -> %d, kind=%s", i, msg.From, msg.To, msg.Kind)
				}
			}
		})
	}
}

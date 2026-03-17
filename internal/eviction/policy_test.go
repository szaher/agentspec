package eviction

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()

	if p.MaxEntries != 10000 {
		t.Errorf("MaxEntries = %d, want 10000", p.MaxEntries)
	}
	if p.TTL != 10*time.Minute {
		t.Errorf("TTL = %v, want 10m", p.TTL)
	}
	if p.EvictionInterval != 5*time.Minute {
		t.Errorf("EvictionInterval = %v, want 5m", p.EvictionInterval)
	}
}

func TestPolicyValidate(t *testing.T) {
	tests := []struct {
		name    string
		policy  Policy
		wantErr string // empty means no error expected
	}{
		{
			name: "valid policy with all positive values",
			policy: Policy{
				MaxEntries:       100,
				TTL:              5 * time.Minute,
				EvictionInterval: 1 * time.Minute,
			},
			wantErr: "",
		},
		{
			name:    "default policy validates successfully",
			policy:  DefaultPolicy(),
			wantErr: "",
		},
		{
			name: "zero MaxEntries",
			policy: Policy{
				MaxEntries:       0,
				TTL:              5 * time.Minute,
				EvictionInterval: 1 * time.Minute,
			},
			wantErr: "MaxEntries must be > 0",
		},
		{
			name: "negative MaxEntries",
			policy: Policy{
				MaxEntries:       -1,
				TTL:              5 * time.Minute,
				EvictionInterval: 1 * time.Minute,
			},
			wantErr: "MaxEntries must be > 0",
		},
		{
			name: "zero TTL",
			policy: Policy{
				MaxEntries:       100,
				TTL:              0,
				EvictionInterval: 1 * time.Minute,
			},
			wantErr: "TTL must be > 0",
		},
		{
			name: "zero EvictionInterval",
			policy: Policy{
				MaxEntries:       100,
				TTL:              5 * time.Minute,
				EvictionInterval: 0,
			},
			wantErr: "EvictionInterval must be > 0",
		},
		{
			name: "EvictionInterval equals TTL",
			policy: Policy{
				MaxEntries:       100,
				TTL:              5 * time.Minute,
				EvictionInterval: 5 * time.Minute,
			},
			wantErr: "EvictionInterval",
		},
		{
			name: "EvictionInterval exceeds TTL",
			policy: Policy{
				MaxEntries:       100,
				TTL:              5 * time.Minute,
				EvictionInterval: 10 * time.Minute,
			},
			wantErr: "EvictionInterval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() returned unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() returned nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestPolicyZeroValue(t *testing.T) {
	var zero Policy
	def := DefaultPolicy()

	// The zero value must be distinguishable from a configured policy.
	// This is relied upon by ratelimit.go: if (policy == eviction.Policy{}) { ... }
	if zero == def {
		t.Fatal("zero-value Policy must not equal DefaultPolicy()")
	}

	// Each field of the zero value should itself be zero.
	if zero.MaxEntries != 0 {
		t.Errorf("zero Policy.MaxEntries = %d, want 0", zero.MaxEntries)
	}
	if zero.TTL != 0 {
		t.Errorf("zero Policy.TTL = %v, want 0", zero.TTL)
	}
	if zero.EvictionInterval != 0 {
		t.Errorf("zero Policy.EvictionInterval = %v, want 0", zero.EvictionInterval)
	}

	// The zero value must fail validation (it is not a usable policy).
	if err := zero.Validate(); err == nil {
		t.Error("zero-value Policy.Validate() returned nil, want error")
	}
}

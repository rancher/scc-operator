package config

import (
	"testing"

	rootLog "github.com/rancher/scc-operator/internal/log"
	"github.com/sirupsen/logrus"
)

func TestDecideLogFormat(t *testing.T) {
	t.Parallel()
	// Valid format
	if got := decideLogFormat("json"); got != rootLog.FormatJSON {
		t.Fatalf("decideLogFormat(json) = %v, want %v", got, rootLog.FormatJSON)
	}

	// Invalid should default
	if got := decideLogFormat("yaml"); got != rootLog.DefaultFormat {
		t.Fatalf("decideLogFormat(yaml) = %v, want default %v", got, rootLog.DefaultFormat)
	}
}

func TestDecideLogLevel(t *testing.T) {
	t.Parallel()
	if got := decideLogLevel("info", true, false); got != logrus.TraceLevel {
		t.Fatalf("decideLogLevel(trace) = %v, want TraceLevel", got)
	}
	if got := decideLogLevel("info", false, true); got != logrus.DebugLevel {
		t.Fatalf("decideLogLevel(debug) = %v, want DebugLevel", got)
	}
	if got := decideLogLevel("error", false, false); got != logrus.ErrorLevel {
		t.Fatalf("decideLogLevel(error) = %v, want ErrorLevel", got)
	}
	if got := decideLogLevel("not-a-level", false, false); got != logrus.InfoLevel {
		t.Fatalf("decideLogLevel(invalid) = %v, want InfoLevel", got)
	}
}

func TestOperatorSettingsValidate(t *testing.T) {
	t.Parallel()
	// Missing operator name
	s := OperatorSettings{}
	if err := s.Validate(); err == nil {
		t.Fatal("Validate() expected error for missing OperatorName, got nil")
	}

	// Missing system namespace
	s.OperatorName = "op"
	if err := s.Validate(); err == nil {
		t.Fatal("Validate() expected error for missing SystemNamespace, got nil")
	}

	// Empty lease namespace should not error (only logs a warning)
	s.SystemNamespace = "scc-ns"
	s.LeaseNamespace = ""
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error with empty LeaseNamespace: %v", err)
	}

	// With all fields set
	s.LeaseNamespace = "kube-system"
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
}

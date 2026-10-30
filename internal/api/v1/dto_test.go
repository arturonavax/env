package v1_test

import (
	"encoding/json"
	"strings"
	"testing"

	v1 "github.com/arturonavax/env/internal/api/v1"
	"github.com/arturonavax/env/internal/domain"
)

func TestEnvelopeSerialization(t *testing.T) {
	planData := v1.PlanResponseDTO{
		PlanDigest: "abc123digest",
		Profile:    "developer",
		StepCount:  2,
		Spec: domain.PlanSpec{
			SchemaVersion: "anx.dev/v1",
			Platform:      "linux",
			Profile:       "developer",
		},
	}

	env := v1.NewEnvelope("test-host-01", "rev-42", planData)
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed marshaling envelope: %v", err)
	}

	raw := string(b)
	if !strings.Contains(raw, v1.CurrentSchemaVersion) {
		t.Errorf("expected envelope to contain schema version %s, got: %s", v1.CurrentSchemaVersion, raw)
	}
	if !strings.Contains(raw, "test-host-01") || !strings.Contains(raw, "abc123digest") {
		t.Errorf("expected envelope metadata in JSON output, got: %s", raw)
	}
}

package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// General
	if !cfg.General.Enabled {
		t.Error("General.Enabled should default to true")
	}

	// Template
	if cfg.Template.Format != "conventional" {
		t.Errorf("Template.Format should be 'conventional', got '%s'", cfg.Template.Format)
	}
	if cfg.Template.CustomTemplate != "" {
		t.Errorf("Template.CustomTemplate should be empty, got '%s'", cfg.Template.CustomTemplate)
	}

	// Trailer (nested inside Template)
	if cfg.Template.Trailer.Format != "Assisted-by" {
		t.Errorf("Trailer.Format should be 'Assisted-by', got '%s'", cfg.Template.Trailer.Format)
	}
	if !cfg.Template.Trailer.IncludeModel {
		t.Error("Trailer.IncludeModel should default to true")
	}

	// AI Detection
	if !cfg.AI.Enabled {
		t.Error("AI.Enabled should default to true")
	}
	if cfg.AI.Mode != "auto" {
		t.Errorf("AI.Mode should be 'auto', got '%s'", cfg.AI.Mode)
	}
	if cfg.AI.OverrideTool != "" {
		t.Errorf("AI.OverrideTool should be empty, got '%s'", cfg.AI.OverrideTool)
	}
	if cfg.AI.ConfidenceThreshold != 0.7 {
		t.Errorf("AI.ConfidenceThreshold should be 0.7, got %f", cfg.AI.ConfidenceThreshold)
	}

	// Journal
	if !cfg.Journal.Enabled {
		t.Error("Journal.Enabled should default to true")
	}
	if cfg.Journal.DBPath != "~/.gitx/commits.db" {
		t.Errorf("Journal.DBPath should be '~/.gitx/commits.db', got '%s'", cfg.Journal.DBPath)
	}
}

func TestDefaultConfigReturnsNewInstance(t *testing.T) {
	// Each call should return a fresh instance, not a shared pointer.
	// Modifying one should not affect the other.
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()

	cfg1.General.Enabled = false
	cfg1.Template.Format = "angular"

	if !cfg2.General.Enabled {
		t.Error("modifying cfg1 should not affect cfg2 — they should be independent instances")
	}
	if cfg2.Template.Format != "conventional" {
		t.Error("modifying cfg1 should not affect cfg2")
	}
}

func TestDefaultConfigZeroValueSafety(t *testing.T) {
	// Verify no field accidentally relies on Go zero values
	// (empty string, false, 0.0) when it shouldn't.
	cfg := DefaultConfig()

	// These should NOT be zero values
	checks := []struct {
		name string
		ok   bool
	}{
		{"General.Enabled", cfg.General.Enabled},
		{"AI.Enabled", cfg.AI.Enabled},
		{"Journal.Enabled", cfg.Journal.Enabled},
		{"Trailer.IncludeModel", cfg.Template.Trailer.IncludeModel},
	}

	for _, c := range checks {
		if !c.ok {
			t.Errorf("%s is false — likely a missing default (Go zero value leak)", c.name)
		}
	}

	// These strings should not be empty
	stringChecks := []struct {
		name  string
		value string
	}{
		{"Template.Format", cfg.Template.Format},
		{"Trailer.Format", cfg.Template.Trailer.Format},
		{"AI.Mode", cfg.AI.Mode},
		{"Journal.DBPath", cfg.Journal.DBPath},
	}

	for _, c := range stringChecks {
		if c.value == "" {
			t.Errorf("%s is empty — likely a missing default", c.name)
		}
	}

	// Confidence threshold should be non-zero
	if cfg.AI.ConfidenceThreshold == 0.0 {
		t.Error("AI.ConfidenceThreshold is 0.0 — likely a missing default")
	}
}

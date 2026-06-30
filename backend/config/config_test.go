package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Title != "Mi Plaza" {
		t.Errorf("expected default title 'Mi Plaza', got %q", cfg.Title)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.MaxUsersPerBench != 33 {
		t.Errorf("expected default max_users 33, got %d", cfg.MaxUsersPerBench)
	}
	if !cfg.AllowAnonymous {
		t.Error("expected AllowAnonymous to be true by default")
	}
	if cfg.Ads {
		t.Error("expected Ads to be false by default")
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	cfg, err := Load("/tmp/nonexistent_plaza_12345.toml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if cfg.Title != "Mi Plaza" {
		t.Errorf("expected defaults, got title %q", cfg.Title)
	}
}

func TestLoadValidFile(t *testing.T) {
	content := []byte(`
title = "Test Plaza"
domain = "test.local"
port = 9090
max_users_per_bench = 10
allow_anonymous = false
history = true
ads = false
data_dir = "./testdata"
`)
	tmpfile, err := os.CreateTemp("", "plaza_test_*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Title != "Test Plaza" {
		t.Errorf("expected 'Test Plaza', got %q", cfg.Title)
	}
	if cfg.Domain != "test.local" {
		t.Errorf("expected 'test.local', got %q", cfg.Domain)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected 9090, got %d", cfg.Port)
	}
	if cfg.MaxUsersPerBench != 10 {
		t.Errorf("expected 10, got %d", cfg.MaxUsersPerBench)
	}
	if cfg.AllowAnonymous {
		t.Error("expected AllowAnonymous to be false")
	}
	if !cfg.History {
		t.Error("expected History to be true")
	}
	if cfg.DataDir != "./testdata" {
		t.Errorf("expected './testdata', got %q", cfg.DataDir)
	}
}

func TestLoadPartialFile(t *testing.T) {
	content := []byte(`title = "Partial Plaza"`)
	tmpfile, err := os.CreateTemp("", "plaza_partial_*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Title != "Partial Plaza" {
		t.Errorf("expected 'Partial Plaza', got %q", cfg.Title)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
}

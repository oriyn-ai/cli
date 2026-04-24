package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleSkill = `---
name: oriyn
version: 1
---

# Oriyn

Test content.
`

func TestFetchSkill_HTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "oriyn-cli" {
			t.Errorf("missing User-Agent: got %q", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write([]byte(sampleSkill))
	}))
	defer srv.Close()

	data, err := fetchSkill(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("fetchSkill: %v", err)
	}
	if string(data) != sampleSkill {
		t.Errorf("content mismatch:\ngot:  %q\nwant: %q", data, sampleSkill)
	}
}

func TestFetchSkill_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill.md")
	if err := os.WriteFile(path, []byte(sampleSkill), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := fetchSkill(context.Background(), path)
	if err != nil {
		t.Fatalf("fetchSkill: %v", err)
	}
	if string(data) != sampleSkill {
		t.Errorf("content mismatch")
	}
}

func TestFetchSkill_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := fetchSkill(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention status: %v", err)
	}
}

func TestFetchSkill_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := fetchSkill(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty: %v", err)
	}
}

func TestFetchSkill_NetworkFailureMentionsURLFlag(t *testing.T) {
	_, err := fetchSkill(context.Background(), "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected connection error")
	}
	if !strings.Contains(err.Error(), "--url") {
		t.Errorf("error should suggest --url flag: %v", err)
	}
}

func TestInstallSkill_WritesFile(t *testing.T) {
	dir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleSkill))
	}))
	defer srv.Close()

	var out bytes.Buffer
	if err := installSkill(context.Background(), &out, dir, srv.URL, false); err != nil {
		t.Fatalf("installSkill: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != sampleSkill {
		t.Errorf("installed content mismatch")
	}
	if !strings.Contains(out.String(), "Installed") {
		t.Errorf("output should confirm install: %q", out.String())
	}
}

func TestInstallSkill_RefusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleSkill))
	}))
	defer srv.Close()

	err := installSkill(context.Background(), new(bytes.Buffer), dir, srv.URL, false)
	if err == nil {
		t.Fatal("expected refusal")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should suggest --force: %v", err)
	}

	got, _ := os.ReadFile(target)
	if string(got) != "existing" {
		t.Errorf("file should be unchanged, got %q", got)
	}
}

func TestInstallSkill_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleSkill))
	}))
	defer srv.Close()

	if err := installSkill(context.Background(), new(bytes.Buffer), dir, srv.URL, true); err != nil {
		t.Fatalf("installSkill: %v", err)
	}

	got, _ := os.ReadFile(target)
	if string(got) != sampleSkill {
		t.Errorf("force install should overwrite, got %q", got)
	}
}

func TestInstallSkill_CreatesNestedDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deeply", "skills")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleSkill))
	}))
	defer srv.Close()

	if err := installSkill(context.Background(), new(bytes.Buffer), dir, srv.URL, false); err != nil {
		t.Fatalf("installSkill: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		t.Errorf("expected SKILL.md in nested dir: %v", err)
	}
}

package whispercpp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLatestReleaseFollowsStableNightlyBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			_ = json.NewEncoder(w).Encode(GithubRelease{
				TagName: "v1.9.4",
				Body:    "## Overview\n\n**Nightly build:** [b5130](https://example.invalid/b5130)",
			})
		case "/releases/tags/b5130":
			_ = json.NewEncoder(w).Encode(GithubRelease{
				TagName: "b5130",
				Assets: []GithubAsset{{
					Name: "whisper-cublas-12.4.0-bin-x64.zip",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	rel, err := latestRelease(server.URL + "/releases")
	if err != nil {
		t.Fatalf("latestRelease() error = %v", err)
	}
	if rel.TagName != "v1.9.4" {
		t.Fatalf("latestRelease() tag = %q, want v1.9.4", rel.TagName)
	}
	if len(rel.Assets) != 1 || rel.Assets[0].Name != "whisper-cublas-12.4.0-bin-x64.zip" {
		t.Fatalf("latestRelease() assets = %#v", rel.Assets)
	}
}

func TestLatestReleaseKeepsReleaseWithDirectAssets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/releases/latest" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(GithubRelease{
			TagName: "v1.8.0",
			Assets: []GithubAsset{{
				Name: "whisper-bin-x64.zip",
			}},
		})
	}))
	defer server.Close()

	rel, err := latestRelease(server.URL + "/releases")
	if err != nil {
		t.Fatalf("latestRelease() error = %v", err)
	}
	if rel.TagName != "v1.8.0" {
		t.Fatalf("latestRelease() tag = %q, want v1.8.0", rel.TagName)
	}
}

func TestLatestReleaseRejectsMissingBuildPointer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(GithubRelease{
			TagName: "v1.9.4",
			Body:    "release without binary metadata",
		})
	}))
	defer server.Close()

	_, err := latestRelease(server.URL + "/releases")
	if err == nil || !strings.Contains(err.Error(), "no binary assets or nightly build pointer") {
		t.Fatalf("latestRelease() error = %v, want missing pointer error", err)
	}
}

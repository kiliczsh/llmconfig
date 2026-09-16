package llamacpp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLatestReleaseFollowsStableNightlyTag(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			_ = json.NewEncoder(w).Encode(GithubRelease{
				TagName: "v0.4.1",
				Assets: []GithubAsset{{
					Name:               "nightly-tag.txt",
					BrowserDownloadURL: server.URL + "/nightly-tag.txt",
				}},
			})
		case "/nightly-tag.txt":
			_, _ = w.Write([]byte("b10964\n"))
		case "/releases/tags/b10964":
			_ = json.NewEncoder(w).Encode(GithubRelease{
				TagName: "b10964",
				Assets: []GithubAsset{{
					Name: "llama-b10964-bin-win-cpu-x64.zip",
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
	if rel.TagName != "v0.4.1" {
		t.Fatalf("latestRelease() tag = %q, want v0.4.1", rel.TagName)
	}
	if len(rel.Assets) != 1 || rel.Assets[0].Name != "llama-b10964-bin-win-cpu-x64.zip" {
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
			TagName: "b10995",
			Assets: []GithubAsset{{
				Name: "llama-b10995-bin-win-cpu-x64.zip",
			}},
		})
	}))
	defer server.Close()

	rel, err := latestRelease(server.URL + "/releases")
	if err != nil {
		t.Fatalf("latestRelease() error = %v", err)
	}
	if rel.TagName != "b10995" {
		t.Fatalf("latestRelease() tag = %q, want b10995", rel.TagName)
	}
}

func TestLatestReleaseRejectsInvalidNightlyTag(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			_ = json.NewEncoder(w).Encode(GithubRelease{
				TagName: "v0.4.1",
				Assets: []GithubAsset{{
					Name:               "nightly-tag.txt",
					BrowserDownloadURL: server.URL + "/nightly-tag.txt",
				}},
			})
		case "/nightly-tag.txt":
			_, _ = w.Write([]byte("not-a-build\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, err := latestRelease(server.URL + "/releases")
	if err == nil || !strings.Contains(err.Error(), "invalid tag") {
		t.Fatalf("latestRelease() error = %v, want invalid tag", err)
	}
}

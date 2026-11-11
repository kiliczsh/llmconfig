package prismllamacpp

import "testing"

func TestPickAssetsMatchesArchitecture(t *testing.T) {
	for _, tc := range []struct {
		os, arch, backend string
		names             []string
		want              int
	}{
		{"darwin", "amd64", "cpu", []string{"bin-macos-arm64.tar.gz", "bin-macos-x64.tar.gz"}, 1},
		{"darwin", "arm64", "metal", []string{"bin-macos-x64.tar.gz", "bin-macos-arm64-kleidiai.tar.gz"}, 1},
		{"linux", "arm64", "cpu", []string{"bin-ubuntu-x64.tar.gz", "bin-ubuntu-arm64.tar.gz"}, 1},
		{"windows", "amd64", "cpu", []string{"bin-win-cpu-arm64.zip", "bin-win-cpu-x64.zip"}, 1},
		{"linux", "arm64", "cuda", []string{"bin-linux-cuda-12.4-x64.tar.gz", "bin-ubuntu-arm64.tar.gz"}, -1},
	} {
		t.Run(tc.os+"/"+tc.arch+"/"+tc.backend, func(t *testing.T) {
			rel := &GithubRelease{}
			for _, name := range tc.names {
				rel.Assets = append(rel.Assets, GithubAsset{Name: "llama-prism-" + name})
			}
			got, err := pickAssets(rel, tc.backend, tc.os, tc.arch)
			if tc.want < 0 {
				if err == nil {
					t.Fatal("selected an incompatible archive")
				}
				return
			}
			if err != nil || len(got) != 1 || got[0] != &rel.Assets[tc.want] {
				t.Fatalf("got %v, error %v; want %s", got, err, tc.names[tc.want])
			}
		})
	}
}

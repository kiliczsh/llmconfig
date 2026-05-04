package ikllamacpp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const RepoURL = "https://github.com/ikawrakow/ik_llama.cpp"

// BuildOptions controls the source build performed by Build().
type BuildOptions struct {
	// Backend selects the cmake feature flags. "cpu" or "cuda".
	// Empty defaults to "cpu" since ik_llama.cpp's only fully supported
	// GPU backend is CUDA.
	Backend string

	// Ref is a git ref (tag, branch, or commit SHA) to check out before
	// building. Empty means use the default branch HEAD.
	Ref string

	// Jobs is the parallel build job count. 0 → runtime.NumCPU().
	Jobs int

	// Verbose mirrors all subprocess output to stderr in addition to the
	// log file. When false, only the log file receives the build output
	// and the caller is expected to render its own progress UI.
	Verbose bool

	// LogPath is the file that receives full git/cmake output. Created if
	// missing. When empty, output is discarded (only Verbose is honored).
	LogPath string

	// OnStep, if non-nil, receives a one-line description each time the
	// build advances to a new phase (clone, configure, compile, install).
	// Useful for driving a Bubble Tea progress view.
	OnStep func(string)
}

// Build clones (or updates) the ik_llama.cpp source tree under CacheDir() and
// runs cmake to produce llama-server / llama-cli, then copies the artifacts
// into BinDir(). The function is intentionally synchronous and idempotent:
// repeated runs reuse the source checkout and incremental cmake build.
func Build(ctx context.Context, opts BuildOptions) error {
	if opts.Backend == "" {
		opts.Backend = "cpu"
	}
	if opts.Backend != "cpu" && opts.Backend != "cuda" {
		return fmt.Errorf("build: unsupported backend %q (want cpu or cuda)", opts.Backend)
	}
	if opts.Jobs <= 0 {
		opts.Jobs = runtime.NumCPU()
	}

	if err := checkPrereqs(opts.Backend); err != nil {
		return err
	}

	logW, closeLog, err := openLog(opts.LogPath)
	if err != nil {
		return err
	}
	defer closeLog()

	step := func(msg string) {
		if opts.OnStep != nil {
			opts.OnStep(msg)
		}
		fmt.Fprintf(logW, "\n=== %s [%s] ===\n", msg, time.Now().Format(time.RFC3339))
	}

	if err := os.MkdirAll(filepath.Dir(CacheDir()), 0755); err != nil {
		return fmt.Errorf("build: create cache dir: %w", err)
	}

	srcDir := CacheDir()
	if _, err := os.Stat(filepath.Join(srcDir, ".git")); err == nil {
		step("updating ik_llama.cpp source")
		if err := runStream(ctx, logW, opts.Verbose, srcDir, "git", "fetch", "--tags", "--prune", "origin"); err != nil {
			return fmt.Errorf("build: git fetch: %w", err)
		}
	} else {
		step("cloning ik_llama.cpp")
		if err := runStream(ctx, logW, opts.Verbose, "", "git", "clone", RepoURL, srcDir); err != nil {
			return fmt.Errorf("build: git clone: %w", err)
		}
	}

	if opts.Ref != "" {
		step("checking out " + opts.Ref)
		if err := runStream(ctx, logW, opts.Verbose, srcDir, "git", "checkout", "--detach", opts.Ref); err != nil {
			return fmt.Errorf("build: git checkout %s: %w", opts.Ref, err)
		}
	} else {
		step("checking out default branch")
		// Pull only when on a branch; "git pull" is a no-op on detached HEAD,
		// but we re-set the branch each time to be safe.
		if err := runStream(ctx, logW, opts.Verbose, srcDir, "git", "checkout", "main"); err != nil {
			// Fall back to master for forks that haven't renamed yet.
			if err2 := runStream(ctx, logW, opts.Verbose, srcDir, "git", "checkout", "master"); err2 != nil {
				return fmt.Errorf("build: git checkout main/master: %w", err)
			}
		}
		if err := runStream(ctx, logW, opts.Verbose, srcDir, "git", "pull", "--ff-only"); err != nil {
			return fmt.Errorf("build: git pull: %w", err)
		}
	}

	buildDir := filepath.Join(srcDir, "build")
	step("configuring (cmake, backend=" + opts.Backend + ")")
	configureArgs := []string{"-B", buildDir, "-S", srcDir, "-DGGML_NATIVE=ON", "-DLLAMA_BUILD_TESTS=OFF", "-DLLAMA_BUILD_EXAMPLES=ON"}
	if opts.Backend == "cuda" {
		configureArgs = append(configureArgs, "-DGGML_CUDA=ON")
	}
	if err := runStream(ctx, logW, opts.Verbose, srcDir, "cmake", configureArgs...); err != nil {
		return fmt.Errorf("build: cmake configure: %w", err)
	}

	step(fmt.Sprintf("compiling (jobs=%d)", opts.Jobs))
	buildArgs := []string{"--build", buildDir, "--config", "Release", "--parallel", strconv.Itoa(opts.Jobs), "--target", "llama-server", "--target", "llama-cli"}
	if err := runStream(ctx, logW, opts.Verbose, srcDir, "cmake", buildArgs...); err != nil {
		return fmt.Errorf("build: cmake build: %w", err)
	}

	step("installing artifacts to " + BinDir())
	if err := os.MkdirAll(BinDir(), 0755); err != nil {
		return fmt.Errorf("build: create bin dir: %w", err)
	}
	if err := copyArtifacts(buildDir, BinDir()); err != nil {
		return err
	}

	return nil
}

// copyArtifacts walks the cmake build directory and copies any file matching
// isUsefulFile() into destDir. Multi-config generators (MSVC) emit binaries
// under build/bin/Release/ while single-config generators (Ninja, Makefile)
// emit them under build/bin/, so we recurse over the whole tree to find them
// regardless of generator.
func copyArtifacts(buildDir, destDir string) error {
	found := 0
	err := filepath.Walk(buildDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if !isUsefulFile(name) {
			return nil
		}
		dest := filepath.Join(destDir, name)
		if err := copyFile(path, dest); err != nil {
			return fmt.Errorf("copy %s: %w", name, err)
		}
		if runtime.GOOS != "windows" {
			_ = os.Chmod(dest, 0755)
		}
		// Only count primary binaries — DLLs pad the score and confuse the
		// "build produced nothing" check below.
		if name == "llama-server" || name == "llama-server.exe" || name == "llama-cli" || name == "llama-cli.exe" {
			found++
		}
		return nil
	})
	if err != nil {
		return err
	}
	if found == 0 {
		return fmt.Errorf("build: no llama-server/llama-cli artifacts found under %s", buildDir)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// runStream runs a subprocess with stdout/stderr merged and tee'd to logW
// (and optionally to the parent stderr when verbose is true). Working
// directory is dir; pass "" for the current directory.
func runStream(ctx context.Context, logW io.Writer, verbose bool, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	fmt.Fprintf(logW, "$ %s %s\n", name, joinArgs(args))

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(logW, line)
			if verbose {
				fmt.Fprintln(os.Stderr, line)
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		<-done
		return err
	}
	runErr := cmd.Wait()
	_ = pw.Close()
	<-done
	return runErr
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}

func openLog(path string) (io.Writer, func(), error) {
	if path == "" {
		return io.Discard, func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, nil, fmt.Errorf("build: create log dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("build: open log: %w", err)
	}
	return f, func() { _ = f.Close() }, nil
}

// checkPrereqs verifies the toolchain we'll need is on PATH. We do this up
// front so users get one clear error instead of a confusing cmake stack
// trace 30 seconds in. CUDA toolkit presence is best-effort: nvcc is the
// canonical probe but its absence is a warning, not a hard fail, since
// some distros put it under /usr/local/cuda/bin instead of PATH.
func checkPrereqs(backend string) error {
	for _, tool := range []string{"git", "cmake"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("build: %s not found on PATH — %s", tool, installHint(tool))
		}
	}
	// A C++ compiler is required. cmake will figure out which one to use,
	// but on Windows MSVC isn't on PATH by default — give an early hint.
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("cl.exe"); err != nil {
			if _, err2 := exec.LookPath("clang-cl.exe"); err2 != nil {
				return fmt.Errorf("build: no C++ compiler found on PATH (cl.exe / clang-cl.exe) — open a \"Developer PowerShell for VS\" or run vcvars64.bat first")
			}
		}
	}
	return nil
}

func installHint(tool string) string {
	switch runtime.GOOS {
	case "linux":
		return "install via: sudo apt install " + tool + " (or your distro's equivalent)"
	case "darwin":
		return "install via: brew install " + tool
	case "windows":
		switch tool {
		case "git":
			return "install via: winget install Git.Git"
		case "cmake":
			return "install via: winget install Kitware.CMake"
		}
	}
	return "install " + tool + " and re-run"
}

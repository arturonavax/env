package configsync_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/configsync"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/security"
)

func TestSmartAppend(t *testing.T) {
	tests := []struct {
		name         string
		initial      string
		newBlock     string
		markerStart  string
		markerEnd    string
		wantContains []string
		wantExcludes []string
	}{
		{
			name:     "append new block preserving user lines",
			initial:  "# User config\nalias ll='ls -la'\n",
			newBlock: "export ANX_ACTIVE=1",
			wantContains: []string{
				"alias ll='ls -la'",
				configsync.DefaultMarkerStart,
				"export ANX_ACTIVE=1",
				configsync.DefaultMarkerEnd,
			},
		},
		{
			name:     "replace existing block preserving surrounding lines",
			initial:  "line 1\n# >>> arturonavax/env >>>\nold block\n# <<< arturonavax/env <<<\nline 2\n",
			newBlock: "updated block",
			wantContains: []string{
				"line 1",
				"updated block",
				"line 2",
			},
			wantExcludes: []string{
				"old block",
			},
		},
		{
			name:        "custom markers on empty file",
			initial:     "",
			newBlock:    "set -g mouse on",
			markerStart: "# BEGIN ANX",
			markerEnd:   "# END ANX",
			wantContains: []string{
				"# BEGIN ANX\nset -g mouse on\n# END ANX\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := configsync.SmartAppend(tt.initial, tt.newBlock, tt.markerStart, tt.markerEnd)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, got)
				}
			}
			for _, unwanted := range tt.wantExcludes {
				if strings.Contains(got, unwanted) {
					t.Errorf("expected output not to contain %q, got:\n%s", unwanted, got)
				}
			}
		})
	}
}

func TestThreeWayMerge(t *testing.T) {
	tests := []struct {
		name             string
		base             string
		local            string
		incoming         string
		wantConflictType domain.ConflictType
		wantHasConflict  bool
		wantResolution   domain.MergeResolution
		wantMerged       string
		wantContains     []string
	}{
		{
			name:             "identical local and incoming is clean",
			base:             "a\nb\n",
			local:            "a\nb\n",
			incoming:         "a\nb\n",
			wantConflictType: domain.ConflictNone,
			wantHasConflict:  false,
			wantResolution:   domain.ResolutionAutoMerged,
			wantMerged:       "a\nb\n",
		},
		{
			name:             "only incoming modified fast-forwards to incoming",
			base:             "hello",
			local:            "hello",
			incoming:         "hello world",
			wantConflictType: domain.ConflictIncomingModified,
			wantHasConflict:  false,
			wantResolution:   domain.ResolutionAutoMerged,
			wantMerged:       "hello world",
		},
		{
			name:             "only local modified keeps local",
			base:             "hello",
			local:            "hello local",
			incoming:         "hello",
			wantConflictType: domain.ConflictLocalModified,
			wantHasConflict:  false,
			wantResolution:   domain.ResolutionAutoMerged,
			wantMerged:       "hello local",
		},
		{
			name:             "non-overlapping hunk edits at top and bottom auto-merge cleanly",
			base:             "font = JetBrainsMono\nsize = 13\nopacity = 1.0\ntheme = dark\nvsync = true\n",
			local:            "font = JetBrainsMono\nsize = 15\nopacity = 1.0\ntheme = dark\nvsync = true\n",
			incoming:         "font = JetBrainsMono\nsize = 13\nopacity = 1.0\ntheme = catppuccin\nvsync = false\n",
			wantConflictType: domain.ConflictBothModified,
			wantHasConflict:  false,
			wantResolution:   domain.ResolutionAutoMerged,
			wantMerged:       "font = JetBrainsMono\nsize = 15\nopacity = 1.0\ntheme = catppuccin\nvsync = false\n",
		},
		{
			name:             "non-overlapping header insertion and footer insertion auto-merge cleanly",
			base:             "middle-1\nmiddle-2\n",
			local:            "local-header\nmiddle-1\nmiddle-2\n",
			incoming:         "middle-1\nmiddle-2\nincoming-footer\n",
			wantConflictType: domain.ConflictBothModified,
			wantHasConflict:  false,
			wantResolution:   domain.ResolutionAutoMerged,
			wantMerged:       "local-header\nmiddle-1\nmiddle-2\nincoming-footer\n",
		},
		{
			name:             "identical hunk edit on both sides plus incoming edit elsewhere auto-merges",
			base:             "line1\nline2\nline3\nline4\n",
			local:            "line1-same\nline2\nline3\nline4\n",
			incoming:         "line1-same\nline2\nline3\nline4-inc\n",
			wantConflictType: domain.ConflictBothModified,
			wantHasConflict:  false,
			wantResolution:   domain.ResolutionAutoMerged,
			wantMerged:       "line1-same\nline2\nline3\nline4-inc\n",
		},
		{
			name:             "overlapping edits on same line produce localized conflict markers while auto-merging separated hunks",
			base:             "header = 1\nsection = core\nshared = base\nmiddle = ok\nfooter = 1\n",
			local:            "header = 2\nsection = core\nshared = local_val\nmiddle = ok\nfooter = 1\n",
			incoming:         "header = 1\nsection = core\nshared = incoming_val\nmiddle = ok\nfooter = 2\n",
			wantConflictType: domain.ConflictBothModified,
			wantHasConflict:  true,
			wantResolution:   domain.ResolutionManual,
			wantContains: []string{
				"header = 2\nsection = core\n",
				"<<<<<<< LOCAL\nshared = local_val\n=======\nshared = incoming_val\n>>>>>>> INCOMING",
				"\nmiddle = ok\nfooter = 2\n",
			},
		},
		{
			name:             "adjacent multi-line local edit overlapping incoming edit coalesces conflict hunk and auto-merges footer",
			base:             "header = 1\nshared = base\nmiddle = ok\nfooter = 1\n",
			local:            "header = 2\nshared = local_val\nmiddle = ok\nfooter = 1\n",
			incoming:         "header = 1\nshared = incoming_val\nmiddle = ok\nfooter = 2\n",
			wantConflictType: domain.ConflictBothModified,
			wantHasConflict:  true,
			wantResolution:   domain.ResolutionManual,
			wantContains: []string{
				"<<<<<<< LOCAL\nheader = 2\nshared = local_val\n=======\nheader = 1\nshared = incoming_val\n>>>>>>> INCOMING",
				"\nmiddle = ok\nfooter = 2\n",
			},
		},
		{
			name:             "deleted local while incoming modified produces conflict",
			base:             "line1\nline2\n",
			local:            "",
			incoming:         "line1\nline2-mod\n",
			wantConflictType: domain.ConflictDeletedLocal,
			wantHasConflict:  true,
			wantResolution:   domain.ResolutionManual,
			wantContains:     []string{"<<<<<<< LOCAL", ">>>>>>> INCOMING"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := configsync.ThreeWayMerge("config.toml", tt.base, tt.local, tt.incoming)
			if diff.ConflictType != tt.wantConflictType {
				t.Fatalf("ConflictType = %s, want %s", diff.ConflictType, tt.wantConflictType)
			}
			if diff.HasConflict != tt.wantHasConflict {
				t.Fatalf("HasConflict = %v, want %v (merged:\n%s)", diff.HasConflict, tt.wantHasConflict, diff.MergedContent)
			}
			if tt.wantResolution != 0 && diff.Resolution != tt.wantResolution {
				t.Fatalf("Resolution = %s, want %s", diff.Resolution, tt.wantResolution)
			}
			if tt.wantMerged != "" && diff.MergedContent != tt.wantMerged {
				t.Fatalf("MergedContent =\n%q\nwant:\n%q", diff.MergedContent, tt.wantMerged)
			}
			for _, sub := range tt.wantContains {
				if !strings.Contains(diff.MergedContent, sub) {
					t.Errorf("MergedContent missing %q, got:\n%s", sub, diff.MergedContent)
				}
			}
		})
	}
}

func TestSyncDirectoryExclusionsAndPermissions(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	dstDir := filepath.Join(root, "dst")

	mustWrite := func(rel string, content string, perm os.FileMode) {
		t.Helper()
		full := filepath.Join(srcDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), perm); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(full, perm); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("init.lua", "require('config.lazy')\n", 0o644)
	mustWrite("bin/run.sh", "#!/usr/bin/env bash\necho ok\n", 0o755)
	mustWrite("lua/plugins/lsp.lua", "return {}\n", 0o644)
	mustWrite(".git/config", "[core]\n", 0o644)
	mustWrite("cache.zwc", "binary-zwc", 0o644)
	mustWrite("debug.log", "runtime-log", 0o644)
	mustWrite("session.json", "{}", 0o644)
	mustWrite("session-snapshots/snap1.json", "{}", 0o644)
	mustWrite(".plugins.lock", "lock", 0o644)

	excludes := []string{".git", "*.zwc", "*.log", "session.json", "session-snapshots", ".plugins.lock"}
	deployer := configsync.NewDeployer()
	count, err := deployer.SyncDirectory(srcDir, dstDir, excludes)
	if err != nil {
		t.Fatalf("SyncDirectory failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("filesSynced = %d, want 3", count)
	}

	// Verify included files and permission bits.
	if runtime.GOOS != "windows" {
		execInfo, err := os.Stat(filepath.Join(dstDir, "bin/run.sh"))
		if err != nil {
			t.Fatalf("stat synced executable: %v", err)
		}
		if execInfo.Mode().Perm() != 0o755 {
			t.Errorf("executable mode = %o, want 0755", execInfo.Mode().Perm())
		}

		regInfo, err := os.Stat(filepath.Join(dstDir, "init.lua"))
		if err != nil {
			t.Fatalf("stat synced regular file: %v", err)
		}
		if regInfo.Mode().Perm() != 0o644 {
			t.Errorf("regular file mode = %o, want 0644", regInfo.Mode().Perm())
		}
	}

	// Verify excluded paths were skipped.
	for _, skipped := range []string{
		".git/config",
		"cache.zwc",
		"debug.log",
		"session.json",
		"session-snapshots/snap1.json",
		".plugins.lock",
	} {
		if _, err := os.Stat(filepath.Join(dstDir, skipped)); !os.IsNotExist(err) {
			t.Errorf("expected excluded path %q not to exist in dstDir", skipped)
		}
	}

	// Verify symlink escape outside srcDir is blocked by SafePath.
	if runtime.GOOS != "windows" {
		outside := filepath.Join(root, "outside-secret.txt")
		if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(srcDir, "escape-link")); err != nil {
			t.Fatal(err)
		}
		_, err := deployer.SyncDirectory(srcDir, filepath.Join(root, "dst2"), excludes)
		if !errors.Is(err, security.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal on escaping symlink, got: %v", err)
		}
	}
}

func TestCaptureToRepo(t *testing.T) {
	t.Run("marker_block extracts only managed block and excludes personal secrets", func(t *testing.T) {
		homeDir := t.TempDir()
		repoRoot := t.TempDir()

		userZshrc := strings.Join([]string{
			"export OPENAI_API_KEY='sk-super-secret-personal-token'",
			"alias mycorp='ssh prod'",
			configsync.DefaultMarkerStart,
			"[[ -f ~/.base.zsh ]] && source ~/.base.zsh",
			"[[ -f ~/.tools.sh ]] && source ~/.tools.sh",
			configsync.DefaultMarkerEnd,
			"export AWS_SECRET_ACCESS_KEY='top-secret'",
			"",
		}, "\n")

		if err := os.WriteFile(filepath.Join(homeDir, ".zshrc"), []byte(userZshrc), 0o600); err != nil {
			t.Fatal(err)
		}

		cfg := domain.ConfigurationResource{
			ID:          "zshrc-loader",
			Source:      "files/zsh/.zshrc",
			Target:      "~/.zshrc",
			Strategy:    domain.StrategyMarkerBlock,
			MarkerStart: configsync.DefaultMarkerStart,
			MarkerEnd:   configsync.DefaultMarkerEnd,
		}

		captured, err := configsync.CaptureToRepo(homeDir, repoRoot, cfg, "linux")
		if err != nil {
			t.Fatalf("CaptureToRepo failed: %v", err)
		}
		if captured != 1 {
			t.Fatalf("captured = %d, want 1", captured)
		}

		gotBytes, err := os.ReadFile(filepath.Join(repoRoot, "files/zsh/.zshrc"))
		if err != nil {
			t.Fatal(err)
		}
		got := string(gotBytes)
		if strings.Contains(got, "OPENAI_API_KEY") || strings.Contains(got, "AWS_SECRET_ACCESS_KEY") {
			t.Fatalf("personal secrets leaked into repository capture:\n%s", got)
		}
		want := "[[ -f ~/.base.zsh ]] && source ~/.base.zsh\n[[ -f ~/.tools.sh ]] && source ~/.tools.sh\n"
		if got != want {
			t.Fatalf("captured block =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("marker_block without markers skips overwriting repo template", func(t *testing.T) {
		homeDir := t.TempDir()
		repoRoot := t.TempDir()

		if err := os.WriteFile(filepath.Join(homeDir, ".zshrc"), []byte("export PERSONAL_TOKEN=123\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		repoFile := filepath.Join(repoRoot, "files/zsh/.zshrc")
		if err := os.MkdirAll(filepath.Dir(repoFile), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(repoFile, []byte("# canonical template\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg := domain.ConfigurationResource{
			ID:       "zshrc-loader",
			Source:   "files/zsh/.zshrc",
			Target:   "~/.zshrc",
			Strategy: domain.StrategyMarkerBlock,
		}

		captured, err := configsync.CaptureToRepo(homeDir, repoRoot, cfg, "linux")
		if err != nil {
			t.Fatalf("CaptureToRepo failed: %v", err)
		}
		if captured != 0 {
			t.Fatalf("captured = %d, want 0", captured)
		}
		got, _ := os.ReadFile(repoFile)
		if string(got) != "# canonical template\n" {
			t.Fatalf("repo template was unexpectedly modified: %q", string(got))
		}
	})

	t.Run("directory_sync honors config excludes plus runtime excludes and OS target resolution", func(t *testing.T) {
		homeDir := t.TempDir()
		repoRoot := t.TempDir()

		nvimDir := filepath.Join(homeDir, ".config/nvim")
		for rel, content := range map[string]string{
			"init.lua":                   "require('config.lazy')\n",
			"lua/plugins/coding.lua":     "return {}\n",
			"lua/plugins/example.lua":    "template\n",
			"debug.log":                  "noisy log\n",
			".git/HEAD":                  "ref: refs/heads/main\n",
			"session.json":               "{}\n",
			"session-snapshots/one.json": "{}\n",
			".plugins.lock":              "lock\n",
		} {
			p := filepath.Join(nvimDir, rel)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		cfg := domain.ConfigurationResource{
			ID:       "nvim-config",
			Source:   "files/nvim",
			Target:   "~/.config/nvim",
			Strategy: domain.StrategyDirectorySync,
			Excludes: []string{"lua/plugins/example.lua"},
		}

		captured, err := configsync.CaptureToRepo(homeDir, repoRoot, cfg, "linux")
		if err != nil {
			t.Fatalf("CaptureToRepo directory_sync failed: %v", err)
		}
		if captured != 2 {
			t.Fatalf("captured = %d, want 2", captured)
		}

		for _, skipped := range []string{"lua/plugins/example.lua", "debug.log", ".git/HEAD", "session.json", ".plugins.lock"} {
			if _, err := os.Stat(filepath.Join(repoRoot, "files/nvim", skipped)); !os.IsNotExist(err) {
				t.Errorf("expected %q to be excluded from repo capture", skipped)
			}
		}
	})

	t.Run("rejects repository path traversal", func(t *testing.T) {
		homeDir := t.TempDir()
		repoRoot := t.TempDir()

		cfg := domain.ConfigurationResource{
			ID:       "malicious",
			Source:   "../../etc/passwd",
			Target:   "~/.bashrc",
			Strategy: domain.StrategyThreeWay,
		}
		_, err := configsync.CaptureToRepo(homeDir, repoRoot, cfg, "linux")
		if !errors.Is(err, security.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got: %v", err)
		}
	})
}

func TestUnifiedDiffAndDiffTree(t *testing.T) {
	t.Run("UnifiedDiff returns empty string for identical input", func(t *testing.T) {
		if got := configsync.UnifiedDiff("a", "b", "same\ntext\n", "same\ntext\n", 3); got != "" {
			t.Fatalf("expected empty diff, got:\n%s", got)
		}
	})

	t.Run("UnifiedDiff generates accurate unified hunks", func(t *testing.T) {
		textA := "line1\nline2\nline3\nline4\nline5\n"
		textB := "line1\nline2-updated\nline3\nline4\nline5\nline6-added\n"

		patch := configsync.UnifiedDiff("repo/config.toml", "home/config.toml", textA, textB, 1)
		for _, want := range []string{
			"--- repo/config.toml",
			"+++ home/config.toml",
			"@@ -1,3 +1,3 @@",
			"-line2",
			"+line2-updated",
			"+line6-added",
		} {
			if !strings.Contains(patch, want) {
				t.Errorf("patch missing %q, got:\n%s", want, patch)
			}
		}
	})

	t.Run("DiffTree detects modified added and removed files while honoring excludes", func(t *testing.T) {
		dirA := t.TempDir()
		dirB := t.TempDir()

		write := func(base, rel, content string) {
			t.Helper()
			p := filepath.Join(base, rel)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		write(dirA, "same.toml", "key = 1\n")
		write(dirB, "same.toml", "key = 1\n")

		write(dirA, "changed.toml", "theme = 'dark'\n")
		write(dirB, "changed.toml", "theme = 'light'\n")

		write(dirA, "only-in-a.lua", "-- removed\n")
		write(dirB, "only-in-b.lua", "-- added\n")

		write(dirA, "ignored.log", "old log\n")
		write(dirB, "ignored.log", "new log\n")

		diffs, err := configsync.DiffTree(dirA, dirB, []string{"*.log"})
		if err != nil {
			t.Fatalf("DiffTree failed: %v", err)
		}
		if len(diffs) != 3 {
			t.Fatalf("len(diffs) = %d, want 3: %+v", len(diffs), diffs)
		}

		gotStatuses := map[string]configsync.DiffStatus{}
		for _, d := range diffs {
			gotStatuses[d.RelPath] = d.Status
			if d.Patch == "" {
				t.Errorf("expected non-empty patch for %s", d.RelPath)
			}
		}

		if gotStatuses["changed.toml"] != configsync.DiffModified {
			t.Errorf("changed.toml status = %q, want %q", gotStatuses["changed.toml"], configsync.DiffModified)
		}
		if gotStatuses["only-in-a.lua"] != configsync.DiffRemoved {
			t.Errorf("only-in-a.lua status = %q, want %q", gotStatuses["only-in-a.lua"], configsync.DiffRemoved)
		}
		if gotStatuses["only-in-b.lua"] != configsync.DiffAdded {
			t.Errorf("only-in-b.lua status = %q, want %q", gotStatuses["only-in-b.lua"], configsync.DiffAdded)
		}
	})
}

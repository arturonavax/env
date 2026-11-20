package osconfig

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

type fakeExec struct {
	calls   []string
	version int
}

func (f *fakeExec) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	switch name {
	case "sw_vers":
		return []byte("15.1\n"), nil
	case "pmset":
		return []byte(" lowpowermode         0\n"), nil
	case "defaults":
		if strings.Contains(" "+strings.Join(args, " ")+" ", " export ") {
			return []byte("<plist>v" + string(rune('0'+f.version)) + " " + strings.Join(args, " ") + "</plist>"), nil
		}
	}
	return nil, nil
}

func newTestManager(t *testing.T, f *fakeExec) (*Manager, *time.Time) {
	t.Helper()
	dir := t.TempDir()
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := NewManager(filepath.Join(dir, "backups"), dir)
	m.SudoLocalPath = filepath.Join(dir, "sudo_local")
	m.Exec = f.run
	m.Now = func() time.Time { return clock }
	return m, &clock
}

func TestCreateDedupesIdenticalState(t *testing.T) {
	f := &fakeExec{}
	m, clock := newTestManager(t, f)
	a, created, err := m.Create(context.Background(), "a", true)
	if err != nil || !created {
		t.Fatalf("first create: %v created=%v", err, created)
	}
	*clock = clock.Add(time.Minute)
	b, created, err := m.Create(context.Background(), "b", true)
	if err != nil || created || b.ID != a.ID {
		t.Fatalf("expected dedupe to reuse %s, got %v created=%v err=%v", a.ID, b, created, err)
	}
	if all, _ := m.List(); len(all) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(all))
	}
}

func TestPruneKeepsFirstPlusNewest(t *testing.T) {
	f := &fakeExec{}
	m, clock := newTestManager(t, f)
	var ids []string
	for i := 1; i <= 8; i++ {
		f.version = i
		*clock = clock.Add(time.Minute)
		man, created, err := m.Create(context.Background(), "run", true)
		if err != nil || !created {
			t.Fatalf("create %d: %v %v", i, err, created)
		}
		ids = append(ids, man.ID)
	}
	all, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != DefaultKeep {
		t.Fatalf("expected %d snapshots, got %d", DefaultKeep, len(all))
	}
	want := []string{ids[0], ids[4], ids[5], ids[6], ids[7]}
	for i, man := range all {
		if man.ID != want[i] {
			t.Fatalf("snapshot %d = %s, want %s", i, man.ID, want[i])
		}
	}
	if !all[0].First {
		t.Fatal("baseline must be flagged First")
	}
}

func TestRestoreIssuesImportsAndSudoSteps(t *testing.T) {
	f := &fakeExec{}
	m, clock := newTestManager(t, f)
	if err := os.WriteFile(m.SudoLocalPath, []byte("auth sufficient pam_tid.so\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.Create(context.Background(), "baseline", true); err != nil {
		t.Fatal(err)
	}
	f.version = 2
	*clock = clock.Add(time.Minute)
	f.calls = nil
	rep, err := m.Restore(context.Background(), "first", false, true)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(f.calls, "\n")
	for _, want := range []string{
		"defaults import NSGlobalDomain",
		"defaults -currentHost import com.apple.controlcenter",
		"sudo -n defaults import /Library/Preferences/com.apple.loginwindow",
		"sudo -n install -m 0644",
		"sudo -n pmset -a lowpowermode 0",
		"killall cfprefsd",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing call %q in:\n%s", want, joined)
		}
	}
	if rep.SafetyCopy == "" {
		t.Error("expected a pre-restore safety copy")
	}
	if !rep.NeedsSudo {
		t.Error("expected NeedsSudo")
	}
}

func TestRestoreRemovesSudoLocalCreatedByAnx(t *testing.T) {
	f := &fakeExec{}
	m, _ := newTestManager(t, f)
	if _, _, err := m.Create(context.Background(), "baseline", true); err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	if _, err := m.Restore(context.Background(), "first", true, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(f.calls, "\n"), "sudo -n rm -f "+m.SudoLocalPath) {
		t.Errorf("expected sudo_local removal, calls:\n%v", f.calls)
	}
}

// TestManagedDomainsCoverScript keeps ManagedDomains in sync with scripts/osconfig/macos.sh.
func TestManagedDomainsCoverScript(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", "osconfig", "macos.sh"))
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, d := range ManagedDomains("/home/x") {
		known[strings.ToLower(d.Domain)] = true
	}
	re := regexp.MustCompile(`(?m)^\s*(?:defaults_write|defaults_curhost_write|defaults_write_optional)\s+("?[A-Za-z0-9._-]+"?)\s`)
	for _, mt := range re.FindAllStringSubmatch(string(raw), -1) {
		dom := strings.Trim(mt[1], `"`)
		if !known[strings.ToLower(dom)] {
			t.Errorf("domain %q written by macos.sh is not covered by ManagedDomains", dom)
		}
	}
}

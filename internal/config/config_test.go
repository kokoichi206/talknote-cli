package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadResolve(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	account := Account{SID: "secret", Origin: "https://s01.company.talknote.com", UserID: "1"}
	cfg := &Config{Default: "work", Accounts: map[string]Account{"work": account}}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %#o", info.Mode().Perm())
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	name, got, err := loaded.Resolve("")
	if err != nil || name != "work" || got != account {
		t.Fatalf("name=%q account=%+v err=%v", name, got, err)
	}
}

func TestFindProject(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(filepath.Join(root, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".config", ProjectConfigName), []byte(`{"notes":{"main":"1"},"dms":{"owner":"2"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := FindProject(nested, filepath.Dir(root))
	if err != nil || project.Notes["main"] != "1" || project.DMs["owner"] != "2" {
		t.Fatalf("project=%+v err=%v", project, err)
	}
}

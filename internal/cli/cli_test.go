package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kokoichi206/talknote-cli/internal/config"
	"github.com/kokoichi206/talknote-cli/internal/talknote"
)

type runOptions struct {
	stdin      string
	clipboard  string
	configPath string
}

func runCLI(t *testing.T, handler http.HandlerFunc, options runOptions, args ...string) (string, string, error) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	if options.configPath == "" {
		options.configPath = filepath.Join(t.TempDir(), "accounts.json")
	}
	testRoot := t.TempDir()
	var stdout, stderr bytes.Buffer
	root := New(Deps{
		Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(options.stdin),
		ConfigPath: options.configPath, Clipboard: func() (string, error) { return options.clipboard, nil },
		Home: testRoot, WorkDir: testRoot,
		NewClient: func(account config.Account) *talknote.Client { return talknote.New(server.URL, account.SID) },
	})
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func seedConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "accounts.json")
	account := config.Account{SID: "saved-session", Origin: "https://s01.company.talknote.com"}
	if err := (&config.Config{Default: "work", Accounts: map[string]config.Account{"work": account}}).Save(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAuthLoginValidatesAndHidesSID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/web-api/v1/network/user/me":
			_, _ = w.Write([]byte(`{"id":100,"firstName":"貴弘","lastName":"冨永"}`))
		case "/web-api/v1/network/current":
			_, _ = w.Write([]byte(`{"id":200,"name":"株式会社ジャパゲートシステムズ"}`))
		default:
			http.NotFound(w, r)
		}
	}
	curlText := `curl 'https://s01.company.talknote.com/code/ajax/feed/list' -b 'TALKNOTE_SID2=new-secret'`
	configPath := filepath.Join(t.TempDir(), "accounts.json")
	stdout, _, err := runCLI(t, handler, runOptions{clipboard: curlText, configPath: configPath}, "auth", "login", "--from-clipboard", "--name", "work", "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout, "new-secret") {
		t.Fatalf("secret leaked: %s", stdout)
	}
	cfg, err := config.Load(configPath)
	if err != nil || cfg.Accounts["work"].SID != "new-secret" || cfg.Accounts["work"].UserID != "100" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}

func TestNotesListFilterJSON(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"content":[{"data":{"id":1,"name":"Daily Report"}},{"data":{"id":2,"name":"General"}}]}`))
	}
	stdout, _, err := runCLI(t, handler, runOptions{configPath: seedConfig(t)}, "notes", "list", "--filter", "daily", "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Notes []talknote.Group `json:"notes"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil || len(result.Notes) != 1 || result.Notes[0].Data.ID != "1" {
		t.Fatalf("result=%+v err=%v stdout=%s", result, err, stdout)
	}
}

func TestNoteSendUsesWebAPIJSON(t *testing.T) {
	var content string
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/web-api/v1/group/42/feed" {
			t.Errorf("path=%s", r.URL.Path)
		}
		var body struct {
			Content string `json:"content"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		content = body.Content
		_, _ = w.Write([]byte(`{"id":99}`))
	}
	stdout, _, err := runCLI(t, handler, runOptions{stdin: "line one\nline two\n", configPath: seedConfig(t)}, "notes", "send", "42", "-", "--output", "json")
	if err != nil || content != "line one\nline two" || !strings.Contains(stdout, `"id": "99"`) {
		t.Fatalf("content=%q stdout=%s err=%v", content, stdout, err)
	}
}

func TestDMReadText(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/web-api/v1/thread/7/message" {
			t.Errorf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"content":[{"data":{"id":8,"content":"hello","postedUser":{"firstName":"太郎","lastName":"山田"},"createdAt":1788480000000}}]}`))
	}
	stdout, _, err := runCLI(t, handler, runOptions{configPath: seedConfig(t)}, "dms", "read", "7", "--output", "text")
	if err != nil || !strings.Contains(stdout, "hello") || !strings.Contains(stdout, "太郎 山田") {
		t.Fatalf("stdout=%s err=%v", stdout, err)
	}
}

func TestUsageErrorExitCode(t *testing.T) {
	_, _, err := runCLI(t, func(http.ResponseWriter, *http.Request) {}, runOptions{}, "notes", "read")
	if ExitCode(err) != 2 {
		t.Fatalf("err=%v code=%d", err, ExitCode(err))
	}
}

func TestDeleteRequiresYesNonInteractive(t *testing.T) {
	_, _, err := runCLI(t, func(http.ResponseWriter, *http.Request) {}, runOptions{configPath: seedConfig(t)}, "notes", "delete", "42", "99")
	if ExitCode(err) != 2 {
		t.Fatalf("err=%v code=%d", err, ExitCode(err))
	}
}

func TestInvalidOutputStopsMutationBeforeRequest(t *testing.T) {
	requests := 0
	handler := func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"id":99}`))
	}
	_, _, err := runCLI(t, handler, runOptions{configPath: seedConfig(t)}, "notes", "send", "42", "hello", "--output", "yaml")
	if ExitCode(err) != 2 || requests != 0 {
		t.Fatalf("err=%v code=%d requests=%d", err, ExitCode(err), requests)
	}
}

func TestEmptyMessageFileStopsMutation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	requests := 0
	_, _, err := runCLI(t, func(http.ResponseWriter, *http.Request) { requests++ }, runOptions{configPath: seedConfig(t)}, "notes", "send", "42", "--file", file)
	if ExitCode(err) != 2 || requests != 0 {
		t.Fatalf("err=%v code=%d requests=%d", err, ExitCode(err), requests)
	}
}

func TestMessageArgumentAndFileAreMutuallyExclusive(t *testing.T) {
	file := filepath.Join(t.TempDir(), "message.txt")
	if err := os.WriteFile(file, []byte("from file"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := runCLI(t, func(http.ResponseWriter, *http.Request) {}, runOptions{configPath: seedConfig(t)}, "notes", "send", "42", "from arg", "--file", file)
	if ExitCode(err) != 2 {
		t.Fatalf("err=%v code=%d", err, ExitCode(err))
	}
}

func TestUnknownAliasIsUsageError(t *testing.T) {
	requests := 0
	_, _, err := runCLI(t, func(http.ResponseWriter, *http.Request) { requests++ }, runOptions{configPath: seedConfig(t)}, "notes", "read", "typo")
	if ExitCode(err) != 2 || requests != 0 {
		t.Fatalf("err=%v code=%d requests=%d", err, ExitCode(err), requests)
	}
}

func TestMissingAccountIsUsageError(t *testing.T) {
	_, _, err := runCLI(t, func(http.ResponseWriter, *http.Request) {}, runOptions{}, "notes", "list")
	if ExitCode(err) != 2 {
		t.Fatalf("err=%v code=%d", err, ExitCode(err))
	}
}

func TestParentCommandRejectsArguments(t *testing.T) {
	_, _, err := runCLI(t, func(http.ResponseWriter, *http.Request) {}, runOptions{}, "notes", "typo")
	if ExitCode(err) != 2 {
		t.Fatalf("err=%v code=%d", err, ExitCode(err))
	}
}

func TestParseDeadlineRejectsEightDigitNumber(t *testing.T) {
	if _, err := parseDeadline("20260905"); ExitCode(err) != 2 {
		t.Fatalf("err=%v code=%d", err, ExitCode(err))
	}
}

func TestInvalidNestedIDStopsRequest(t *testing.T) {
	requests := 0
	_, _, err := runCLI(t, func(http.ResponseWriter, *http.Request) { requests++ }, runOptions{configPath: seedConfig(t)}, "notes", "delete", "42", "not-an-id", "--yes")
	if ExitCode(err) != 2 || requests != 0 {
		t.Fatalf("err=%v code=%d requests=%d", err, ExitCode(err), requests)
	}
}

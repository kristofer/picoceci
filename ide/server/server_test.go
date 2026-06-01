package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFrontendShellServesIDEControls(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, needle := range []string{
		`hx-get="/api/project/tree"`,
		`id="editor-content"`,
		`id="save-button"`,
		`id="run-button"`,
		`id="repl-form"`,
		`/static/ide.js`,
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("expected %q in index page, got %q", needle, body)
		}
	}
}

func TestProjectTreeRendersSampleFiles(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/project/tree", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, needle := range []string{`data-path="counter.pc"`, `data-path="hello.pc"`} {
		if !strings.Contains(body, needle) {
			t.Fatalf("expected %q in project tree, got %q", needle, body)
		}
	}
}

func TestFileCRUDAndPathSafety(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	createReq := httptest.NewRequest(http.MethodPost, "/api/files/scratch.pc", strings.NewReader(`{"content":"1 + 2."}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("expected create 200, got %d: %s", createRec.Code, createRec.Body.String())
	}

	readReq := httptest.NewRequest(http.MethodGet, "/api/files/scratch.pc", nil)
	readRec := httptest.NewRecorder()
	handler.ServeHTTP(readRec, readReq)
	if readRec.Code != http.StatusOK {
		t.Fatalf("expected read 200, got %d: %s", readRec.Code, readRec.Body.String())
	}
	var file fileResponse
	if err := json.NewDecoder(readRec.Body).Decode(&file); err != nil {
		t.Fatalf("decode read response: %v", err)
	}
	if file.Content != "1 + 2." {
		t.Fatalf("expected file content to round-trip, got %q", file.Content)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/files/scratch.pc", strings.NewReader("content=3+4."))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateRec := httptest.NewRecorder()
	handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/files/scratch.pc", nil)
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	escapeReq := httptest.NewRequest(http.MethodGet, "/api/files/../outside.pc", nil)
	escapeRec := httptest.NewRecorder()
	handler.ServeHTTP(escapeRec, escapeReq)
	if escapeRec.Code != http.StatusBadRequest {
		t.Fatalf("expected path traversal rejection, got %d: %s", escapeRec.Code, escapeRec.Body.String())
	}
}

func TestExecuteAndREPLSessionHistory(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	executeReq := httptest.NewRequest(http.MethodPost, "/api/execute", strings.NewReader(`{"code":"Console println: 'hi'. 2 + 3."}`))
	executeReq.Header.Set("Content-Type", "application/json")
	executeRec := httptest.NewRecorder()
	handler.ServeHTTP(executeRec, executeReq)
	if executeRec.Code != http.StatusOK {
		t.Fatalf("expected execute 200, got %d: %s", executeRec.Code, executeRec.Body.String())
	}
	var execResp executionResponse
	if err := json.NewDecoder(executeRec.Body).Decode(&execResp); err != nil {
		t.Fatalf("decode execute response: %v", err)
	}
	if execResp.Result != "5" {
		t.Fatalf("expected result 5, got %+v", execResp)
	}
	if execResp.Output != "hi" {
		t.Fatalf("expected console output hi, got %+v", execResp)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/repl/create", nil)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create session 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created replCreateResponse
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty REPL session ID")
	}

	firstEval := httptest.NewRequest(http.MethodPost, "/api/repl/"+created.ID+"/eval", strings.NewReader(`{"code":"let counter: Int. counter := 41."}`))
	firstEval.Header.Set("Content-Type", "application/json")
	firstEvalRec := httptest.NewRecorder()
	handler.ServeHTTP(firstEvalRec, firstEval)
	if firstEvalRec.Code != http.StatusOK {
		t.Fatalf("expected first eval 200, got %d: %s", firstEvalRec.Code, firstEvalRec.Body.String())
	}

	secondEval := httptest.NewRequest(http.MethodPost, "/api/repl/"+created.ID+"/eval", strings.NewReader(`{"code":"counter + 1."}`))
	secondEval.Header.Set("Content-Type", "application/json")
	secondEvalRec := httptest.NewRecorder()
	handler.ServeHTTP(secondEvalRec, secondEval)
	if secondEvalRec.Code != http.StatusOK {
		t.Fatalf("expected second eval 200, got %d: %s", secondEvalRec.Code, secondEvalRec.Body.String())
	}
	var historyEntry HistoryEntry
	if err := json.NewDecoder(secondEvalRec.Body).Decode(&historyEntry); err != nil {
		t.Fatalf("decode eval response: %v", err)
	}
	if historyEntry.Result != "42" {
		t.Fatalf("expected persisted REPL state result 42, got %+v", historyEntry)
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/api/repl/"+created.ID+"/history", nil)
	historyRec := httptest.NewRecorder()
	handler.ServeHTTP(historyRec, historyReq)
	if historyRec.Code != http.StatusOK {
		t.Fatalf("expected history 200, got %d: %s", historyRec.Code, historyRec.Body.String())
	}
	var history []HistoryEntry
	if err := json.NewDecoder(historyRec.Body).Decode(&history); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}

	artifactPath := sessionArtifactPath(server.artifactRoot, created.ID)
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("expected persisted session artifact: %v", err)
	}
	if !bytes.Contains(data, []byte(`"result": "42"`)) {
		t.Fatalf("expected persisted history to include result 42, got %s", string(data))
	}
}

func TestExpiredSessionIsCleanedUp(t *testing.T) {
	server := newTestServer(t)
	server.sessionTTL = time.Minute
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	server.now = func() time.Time { return base }

	session, err := server.newSession()
	if err != nil {
		t.Fatalf("new session: %v", err)
	}

	server.now = func() time.Time { return base.Add(2 * time.Minute) }
	req := httptest.NewRequest(http.MethodGet, "/api/repl/"+session.id+"/history", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected expired session lookup to return 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(sessionArtifactPath(server.artifactRoot, session.id)); !os.IsNotExist(err) {
		t.Fatalf("expected expired session artifact to be deleted, stat err=%v", err)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	repoRoot := repoRoot(t)
	projectRoot := filepath.Join(repoRoot, "ide", "testdata", "sample-project")
	staticRoot := filepath.Join(repoRoot, "ide", "static")
	artifactRoot := filepath.Join(t.TempDir(), "artifacts")

	server, err := NewServer(projectRoot, staticRoot, artifactRoot)
	if err != nil {
		t.Fatalf("new test server: %v", err)
	}
	return server
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kristofer/picoceci/pkg/eval"
)

const defaultSessionTTL = 30 * time.Minute

type Server struct {
	projectRoot  string
	staticRoot   string
	artifactRoot string
	sessionTTL   time.Duration
	now          func() time.Time

	mu       sync.Mutex
	sessions map[string]*REPLSession
}

type REPLSession struct {
	id        string
	interp    *eval.Interpreter
	console   *captureBuffer
	history   []HistoryEntry
	updatedAt time.Time

	mu sync.Mutex
}

type HistoryEntry struct {
	Code      string    `json:"code"`
	Result    string    `json:"result,omitempty"`
	Output    string    `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type fileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

type fileResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type fileRequest struct {
	Content string `json:"content"`
}

type executionRequest struct {
	Code string `json:"code"`
}

type executionResponse struct {
	Result string `json:"result,omitempty"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

type replCreateResponse struct {
	ID string `json:"id"`
}

type captureBuffer struct {
	bytes.Buffer
}

func NewServer(projectRoot, staticRoot, artifactRoot string) (*Server, error) {
	absProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}
	if err := ensureDirectory(absProjectRoot, "project root"); err != nil {
		return nil, err
	}

	absStaticRoot, err := filepath.Abs(staticRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve static root: %w", err)
	}
	if err := ensureDirectory(absStaticRoot, "static root"); err != nil {
		return nil, err
	}

	if artifactRoot == "" {
		artifactRoot = filepath.Join(absProjectRoot, ".picoceci-ide")
	}
	absArtifactRoot, err := filepath.Abs(artifactRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve artifact root: %w", err)
	}
	if err := os.MkdirAll(absArtifactRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create artifact root: %w", err)
	}

	return &Server{
		projectRoot:  absProjectRoot,
		staticRoot:   absStaticRoot,
		artifactRoot: absArtifactRoot,
		sessionTTL:   defaultSessionTTL,
		now:          time.Now,
		sessions:     make(map[string]*REPLSession),
	}, nil
}

func ensureDirectory(path string, description string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s: %w", description, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory: %s", description, path)
	}
	return nil
}

func (s *Server) Handler() http.Handler {
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir(s.staticRoot)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			s.handleIndex(w, r)
		case strings.HasPrefix(r.URL.Path, "/static/"):
			staticHandler.ServeHTTP(w, r)
		case r.URL.Path == "/api/files" || r.URL.Path == "/api/files/" || strings.HasPrefix(r.URL.Path, "/api/files/"):
			s.handleFiles(w, r)
		case r.URL.Path == "/api/execute":
			s.handleExecute(w, r)
		case r.URL.Path == "/api/repl/create":
			s.handleREPLCreate(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/repl/"):
			s.handleREPL(w, r)
		case r.URL.Path == "/api/project/tree":
			s.handleProjectTree(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.staticRoot, "index.html"))
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/files" || r.URL.Path == "/api/files/" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		files, err := s.listProjectFiles()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, files)
		return
	}

	relPath := strings.TrimPrefix(r.URL.Path, "/api/files/")
	fullPath, err := s.resolveProjectPath(relPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(fullPath)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, fileResponse{Path: relPath, Content: string(data)})
	case http.MethodPost:
		if _, err := os.Stat(fullPath); err == nil {
			writeError(w, http.StatusConflict, "file already exists")
			return
		}
		fallthrough
	case http.MethodPut:
		content, err := decodeContent(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, fileResponse{Path: relPath, Content: content})
	case http.MethodDelete:
		if err := os.Remove(fullPath); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	code, err := decodeCode(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result := s.executeSource(code)
	if wantsHTML(r) {
		writeConsoleHTML(w, result)
		return
	}

	status := http.StatusOK
	if result.Error != "" {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, result)
}

func (s *Server) handleREPLCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	session, err := s.newSession()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, replCreateResponse{ID: session.id})
}

func (s *Server) handleREPL(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/repl/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	sessionID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case r.Method == http.MethodPost && action == "eval":
		code, err := decodeCode(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		entry, status, err := s.evalSession(sessionID, code)
		if err != nil {
			writeError(w, status, err.Error())
			return
		}
		if wantsHTML(r) {
			writeConsoleHTML(w, executionResponse{
				Result: entry.Result,
				Output: entry.Output,
				Error:  entry.Error,
			})
			return
		}
		writeJSON(w, http.StatusOK, entry)
	case r.Method == http.MethodGet && action == "history":
		history, status, err := s.sessionHistory(sessionID)
		if err != nil {
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, history)
	case r.Method == http.MethodDelete && action == "":
		status, err := s.deleteSession(sessionID)
		if err != nil {
			writeError(w, status, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleProjectTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	htmlTree, err := s.renderProjectTree()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, htmlTree)
}

func (s *Server) listProjectFiles() ([]fileEntry, error) {
	files := make([]fileEntry, 0)
	err := filepath.WalkDir(s.projectRoot, func(fullPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if fullPath == s.projectRoot {
			return nil
		}
		if sameOrChildPath(fullPath, s.artifactRoot) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(s.projectRoot, fullPath)
		if err != nil {
			return err
		}
		files = append(files, fileEntry{
			Name:  d.Name(),
			Path:  filepath.ToSlash(rel),
			IsDir: d.IsDir(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func (s *Server) renderProjectTree() (string, error) {
	var buf strings.Builder
	buf.WriteString(`<div class="file-tree__root">`)
	if err := s.renderTreeDirectory(&buf, s.projectRoot, ""); err != nil {
		return "", err
	}
	buf.WriteString(`</div>`)
	return buf.String(), nil
}

func (s *Server) renderTreeDirectory(buf *strings.Builder, dirPath, relPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	buf.WriteString(`<ul class="file-tree">`)
	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())
		if sameOrChildPath(fullPath, s.artifactRoot) {
			continue
		}
		nextRel := entry.Name()
		if relPath != "" {
			nextRel = relPath + "/" + entry.Name()
		}
		if entry.IsDir() {
			buf.WriteString(`<li class="file-tree__dir"><details open><summary>`)
			buf.WriteString(html.EscapeString(entry.Name()))
			buf.WriteString(`/</summary>`)
			if err := s.renderTreeDirectory(buf, fullPath, nextRel); err != nil {
				return err
			}
			buf.WriteString(`</details></li>`)
			continue
		}
		buf.WriteString(`<li class="file-tree__file"><button type="button" class="file-tree__button" data-path="`)
		buf.WriteString(html.EscapeString(nextRel))
		buf.WriteString(`" onclick="picoceciIDE.openFile(this.dataset.path)">`)
		buf.WriteString(html.EscapeString(entry.Name()))
		buf.WriteString(`</button></li>`)
	}
	buf.WriteString(`</ul>`)
	return nil
}

func (s *Server) executeSource(code string) executionResponse {
	console := &captureBuffer{}
	interp := eval.NewWithSinks(eval.GlobalSinks{
		ConsoleWriter:    console,
		TranscriptWriter: console,
	})
	result, err := interp.EvalSource(code)
	resp := executionResponse{Output: strings.TrimSpace(console.String())}
	if err != nil {
		resp.Error = err.Error()
		return resp
	}
	if result != nil {
		resp.Result = result.PrintString()
	}
	return resp
}

func (s *Server) newSession() (*REPLSession, error) {
	s.cleanupExpiredSessions()

	sessionID, err := randomID()
	if err != nil {
		return nil, err
	}
	console := &captureBuffer{}
	session := &REPLSession{
		id:        sessionID,
		interp:    eval.NewWithSinks(eval.GlobalSinks{ConsoleWriter: console, TranscriptWriter: console}),
		console:   console,
		history:   make([]HistoryEntry, 0),
		updatedAt: s.now(),
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	if err := s.persistSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Server) evalSession(sessionID, code string) (HistoryEntry, int, error) {
	session, status, err := s.lookupSession(sessionID)
	if err != nil {
		return HistoryEntry{}, status, err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.console.Reset()
	result, evalErr := session.interp.EvalSource(code)
	entry := HistoryEntry{
		Code:      code,
		Timestamp: s.now(),
		Output:    strings.TrimSpace(session.console.String()),
	}
	if evalErr != nil {
		entry.Error = evalErr.Error()
	} else if result != nil {
		entry.Result = result.PrintString()
	}
	session.history = append(session.history, entry)
	session.updatedAt = entry.Timestamp

	if err := s.persistSession(session); err != nil {
		return HistoryEntry{}, http.StatusInternalServerError, err
	}

	return entry, http.StatusOK, nil
}

func (s *Server) sessionHistory(sessionID string) ([]HistoryEntry, int, error) {
	session, status, err := s.lookupSession(sessionID)
	if err != nil {
		return nil, status, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	history := append([]HistoryEntry(nil), session.history...)
	return history, http.StatusOK, nil
}

func (s *Server) deleteSession(sessionID string) (int, error) {
	s.mu.Lock()
	session, ok := s.sessions[sessionID]
	if ok {
		delete(s.sessions, sessionID)
	}
	s.mu.Unlock()
	if !ok {
		return http.StatusNotFound, fmt.Errorf("unknown session %q", sessionID)
	}
	if err := os.Remove(sessionArtifactPath(s.artifactRoot, session.id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return http.StatusInternalServerError, err
	}
	return http.StatusNoContent, nil
}

func (s *Server) lookupSession(sessionID string) (*REPLSession, int, error) {
	s.cleanupExpiredSessions()
	s.mu.Lock()
	session, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		return nil, http.StatusNotFound, fmt.Errorf("unknown session %q", sessionID)
	}
	session.mu.Lock()
	expired := s.now().Sub(session.updatedAt) > s.sessionTTL
	session.mu.Unlock()
	if expired {
		_, _ = s.deleteSession(sessionID)
		return nil, http.StatusNotFound, fmt.Errorf("session %q expired", sessionID)
	}
	return session, http.StatusOK, nil
}

func (s *Server) cleanupExpiredSessions() {
	now := s.now()

	s.mu.Lock()
	expiredIDs := make([]string, 0)
	for id, session := range s.sessions {
		session.mu.Lock()
		expired := now.Sub(session.updatedAt) > s.sessionTTL
		session.mu.Unlock()
		if expired {
			expiredIDs = append(expiredIDs, id)
			delete(s.sessions, id)
		}
	}
	s.mu.Unlock()

	for _, id := range expiredIDs {
		_ = os.Remove(sessionArtifactPath(s.artifactRoot, id))
	}
}

func (s *Server) persistSession(session *REPLSession) error {
	payload := struct {
		ID        string         `json:"id"`
		UpdatedAt time.Time      `json:"updatedAt"`
		History   []HistoryEntry `json:"history"`
	}{
		ID:        session.id,
		UpdatedAt: session.updatedAt,
		History:   append([]HistoryEntry(nil), session.history...),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessionArtifactPath(s.artifactRoot, session.id), data, 0o644)
}

func sessionArtifactPath(artifactRoot, sessionID string) string {
	return filepath.Join(artifactRoot, sessionID+".json")
}

func sameOrChildPath(candidate, root string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func randomID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func (s *Server) resolveProjectPath(relPath string) (string, error) {
	for _, segment := range strings.Split(relPath, "/") {
		if segment == ".." {
			return "", fmt.Errorf("path escapes project root")
		}
	}
	cleaned := path.Clean("/" + relPath)
	if cleaned == "/" || cleaned == "." {
		return s.projectRoot, nil
	}
	rel := strings.TrimPrefix(cleaned, "/")
	fullPath := filepath.Join(s.projectRoot, filepath.FromSlash(rel))
	resolved := filepath.Clean(fullPath)
	relToRoot, err := filepath.Rel(s.projectRoot, resolved)
	if err != nil {
		return "", err
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project root")
	}
	if sameOrChildPath(resolved, s.artifactRoot) {
		return "", fmt.Errorf("path is reserved for IDE artifacts")
	}
	return resolved, nil
}

func decodeContent(r *http.Request) (string, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var req fileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return "", fmt.Errorf("decode JSON body: %w", err)
		}
		return req.Content, nil
	}
	if err := r.ParseForm(); err != nil {
		return "", fmt.Errorf("parse form: %w", err)
	}
	return r.Form.Get("content"), nil
}

func decodeCode(r *http.Request) (string, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var req executionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return "", fmt.Errorf("decode JSON body: %w", err)
		}
		return req.Code, nil
	}
	if err := r.ParseForm(); err != nil {
		return "", fmt.Errorf("parse form: %w", err)
	}
	return r.Form.Get("code"), nil
}

func wantsHTML(r *http.Request) bool {
	if r.Header.Get("HX-Request") == "true" {
		return true
	}
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

func writeConsoleHTML(w http.ResponseWriter, result executionResponse) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var buf strings.Builder
	if result.Output != "" {
		buf.WriteString(`<pre class="console__entry">`)
		buf.WriteString(html.EscapeString(result.Output))
		buf.WriteString(`</pre>`)
	}
	if result.Result != "" {
		buf.WriteString(`<pre class="console__entry console__entry--result">=> `)
		buf.WriteString(html.EscapeString(result.Result))
		buf.WriteString(`</pre>`)
	}
	if result.Error != "" {
		buf.WriteString(`<pre class="console__entry console__entry--error">`)
		buf.WriteString(html.EscapeString(result.Error))
		buf.WriteString(`</pre>`)
	}
	if buf.Len() == 0 {
		buf.WriteString(`<pre class="console__entry console__entry--muted">(no output)</pre>`)
	}
	_, _ = io.WriteString(w, buf.String())
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

# picoceci IDE Plan

Version: 0.1-draft
Status: Planning phase
Target: Acme-inspired web-based IDE for picoceci development

---

## Overview

This document defines a plan for building a simple, Acme-inspired IDE for picoceci that provides:

1. **File editing** - Edit picoceci source files with syntax awareness
2. **Project outline** - Navigate and manage project files
3. **Interpreter integration** - Execute code and interact with the picoceci interpreter/VM
4. **Web-based interface** - HTMX frontend, REST API middleware, filesystem backend

## Architecture

```
┌─────────────────────────────────────────┐
│        HTMX Frontend (Browser)          │
│  - File tree navigation                 │
│  - Code editor with syntax highlighting │
│  - REPL/console output                  │
│  - Command palette                      │
└─────────────────┬───────────────────────┘
                  │ HTTP/REST
┌─────────────────▼───────────────────────┐
│      REST API Middleware (Go)           │
│  - File operations (CRUD)               │
│  - Code execution endpoints             │
│  - REPL session management              │
│  - Project structure queries            │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│      Backend Services                   │
│  ├── Filesystem (local/SD card)         │
│  ├── picoceci interpreter (AST)         │
│  └── picoceci VM (bytecode)             │
└─────────────────────────────────────────┘
```

## Design Principles (Acme-inspired)

### 1. Text as Interface
- Any text can be selected and executed as a command
- Right-click execution model (or keyboard equivalent)
- Commands are just picoceci code snippets

### 2. Minimal UI
- No heavy IDE chrome
- Simple column/window layout
- Focus on content, not toolbars

### 3. Integrated Execution
- Editor windows can execute code directly
- REPL is integrated, not separate
- Output appears in-context

### 4. File-centric
- Files are the primary unit of organization
- No heavy project files or configurations
- Directory structure is the project structure

## Components

### Phase 1: Core REST API Server

**Directory:** `ide/server/`

**Endpoints:**

```
GET    /api/files                    # List files in project directory
GET    /api/files/*path              # Read file content
POST   /api/files/*path              # Create new file
PUT    /api/files/*path              # Update file content
DELETE /api/files/*path              # Delete file

POST   /api/execute                  # Execute picoceci code (one-shot)
POST   /api/repl/create              # Create new REPL session
POST   /api/repl/:id/eval            # Evaluate code in REPL session
GET    /api/repl/:id/history         # Get REPL history
DELETE /api/repl/:id                 # Destroy REPL session

GET    /api/project/tree             # Get full project file tree
GET    /api/project/outline/:path    # Get outline of a picoceci file
                                      # (objects, methods, functions)
```

**Implementation:**
```go
// ide/server/main.go
package main

import (
    "github.com/kristofer/picoceci/pkg/eval"
    "github.com/kristofer/picoceci/pkg/bytecode"
    "github.com/kristofer/picoceci/pkg/parser"
    "github.com/kristofer/picoceci/pkg/lexer"
)

type Server struct {
    projectRoot string
    sessions    map[string]*REPLSession
}

type REPLSession struct {
    id          string
    interpreter *eval.Interpreter
    vm          *bytecode.VM
    history     []HistoryEntry
}
```

**Features:**
- Serve static HTMX frontend
- Handle file I/O with safety checks (stay within project root)
- Manage REPL sessions with UUIDs
- Support both AST interpreter and bytecode VM execution modes
- Parse files for outline extraction (object/method names)

### Phase 2: HTMX Frontend

**Directory:** `ide/static/`

**Files:**
- `index.html` - Main IDE shell
- `styles.css` - Minimal styling
- `ide.js` - Client-side helpers (minimal JavaScript)

**Layout:**

```
┌─────────────────────────────────────────────────────┐
│ picoceci IDE                    [Run] [REPL] [Help] │
├───────────────┬─────────────────────────────────────┤
│ Files         │ editor.pc                           │
│               │                                     │
│ 📁 examples/  │ 1  let counter: Int.                │
│   counter.pc  │ 2  counter := 0.                    │
│   hello.pc    │ 3                                   │
│               │ 4  let inc: Block.                  │
│ 📁 src/       │ 5  inc := [                         │
│   main.pc     │ 6      counter := counter + 1.      │
│   util.pc     │ 7      Console println: counter     │
│               │ 8          printString.             │
│ 📄 test.pc    │ 9  ].                               │
│               │ 10                                  │
│               │ 11 inc value.                       │
│               │                                     │
├───────────────┼─────────────────────────────────────┤
│ Console / REPL                                      │
│                                                     │
│ picoceci> Console println: 'Hello'.                 │
│ Hello                                               │
│ picoceci> 2 + 2.                                    │
│ 4                                                   │
│ picoceci> █                                         │
└─────────────────────────────────────────────────────┘
```

**HTMX Interactions:**

```html
<!-- File tree - click to load -->
<div id="file-tree" hx-get="/api/project/tree" hx-trigger="load">
  <!-- Tree populated here -->
</div>

<!-- Editor - load file on selection -->
<div id="editor">
  <textarea hx-get="/api/files/{path}"
            hx-trigger="fileSelected"
            hx-target="#editor-content">
  </textarea>
</div>

<!-- Save button -->
<button hx-put="/api/files/{path}"
        hx-include="#editor-content"
        hx-swap="none">
  Save
</button>

<!-- Execute current file -->
<button hx-post="/api/execute"
        hx-include="#editor-content"
        hx-target="#console">
  Run
</button>

<!-- REPL interaction -->
<form hx-post="/api/repl/{session-id}/eval"
      hx-target="#console"
      hx-swap="beforeend">
  <input name="code" />
</form>
```

**Features:**
- Dynamic file tree navigation (no page reloads)
- Code editor (textarea initially, can upgrade to CodeMirror later)
- Syntax highlighting (via CSS or lightweight library)
- REPL console with command history
- Keyboard shortcuts (Ctrl+S save, Ctrl+Enter run)

### Phase 3: Advanced Features

**To be added after MVP:**

1. **Syntax Highlighting**
   - Implement picoceci lexer-based highlighting
   - Color keywords, strings, comments, numbers

2. **Code Intelligence**
   - Autocomplete for object methods
   - Show method signatures on hover
   - Jump to definition (within file)

3. **Multiple Editor Windows**
   - Acme-style column layout
   - Split editor into multiple panes
   - Each pane can show different file

4. **Execute Selected Text**
   - Select code snippet and execute
   - Show result inline or in console
   - True Acme-style interaction

5. **File Watching**
   - Auto-reload files when changed externally
   - Show indicator when file is modified
   - WebSocket for live updates

6. **Debugger Integration**
   - Breakpoints in editor
   - Step through execution
   - Inspect variables

7. **SD Card Integration**
   - Mount SD card filesystems
   - Browse/edit files on SD card
   - Test code that uses SD card

8. **Device Connection**
   - Connect to ESP32-S3 over WiFi
   - Execute code on device
   - Show device console output

## File Structure

```
ide/
├── IDE_PLAN.md              # This file
├── README.md                # Quick start guide
├── server/
│   ├── main.go              # HTTP server entrypoint
│   ├── api.go               # REST API handlers
│   ├── files.go             # File operations
│   ├── repl.go              # REPL session management
│   ├── project.go           # Project structure queries
│   └── outline.go           # Code outline extraction
├── static/
│   ├── index.html           # Main IDE page
│   ├── styles.css           # IDE styles
│   └── ide.js               # Minimal client logic
└── testdata/
    └── sample-project/      # Sample picoceci project for testing
        ├── hello.pc
        └── counter.pc
```

## Implementation Roadmap

### Milestone 1: Basic Server + File Operations (1-2 days)

- [ ] Create `ide/` directory structure
- [ ] Implement HTTP server with file CRUD endpoints
- [ ] Add project root safety checks
- [ ] Write tests for file operations
- [ ] Create sample project in testdata

**Acceptance:**
- Server starts on `localhost:8080`
- Can list, read, write, delete files via curl
- Files outside project root are rejected

### Milestone 2: REPL Integration (1-2 days)

- [ ] Implement REPL session management
- [ ] Add execute endpoint for one-shot code
- [ ] Add REPL eval endpoint with session
- [ ] Store REPL history per session
- [ ] Add session cleanup/timeout

**Acceptance:**
- Can create REPL session and get ID
- Can evaluate picoceci code in session
- Session maintains state between evals
- Can retrieve history

### Milestone 3: Basic Frontend (2-3 days)

- [ ] Create HTML layout (file tree + editor + console)
- [ ] Implement HTMX-based file tree
- [ ] Add file selection and loading
- [ ] Implement editor with save
- [ ] Add REPL console interface
- [ ] Wire up Run button

**Acceptance:**
- Can browse files in web UI
- Can open, edit, and save files
- Can execute file and see output
- Can use REPL interactively

### Milestone 4: Project Outline (1 day)

- [ ] Parse picoceci files for structure
- [ ] Extract object definitions
- [ ] Extract method definitions
- [ ] Build outline tree
- [ ] Add outline panel to UI

**Acceptance:**
- Outline shows objects and methods
- Clicking outline item jumps to line
- Updates when file changes

### Milestone 5: Polish & Documentation (1 day)

- [ ] Add syntax highlighting
- [ ] Improve styling (minimal but clean)
- [ ] Add keyboard shortcuts
- [ ] Write user documentation
- [ ] Add error handling and user feedback

**Acceptance:**
- IDE looks professional
- Keyboard shortcuts work
- Errors show helpful messages
- README explains how to use

**Total estimated time: 6-9 days for MVP**

## Usage

### Starting the IDE

```bash
cd ide/server
go run main.go --project=/path/to/picoceci/project
```

Or from project root:

```bash
make ide-serve
```

Then open browser to `http://localhost:8080`

### Development Workflow

1. Browse files in left panel
2. Click file to open in editor
3. Edit code
4. Save with Ctrl+S or Save button
5. Run with Ctrl+Enter or Run button
6. See output in console
7. Use REPL for interactive experimentation

### Connecting to ESP32-S3

When picoceci device runtime is complete:

```bash
# On device, start WiFi REPL server
Wifi connectSSID: 'MyNetwork' password: 'password'.
PicoceciREPL serve: 2323.

# In IDE, connect to device
ide/server/main.go --device=192.168.1.100:2323
```

IDE will execute code on device instead of locally.

## Technical Considerations

### Security

- **Path traversal prevention:** Validate all file paths
- **Execution sandboxing:** REPL sessions isolated
- **No authentication in MVP:** Add auth before public deployment
- **CORS configuration:** Restrict to localhost initially

### Performance

- **File caching:** Cache file tree, invalidate on write
- **REPL timeout:** Clean up stale sessions after 1 hour
- **Streaming output:** Use SSE or WebSocket for long-running code
- **Editor size limit:** Warn on files > 1MB

### Compatibility

- **Browser support:** Modern browsers (Chrome, Firefox, Safari, Edge)
- **Mobile:** Basic support, desktop-optimized
- **Dark mode:** Respect system preference

### Testing

- **Unit tests:** All API handlers
- **Integration tests:** Full execution flows
- **E2E tests:** Basic UI workflows (optional)
- **Manual testing:** Test with real picoceci projects

## Related Documents

- [picoceci Language Spec](../LANGUAGE_SPEC.md) - Language reference
- [Implementation Plan](../docs/IMPLEMENTATION_PLAN.md) - Runtime implementation
- [Standard Library](../docs/stdlib.md) - Built-in objects and methods

## Future Directions

### Collaboration Features
- Multiple cursors (collaborative editing)
- Share REPL sessions
- Code review annotations

### Advanced Editor
- Vim/Emacs keybindings
- Code folding
- Minimap

### Visual Programming
- Block-based visual editor for beginners
- Dataflow visualization
- Object graph visualization

### Mobile App
- Native mobile IDE
- Touch-optimized interface
- Bluetooth connection to ESP32

### Cloud Storage
- Save projects to cloud
- Sync across devices
- Project templates/examples

---

## References

- **Acme:** http://acme.cat-v.org/ - Original Plan 9 editor
- **HTMX:** https://htmx.org/ - Hypermedia framework
- **CodeMirror:** https://codemirror.net/ - Editor component (optional upgrade)

## License

Same as picoceci - MIT License

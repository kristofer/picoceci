# picoceci IDE

A simple, Acme-inspired web-based IDE for picoceci development.

## Status

🚧 **Planning phase** - See [IDE_PLAN.md](IDE_PLAN.md) for the complete implementation plan.

## What is this?

The picoceci IDE provides:

- **File editor** - Edit picoceci source files with syntax awareness
- **Project outline** - Navigate and manage your picoceci projects
- **Integrated REPL** - Execute code and interact with the interpreter
- **Simple interface** - Minimal, Acme-inspired design focused on productivity

## Architecture

- **Frontend:** HTMX-based web interface (no heavy JavaScript frameworks)
- **Middleware:** Go REST API server
- **Backend:** Local filesystem + picoceci interpreter/VM

## Quick Start

*(To be implemented)*

```bash
# Start the IDE server
cd ide/server
go run main.go --project=/path/to/your/project

# Open in browser
open http://localhost:8080
```

## Design Philosophy

Inspired by Acme from Plan 9:

1. **Text as interface** - Any text can be executed as a command
2. **Minimal UI** - No heavy IDE chrome, focus on content
3. **Integrated execution** - Editor and REPL work together seamlessly
4. **File-centric** - Directory structure is your project structure

## Implementation Status

See [IDE_PLAN.md](IDE_PLAN.md) for:
- Complete architecture documentation
- API endpoint specifications
- Frontend design and layouts
- Implementation milestones and roadmap
- Technical considerations

## Directory Structure

```
ide/
├── IDE_PLAN.md          # Complete implementation plan
├── README.md            # This file
├── server/              # REST API server (to be implemented)
├── static/              # HTMX frontend (to be implemented)
└── testdata/            # Sample projects for testing (to be implemented)
```

## Contributing

This IDE is being built in phases. See the Implementation Roadmap in [IDE_PLAN.md](IDE_PLAN.md) for current status and next milestones.

## License

MIT - Same as picoceci

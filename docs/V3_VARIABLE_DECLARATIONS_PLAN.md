# picoceci v3 — Variable Declaration Syntax Recommendation and Migration Plan

Version: 0.1-draft  
Status: **Planning only — no implementation has been started**  
Author: picoceci contributors

---

## 1. Problem statement

picoceci v2 uses typed declarations inside pipe delimiters:

```picoceci
| x: Int obiwan: Object |
x := 1.
```

This is explicit and type-safe, but awkward to read and edit in longer methods. The v3 goal is to keep v2’s type guarantees while improving declaration ergonomics.

---

## 2. Recommendation

### 2.1 Adopt `let` declarations in v3

Introduce statement-style declarations:

```picoceci
let x: Int.
let obiwan: Object.
```

Also support an explicit declaration-with-inference form:

```picoceci
let retries := 3.          "declares retries as Int"
let title := 'picoceci'.   "declares title as String"
```

### 2.2 Keep `:=` as assignment for already-declared variables

`:=` should **not** become implicit declaration on first use (`x := ...` when `x` is undeclared). That Go-like behavior increases typo risk and weakens declaration-time intent.

### 2.3 Migration compatibility

During migration, parse both forms:
- v2: `| name: Type ... |`
- v3: `let name: Type.` and `let name := expr.`

After repository-wide rewrites are complete, remove v2 pipe declarations in v3-final.

---

## 3. Why this recommendation

### Benefits

- **Readability:** declarations become normal statements and align with method flow.
- **Parser simplification (eventual):** eliminates the special `| ... |` declaration delimiter path once migration is complete.
- **Type safety preserved:** typed declarations remain explicit; inferred declarations still lock type at declaration time.

### Downsides and complexity trade-offs

- **Parser transition complexity (temporary):** supporting both `varDecl` syntaxes increases grammar branches and test matrix until v2 syntax is removed.
- **Declaration ledger complexity:** interpreter and VM must distinguish `let x := expr` (declare+type-bind) from `x := expr` (assign existing), requiring strict symbol-table checks in both engines.
- **Migration churn:** every embedded picoceci snippet in tests, examples, and docs must be rewritten together to avoid mixed-style confusion.

---

## 4. Multi-phase implementation plan

### Phase 1 — Grammar and parser introduction (dual syntax)

Scope:
- Add `let` token and `let` declaration grammar.
- Add AST node(s) for `let` declarations:
  - typed form (`let x: Int.`)
  - inferred form (`let x := expr.`)
- Keep existing v2 `| ... |` parsing for compatibility.

Acceptance:
- Parser tests cover both syntaxes.
- Existing v2 tests still pass unchanged.

### Phase 2 — Interpreter/VM declaration semantics

Scope:
- Implement `let` evaluation in tree-walking interpreter and bytecode VM.
- For inferred declarations, compute declared type from RHS value kind at declaration time.
- Enforce rule:
  - `let name ...` declares in current scope only
  - `name := expr` requires prior declaration in visible scope
- Preserve existing type checks on assignment.

Acceptance:
- New eval/vm tests for:
  - declare-then-assign success
  - inferred declaration type lock
  - type mismatch errors
  - undeclared assignment error

### Phase 3 — Repository-wide source rewrites (tests/examples/docs)

Scope:
- Rewrite picoceci source in:
  - `examples/**/*.pc`
  - `testdata/**/*.pc` and `testdata/**/*.ceci`
  - embedded picoceci snippets in `pkg/**/_test.go`
  - markdown docs containing picoceci code blocks
- Prefer `let x: Type.` by default.
- Use `let x := expr.` only where it materially improves clarity.

Acceptance:
- No remaining v2 `| ... |` declarations in repository-authored picoceci samples/tests.
- All affected tests updated and passing.

### Phase 4 — Language spec and grammar documents update

Scope:
- Update `LANGUAGE_SPEC.md`:
  - lexical tokens (`let`)
  - declaration and assignment sections
  - grammar summary (`varDecl` replacement)
  - examples rewritten to `let` style
- Update `docs/grammar.ebnf` with final v3 declaration rules.
- Add migration notes and compatibility window policy.

Acceptance:
- Spec examples and formal grammar are consistent with parser behavior.
- No contradictory v2 declaration guidance remains in primary docs.

### Phase 5 — Compatibility removal and stabilization

Scope:
- Remove parser/runtime support for v2 pipe declarations.
- Remove transitional tests and keep only v3 syntax.
- Final pass on error messages for declaration/assignment failures.

Acceptance:
- `go build ./...`, `go vet ./...`, `go test ./...` pass.
- v3 syntax is the single documented and implemented declaration form.

---

## 5. Open decisions before implementation

1. Should inferred `let x := expr` infer user-defined object/interface names, or collapse to a broad type (`Any` or a dedicated root object type)?
2. Should shadowing with `let` be allowed by default in inner scopes, or require an explicit keyword later?
3. Should a temporary parser warning be added when v2 `| ... |` syntax is used during the compatibility window?

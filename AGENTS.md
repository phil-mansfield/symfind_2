# Symfind-2 contributor instructions

## Authority and repository boundaries

- Assume the repository is read-only unless the user explicitly asks for a
  change. Inspection, analysis, and reporting do not authorize edits.
- Do not modify files outside this repository unless the user explicitly asks.
- Repository scripts must use portable shell tools such as `find` and `grep`;
  they must not depend on `rg`.
- AI edits of human-written code require higher scrutiny than AI edits of
  AI-written code.
- AI should not write core science logic and should generally be focused on
  clearly-specified refactoring, interacting with complex interfaces,
  automation, and generalising human-written example code.

## Project layout and validation

- The root module contains the `symfind_2` command and requires Go 1.26.1.
- `pipeline` contains analysis stages, `scripts` contains analysis and
  repository-management routines, and `symlib` contains shared utilities.
- Python wrappers live in `python`.
- Run the ordinary Go test suite from the repository root with:

      go test ./...

- The local Chimera dependency is selected by the `replace` directive in
  `go.mod`. Do not remove or replace that development configuration unless the
  user asks.

## Go style

- All AI-written code must be read by a human, so all AI-written code must be
  designed to be easily reviewed.
- Do not compress multiple statements into a one-line function with
  semicolons. A one-line function containing one statement, such as a simple
  return-only callback, is acceptable.

## Test design

- Name expected and actual values `want` and `got`, respectively.
- Strongly prefer table-driven tests using slices of anonymous structs. Put
  each case on its own line. Use named fields when an unnamed case would be
  difficult to scan.
- Store the operation, inputs, and expected results in the table. Invoke the
  operation inside the test loop; never store an eagerly evaluated `got` value
  in the table.
- Prefer separate tables to clever shared loops when functions have
  meaningfully different types or signatures, but 
- Every path leading to a distinct `t.Error`, `t.Errorf`, `t.Fatal`, or
  `t.Fatalf` call must have a short comment on the largest block used only by
  that failure path.
- Failure messages must identify the behavior under test and, when tractable,
  include the inputs, `got`, and `want` values. Avoid generic messages such as
  `t.Fatal(got)`.
- Default to messages of this form when it remains readable:

      Decreasing unsigned ints - Expected Linspace(<inputs>) = <want>, but got <got>

  Use clearer phrasing when that form is clumsy or the inputs or outputs are
  too large to print usefully.

## AI provenance and review

- Agents may modify `TODO.md` without adding AI attribution or provenance
  comments.
- Treat every Bash script as entirely AI-written, regardless of its actual
  origin. Do not add provenance markers to shell scripts.
- Wrap each coherent block of code added by AI in `AI-GENERATED-BEGIN` and
  `AI-GENERATED-END` comments. The review unit is the added block, not the
  containing function: a block may be a declaration, statement sequence,
  function, method, test, helper, or C fragment in a cgo preamble.
- Use a single-line begin marker with this schema:

      // AI-GENERATED-BEGIN: v=1; task="..."; reviewer="Phil Mansfield"; tool="Codex"; review="pending"

  Use the comment syntax of the containing language. Place the marker directly
  before the reviewed block and the end marker directly after it. When the
  block begins with a Go doc comment, put a blank line between the begin marker
  and doc comment so the marker does not appear in godoc.
- Set `review="pending"` for new AI-authored blocks. Only Phil may manually
  change it to `review="approved"`; agents must never approve their own work.
- After adding a marked block, report its exact `task` value and ask Phil to
  confirm it.
- Keep markers through ordinary human edits and remove them only after the
  marked block has been completely rewritten. For a substantial AI change to
  an existing human-authored block, ask before marking the entire resulting
  block. Provenance markers are unnecessary for trivial edits.
- When provenance markers change, run:

      ./scripts/repo/check_ai_reviews.sh

  The check lists blocks without `review="approved"` and fails until Phil has
  approved them. A pending failure immediately after AI adds code is expected.

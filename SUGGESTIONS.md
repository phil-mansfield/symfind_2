# Approaches for annotate-tree follow-ups

1. Make input-file discovery obey `TreeFileN`.
   First define the input naming convention precisely: whether `TreeFileName`
   is an exact single-file path or a numbered-file prefix. Generate or filter
   only the expected names, sort them in the intended numeric order, and
   return an error if their count differs from `TreeFileN`. Treat zero matches
   as an error before opening or retaining an output file.

2. Define and verify the `Header/NBlocks` convention for multi-file trees.
   Document whether `NBlocks` means a count or a maximum zero-based index.
   Add a two-file fixture and assert the attribute and the presence of
   `Header/0`, `Header/1`, `Data/0`, and `Data/1`. If it is a count, write
   `args.Index + 1` after each block or once after the complete write.

3. Implement `Tree.Read` and add a round-trip test.
   Start with the smallest useful reader: load a specified block's Names,
   Types, and datasets into `TreeData`. A table-driven test can write a small
   mixed integer/float tree to a temporary file, read it back, and compare
   every column and header field. This also turns the HDF5 layout into an
   executable contract.

4. Make tree-config column indices independent of blank lines.
   Keep a separate `columnIndex` counter that increments only for accepted
   column declarations, rather than deriving the index from `i` in the raw
   lines slice. Tests should cover the ordinary layout, leading blank lines,
   and blank lines between declarations.

5. Support input trees with no float columns.
   In `InputTree.Read`, call `ReadFloat64s` only when `len(fCols) != 0`; use an
   empty result otherwise. Apply the analogous guard to integer columns if the
   dataset is ever generalized so that no integer column is mandatory. Test a
   depth-first-ID-only input tree.

6. Close a newly created `Config.h5` in `Pipeline.Init`.
   Close the file after all initialization operations have succeeded, and
   return a close error if the HDF5 wrapper exposes one. Avoid a simple defer
   if error paths should also preserve and report the first operation error;
   a small helper can make that behavior explicit.

7. Resolve paths in a configuration file relative to that file.
   Compute the directory containing `--config`, then resolve relative
   `TreeFileName` and `TreeConfigFileName` values against it while preserving
   absolute paths. Store the resolved paths in `ConfigData`, and test by
   running from a different working directory.

8. Record configuration failures as failed pipeline versions.
   Replace the `log.Fatal` branch with the same error-reporting/status-update
   path used for `stage.Run` errors. Keep process termination in `main`, after
   the status write, because `log.Fatal` calls `os.Exit` and skips defers.

9. Reconcile the documented and accepted spelling of check mode.
   Pick lowercase `check` to match the usage text, or deliberately accept both
   spellings during a transition. Add focused CLI parsing tests for normal
   mode, check mode, and invalid three-argument invocations.

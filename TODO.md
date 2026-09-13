# Annotate-tree follow-ups

1. Make input-file discovery obey `TreeFileN`.
   `InputTree.Configure` currently accepts every path matching
   `TreeFileName + "*"` without checking the result count. This can process
   unintended files, silently accept too few files, or succeed with no input
   files and leave a stale `Tree.h5` behind.

2. Define and verify the `Header/NBlocks` convention for multi-file trees.
   The writer currently stores the zero-based final block index. The one-block
   test file therefore contains `NBlocks = 0`; if this attribute is a count,
   it should instead be one.

3. Implement `Tree.Read` and add a round-trip test.
   The writer has no in-project reader, so the current manual HDF5 inspection
   verifies only that the file looks plausible rather than that a consumer can
   recover its data and metadata.

4. Make tree-config column indices independent of blank lines.
   `parseInputTreeColumns` uses the physical line number minus one for the
   input column index. Extra blank lines after `---` or between declarations
   shift later columns to the wrong positions.

5. Support input trees with no float columns.
   `InputTree.Read` unconditionally calls `ReadFloat64s`, which can panic in
   the text reader when the requested float-column list is empty.

6. Close a newly created `Config.h5` in `Pipeline.Init`.
   The initialization branch creates the groups and attributes but leaves the
   HDF5 handle open; this only occurs for a fresh pipeline directory.

7. Resolve paths in a configuration file relative to that file.
   `TreeFileName` and `TreeConfigFileName` currently depend on the shell's
   current directory, making a config file work differently when the command
   is launched elsewhere.

8. Record configuration failures as failed pipeline versions.
   `log.Fatal` on a dataset-configuration error exits before the run status is
   changed from "started" to "caught error".

9. Reconcile the documented and accepted spelling of check mode.
   Help says `check`; the parser currently accepts only `Check`.

## Annotate-tree validation and error propagation

10. Validate the raw columns before calculating tracks.
    Require the ID, descendant ID, UPID, depth-first ID, snapshot, and mass
    columns to be present, non-empty, and equal in length. Check that snapshot
    numbers are non-negative, decide which mass values are admissible, and
    check that the depth-first ordering satisfies the branch-contiguity
    assumptions used by `Edges`.

11. Validate IDs and tree references before using them as slice indices.
    Require halo IDs to be non-negative and unique. Every UPID other than -1
    must resolve to a halo in the tree at the same snapshot, and every
    descendant ID other than -1 must resolve to a valid descendant.

12. Replace unchecked lookup-table indexing with an explicit failure result.
    A missing or out-of-range ID currently becomes an index panic. Make lookup
    return an `ok` value or an error so callers can identify the bad field,
    source halo, referenced ID, and snapshot. Avoid allocating storage
    proportional to the largest ID, or reject an unexpectedly sparse ID range
    before allocating it.

13. Propagate track-calculation errors through the annotate-tree stage.
    Give `CalcTracks` and any fallible helpers error-returning interfaces, and
    return those errors through `annotateTree` and `AnnotateTree.Run` with
    enough context to identify the input tree or block.

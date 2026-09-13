# Symfind-2

The particle-tracking halo finder Symfind-2.

## Repository layout

- `pipeline/` contains stages of the analysis pipeline.
- `scripts/` contains scientific analysis and repository-management routines.
- `symlib/` contains utilities shared by the Go packages.
- `python/` contains the installable Python wrapper package.
- `symfind_2.go` is the command-line entry point.

## Development

Build the command and test all Go packages from the repository root:

```sh
go build ./...
go test ./...
```

Install the Python wrappers in editable mode:

```sh
python -m pip install -e ./python
```

Check that all marked AI-generated code blocks have been reviewed:

```sh
./scripts/repo/check_ai_reviews.sh
```

The Go module requires Go 1.26.1 or newer. The Python package requires Python 3.9
or newer.

## License

Symfind-2 is licensed under the [GNU General Public License v3.0](LICENSE).

package pipeline

import (
	"unicode"
)

// Stage is a single stage in the pipeline.
type Stage interface {
	// Init initialises the contents of the Stage and allows the other Stage
	// methods to resolve.
	Init(pipe *Pipeline)

	// Name returns of the CamelCase stage name used for internal referencing.
	Name() string

	// CLIName gives the name used by the CLI, which has different conventions
	// from the code. (CamelCaseXYZ -> camel_case_x_y_z).
	CLIName() string

	// Run executes the pipeline stage. Command-line arguments are passed as
	// key-value pairs in args, and the flag check indicates whether the stage
	// should be run in "check" mode, which doesn't perform any expensive
	// compute but tries to activate as many error-checking paths as possible.
	// Some Stage implementations have arrays of underlying state and
	// indicates which element of that underlyingstage should be run.
	Run(id uint64, check bool, args map[string]string) error

	// Dependencies returns the names of datasets that the pipeline stage
	// depends on.
	Dependencies() []string

	// Help returns a help string for the stage. These should follow the format
	// of ExampleStageHelpString.
	Help() string
}

// BaseStage is a base-type that all Stage-implementing structs should embed.
type BaseStage struct {
	StageName         string
	StageDependencies []string
}

func (stage *BaseStage) Name() string {
	return stage.StageName
}

func (stage *BaseStage) CLIName() string {
	name := stage.Name()
	rCC := []rune(name) // CamelCase runes
	rUC := []rune{}     // under_case runes
	if len(rCC) > 0 {
		rUC = append(rUC, unicode.ToLower(rCC[0]))
	}

	for i := 1; i < len(rCC); i++ {
		if unicode.IsUpper(rCC[i]) {
			rUC = append(rUC, []rune{'_', unicode.ToLower(rCC[i])}...)
		} else {
			rUC = append(rUC, rCC[i])
		}
	}

	return string(rUC)
}

func (stage *BaseStage) Dependencies() []string {
	return stage.StageDependencies
}

var ExampleStageHelpString string = `Stage my_example

    my_example computes something and this is an efficient explanation of it
    which keeps lines under 80 characters. Lines are preceeded by four spaces.

    my_example Arguments:
        --arg1=<itentifier>
            Explanation for arg1.
        --arg2=<identifier>
            Explanation for arg2.`

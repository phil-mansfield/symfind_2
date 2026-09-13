// Command symfind_2 runs the Symfind-2 analysis pipeline.
package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/phil-mansfield/chimera/hdf5"
	"github.com/phil-mansfield/chimera/vec"
	"github.com/phil-mansfield/symfind_2/pipeline"
)

func main() {
	pipe := pipeline.Symfind2

	log.SetOutput(os.Stderr)
	hdf5.SetPanicOnError(true)

	// Parse input from the CLI
	input, err := ParseCLI()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	} else if input.Stage == "help" || input.Stage == "" {
		pipe.ErrorInit() // This is messy, but needed to the error message.
		fmt.Fprintln(os.Stderr, HelpText(pipe))
		os.Exit(1)
	}

	// Initialise pipeline.
	if err = pipe.Init(input.Dir); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	// TODO: doesn't completely work: for now everything needs to be done with
	// one CLI call.
	//
	// Register the new ConfigData
	version, err := pipe.RegisterVersion(VersionText(input))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Internal error - could not initialise run: %s\n", err.Error())
		os.Exit(1)
	}

	input.Args["version_id"] = fmt.Sprintf("%d", version)
	input.Args["base_dir"] = input.Dir

	checkModeStr := ""
	if input.Check {
		checkModeStr = " in check mode"
	}
	log.Printf("Running version %d of pipeline %s%s.",
		version, input.Dir, checkModeStr)

	// Figure out which stages to run and their order.
	stages, err := pipe.StagesToRun(input.Stage)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		pipe.SetVersionRunStatus(version, pipeline.RunStatusCaughtError)
		os.Exit(1)
	}

	stageNames := make([]string, len(stages))
	for i := range stageNames {
		stageNames[i] = stages[i].CLIName()
	}

	log.Printf("The following stages will be run: [%s]",
		strings.Join(stageNames, ", "))

	// Now we enter the zone where arbitrary panics can occur.
	defer CatchPanicAndFail(pipe, version)
	for _, stage := range stages {
		log.Printf("Running stage %s.", stage.CLIName())

		// Configure all the datasets that the current stage depends on, if
		// they haven't been configured yet.
		for _, dep := range stage.Dependencies() {
			err := pipe.ConfigureDataset(dep)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		// Run the pipeline stage.
		if err := stage.Run(version, input.Check, input.Args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			pipe.SetVersionRunStatus(version, pipeline.RunStatusCaughtError)
			os.Exit(1)
		}

		log.Printf("Finishing stage %s.", stage.CLIName())
	}

	pipe.SetVersionRunStatus(version, pipeline.RunStatusFinished)
}

func HelpText(pipe *pipeline.Pipeline) string {

	blocks := []string{`Usage:
    symfind_2 [check] <stage> <target-directory> [--arg1=val1] [--arg2=val2]

    symfind_2 will run the Symfind-2 subhalo finder until the target stage
    is completed and will store the output of that stage to disk. Earlier
    stages of the pipeline which have not been run or have not had data cached
    on disk are (re)-run. Each stage has different set of accepted arguments.

    Running symfind_2 with the argument 'check' prior to other arugments will
    execute non-compute-intensive the error-checking routines of that pipeline
    stage without performing computation.`}

	names := make([]string, len(pipe.Stages))
	for i, stage := range pipe.Stages {
		names[i] = stage.CLIName()
	}

	blocks = append(blocks, "Available stages: "+strings.Join(names, ", "))

	blocks = append(blocks, "Run 'symfind_2 help --<stage>' for information on a specific stage.")

	return strings.Join(blocks, "\n\n")
}

func StageNameHelpText(pipe *pipeline.Pipeline, stageName string) string {
	names := make([]string, len(pipe.Stages))
	for i, stage := range pipe.Stages {
		names[i] = stage.CLIName()
	}

	return fmt.Sprintf(`Unknown stage %s, all known stages are:
    %s

Run 'symfind_2 help' for usage instructions.`,
		stageName, strings.Join(names, ", "))
}

// CatchPanicAndFail is light wrapper around recover() which prints out the
// stack trace, too.
func CatchPanicAndFail(pipe *pipeline.Pipeline, version uint64) {
	if panicValue := recover(); panicValue != nil {
		fmt.Fprintf(os.Stderr, `Symfind-2 failed with an internal or uncaught error:
%v
%s
`, panicValue, debug.Stack())

		pipe.SetVersionRunStatus(version, pipeline.RunStatusCaughtError)
		os.Exit(1)
	}
}

// CLIInput contains the CLI input.
type CLIInput struct {
	Dir   string
	Check bool
	Stage string
	Args  map[string]string
}

// Parse CLI parses the command line input.
func ParseCLI() (*CLIInput, error) {
	// Determine which arguments are flags.
	isFlag := make([]bool, len(os.Args)-1)
	for i := range isFlag {
		before, after, found := strings.Cut(os.Args[i+1], "--")
		isFlag[i] = found && before == "" && after != ""
	}

	// This lets you put the flags wherever you want relative to the non-flags,
	// which is fine by me.
	flags := vec.Mask(os.Args[1:], isFlag)
	nonFlags := vec.Mask(os.Args[1:], vec.Not(isFlag))

	input := &CLIInput{Args: map[string]string{}}

	// Parse flag arguments.
	for _, flag := range flags {
		_, kvPair, _ := strings.Cut(flag, "--")
		key, value, _ := strings.Cut(kvPair, "=")
		// If nothing is set, value is "".
		input.Args[key] = value
	}

	// Parse non-flag arguments.
	switch len(nonFlags) {
	case 0:
		return nil, fmt.Errorf(`Stage and taget directory not passed to symfind_2. Run 'symfind_2 help' for
usage instructions.`)
	case 1:
		switch nonFlags[0] {
		case "help":
			input.Stage = "help"
		case "check":
			return nil, fmt.Errorf(`Stage and target directory not passed to symfind_2. Run 'symfind_2 help' for
usage instructions.`)
		default:
			return nil, fmt.Errorf(`Target directory not passed to symfind_2. Run 'symfind_2 help' for usage
instructions.`)
		}
	case 2:
		input.Stage, input.Dir = nonFlags[0], nonFlags[1]
		if input.Stage == "check" {
			return nil, fmt.Errorf(`Target directory not passed to symfind_2. Run 'symfind_2 help' for usage
instructions.`)
		}
	case 3:
		var check string
		check, input.Stage, input.Dir = nonFlags[0], nonFlags[1], nonFlags[2]
		input.Check = true

		if check != "Check" {
			return nil, fmt.Errorf(`Three non-flag arguments passed to symfind_2, but the first flag is not 'check'.
Run 'symfind_2 help' for usage instructions.`)
		}
	default:
		return nil, fmt.Errorf(`More than three non-flag arguments passed to symfind_2. Run
'symfind_2 help' usage instructions.`)
	}

	return input, nil
}

func VersionText(input *CLIInput) string {
	info, ok := debug.ReadBuildInfo()
	var buildStr string
	if ok {
		buildStr = info.String()
	} else {
		buildStr = ""
	}

	n := len(input.Args)
	argLines, keys := make([]string, n), []string{}
	for key, _ := range input.Args {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for i, key := range keys {
		argLines[i] = fmt.Sprintf("  --%s=%s", key, input.Args[key])
	}

	modeStr := fmt.Sprintf(`Command-line arguments:
  check=%v
  stage=%s
  dir=%s
`, input.Check, input.Stage, input.Dir)

	argsStr := fmt.Sprintf("Command-line flags:\n%s",
		strings.Join(argLines, "\n"))

	timeStamp := time.Now().UTC().Format(time.UnixDate)
	timeStr := fmt.Sprintf("Start time: %s", timeStamp)

	return fmt.Sprintf("%s\n%s\n\n%s\n\n%s\n",
		buildStr, modeStr, argsStr, timeStr)
}

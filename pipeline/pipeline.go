package pipeline

import (
	"fmt"
	"os"
	"path"
	"slices"

	"github.com/phil-mansfield/chimera/hdf5"
	"github.com/phil-mansfield/chimera/vec"
)

// Pipeline is an analysis pipeline.
type Pipeline struct {
	Config   *ConfigData // Configuration data, only set after "Config" stage
	Dir      string      // Home directory of the pipeline
	Stages   []Stage     // Stages of the pipeline
	Datasets []Dataset   // Datasets tracked

	// Internal state tracking which datasets have been configured
	configuredDatasets map[string]bool
}

func (pipe *Pipeline) Init(dir string) error {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf("%s is not an existing directory.", dir)
	}

	pipe.Dir = dir
	pipe.configuredDatasets = map[string]bool{}

	fname := path.Join(dir, "Config.h5")
	if _, err := os.Stat(fname); err != nil {
		file, err := hdf5.Open(fname, "w+")
		if err != nil {
			return fmt.Errorf("Unable to create %s: %s.", fname, err.Error())
		}

		zero := uint64(0)
		// Initialise the versions. Everything else is done in the
		if err := file.Group("Versions"); err != nil {
			return err
		} else if err = file.Attr("Versions", "N", &zero); err != nil {
			return err
		} else if err = file.Group("Config"); err != nil {
			return err
		} else if err = file.Group("Datasets"); err != nil {
			return err
		}
	}

	// Note that none of the stage/dset names/etc are garuanteed to be valid
	// until Init is called

	for _, stage := range pipe.Stages {
		stage.Init(pipe)
	}

	for _, dataset := range pipe.Datasets {
		dataset.Init(pipe)
	}

	return nil
}

// ErrorInit initialises enough of Pipeline to be useful for generating error
// messages.
func (pipe *Pipeline) ErrorInit() {
	for _, stage := range pipe.Stages {
		stage.Init(pipe)
	}

	for _, dataset := range pipe.Datasets {
		dataset.Init(pipe)
	}
}

func (pipe *Pipeline) GetDataset(name string) (Dataset, bool) {
	for _, dset := range pipe.Datasets {
		if dset.Name() == name {
			return dset, true
		}
	}
	return nil, false
}

func (pipe *Pipeline) GetStage(name string) (Stage, bool) {
	for _, stage := range pipe.Stages {
		if stage.Name() == name || stage.CLIName() == name {
			return stage, true
		}
	}
	return nil, false
}

func (pipe *Pipeline) ConfigureDataset(name string) error {
	if pipe.configuredDatasets[name] {
		return nil
	}

	stage, ok := pipe.GetDataset(name)
	if !ok {
		return fmt.Errorf("Internal error: unrecognized dataset name, %s.", name)
	}

	// pipe.Config will only be non-nil after the Config stage has been
	// run. After that point, it will be the initialised.
	if err := stage.Configure(pipe.Config); err != nil {
		return err
	}
	pipe.configuredDatasets[name] = true
	return nil
}

const (
	RunStatusStarted = iota
	RunStatusCaughtError
	RunStatusFinished
)

// RegisterVersion registers the current version in Config.h and returns the ID.
// Text is the text of the version string, as determined by the CLI.
func (pipe *Pipeline) RegisterVersion(text string) (uint64, error) {
	// Unfortuantely, error-handling has completely undermined the elegance of
	// the hdf5 interface. What else is new :P ?

	fname := path.Join(pipe.Dir, "Config.h5")
	file, err := hdf5.Open(fname, "w")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	n := uint64(0)
	if err = file.GetAttr("Versions", "N", &n); err != nil {
		return 0, err
	}
	n++
	if err = file.Attr("Versions", "N", &n); err != nil {
		return 0, err
	}

	status := uint64(RunStatusStarted)

	grp := fmt.Sprintf("Versions/%d", n-1)
	if err = file.Group(grp); err != nil {
		return 0, err
	} else if err = file.Attr(grp, "Info", &text); err != nil {
		return 0, err
	} else if err = file.Attr(grp, "RunStatus", &status); err != nil {
		return 0, err
	}

	return uint64(n - 1), nil
}

func (pipe *Pipeline) SetVersionRunStatus(version, status uint64) {
	fname := path.Join(pipe.Dir, "Config.h5")
	file, err := hdf5.Open(fname, "w")
	if err != nil { // Alreayd checked
		panic(err.Error())
	}
	defer file.Close()

	err = file.Attr(fmt.Sprintf("Versions/%d", version), "RunStatus", &status)
	if err != nil { // Already checked.
		panic(err.Error())
	}
}

func (pipe *Pipeline) StagesToRun(name string) ([]Stage, error) {
	stage, ok := pipe.GetStage(name)
	if !ok {
		return nil, fmt.Errorf("unknown stage %q", name)
	}

	stages := []Stage{}
	deps := stage.Dependencies()

	for _, dsetName := range deps {
		dset, ok := pipe.GetDataset(dsetName)
		if !ok {
			return nil, fmt.Errorf(
				"stage %q depends on unregistered dataset %q",
				stage.Name(), dsetName,
			)
		}

		// TODO: put limit in here to stop on version conditions

		stageName := dset.Dependency()
		if stageName != "" {
			sstages, err := pipe.StagesToRun(stageName)
			if err != nil {
				return nil, fmt.Errorf(
					"stage %q depends on dataset %q: %w",
					stage.Name(), dsetName, err,
				)
			}
			stages = append(stages, sstages...)
		}
	}

	isRepeat := make([]bool, len(stages))
	for i := range stages {
		isRepeat[i] = i != slices.IndexFunc(stages, func(s Stage) bool {
			return s.Name() == stages[i].Name()
		})
	}
	stages = vec.Mask(stages, vec.Not(isRepeat))

	stages = append(stages, stage)
	return stages, nil
}

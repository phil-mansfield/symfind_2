package pipeline

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/phil-mansfield/chimera/config"
)

type Configure struct {
	BaseStage
	ConfigureInput

	ConfigData
	Pipe *Pipeline
}

// Type checking
var _ Stage = &Configure{}

func (stage *Configure) Init(pipe *Pipeline) {
	stage.Pipe = pipe
	stage.StageName = "Configure"
}

func newConfigDataConfig(data *ConfigData) *config.Config {
	cfg := config.NewConfig()

	cfg.String("TreeFileName", "Name of the tree file. For input tree formats with mulutiple files, this is the first file accoridng to that format's index system.", &data.TreeFileName, data.TreeFileName)

	cfg.String("TreeConfigFileName", "Name of the config file describing the format of the tree file.", &data.TreeConfigFileName, data.TreeConfigFileName)

	cfg.Int("TreeFileN", "The number of tree files. ", &data.TreeFileN, data.TreeFileN)

	cfg.Float("L", "The size of the periodic box in cMpc/h", &data.L, data.L)

	return cfg
}

func newStageInputConfig(input *StageInput) *config.Config {
	cfg := config.NewConfig()

	// No default because the logic for figuring out what the pipeline's
	// default is will be elsewhere.
	cfg.Int("version_id", "The version ID to run the pipeline stage with. This is automatically set by the pipeline.", &input.version_id)

	cfg.String("base_dir", "Base directory of the pipeline.", &input.base_dir)

	return cfg
}

type StageInput struct {
	base_dir   string
	version_id int64
}

type ConfigureInput struct {
	StageInput
	config string
}

func (stage *Configure) Run(id uint64, check bool, args map[string]string) error {

	// id and check aren't used when running Configure
	_, _ = id, check

	// Parse the CLI arguments so that config file can be parsed
	if err := stage.parseStageArgs(args); err != nil {
		return err
	}

	//////////////
	// CLI args //
	//////////////

	// config

	if stage.config == "" {
		return fmt.Errorf("'config' argument must be passed to 'configure' stage")
	}

	// Parse config file
	if err := stage.parseConfigFile(); err != nil {
		return err
	}
	// Parse CLI overwrite of config file
	if err := stage.parseConfigArgs(args); err != nil {
		return err
	}

	/////////////////
	// Config args //
	/////////////////

	// TreeConfigFileName

	if _, err := os.Stat(stage.TreeConfigFileName); err != nil {
		return fmt.Errorf("'TreeConfigFileName' is set to %s, but no file by that name exists.", stage.TreeConfigFileName)
	} else {
		f, err := os.Open(stage.TreeConfigFileName)
		if err != nil {
			return fmt.Errorf("'TreeConfigFileName' is set to %s, but that file cannot be opened: %s", stage.TreeConfigFileName, err.Error())
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("'TreeConfigFileName' is set to %s, but that file cannot be read: %s", stage.TreeConfigFileName, err.Error())
		}
		stage.TreeConfigText = string(data)
	}

	// TreeFileN
	if stage.TreeFileN <= 0 {
		return fmt.Errorf("'TreeFileN' must be set and must be a positive number.")
	}

	// TreeFileName
	// TODO: check for existence
	if stage.TreeFileName == "" {
		return fmt.Errorf("'TreeFileName' not set.")
	}

	// L
	if stage.L == -1 {
		return fmt.Errorf("'L' not set.")
	} else if stage.L <= 0 {
		return fmt.Errorf("'L' must be a positive number.")
	}

	// Add config fields to file

	cfg, _ := stage.Pipe.GetDataset("Config")
	fname := path.Join(args["base_dir"], "Config.h5")
	err := cfg.Write(fname, &ConfigArgs{}, &stage.ConfigData)

	stage.Pipe.Config = &stage.ConfigData

	return err
}

func (stage *Configure) parseStageArgs(args map[string]string) error {
	cfg := newStageInputConfig(&stage.StageInput)
	cfg.String("config", "Name of a plain-text configuration file",
		&stage.config, "")

	// Stage CLI arguments

	cfgLines := []string{}
	for k, v := range args {
		if unicode.IsLower([]rune(k)[0]) {
			cfgLines = append(cfgLines, fmt.Sprintf("%s=%s", k, v))
		}
	}
	cfgText := strings.Join(cfgLines, "\n")

	if err := cfg.Parse(cfgText); err != nil {
		return err
	}

	return nil
}

func (stage *Configure) parseConfigFile() error {
	if stat, err := os.Stat(stage.config); err != nil || stat.IsDir() {
		return fmt.Errorf("No file named '%s'", stage.config)
	}

	file, err := os.Open(stage.config)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	cfg := newConfigDataConfig(&stage.ConfigData)
	return cfg.Parse(string(data))
}

func (stage *Configure) parseConfigArgs(args map[string]string) error {
	cfgLines := []string{}
	for k, v := range args {
		if unicode.IsUpper([]rune(k)[0]) {
			cfgLines = append(cfgLines, fmt.Sprintf("%s=%s", k, v))
		}
	}
	cfgText := strings.Join(cfgLines, "\n")

	cfg := newConfigDataConfig(&stage.ConfigData)
	return cfg.Parse(cfgText)
}

func (stage *Configure) Help() string {
	return `Stage: configure

    configure parses and validates the configuration file and sets up a pipeline directory.

    configure Arguments:
        --config=<string>
            The name of the configuration file.`
}

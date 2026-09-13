package pipeline

import (
	"fmt"
	"runtime/debug"
)

type BuildInfo struct {
	BaseStage
	Pipe *Pipeline
}

func (stage *BuildInfo) Init(pipe *Pipeline) {
	stage.Pipe = pipe
	stage.StageName = "BuildInfo"
	stage.StageDependencies = []string{}
}

func (stage *BuildInfo) Run(id uint64, check bool, args map[string]string) error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("symfind_2 must be compiled with Go's module system. I'm not even mad, I'm impressed: I don't know how you managed to do this.")
	}

	fmt.Print(info.String())

	return nil
}

func (stage *BuildInfo) Help() string {
	return `Stage: build_info

    build_info is a utility stage that reports information about the build
    system and build flags.`
}

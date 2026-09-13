package pipeline

type Data struct {
	BaseStage

	Pipe *Pipeline
}

// Type checking
var _ Stage = &Data{}

func (stage *Data) Init(pipe *Pipeline) {
	stage.Pipe = pipe
	stage.StageName = "Data"
}

func (stage *Data) Run(id uint64, check bool, args map[string]string) error {
	println("Running the `data` stage")
	return nil
}

func (stage *Data) Dependencies() []string {
	return []string{}
}

func (stage *Data) Help() string {
	return `Stage: data

    data is a utility stage that reports information about the status of data
    needed by the pipeline.

    configure Arguments:
        --reset=<dataset name>
            Flag this dataset as being invalid. Future runs of the pipeline will
            require it to be regenerated. --reset resets all datasets. Can
            be a comma-separated list.
        --update=<dataset name>
            Inidcates that this dataset has been updated externally (e.g., a
            new external dataset, an internal dataset copied from another
            pipelines, etc.).  --update updates all datasets. Can be a
            comma-separated list.
        --print=<dataset name>
            Print the status of a dataset. --print prints all datasets. Can be
            a comma-separated list.`
}

package pipeline

import (
	"fmt"
)

type Dataset interface {
	// Init initalisaes a dataset within the correspodning pipeline. Do not call
	// any other methods before this one.
	Init(pipe *Pipeline)

	// Name returns the name of the dataset.
	Name() string

	// Configure sets up the dataset based on the data in Configure. When run
	// in check mode, Configure is the only non-trivial function associated
	// with this dataset that gets run, so it should be as aggressive as it
	// can be with error checking.
	//
	// I would recommend using AI to augment your own implementation of this
	// function. You are not prepared for the sheer number of ways
	// configuration can go wrong.
	Configure(cfg *ConfigData) error

	// IDs is the number of IDs in the dataset (i.e., a generic version of
	// snapshots).
	IDs() int

	// FileNames returns the file anmes associated with the dataset at a given
	// ID.
	FileNames(id int) []string

	// Load loads in a dataset from the named file with the arguemnts args
	// and writes the result to data.
	Read(fname string, args, data any) error

	// Write writes the contents of data to the named file according to the
	// arguments args.
	Write(fname string, args, data any) error

	// Version returns the version flags for each id.
	Version() []uint64

	// Dependency returns the name of the stage the Dataset depends on. An empy
	// string indicates an external dataset.
	Dependency() string
}

// BaseDataset is a base type that implements all the trivial getter and setter
// methods on a Dataset.
type BaseDataset struct {
	Pipeline     *Pipeline
	DsetName     string
	FNames       [][]string
	VersionFlags []uint64
	Dep          string
}

func (dset *BaseDataset) Name() string {
	return dset.DsetName
}

func (dset *BaseDataset) IDs() int {
	return len(dset.FNames)
}

func (dset *BaseDataset) FileNames(id int) []string {
	if id < 0 || id >= len(dset.FNames) {
		panic(fmt.Sprintf("The id %d is out of range for %s. Must be in range [0, %d).", id, dset.DsetName, len(dset.FNames)))
	}

	return dset.FNames[id]
}

func (dset *BaseDataset) SetPipeline(pipe *Pipeline) {
	dset.Pipeline = pipe
}

func (dset *BaseDataset) Version() []uint64 {
	return dset.VersionFlags
}

func (dset *BaseDataset) Dependency() string {
	return dset.Dep
}

type HaloDatasetArgs struct{}
type HaloDatasetData struct{}

type ParticleDatasetArgs struct{}
type ParticleDatasetData struct{}

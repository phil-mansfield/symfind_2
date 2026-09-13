package pipeline

import (
	"fmt"
	"strings"

	"github.com/phil-mansfield/chimera/hdf5"
)

type Tree struct {
	BaseDataset
}

var _ Dataset = &Tree{}

type TreeArgs struct {
	Overwrite bool

	Index int
	Names []string
}

type TreeData struct {
	L float64

	NSnap, NHalo, NBranch int

	// Data/ floats and ints
	F map[string][]float64
	I map[string][]int
	// Branches/ floats and ints
	BF map[string][]float64
	BI map[string][]int
	BB map[string][]byte
	// Hosts/ floats and ints
	HF map[string][]float64
	HI map[string][]int
	// BoundingBoxes/ floats and ints
	BBI map[string][]int
	BBF map[string][]float64
	// Connections/ floats and ints
	CI map[string][]int
	CF map[string][]float64
}

func (dset *Tree) Init(pipe *Pipeline) {
	dset.BaseDataset = BaseDataset{
		Pipeline: pipe,
		DsetName: "Tree",
		Dep:      "AnnotateTree",
	}
}

func (dset *Tree) Configure(cfg *ConfigData) error {
	return nil
}

func (stage *Tree) Read(fname string, args, data any) error {
	panic("NYI")
}

func (stage *Tree) Write(fname string, genArgs, genData any) error {
	args, ok := genArgs.(*TreeArgs)
	if !ok {
		return fmt.Errorf("Tree.Write argument genArgs must be of type *TreeArgs")
	}
	data, ok := genData.(*TreeData)
	if !ok {
		return fmt.Errorf("Tree.Write argument genData must be of type *TreeData")
	}

	var (
		f   *hdf5.File
		err error
	)
	mode := "rw"
	if args.Overwrite {
		mode = "w+"
	}

	if f, err = hdf5.Open(fname, mode); err != nil {
		return err
	}
	defer f.Close()

	if args.Index == 0 {
		f.Group("Header")
		f.Group("Branches")
		f.Group("Hosts")
		f.Group("Data")
		f.Group("BoundingBoxes")
		f.Group("Connections")
	}

	hdr := fmt.Sprintf("Header/%d", args.Index)
	brn := fmt.Sprintf("Branches/%d", args.Index)
	grp := fmt.Sprintf("Data/%d", args.Index)
	hst := fmt.Sprintf("Hosts/%d", args.Index)
	bbox := fmt.Sprintf("BoundingBoxes/%d", args.Index)
	conn := fmt.Sprintf("Connections/%d", args.Index)

	f.Group(hdr)
	f.Group(brn)
	f.Group(grp)
	f.Group(hst)
	f.Group(bbox)
	f.Group(conn)

	n := args.Index + 1
	f.Attr("Header", "NBlocks", &n)
	names, types := treeInfoStrings(args.Names, data)
	f.Attr(hdr, "Names", &names)
	f.Attr(hdr, "Types", &types)
	f.Attr(hdr, "NSnap", &data.NSnap)
	f.Attr(hdr, "NHalo", &data.NHalo)
	f.Attr(hdr, "NBranch", &data.NBranch)
	f.Attr(hdr, "L", &data.L)

	for name, x := range data.I {
		f.Dset(grp+"/"+name, x, []int{len(x)})
	}

	for name, x := range data.F {
		f.Dset(grp+"/"+name, x, []int{len(x)})
	}

	for name, x := range data.BI {
		f.Dset(brn+"/"+name, x, []int{len(x)})
	}

	for name, x := range data.BF {
		f.Dset(brn+"/"+name, x, []int{len(x)})
	}

	for name, x := range data.BB {
		f.Dset(brn+"/"+name, x, []int{len(x)})
	}

	for name, x := range data.HI {
		f.Dset(hst+"/"+name, x, []int{len(x)})
	}

	for name, x := range data.HF {
		f.Dset(hst+"/"+name, x, []int{len(x)})
	}

	for name, x := range data.BBI {
		f.Dset(bbox+"/"+name, x, []int{len(x)})
	}

	for name, x := range data.BBF {
		f.Dset(bbox+"/"+name, x, []int{len(x)})
	}

	for name, x := range data.CI {
		f.Dset(conn+"/"+name, x, []int{len(x)})
	}

	for name, x := range data.CF {
		f.Dset(conn+"/"+name, x, []int{len(x)})
	}

	return nil
}

func treeInfoStrings(names []string, data *TreeData) (nameStr, typeStr string) {
	types := make([]string, len(names))
	for i := range names {
		types[i] = "int"
		if _, ok := data.F[names[i]]; ok {
			types[i] = "float"
		}
	}
	return strings.Join(names, ";"), strings.Join(types, ";")
}

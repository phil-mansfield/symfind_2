package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/phil-mansfield/chimera/config"
	"github.com/phil-mansfield/chimera/text"
)

type InputTree struct {
	BaseDataset

	Separator   byte  // Column delimiter
	Comment     byte  // Comment character
	HeaderLines int64 // Number of lines to skip before data starts

	Names    []string          // List of column names, in file order
	TypeMap  map[string]string // maps names to either "int" or "float"
	IndexMap map[string]int    // maps names to their indices in the file
}

var _ Dataset = &InputTree{}

type InputTreeArgs struct {
	Names []string
}

type InputTreeData struct {
	Names []string

	F map[string][]float64
	I map[string][]int
}

func (dset *InputTree) Init(pipe *Pipeline) {
	dset.BaseDataset = BaseDataset{
		Pipeline: pipe,
		DsetName: "InputTree",
		Dep:      "",
	}
}

var inputTreeErrorText = `TreeConfigFile, %s, is not formatted correctly: %s

TreeConfigFile should be formatted like this:

Separator=' '
Comment='#'
HeaderLines=15
---
id int
pid int
mvir float
...`

func (dset *InputTree) Configure(cfg *ConfigData) error {
	keyValBlock, columnBlock, ok := strings.Cut(cfg.TreeConfigText, "---")

	if !ok {
		return fmt.Errorf(inputTreeErrorText, "No '---' split in tree configuration file %s.", cfg.TreeConfigFileName)
	}

	if err := parseInputTreeConfig(cfg, keyValBlock, dset); err != nil {
		return err
	}

	if err := parseInputTreeColumns(cfg, columnBlock, dset); err != nil {
		return err
	}

	matches, err := filepath.Glob(cfg.TreeFileName + "*")
	if err != nil {
		return err
	}

	dset.FNames = [][]string{matches}

	return nil
}

func parseInputTreeConfig(cfg *ConfigData, keyValBlock string, dset *InputTree) error {
	var (
		separatorText, commentText string
	)

	con := config.NewConfig()
	// TODO: underlying library doesn't handle this correctly
	con.String("Separator", "Delimiter character between columns. Must be in the format 'X'", &separatorText, "' '")
	// TODO: not actually handled correctly in the
	con.String("Comment", "Character used to signify columns. Must be in the format 'X'", &commentText, "'#'")
	// TODO: supporting scipt outputs wrong variabl name and has an off-by-one
	// error.
	con.Int("HeaderLines", "Number of lines in the header before data starts.", &dset.HeaderLines, 0)

	err := con.Parse(keyValBlock)
	if err != nil {
		return fmt.Errorf("Cannot parse input tree configuration file, %s:\n%s", cfg.TreeConfigFileName, err.Error())
	}

	dset.Separator, err = stringToByte("Separator", separatorText)
	if err != nil {
		return err
	}
	dset.Comment, err = stringToByte("Comment", commentText)
	if err != nil {
		return err
	}

	if dset.HeaderLines < 0 {
		return fmt.Errorf("HeaderLines variable set to %d, but must be a positive number.", int(dset.HeaderLines))
	}

	return nil
}

func parseInputTreeColumns(cfg *ConfigData, columnBlock string, dset *InputTree) error {
	lines := strings.Split(columnBlock, "\n")

	dset.IndexMap = map[string]int{}
	dset.TypeMap = map[string]string{}

	for i := range lines {
		tok := strings.Fields(lines[i])
		if len(tok) == 0 {
			continue
		} else if len(tok) != 2 {
			return fmt.Errorf("Cannot parse the line '%s' in the column block of the input-tree configuration file, %s.", lines[i], cfg.TreeConfigFileName)
		}

		name, kind := tok[0], tok[1]

		if _, ok := dset.IndexMap[name]; ok {
			return fmt.Errorf("Column '%s' found multiple times in the column block of the input-tree configuration file, %s.", name, cfg.TreeConfigFileName)
		} else if kind != "int" && kind != "float" {
			return fmt.Errorf("Column '%s' was given type '%s' in the column block of the input-tree configuration file, %s.", name, kind, cfg.TreeConfigFileName)
		}

		dset.Names = append(dset.Names, name)
		dset.IndexMap[name] = i - 1 // Remove the newline after "---"
		dset.TypeMap[name] = kind
	}

	return nil
}

func stringToByte(name, s string) (byte, error) {
	if len(s) != 1 {
		return ' ', fmt.Errorf("Cannot parse variable %s, its value, `%s`, was not in the format `'X'`", name, s)
	}

	return s[0], nil
}

func (dset *InputTree) Read(fname string, genArgs, genData any) error {
	args, ok := genArgs.(*InputTreeArgs)
	if !ok {
		panic("InputTree genArgs argument was not of type *InputTreeArgs")
	}
	data, ok := genData.(*InputTreeData)
	if !ok {
		panic("InputTree genData argument was not of type *InputTreeData")
	}

	if len(args.Names) == 0 {
		args.Names = dset.Names
	}
	data.Names = args.Names

	iNames, fNames, iCols, fCols, err := splitInputTreeNames(args.Names, dset)

	if err != nil {
		return err
	}

	cfg := text.DefaultConfig
	cfg.Separator = dset.Separator
	cfg.Comment = dset.Comment
	cfg.SkipLines = int(dset.HeaderLines)
	cfg.ErrorHandling = text.SkipColumnErrors
	rd := text.TextFile(fname, cfg)

	ints := rd.ReadInts(iCols)
	floats := rd.ReadFloat64s(fCols)

	data.I, data.F = map[string][]int{}, map[string][]float64{}
	for i := range ints {
		data.I[iNames[i]] = ints[i]
	}
	for i := range floats {
		data.F[fNames[i]] = floats[i]
	}

	return nil
}

func (dset *InputTree) Write(fname string, args, data any) error {
	panic("NYI")
}

func splitInputTreeNames(names []string, dset *InputTree) (iNames, fNames []string, iCols, fCols []int, err error) {
	for i := range names {
		if _, ok := dset.IndexMap[names[i]]; !ok {
			return nil, nil, nil, nil, fmt.Errorf("No column named %s found in input-tree configuration file.", names[i])
		}

		idx, kind := dset.IndexMap[names[i]], dset.TypeMap[names[i]]
		switch kind {
		case "int":
			iCols = append(iCols, idx)
			iNames = append(iNames, names[i])
		case "float":
			fCols = append(fCols, idx)
			fNames = append(fNames, names[i])
		default:
			panic("Impossible.")
		}
	}

	return iNames, fNames, iCols, fCols, nil
}

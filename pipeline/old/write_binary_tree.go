package main
1;95;0c
import (
	"fmt"
	"runtime"
	"io"
	"os"
	"path"
	"encoding/binary"
	"io/ioutil"
	"strings"
	"strconv"
	"log"
	
	"github.com/phil-mansfield/symphony_pipeline/lib"
	"github.com/phil-mansfield/guppy/lib/catio"
)

var (
	OldTreeVersion = false

	DFIDCol = 28
	// ID, DescID, UPID, Phantom, Snap, NextProg
	IntCols = []int{ 1, 3, 6, 8, 31, 32 }
	// Mvir, Rs, Vmax, M200b, M200c, M500c, Xoff, SpinBullock,
	// b/a, c/a, T/U, RVmax
	FloatCols = []int{ 10, 12, 16, 39, 40, 41, 43, 45, 46, 47, 56, 60}
	// X(3), V(3), J(3), A(3)
	VectorCols = []int{17, 18, 19, 20, 21, 22, 23, 24, 25, 48, 49, 50 }

	BlankCol = 2

	OldTreeMap = map[int]int{
		28: 28,
		1: 1,
		3: 3,
		6: 6,
		8: 8,
		31: 31,
		32: 32, 
		10: 10,
		12: 12,
		16: 16,
		39: 36,
		40: 37,
		41: 38,
		43: 40,
		45: 42,
		46: 43,
		47: 44,
		56: 53,
		60: 16, // Should never happen
		17: 17,
		18: 18,
		19: 19,
		20: 20,
		21: 21,
		22: 22,
		23: 23,
		24: 24,
		25: 25,
		48: 45,
		49: 46,
		50: 47,
	}
)

// HaloTrack contains the evolution history of a single halo.
func main() {
	haloIndex := -1
    if len(os.Args) > 3 || len(os.Args) <= 1 {
        panic("You must provide a config file.")
    } else if len(os.Args) == 3 {
		var err error
		haloIndex, err = strconv.Atoi(os.Args[2])
		if err != nil { panic(err.Error()) }
	}

	inputName := os.Args[1]
	cfg := lib.ParseConfig(inputName)

	for i := range cfg.BaseDir {
		if haloIndex != -1 && haloIndex !=  i { continue }

		log.Printf("Converting halo %d (%s)", i, cfg.BaseDir[i])
		lib.MemoryUsage()
		
		lib.MaybeMkdir(lib.HaloDirName(cfg.BaseDir[i]))
		ConvertTree(cfg.TreeDir[i], cfg.BaseDir[i], cfg, i)
	}

	log.Println("Finishing write_binary_tree")
}

func CountHeaderLines(fileName string) (nLines, nTrees int) {
	f, err := os.Open(fileName)
	if err != nil { panic(err.Error()) }
	defer f.Close()

	stat, err := f.Stat()
	if err != nil { panic(err.Error()) }
	nMax := stat.Size()

	if nMax > 5000 { nMax = 5000 }
	
	buf := make([]byte, nMax)
	_, err = io.ReadFull(f, buf)
	if err != nil { panic(err.Error()) }

	nLines = 0
	line := 1
	lineStart := 0
	nTrees = -1
	for i := range buf {
		switch buf[i] {
		case '\n', 0:
			line++

			if nTrees == -1 {
				lineEnd := i
				line := buf[lineStart: lineEnd]

				if line[0] != '#' {
					nTrees, err = strconv.Atoi(strings.Trim(string(line), " "))
					if err != nil { nTrees = -1 }
				}
				
				lineStart = lineEnd+1
			}

			if buf[i] == 0 { break }
			
		case '#':
			nLines = line
		}
	}

	return nLines, nTrees
}

func ConvertTree(treeDir, outDir string, cfg *lib.Config, cfgi int) {
	files := TreeFileNames(treeDir)
	binFiles := BinFileNames(outDir, files)
	for i := range files {
		runtime.GC()
		ConvertTreeFile(files[i], binFiles[i], cfg, cfgi)
	}
}

func TreeFileNames(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	if err != nil { panic(err.Error()) }
	
	out := []string{ }
	for i := range files {
		name := files[i].Name()
		if len(name) >= 8 &&
			name[:4] == "tree" &&
			name[len(name)-4:] == ".dat" {
			out = append(out, path.Join(dir, name))
		}
	}

	return out
}

func BinFileNames(outDir string, treeFileNames []string) []string {
	out := []string{ }
	for i := range treeFileNames {
		outName := fmt.Sprintf("tree_%d.dat", i)
		out = append(out, path.Join(lib.HaloDirName(outDir), outName))
	}
	return out
}

func ConvertTreeFile(treeName, outName string, cfg *lib.Config, cfgi int) {
	catioCfg := catio.DefaultConfig

	var nTrees int
	if catioCfg.SkipLines, nTrees = CountHeaderLines(treeName); nTrees == 0 {
		return
	}

	switch cfg.TreeStyle[cfgi] {
	case "ct_rvmax", "ct_rhapsody":
	default:
		panic(fmt.Sprintf("Cannot parse tree files with style '%s'",
			cfg.TreeStyle[cfgi]))
	}

	log.Printf("Parsing %s", treeName)
	rd := catio.TextFile(treeName, catioCfg)

	f, err := os.Create(outName)
	if err != nil { panic(err.Error()) }
	defer f.Close()
	
	order := ConvertInts(rd, f, cfg.TreeStyle[cfgi], cfg.MaxSnap[cfgi])
	runtime.GC()
	ConvertFloats(rd, f, cfg.TreeStyle[cfgi], order)
	runtime.GC()
	ConvertVectors(rd, f, cfg.TreeStyle[cfgi], order)
	runtime.GC()
}

func ConvertInts(rd catio.Reader, f *os.File, treeStyle string, maxSnap int32) []int32 {
	// Read the columns
	colIdxs := []int{ DFIDCol }
	colIdxs = append(colIdxs, IntCols...)

	if OldTreeVersion {
		for i := range colIdxs { colIdxs[i] = OldTreeMap[colIdxs[i]] }
	}
	cols := rd.ReadInts(colIdxs)

	dfid := ToInt32(cols[0])
	order := IDOrder(dfid)

	hd := CreateTreeHeader(len(order))
	lib.WriteTreeHeader(f, hd)

	buf := dfid
	byteOrder := binary.LittleEndian
	for i := range cols {
		if colIdxs[i] == 31 {
			// Fix the range of snapshot indices in consistent-trees
			maxSnapIdx := IntMax(cols[i])
			deltaSnap := int(maxSnap) - maxSnapIdx
			for j := range cols[i] {
				cols[i][j] += deltaSnap
			}
		}

		ReorderToInt32(cols[i], order, buf)
		err := binary.Write(f, byteOrder, buf)
		if err != nil { panic(err.Error()) }
	}

	return order
}

func IntMax(x []int) int {
	max := x[0]
	for i := 1; i < len(x); i++ {
		if x[i] > max { max = x[i] }
	}
	return max
}

func CreateTreeHeader(n int) *lib.TreeHeader {
	cols := []int32{ int32(DFIDCol) }
	for i := range IntCols {
		cols = append(cols, int32(IntCols[i]))
	}
	for i := range FloatCols {
		cols = append(cols, int32(FloatCols[i]))
	}
	for i := 0; i < len(VectorCols); i += 3 {
		cols = append(cols, int32(VectorCols[i]))
	}

	return &lib.TreeHeader{
		lib.FixedHeader{
			int32(n),
			int32(1 + len(IntCols)),
			int32(len(FloatCols)),
			int32(len(VectorCols) / 3),
		}, cols,
	}
}

func ConvertFloats(rd catio.Reader, f *os.File, treeStyle string, order []int32) {
	colIdxs := make([]int, len(FloatCols))
	if OldTreeVersion {
		for i := range colIdxs { colIdxs[i] = OldTreeMap[FloatCols[i]] }
	} else {
		for i := range colIdxs { colIdxs[i] = FloatCols[i] }
	}

	if treeStyle == "ct_rhapsody" {
		for i := range colIdxs {
			if colIdxs[i] == 60 { colIdxs[i] = BlankCol }
		}
	}

	cols := rd.ReadFloat32s(colIdxs)

	for i := range cols {
		if colIdxs[i] == BlankCol {
			for j := range cols[i] { cols[i][j] = -1 }
		}
	}

	buf := make([]float32, len(cols[0]))
	for i := range cols {
		ReorderToFloat32(cols[i], order, buf)
		err := binary.Write(f, binary.LittleEndian, buf)
		if err != nil { panic(err.Error()) }
	}
}

func ConvertVectors(rd catio.Reader, f *os.File, treeStyle string, order []int32) {
	colIdxs := make([]int, len(VectorCols))
	if OldTreeVersion {
		for i := range colIdxs { colIdxs[i] = OldTreeMap[VectorCols[i]] }
	} else {
		for i := range colIdxs { colIdxs[i] = VectorCols[i] }
	}
	cols := rd.ReadFloat32s(colIdxs)

	buf := make([][3]float32, len(cols[0]))
	for i := 0; i < len(cols); i += 3 {
		ReorderToVector32([3][]float32{cols[i],cols[i+1],cols[i+2]}, order, buf)
		err := binary.Write(f, binary.LittleEndian, buf)
		if err != nil { panic(err.Error()) }
	}
}

func ToInt32(x []int) []int32 {
	out := make([]int32, len(x))
	for i := range x {
		out[i] = int32(x[i])
	}

	return out
}

func ReorderToInt32(x []int, order, out []int32) {
	for i := range out {
		out[i] = int32(x[order[i]])
	}
}

func ReorderToFloat32(x []float32, order []int32, out []float32){
	for i := range out {
		out[i] = x[order[i]]
	}
}

func ReorderToVector32(x [3][]float32, order []int32, out [][3]float32) {
	for i := range out {
		for dim := 0; dim < 3; dim++ {
			out[i][dim] = x[dim][order[i]]
		}
	}
}

func IDOrder(id []int32) []int32 {
	fid := make([]float64, len(id))
	for i := range id {
		fid[i] = float64(id[i])
	}

	idx := lib.QuickSortIndex(fid)
	idx32 := make([]int32, len(idx))
	for i := range idx32 { idx32[i] = int32(idx[i]) }
	
	return idx32
}

func WriteInt(f *os.File, n int) {
	err := binary.Write(f, binary.LittleEndian, int32(n))
	if err != nil { panic(err.Error()) }
}

func ParseInputFile(fname string) (trees, outs []string) {
	b, err := ioutil.ReadFile(fname)
	if err != nil {
		panic(fmt.Sprintf("Could not open input file: %s", err.Error()))
	}
	s := string(b)

	trees, outs = []string{ }, []string{ }
	
	lines := strings.Split(s, "\n")
	for i := range lines {
		line := strings.Trim(lines[i], " ")
		if len(line) == 0 { continue }

		tok := strings.Split(lines[i], " ")
		cols := []string{ }
		for i := range tok {
			if len(tok[i]) > 0 {
				cols = append(cols, tok[i])
			}
		}
		
		if len(cols) != 2 {
			panic(fmt.Sprintf("Line %d of %s is '%s', but you need there " +
				"to be three columns.", i+1, fname, line))
		}
		
		trees = append(trees, cols[0])
		outs = append(outs, cols[1])
	}
	
	return trees, outs
}

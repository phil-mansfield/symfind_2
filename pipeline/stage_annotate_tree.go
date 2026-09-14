package pipeline

import (
	"cmp"
	"fmt"
	"log"
	"path"
	"runtime"
	"slices"

	"github.com/phil-mansfield/chimera/vec"
)

type AnnotateTree struct {
	BaseStage
	Pipe *Pipeline
}

// Type checking
var _ Stage = &AnnotateTree{}

func (stage *AnnotateTree) Init(pipe *Pipeline) {
	stage.Pipe = pipe
	stage.StageName = "AnnotateTree"
	stage.StageDependencies = []string{"Config", "InputTree"}
}

func (stage *AnnotateTree) Run(id uint64, check bool, args map[string]string) error {
	tree, _ := stage.Pipe.GetDataset("Tree")
	inputTree, _ := stage.Pipe.GetDataset("InputTree")

	outArgs := &TreeArgs{}
	outData := &TreeData{}

	for i, fname := range inputTree.FileNames(0) {
		runtime.GC()

		args := &InputTreeArgs{}
		data := &InputTreeData{}

		// Empty InputTreeArgs read all columns
		if err := inputTree.Read(fname, args, data); err != nil {
			return err
		}

		if err := sortInputTree(data); err != nil {
			return err
		}

		outData.L = stage.Pipe.Config.L
		outData.F = data.F
		outData.I = data.I

		outData.HF = map[string][]float64{}
		outData.HI = map[string][]int{}

		outData.BF = map[string][]float64{}
		outData.BI = map[string][]int{}
		outData.BB = map[string][]byte{}

		outData.BBF = map[string][]float64{}
		outData.BBI = map[string][]int{}

		outData.CF = map[string][]float64{}
		outData.CI = map[string][]int{}

		if err := annotateTree(outData); err != nil {
			return err
		}

		fname := path.Join(stage.Pipe.Dir, "Tree.h5")

		// TODO: eventually needs to not keep re-opening files every time we
		// call Read and Write? Think this over. See if it matters. Need to
		// flush output eventually, though.

		outArgs.Index = i
		outArgs.Overwrite = i == 0
		outArgs.Names = data.Names
		if err := tree.Write(fname, outArgs, outData); err != nil {
			return err
		}
	}

	return nil
}

func sortInputTree(data *InputTreeData) error {
	// TODO: generalise this and allow the tree to tell you what it is. Pull
	// this error message out of here and into Configure
	dfid, ok := data.I["Depth_first_ID"]
	if !ok {
		return fmt.Errorf("No column labeled 'Depth_first_ID' found in tree")
	}

	idx := vec.Argsort(dfid)

	n := len(dfid)
	iBuf := make([]int, n)
	for name := range data.I {
		vec.At(data.I[name], idx, iBuf)
		copy(data.I[name], iBuf)
	}

	fBuf := make([]float64, n)
	for name := range data.F {
		vec.At(data.F[name], idx, fBuf)
		copy(data.F[name], fBuf)
	}

	return nil
}

func (stage *AnnotateTree) Help() string {
	panic("NYI")
}

func annotateTree(tree *TreeData) error {
	// TODO validate the correct names of columns earlier in the pipeline,
	// allow them to be passed by config file if needed.

	// Consistent-trees naminch scheme
	// TODO - switch on other pre-baked naming schemes
	haloes := &Haloes{
		L: tree.L,

		ID:     tree.I["id"],
		DescID: tree.I["desc_id"],
		UPID:   tree.I["upid"],
		DFID:   tree.I["Depth_first_ID"],
		Snap:   tree.I["Snap_num"],
		Mvir:   tree.F["mvir"], // TODO allow user to choose mass definition

		Position: [3][]float64{
			tree.F["x"], tree.F["y"], tree.F["z"],
		},

		IDTable: NewLookupTable(tree.I["id"]),
	}

	// TODO: tracks makes no sense
	// TODO: error handling!?!
	t := CalcTracks(haloes)
	tree.BI["Edges"] = t.Edges

	tree.BB["Error"] = t.Error
	tree.BF["MpeakRaw"] = t.MpeakRaw
	tree.BF["MpeakPre"] = t.MpeakPre
	tree.BF["Minfall"] = t.Minfall

	tree.HI["Edges"] = t.HostEdges
	tree.HI["Branch"] = t.HostBranch
	tree.HI["FirstSnap"] = t.HostFirstSnap
	tree.HI["LastSnap"] = t.HostLastSnap

	tree.BBF["Offsets"] = t.Offsets
	tree.BBF["Spans"] = t.Spans

	tree.CI["Parents"] = t.Parents
	tree.CI["ParentsEdges"] = t.ParentsEdges
	tree.CI["Children"] = t.Children

	tree.NSnap = t.MaxSnap + 1
	tree.NHalo = len(haloes.Mvir)
	tree.NBranch = len(t.Minfall)

	return nil
}

// Haloes is a collection of raw columns from the tree.dat files.
type Haloes struct {
	L float64

	ID, DescID, UPID, DFID, Snap []int

	Position [3][]float64

	Mvir []float64

	IDTable *LookupTable
}

type LookupTable struct {
	Order, SortedIDs []int
}

func NewLookupTable(id []int) *LookupTable {
	order := vec.Argsort(id)
	return &LookupTable{Order: order, SortedIDs: vec.At(id, order)}
}

func (tab LookupTable) Find(id int) (int, bool) {
	i, ok := slices.BinarySearch(tab.SortedIDs, id)
	if !ok {
		return 0, false
	}
	return tab.Order[i], true
}

type Tracks struct {
	N, MaxSnap int
	// Haloes are sorted by depth-first ID (DFID). Branches are contiguous in
	// DFID, meaning that all branches are subslices of the global tree. All
	// haloes are members of exactly one branch.

	// The tree information for a branch i is contained in the range
	// [Edges[i]: Edges[i+1]]
	// or
	// [Start[i]: End[i]]
	// The latter arrays are just a utility slicing of the first.
	Edges        []int
	Starts, Ends []int

	IsReal      []bool // Branch forms outside of another halo
	IsDisappear []bool // Branch doesn't make it to final snapshot

	Error []byte // bitwise error flags
	// (Error[i] >> 0) & 1 = !IsReal[i]
	// (Error[i] >> 1) & 1 = IsDisappear[i]
	// (Error[i] >> 2) & 1 = ---
	// (Error[i] >> 3) & 1 = ---
	// (Error[i] >> 4) & 1 = ---
	// (Error[i] >> 5) & 1 = ---
	// (Error[i] >> 6) & 1 = ---
	// (Error[i] >> 7) & 1 = ---

	// All nested slices contain one element for every snapshot where the halo
	// has a UPID != -1. All elements refer to the most massive host.
	HostIdx  [][]int // Indices of the subhalo's hosts within the tree. Ordered from last-to-first.
	HostSnap [][]int // The snapshots of the subhalo's hosts
	TrackIdx [][]int // Indices of the tracks containing each host

	IsReverseMerger [][]bool // "host" later merges into this branch
	IsReverseSub    [][]bool // "host" later becomes a subhalo of the satellite
	// TODO: Add Mpeak-based condition? Part of ReverseSub?
	IsValidHost [][]bool // Combine all previous validity checks.

	// TODO: implement different masses

	// Different measures of peak/infall mass.
	MpeakPre []float64 // Highest Mvir achieved while not a subhalo. -1 if branch always a subhalo.
	MpeakRaw []float64 // Highest Mvir achieved
	Minfall  []float64 // Mvir is the mass at the last snap before a halo became a subhalo for the first time.

	// The host information for a branch i is contained in the range
	// [HostEdges[i]: HostEdges[i+1]]
	// or
	// [HostStarts[i]: HostEnds[i]]
	// The latter arrays are just a utility slicing of the first. Merger
	// information is sorted in increasing order by HostFirstSnap
	//
	// This host information is per-host, not per-snapshot like the nested
	// slices.
	HostEdges            []int
	HostStarts, HostEnds []int

	HostBranch    []int // The branch index of the host
	HostFirstSnap []int // The first snapshot when the host was the most-massive host.
	HostLastSnap  []int // The last snapshot when the host was the most-massive host.

	// Spans and Offsets are a pair of 3*NSnap-length arrays which give the
	// lower corner and width of a peridoic bounding box which
	// contains all the haloes in this block. The bounding box is garuantted to
	// be the smallest possible bounding box when Span < L/2. in all thre
	// dimensions. Offset[3*i + j] is lower corner at snapshot i, dimension d.
	// When the snapshot is empty, Offset and Span are -1.
	Offsets, Spans []float64

	// Children gives the indices of branches that each branch mergers
	// into. If a branch never merges, Children is set to -1. There is at most
	// one child per branch.
	Children []int

	// Parents are all the indices of all the branches which merge into a given
	// branch. Branches can have multiple parents. Parents are
	// in increasing order by snapshot. Parents at the same snapshot are in
	// increasing order by MpeakPre
	Parents, ParentsEdges []int
}

func CalcTracks(h *Haloes) *Tracks {
	t := &Tracks{}

	// TODO: can be rewritten to avoid excess heap usage. I /think/ this
	// is unneccessary, since this floats a couple numbers per branch and should
	// be small compares to the size of the tree. But should be checked.

	// Header

	t.MaxSnap = slices.Max(h.Snap)

	// Branches

	t.Edges = Edges(h)
	t.Starts, t.Ends = t.Edges[:len(t.Edges)-1], t.Edges[1:]

	t.N = len(t.Starts)

	t.IsReal = IsReal(h, t)
	t.IsDisappear = IsDisappear(h, t, t.MaxSnap)
	t.Error = BranchErrorCodes(t)

	// Host information

	t.HostIdx, t.HostSnap = FindAllHosts(h, t)
	t.TrackIdx, t.Children = FindTrackIndices(h, t)

	t.IsReverseMerger = IsReverseMerger(h, t)
	t.IsReverseSub = IsReverseSub(h, t)
	t.IsValidHost = IsValidHost(h, t)
	t.MpeakRaw, t.MpeakPre, t.Minfall = Mpeak(h, t)

	t.HostEdges, t.HostBranch, t.HostFirstSnap, t.HostLastSnap = HostInfo(t)
	t.HostStarts, t.HostEnds = t.HostEdges[:len(t.HostEdges)-1], t.HostEdges[1:]

	// Bounding boxes

	t.Offsets, t.Spans = BoundingBoxes(h, t)

	// Connections

	t.Parents, t.ParentsEdges = Parents(h, t)

	return t
}

// TODO: either remove or normalise this.
func MemoryLog() {
	ms := &runtime.MemStats{}
	runtime.ReadMemStats(ms)
	log.Printf("Allocated: %.1f In Use: %.1f Idle: %.1f\n",
		float64(ms.Alloc)/1e9, float64(ms.HeapInuse)/1e9,
		float64(ms.HeapIdle)/1e9)
}

func Edges(h *Haloes) []int {
	edges := []int{0}
	for i := 0; i < len(h.DFID)-1; i++ {
		if h.DescID[i+1] != h.ID[i] {
			edges = append(edges, i+1)
		}
	}
	edges = append(edges, len(h.DFID))

	return edges
}

func IsReal(h *Haloes, t *Tracks) []bool {
	// Reveiwed
	isReal := make([]bool, t.N)
	for i := range isReal {
		isReal[i] = h.UPID[t.Ends[i]-1] == -1
	}
	return isReal
}

func IsDisappear(h *Haloes, t *Tracks, maxSnap int) []bool {
	// TODO: are these already removed from the consistent-trees output?
	// I think so?
	out := make([]bool, t.N)

	for i := range out {
		out[i] = h.DescID[t.Starts[i]] == -1 &&
			h.Snap[t.Starts[i]] != maxSnap
	}

	return out
}

func BranchErrorCodes(t *Tracks) []byte {
	out := make([]byte, len(t.IsReal))

	for i := range t.IsReal {
		codes := [8]byte{}

		if !t.IsReal[i] {
			codes[0] = 1
		} else if t.IsDisappear[i] {
			codes[1] = 1
		}

		code := byte(0)
		for j := range codes {
			code = code | (codes[j] << j)
		}

		out[i] = code
	}

	return out
}

func FindAllHosts(h *Haloes, t *Tracks) ([][]int, [][]int) {
	// Reviewed
	// TODO: use a shared backing array for memory efficiency and to make the
	// writing easier
	outIdx, outSnap := make([][]int, t.N), make([][]int, t.N)

	for i := range t.Starts {
		for j := t.Starts[i]; j < t.Ends[i]; j++ {
			if h.UPID[j] != -1 {
				k, ok := h.IDTable.Find(h.UPID[j])
				if !ok {
					panic(fmt.Sprintf("Unable to find UPID %d for halo %d", h.UPID[j], j))
				}
				outIdx[i] = append(outIdx[i], k)
				outSnap[i] = append(outSnap[i], h.Snap[j])
			}
		}
	}

	return outIdx, outSnap
}

func FindTrackIndices(h *Haloes, t *Tracks) (trackIdx [][]int, descIdx []int) {
	// TODO: rewrite by binary searching on t.Edges?

	trackIdx = make([][]int, t.N)
	descIdx = make([]int, t.N)
	// The original version re-used Snap's buffer here, but I think that's
	// playing with fire.
	idxTable := make([]int, len(h.Snap))

	for i := range t.Starts {
		for j := t.Starts[i]; j < t.Ends[i]; j++ {
			idxTable[j] = i
		}
	}

	// TODO: use a single backing array?

	for i := range t.N {
		trackIdx[i] = make([]int, len(t.HostIdx[i]))

		for j := range trackIdx[i] {
			trackIdx[i][j] = idxTable[t.HostIdx[i][j]]
		}

		descID := h.DescID[t.Starts[i]]
		if descID == -1 {
			descIdx[i] = -1
			continue
		}
		idx, ok := h.IDTable.Find(descID)
		if !ok {
			panic(fmt.Sprintf("Could not find descdant ID, %d for halo %d (index = %d, branch = %d)", descID, h.ID[t.Starts[i]], t.Starts[i], i))
		}
		descIdx[i] = idxTable[idx]
	}

	return trackIdx, descIdx
}

func IsReverseMerger(h *Haloes, t *Tracks) [][]bool {
	out := make([][]bool, len(t.HostIdx))

	for i := range out {
		out[i] = make([]bool, len(t.HostIdx[i]))

		minID, maxID := h.DFID[t.Starts[i]], h.DFID[t.Ends[i]-1]
		for j := range t.HostIdx[i] {
			// The last halo of the host (first in DF order)
			start := t.Starts[t.TrackIdx[i][j]]
			// If the descendent is -1, that means this wasn't a merger
			// and we're okay
			if h.DescID[start] != -1 {
				// If the host's descendant has a DFID within this halo's DFID
				// range, that means the "host" merged with the descendant.
				k, ok := h.IDTable.Find(h.DescID[start])
				if !ok {
					panic(fmt.Sprintf("Unable to find DescID %d for halo %d", h.DescID[start], start))
				}
				out[i][j] = h.DFID[k] >= minID && h.DFID[k] <= maxID
			}
		}
	}

	return out
}

func IsReverseSub(h *Haloes, t *Tracks) [][]bool {
	out := make([][]bool, len(t.HostIdx))

	for i := range out {
		out[i] = make([]bool, len(t.HostIdx[i]))

		minID, maxID := h.DFID[t.Starts[i]], h.DFID[t.Ends[i]-1]
		for j := range t.HostIdx[i] {
			// Last halo of the host (first in DF order)
			start := t.Starts[t.TrackIdx[i][j]]

			// Loop from last host halo to the timestep after the host-sub
			// pairing was found
			for k := start; k < t.HostIdx[i][j]; k++ {
				// Skip snapshots where the "host" is still a central
				if h.UPID[k] != -1 {
					// Check if the "host" is a later subhalo of the current
					// subhalo
					l, ok := h.IDTable.Find(h.UPID[k])
					if !ok {
						panic(fmt.Sprintf("Unable to find UPID %d for halo %d", h.UPID[k], k))
					}

					if h.DFID[l] >= minID && h.DFID[l] <= maxID {
						out[i][j] = true
						break
					}
				}
			}
		}
	}

	// Do a second pass to make sure that if something gets flagged as a
	// reverse sub and then later gets flagged as a normal subhalo, the
	// earlier reverse subhalo classification gets removed.

	for i := range out {
		for j := range out[i] {
			if out[i][j] {
				for k := range j {
					if t.TrackIdx[i][j] == t.TrackIdx[i][k] && !out[i][k] {
						out[i][j] = false
						break
					}
				}
			}
		}
	}

	return out
}

func FindFirstIndex(h *Haloes, i int) int {
	j := i
	for ; j > 0; j-- {
		if h.DescID[j] != h.ID[j-1] {
			return j
		}
	}
	return 0
}

func FindLastIndex(h *Haloes, i int) int {
	j := int(i)
	for ; j < len(h.ID)-1; j++ {
		if h.DescID[j+1] != h.ID[j] {
			return j
		}
	}

	return len(h.ID) - 1
}

func IsValidHost(h *Haloes, t *Tracks) [][]bool {
	out := make([][]bool, t.N)
	for i := range out {
		out[i] = make([]bool, len(t.TrackIdx[i]))
		for j := range out[i] {
			k := t.TrackIdx[i][j]
			out[i][j] = t.IsReal[k] && !t.IsDisappear[k] &&
				!t.IsReverseMerger[i][j] && !t.IsReverseSub[i][j]

		}
	}

	return out
}

func Mpeak(h *Haloes, t *Tracks) (mpeakRaw, mpeakPre, minfall []float64) {
	mpeakRaw = make([]float64, t.N)
	mpeakPre = make([]float64, t.N)
	minfall = make([]float64, t.N)

	isSubhalo := make([]bool, t.MaxSnap+1)

	for i := range t.N {
		minfall[i], mpeakPre[i] = -1, -1

		// Flag snapshots where the branch is a bona fide subhalo
		for j := range isSubhalo {
			isSubhalo[j] = false
		}
		for j := range t.HostIdx[i] {
			if t.IsValidHost[i][j] {
				isSubhalo[t.HostSnap[i][j]] = true
			}
		}

		for j := t.Starts[i]; j < t.Ends[i]; j++ {
			// Simple maximum
			if h.Mvir[j] > mpeakRaw[i] {
				mpeakRaw[i] = h.Mvir[j]
			}

			// Maximum if not subahlo
			if !isSubhalo[h.Snap[j]] {
				if h.Mvir[j] > mpeakPre[i] {
					mpeakPre[i] = h.Mvir[j]
				}
			}
		}

		// minfall gets updated until this flips to false
		wasNeverSub := true
		// loop backwards to go forward in time
		for j := t.Ends[i] - 1; j >= t.Starts[i]; j-- {
			if isSubhalo[h.Snap[j]] {
				wasNeverSub = false
			} else if wasNeverSub {
				minfall[i] = h.Mvir[j]
			}
		}

		if wasNeverSub {
			minfall[i] = h.Mvir[t.Starts[i]]
		}
	}

	return mpeakRaw, mpeakPre, minfall
}

func HostInfo(t *Tracks) (edges, branch, first, last []int) {
	edges = []int{0}

	for i := range t.TrackIdx {
		hostBranch, hostFirstSnap, hostLastSnap := CondenseHostInfo(
			t.HostSnap[i], t.TrackIdx[i])

		branch = append(branch, hostBranch...)
		first = append(first, hostFirstSnap...)
		last = append(last, hostLastSnap...)

		edges = append(edges, len(branch))
	}

	return edges, branch, first, last
}

func CondenseHostInfo(hostSnap, trackIdx []int) (hostBranch, hostFirstSnap, hostLastSnap []int) {
	// TODO: rewrite to reduce heap allocations
	idx := vec.Argsort(trackIdx)

	prevti := -1
	for _, i := range idx {
		ti, snap := trackIdx[i], hostSnap[i]

		if prevti == ti {
			last := len(hostFirstSnap) - 1
			if snap > hostLastSnap[last] {
				hostLastSnap[last] = snap
			} else if snap < hostFirstSnap[last] {
				hostFirstSnap[last] = snap
			}
		} else {
			hostBranch = append(hostBranch, ti)
			hostFirstSnap = append(hostFirstSnap, snap)
			hostLastSnap = append(hostLastSnap, snap)
		}

		prevti = ti
	}

	// Reorder to be increasing by snapshot of first occurance.
	idx = vec.Argsort(hostFirstSnap)
	hostBranch = vec.At(hostBranch, idx)
	hostFirstSnap = vec.At(hostFirstSnap, idx)
	hostLastSnap = vec.At(hostLastSnap, idx)

	return hostBranch, hostFirstSnap, hostLastSnap
}

func BoundingBoxes(h *Haloes, t *Tracks) (offsets, spans []float64) {

	offsets = make([]float64, 3*(t.MaxSnap+1))
	spans = make([]float64, 3*(t.MaxSnap+1))

	for i := range offsets {
		offsets[i], spans[i] = -1, -1
	}

	for i := range t.N {

		for j := t.Starts[i]; j < t.Ends[i]; j++ {
			snap := h.Snap[j]

			for k := range 3 {
				idx := 3*snap + k

				// This is the first time this snapshot has been encounterd
				if offsets[idx] == -1 {
					offsets[idx] = h.Position[k][j]
					spans[idx] = 0
					continue
				}

				offsets[idx], spans[idx] = UpdatePeriodicBound(
					h.L, offsets[idx], spans[idx], h.Position[k][j],
				)
			}
		}
	}

	return offsets, spans
}

func UpdatePeriodicBound(L, offset, span, x float64) (outOffset, outSpan float64) {
	limit := offset + span
	if limit > L {
		limit -= L
	}

	dx1 := PeriodicDisplacement(L, x, offset)
	dx2 := PeriodicDisplacement(L, x, limit)

	// Point is contained in the range: do nothing.
	if PeriodicContains(L, offset, span, x) {
		return offset, span
	}

	// Technically, this is only optimal if the points have spans less than L/2.
	// The alternative is to sort and search for gaps. That latter approach is
	// exact, but requires allocating an array.

	if dx1 > 0 && dx2 > 0 || dx1 < 0 && dx2 > 0 && dx2 < -dx1 {
		// Closer to the upper edge of the range
		return offset, span + dx2
	} else {
		// Closer to the lower edge of the range. If this is true, dx1 will
		// always need to be negative.
		offset, span = offset+dx1, span-dx1
		if offset < 0 {
			offset += L
		}
		return offset, span
	}
}

// PeriodicDisplacement computes x1-x2 in a periodic box of length L.
func PeriodicDisplacement(L, x1, x2 float64) float64 {
	dx := x1 - x2

	if dx < -L/2 {
		dx += L
	} else if dx > L/2 {
		dx -= L
	}

	return dx
}

func PeriodicContains(L, offset, span, x float64) bool {
	return (x >= offset && x <= offset+span) || x <= offset+span-L
}

func Parents(h *Haloes, t *Tracks) (parents, parentsEdges []int) {
	// TODO: fix this alogithm: this is wasteful of allocations.
	nestedParents := make([][]int, t.N)

	for i := range t.N {
		if t.Children[i] != -1 {
			c := t.Children[i]
			nestedParents[c] = append(nestedParents[c], i)
		}
	}

	// Sort first by snapshot, then by mass.

	for i := range nestedParents {
		slices.SortFunc(nestedParents[i], func(a, b int) int {
			snapA, snapB := h.Snap[t.Starts[a]], h.Snap[t.Starts[b]]
			massA, massB := t.MpeakPre[a], t.MpeakPre[b]
			if snapA == snapB {
				return cmp.Compare(massA, massB)
			} else {
				return cmp.Compare(snapA, snapB)
			}
		})
	}

	// Flatten the parents array
	n := 0
	parentsEdges = make([]int, t.N+1)
	for i := range nestedParents {
		n += len(nestedParents[i])
		parentsEdges[i+1] = n
	}

	parents = make([]int, n)
	for i := range nestedParents {
		copy(parents[parentsEdges[i]:parentsEdges[i+1]], nestedParents[i])
	}

	return parents, parentsEdges
}

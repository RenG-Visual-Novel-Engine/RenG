// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssa

import (
	"cmd/compile/internal/abi"
	"cmd/compile/internal/ir"
	"cmd/compile/internal/types"
	"cmd/internal/dwarf"
	"cmd/internal/obj"
	"cmd/internal/src"
	"encoding/hex"
	"fmt"
	"internal/buildcfg"
	"math/bits"
	"sort"
	"strings"
)

type SlotID int32
type VarID int32

// A FuncDebug contains all the debug information for the variables in a
// function. Variables are identified by their LocalSlot, which may be the
// result of decomposing a larger variable.
type FuncDebug struct {
	// Slots is all the slots used in the debug info, indexed by their SlotID.
	Slots []LocalSlot
	// The user variables, indexed by VarID.
	Vars []*ir.Name
	// The slots that make up each variable, indexed by VarID.
	VarSlots [][]SlotID
	// The location list data, indexed by VarID. Must be processed by PutLocationList.
	LocationLists [][]byte

	// Filled in by the user. Translates Block and Value ID to PC.
	GetPC func(ID, ID) int64
}

type BlockDebug struct {
	// Whether the block had any changes to user variables at all.
	relevant bool
	// State at the end of the block if it's fully processed. Immutable once initialized.
	endState []liveSlot
}

// A liveSlot is a slot that's live in loc at entry/exit of a block.
type liveSlot struct {
	// An inlined VarLoc, so it packs into 16 bytes instead of 20.
	Registers RegisterSet
	StackOffset

	slot SlotID
}

func (loc liveSlot) absent() bool {
	return loc.Registers == 0 && !loc.onStack()
}

// StackOffset encodes whether a value is on the stack and if so, where. It is
// a 31-bit integer followed by a presence flag at the low-order bit.
type StackOffset int32

func (s StackOffset) onStack() bool {
	return s != 0
}

func (s StackOffset) stackOffsetValue() int32 {
	return int32(s) >> 1
}

// stateAtPC is the current state of all variables at some point.
type stateAtPC struct {
	// The location of each known slot, indexed by SlotID.
	slots []VarLoc
	// The slots present in each register, indexed by register number.
	registers [][]SlotID
}

// reset fills state with the live variables from live.
func (state *stateAtPC) reset(live []liveSlot) {
	slots, registers := state.slots, state.registers
	for i := range slots {
		slots[i] = VarLoc{}
	}
	for i := range registers {
		registers[i] = registers[i][:0]
	}
	for _, live := range live {
		slots[live.slot] = VarLoc{live.Registers, live.StackOffset}
		if live.Registers == 0 {
			continue
		}

		mask := uint64(live.Registers)
		for {
			if mask == 0 {
				break
			}
			reg := uint8(bits.TrailingZeros64(mask))
			mask &^= 1 << reg

			registers[reg] = append(registers[reg], live.slot)
		}
	}
	state.slots, state.registers = slots, registers
}

func (s *debugState) LocString(loc VarLoc) string {
	if loc.absent() {
		return "<nil>"
	}

	var storage []string
	if loc.onStack() {
		storage = append(storage, "stack")
	}

	mask := uint64(loc.Registers)
	for {
		if mask == 0 {
			break
		}
		reg := uint8(bits.TrailingZeros64(mask))
		mask &^= 1 << reg

		storage = append(storage, s.registers[reg].String())
	}
	return strings.Join(storage, ",")
}

// A VarLoc describes the storage for part of a user variable.
type VarLoc struct {
	// The registers this variable is available in. There can be more than
	// one in various situations, e.g. it's being moved between registers.
	Registers RegisterSet

	StackOffset
}

func (loc VarLoc) absent() bool {
	return loc.Registers == 0 && !loc.onStack()
}

var BlockStart = &Value{
	ID:  -10000,
	Op:  OpInvalid,
	Aux: StringToAux("BlockStart"),
}

var BlockEnd = &Value{
	ID:  -20000,
	Op:  OpInvalid,
	Aux: StringToAux("BlockEnd"),
}

var FuncEnd = &Value{
	ID:  -30000,
	Op:  OpInvalid,
	Aux: StringToAux("FuncEnd"),
}

// RegisterSet is a bitmap of registers, indexed by Register.num.
type RegisterSet uint64

// logf prints debug-specific logging to stdout (always stdout) if the current
// function is tagged by GOSSAFUNC (for ssa output directed either to stdout or html).
func (s *debugState) logf(msg string, args ...interface{}) {
	if s.f.PrintOrHtmlSSA {
		fmt.Printf(msg, args...)
	}
}

type debugState struct {
	// See FuncDebug.
	slots    []LocalSlot
	vars     []*ir.Name
	varSlots [][]SlotID
	lists    [][]byte

	// The user variable that each slot rolls up to, indexed by SlotID.
	slotVars []VarID

	f              *Func
	loggingEnabled bool
	registers      []Register
	stackOffset    func(LocalSlot) int32
	ctxt           *obj.Link

	// The names (slots) associated with each value, indexed by Value ID.
	valueNames [][]SlotID

	// The current state of whatever analysis is running.
	currentState stateAtPC
	liveCount    []int
	changedVars  *sparseSet

	// The pending location list entry for each user variable, indexed by VarID.
	pendingEntries []pendingEntry

	varParts           map[*ir.Name][]SlotID
	blockDebug         []BlockDebug
	pendingSlotLocs    []VarLoc
	liveSlots          []liveSlot
	liveSlotSliceBegin int
	partsByVarOffset   sort.Interface
}

func (state *debugState) initializeCache(f *Func, numVars, numSlots int) {
	// One blockDebug per block. Initialized in allocBlock.
	if cap(state.blockDebug) < f.NumBlocks() {
		state.blockDebug = make([]BlockDebug, f.NumBlocks())
	} else {
		// This local variable, and the ones like it below, enable compiler
		// optimizations. Don't inline them.
		b := state.blockDebug[:f.NumBlocks()]
		for i := range b {
			b[i] = BlockDebug{}
		}
	}

	// A list of slots per Value. Reuse the previous child slices.
	if cap(state.valueNames) < f.NumValues() {
		old := state.valueNames
		state.valueNames = make([][]SlotID, f.NumValues())
		copy(state.valueNames, old)
	}
	vn := state.valueNames[:f.NumValues()]
	for i := range vn {
		vn[i] = vn[i][:0]
	}

	// Slot and register contents for currentState. Cleared by reset().
	if cap(state.currentState.slots) < numSlots {
		state.currentState.slots = make([]VarLoc, numSlots)
	} else {
		state.currentState.slots = state.currentState.slots[:numSlots]
	}
	if cap(state.currentState.registers) < len(state.registers) {
		state.currentState.registers = make([][]SlotID, len(state.registers))
	} else {
		state.currentState.registers = state.currentState.registers[:len(state.registers)]
	}

	// Used many times by mergePredecessors.
	if cap(state.liveCount) < numSlots {
		state.liveCount = make([]int, numSlots)
	} else {
		state.liveCount = state.liveCount[:numSlots]
	}

	// A relatively small slice, but used many times as the return from processValue.
	state.changedVars = newSparseSet(numVars)

	// A pending entry per user variable, with space to track each of its pieces.
	numPieces := 0
	for i := range state.varSlots {
		numPieces += len(state.varSlots[i])
	}
	if cap(state.pendingSlotLocs) < numPieces {
		state.pendingSlotLocs = make([]VarLoc, numPieces)
	} else {
		psl := state.pendingSlotLocs[:numPieces]
		for i := range psl {
			psl[i] = VarLoc{}
		}
	}
	if cap(state.pendingEntries) < numVars {
		state.pendingEntries = make([]pendingEntry, numVars)
	}
	pe := state.pendingEntries[:numVars]
	freePieceIdx := 0
	for varID, slots := range state.varSlots {
		pe[varID] = pendingEntry{
			pieces: state.pendingSlotLocs[freePieceIdx : freePieceIdx+len(slots)],
		}
		freePieceIdx += len(slots)
	}
	state.pendingEntries = pe

	if cap(state.lists) < numVars {
		state.lists = make([][]byte, numVars)
	} else {
		state.lists = state.lists[:numVars]
		for i := range state.lists {
			state.lists[i] = nil
		}
	}

	state.liveSlots = state.liveSlots[:0]
	state.liveSlotSliceBegin = 0
}

func (state *debugState) allocBlock(b *Block) *BlockDebug {
	return &state.blockDebug[b.ID]
}

func (state *debugState) appendLiveSlot(ls liveSlot) {
	state.liveSlots = append(state.liveSlots, ls)
}

func (state *debugState) getLiveSlotSlice() []liveSlot {
	s := state.liveSlots[state.liveSlotSliceBegin:]
	state.liveSlotSliceBegin = len(state.liveSlots)
	return s
}

func (s *debugState) blockEndStateString(b *BlockDebug) string {
	endState := stateAtPC{slots: make([]VarLoc, len(s.slots)), registers: make([][]SlotID, len(s.registers))}
	endState.reset(b.endState)
	return s.stateString(endState)
}

func (s *debugState) stateString(state stateAtPC) string {
	var strs []string
	for slotID, loc := range state.slots {
		if !loc.absent() {
			strs = append(strs, fmt.Sprintf("\t%v = %v\n", s.slots[slotID], s.LocString(loc)))
		}
	}

	strs = append(strs, "\n")
	for reg, slots := range state.registers {
		if len(slots) != 0 {
			var slotStrs []string
			for _, slot := range slots {
				slotStrs = append(slotStrs, s.slots[slot].String())
			}
			strs = append(strs, fmt.Sprintf("\t%v = %v\n", &s.registers[reg], slotStrs))
		}
	}

	if len(strs) == 1 {
		return "(no vars)\n"
	}
	return strings.Join(strs, "")
}

// slotCanonicalizer is a table used to lookup and canonicalize
// LocalSlot's in a type insensitive way (e.g. taking into account the
// base name, offset, and width of the slot, but ignoring the slot
// type).
type slotCanonicalizer struct {
	slmap  map[slotKey]SlKeyIdx
	slkeys []LocalSlot
}

func newSlotCanonicalizer() *slotCanonicalizer {
	return &slotCanonicalizer{
		slmap:  make(map[slotKey]SlKeyIdx),
		slkeys: []LocalSlot{LocalSlot{N: nil}},
	}
}

type SlKeyIdx uint32

const noSlot = SlKeyIdx(0)

// slotKey is a type-insensitive encapsulation of a LocalSlot; it
// is used to key a map within slotCanonicalizer.
type slotKey struct {
	name        *ir.Name
	offset      int64
	width       int64
	splitOf     SlKeyIdx // idx in slkeys slice in slotCanonicalizer
	splitOffset int64
}

// lookup looks up a LocalSlot in the slot canonicalizer "sc", returning
// a canonical index for the slot, and adding it to the table if need
// be. Return value is the canonical slot index, and a boolean indicating
// whether the slot was found in the table already (TRUE => found).
func (sc *slotCanonicalizer) lookup(ls LocalSlot) (SlKeyIdx, bool) {
	split := noSlot
	if ls.SplitOf != nil {
		split, _ = sc.lookup(*ls.SplitOf)
	}
	k := slotKey{
		name: ls.N, offset: ls.Off, width: ls.Type.Width,
		splitOf: split, splitOffset: ls.SplitOffset,
	}
	if idx, ok := sc.slmap[k]; ok {
		return idx, true
	}
	rv := SlKeyIdx(len(sc.slkeys))
	sc.slkeys = append(sc.slkeys, ls)
	sc.slmap[k] = rv
	return rv, false
}

func (sc *slotCanonicalizer) canonSlot(idx SlKeyIdx) LocalSlot {
	return sc.slkeys[idx]
}

// PopulateABIInRegArgOps examines the entry block of the function
// and looks for incoming parameters that have missing or partial
// OpArg{Int,Float}Reg values, inserting additional values in
// cases where they are missing. Example:
//
//      func foo(s string, used int, notused int) int {
//        return len(s) + used
//      }
//
// In the function above, the incoming parameter "used" is fully live,
// "notused" is not live, and "s" is partially live (only the length
// field of the string is used). At the point where debug value
// analysis runs, we might expect to see an entry block with:
//
//   b1:
//     v4 = ArgIntReg <uintptr> {s+8} [0] : BX
//     v5 = ArgIntReg <int> {used} [0] : CX
//
// While this is an accurate picture of the live incoming params,
// we also want to have debug locations for non-live params (or
// their non-live pieces), e.g. something like
//
//   b1:
//     v9 = ArgIntReg <*uint8> {s+0} [0] : AX
//     v4 = ArgIntReg <uintptr> {s+8} [0] : BX
//     v5 = ArgIntReg <int> {used} [0] : CX
//     v10 = ArgIntReg <int> {unused} [0] : DI
//
// This function examines the live OpArg{Int,Float}Reg values and
// synthesizes new (dead) values for the non-live params or the
// non-live pieces of partially live params.
//
func PopulateABIInRegArgOps(f *Func) {
	pri := f.ABISelf.ABIAnalyzeFuncType(f.Type.FuncType())

	// When manufacturing new slots that correspond to splits of
	// composite parameters, we want to avoid creating a new sub-slot
	// that differs from some existing sub-slot only by type, since
	// the debug location analysis will treat that slot as a separate
	// entity. To achieve this, create a lookup table of existing
	// slots that is type-insenstitive.
	sc := newSlotCanonicalizer()
	for _, sl := range f.Names {
		sc.lookup(*sl)
	}

	// Add slot -> value entry to f.NamedValues if not already present.
	addToNV := func(v *Value, sl LocalSlot) {
		values, ok := f.NamedValues[sl]
		if !ok {
			// Haven't seen this slot yet.
			sla := f.localSlotAddr(sl)
			f.Names = append(f.Names, sla)
		} else {
			for _, ev := range values {
				if v == ev {
					return
				}
			}
		}
		values = append(values, v)
		f.NamedValues[sl] = values
	}

	newValues := []*Value{}

	abiRegIndexToRegister := func(reg abi.RegIndex) int8 {
		i := f.ABISelf.FloatIndexFor(reg)
		if i >= 0 { // float PR
			return f.Config.floatParamRegs[i]
		} else {
			return f.Config.intParamRegs[reg]
		}
	}

	// Helper to construct a new OpArg{Float,Int}Reg op value.
	var pos src.XPos
	if len(f.Entry.Values) != 0 {
		pos = f.Entry.Values[0].Pos
	}
	synthesizeOpIntFloatArg := func(n *ir.Name, t *types.Type, reg abi.RegIndex, sl LocalSlot) *Value {
		aux := &AuxNameOffset{n, sl.Off}
		op, auxInt := ArgOpAndRegisterFor(reg, f.ABISelf)
		v := f.newValueNoBlock(op, t, pos)
		v.AuxInt = auxInt
		v.Aux = aux
		v.Args = nil
		v.Block = f.Entry
		newValues = append(newValues, v)
		addToNV(v, sl)
		f.setHome(v, &f.Config.registers[abiRegIndexToRegister(reg)])
		return v
	}

	// Make a pass through the entry block looking for
	// OpArg{Int,Float}Reg ops. Record the slots they use in a table
	// ("sc"). We use a type-insensitive lookup for the slot table,
	// since the type we get from the ABI analyzer won't always match
	// what the compiler uses when creating OpArg{Int,Float}Reg ops.
	for _, v := range f.Entry.Values {
		if v.Op == OpArgIntReg || v.Op == OpArgFloatReg {
			aux := v.Aux.(*AuxNameOffset)
			sl := LocalSlot{N: aux.Name, Type: v.Type, Off: aux.Offset}
			// install slot in lookup table
			idx, _ := sc.lookup(sl)
			// add to f.NamedValues if not already present
			addToNV(v, sc.canonSlot(idx))
		} else if v.Op.IsCall() {
			// if we hit a call, we've gone too far.
			break
		}
	}

	// Now make a pass through the ABI in-params, looking for params
	// or pieces of params that we didn't encounter in the loop above.
	for _, inp := range pri.InParams() {
		if !isNamedRegParam(inp) {
			continue
		}
		n := inp.Name.(*ir.Name)

		// Param is spread across one or more registers. Walk through
		// each piece to see whether we've seen an arg reg op for it.
		types, offsets := inp.RegisterTypesAndOffsets()
		for k, t := range types {
			// Note: this recipe for creating a LocalSlot is designed
			// to be compatible with the one used in expand_calls.go
			// as opposed to decompose.go. The expand calls code just
			// takes the base name and creates an offset into it,
			// without using the SplitOf/SplitOffset fields. The code
			// in decompose.go does the opposite -- it creates a
			// LocalSlot object with "Off" set to zero, but with
			// SplitOf pointing to a parent slot, and SplitOffset
			// holding the offset into the parent object.
			pieceSlot := LocalSlot{N: n, Type: t, Off: offsets[k]}

			// Look up this piece to see if we've seen a reg op
			// for it. If not, create one.
			_, found := sc.lookup(pieceSlot)
			if !found {
				// This slot doesn't appear in the map, meaning it
				// corresponds to an in-param that is not live, or
				// a portion of an in-param that is not live/used.
				// Add a new dummy OpArg{Int,Float}Reg for it.
				synthesizeOpIntFloatArg(n, t, inp.Registers[k],
					pieceSlot)
			}
		}
	}

	// Insert the new values into the head of the block.
	f.Entry.Values = append(newValues, f.Entry.Values...)
}

// BuildFuncDebug returns debug information for f.
// f must be fully processed, so that each Value is where it will be when
// machine code is emitted.
func BuildFuncDebug(ctxt *obj.Link, f *Func, loggingEnabled bool, stackOffset func(LocalSlot) int32) *FuncDebug {
	if f.RegAlloc == nil {
		f.Fatalf("BuildFuncDebug on func %v that has not been fully processed", f)
	}
	state := &f.Cache.debugState
	state.loggingEnabled = loggingEnabled
	state.f = f
	state.registers = f.Config.registers
	state.stackOffset = stackOffset
	state.ctxt = ctxt

	if buildcfg.Experiment.RegabiArgs {
		PopulateABIInRegArgOps(f)
	}

	if state.loggingEnabled {
		state.logf("Generating location lists for function %q\n", f.Name)
	}

	if state.varParts == nil {
		state.varParts = make(map[*ir.Name][]SlotID)
	} else {
		for n := range state.varParts {
			delete(state.varParts, n)
		}
	}

	// Recompose any decomposed variables, and establish the canonical
	// IDs for each var and slot by filling out state.vars and state.slots.

	state.slots = state.slots[:0]
	state.vars = state.vars[:0]
	for i, slot := range f.Names {
		state.slots = append(state.slots, *slot)
		if ir.IsSynthetic(slot.N) {
			continue
		}

		topSlot := slot
		for topSlot.SplitOf != nil {
			topSlot = topSlot.SplitOf
		}
		if _, ok := state.varParts[topSlot.N]; !ok {
			state.vars = append(state.vars, topSlot.N)
		}
		state.varParts[topSlot.N] = append(state.varParts[topSlot.N], SlotID(i))
	}

	// Recreate the LocalSlot for each stack-only variable.
	// This would probably be better as an output from stackframe.
	for _, b := range f.Blocks {
		for _, v := range b.Values {
			if v.Op == OpVarDef || v.Op == OpVarKill {
				n := v.Aux.(*ir.Name)
				if ir.IsSynthetic(n) {
					continue
				}

				if _, ok := state.varParts[n]; !ok {
					slot := LocalSlot{N: n, Type: v.Type, Off: 0}
					state.slots = append(state.slots, slot)
					state.varParts[n] = []SlotID{SlotID(len(state.slots) - 1)}
					state.vars = append(state.vars, n)
				}
			}
		}
	}

	// Fill in the var<->slot mappings.
	if cap(state.varSlots) < len(state.vars) {
		state.varSlots = make([][]SlotID, len(state.vars))
	} else {
		state.varSlots = state.varSlots[:len(state.vars)]
		for i := range state.varSlots {
			state.varSlots[i] = state.varSlots[i][:0]
		}
	}
	if cap(state.slotVars) < len(state.slots) {
		state.slotVars = make([]VarID, len(state.slots))
	} else {
		state.slotVars = state.slotVars[:len(state.slots)]
	}

	if state.partsByVarOffset == nil {
		state.partsByVarOffset = &partsByVarOffset{}
	}
	for varID, n := range state.vars {
		parts := state.varParts[n]
		state.varSlots[varID] = parts
		for _, slotID := range parts {
			state.slotVars[slotID] = VarID(varID)
		}
		*state.partsByVarOffset.(*partsByVarOffset) = partsByVarOffset{parts, state.slots}
		sort.Sort(state.partsByVarOffset)
	}

	state.initializeCache(f, len(state.varParts), len(state.slots))

	for i, slot := range f.Names {
		if ir.IsSynthetic(slot.N) {
			continue
		}
		for _, value := range f.NamedValues[*slot] {
			state.valueNames[value.ID] = append(state.valueNames[value.ID], SlotID(i))
		}
	}

	blockLocs := state.liveness()
	state.buildLocationLists(blockLocs)

	return &FuncDebug{
		Slots:         state.slots,
		VarSlots:      state.varSlots,
		Vars:          state.vars,
		LocationLists: state.lists,
	}
}

// liveness walks the function in control flow order, calculating the start
// and end state of each block.
func (state *debugState) liveness() []*BlockDebug {
	blockLocs := make([]*BlockDebug, state.f.NumBlocks())

	// Reverse postorder: visit a block after as many as possible of its
	// predecessors have been visited.
	po := state.f.Postorder()
	for i := len(po) - 1; i >= 0; i-- {
		b := po[i]

		// Build the starting state for the block from the final
		// state of its predecessors.
		startState, startValid := state.mergePredecessors(b, blockLocs, nil)
		changed := false
		if state.loggingEnabled {
			state.logf("Processing %v, initial state:\n%v", b, state.stateString(state.currentState))
		}

		// Update locs/registers with the effects of each Value.
		for _, v := range b.Values {
			slots := state.valueNames[v.ID]

			// Loads and stores inherit the names of their sources.
			var source *Value
			switch v.Op {
			case OpStoreReg:
				source = v.Args[0]
			case OpLoadReg:
				switch a := v.Args[0]; a.Op {
				case OpArg, OpPhi:
					source = a
				case OpStoreReg:
					source = a.Args[0]
				default:
					if state.loggingEnabled {
						state.logf("at %v: load with unexpected source op: %v (%v)\n", v, a.Op, a)
					}
				}
			}
			// Update valueNames with the source so that later steps
			// don't need special handling.
			if source != nil {
				slots = append(slots, state.valueNames[source.ID]...)
				state.valueNames[v.ID] = slots
			}

			reg, _ := state.f.getHome(v.ID).(*Register)
			c := state.processValue(v, slots, reg)
			changed = changed || c
		}

		if state.loggingEnabled {
			state.f.Logf("Block %v done, locs:\n%v", b, state.stateString(state.currentState))
		}

		locs := state.allocBlock(b)
		locs.relevant = changed
		if !changed && startValid {
			locs.endState = startState
		} else {
			for slotID, slotLoc := range state.currentState.slots {
				if slotLoc.absent() {
					continue
				}
				state.appendLiveSlot(liveSlot{slot: SlotID(slotID), Registers: slotLoc.Registers, StackOffset: slotLoc.StackOffset})
			}
			locs.endState = state.getLiveSlotSlice()
		}
		blockLocs[b.ID] = locs
	}
	return blockLocs
}

// mergePredecessors takes the end state of each of b's predecessors and
// intersects them to form the starting state for b. It puts that state in
// blockLocs, and fills state.currentState with it. If convenient, it returns
// a reused []liveSlot, true that represents the starting state.
// If previousBlock is non-nil, it registers changes vs. that block's end
// state in state.changedVars. Note that previousBlock will often not be a
// predecessor.
func (state *debugState) mergePredecessors(b *Block, blockLocs []*BlockDebug, previousBlock *Block) ([]liveSlot, bool) {
	// Filter out back branches.
	var predsBuf [10]*Block
	preds := predsBuf[:0]
	for _, pred := range b.Preds {
		if blockLocs[pred.b.ID] != nil {
			preds = append(preds, pred.b)
		}
	}

	if state.loggingEnabled {
		// The logf below would cause preds to be heap-allocated if
		// it were passed directly.
		preds2 := make([]*Block, len(preds))
		copy(preds2, preds)
		state.logf("Merging %v into %v\n", preds2, b)
	}

	// TODO all the calls to this are overkill; only need to do this for slots that are not present in the merge.
	markChangedVars := func(slots []liveSlot) {
		for _, live := range slots {
			state.changedVars.add(ID(state.slotVars[live.slot]))
		}
	}

	if len(preds) == 0 {
		if previousBlock != nil {
			// Mark everything in previous block as changed because it is not a predecessor.
			markChangedVars(blockLocs[previousBlock.ID].endState)
		}
		state.currentState.reset(nil)
		return nil, true
	}

	p0 := blockLocs[preds[0].ID].endState
	if len(preds) == 1 {
		if previousBlock != nil && preds[0].ID != previousBlock.ID {
			// Mark everything in previous block as changed because it is not a predecessor.
			markChangedVars(blockLocs[previousBlock.ID].endState)
		}
		state.currentState.reset(p0)
		return p0, true
	}

	baseID := preds[0].ID
	baseState := p0

	// If previous block is not a predecessor, its location information changes at boundary with this block.
	previousBlockIsNotPredecessor := previousBlock != nil // If it's nil, no info to change.

	if previousBlock != nil {
		// Try to use previousBlock as the base state
		// if possible.
		for _, pred := range preds[1:] {
			if pred.ID == previousBlock.ID {
				baseID = pred.ID
				baseState = blockLocs[pred.ID].endState
				previousBlockIsNotPredecessor = false
				break
			}
		}
	}

	if state.loggingEnabled {
		state.logf("Starting %v with state from b%v:\n%v", b, baseID, state.blockEndStateString(blockLocs[baseID]))
	}

	slotLocs := state.currentState.slots
	for _, predSlot := range baseState {
		slotLocs[predSlot.slot] = VarLoc{predSlot.Registers, predSlot.StackOffset}
		state.liveCount[predSlot.slot] = 1
	}
	for _, pred := range preds {
		if pred.ID == baseID {
			continue
		}
		if state.loggingEnabled {
			state.logf("Merging in state from %v:\n%v", pred, state.blockEndStateString(blockLocs[pred.ID]))
		}
		for _, predSlot := range blockLocs[pred.ID].endState {
			state.liveCount[predSlot.slot]++
			liveLoc := slotLocs[predSlot.slot]
			if !liveLoc.onStack() || !predSlot.onStack() || liveLoc.StackOffset != predSlot.StackOffset {
				liveLoc.StackOffset = 0
			}
			liveLoc.Registers &= predSlot.Registers
			slotLocs[predSlot.slot] = liveLoc
		}
	}

	// Check if the final state is the same as the first predecessor's
	// final state, and reuse it if so. In principle it could match any,
	// but it's probably not worth checking more than the first.
	unchanged := true
	for _, predSlot := range baseState {
		if state.liveCount[predSlot.slot] != len(preds) ||
			slotLocs[predSlot.slot].Registers != predSlot.Registers ||
			slotLocs[predSlot.slot].StackOffset != predSlot.StackOffset {
			unchanged = false
			break
		}
	}
	if unchanged {
		if state.loggingEnabled {
			state.logf("After merge, %v matches b%v exactly.\n", b, baseID)
		}
		if previousBlockIsNotPredecessor {
			// Mark everything in previous block as changed because it is not a predecessor.
			markChangedVars(blockLocs[previousBlock.ID].endState)
		}
		state.currentState.reset(baseState)
		return baseState, true
	}

	for reg := range state.currentState.registers {
		state.currentState.registers[reg] = state.currentState.registers[reg][:0]
	}

	// A slot is live if it was seen in all predecessors, and they all had
	// some storage in common.
	for _, predSlot := range baseState {
		slotLoc := slotLocs[predSlot.slot]

		if state.liveCount[predSlot.slot] != len(preds) {
			// Seen in only some predecessors. Clear it out.
			slotLocs[predSlot.slot] = VarLoc{}
			continue
		}

		// Present in all predecessors.
		mask := uint64(slotLoc.Registers)
		for {
			if mask == 0 {
				break
			}
			reg := uint8(bits.TrailingZeros64(mask))
			mask &^= 1 << reg
			state.currentState.registers[reg] = append(state.currentState.registers[reg], predSlot.slot)
		}
	}

	if previousBlockIsNotPredecessor {
		// Mark everything in previous block as changed because it is not a predecessor.
		markChangedVars(blockLocs[previousBlock.ID].endState)

	}
	return nil, false
}

// processValue updates locs and state.registerContents to reflect v, a value with
// the names in vSlots and homed in vReg.  "v" becomes visible after execution of
// the instructions evaluating it. It returns which VarIDs were modified by the
// Value's execution.
func (state *debugState) processValue(v *Value, vSlots []SlotID, vReg *Register) bool {
	locs := state.currentState
	changed := false
	setSlot := func(slot SlotID, loc VarLoc) {
		changed = true
		state.changedVars.add(ID(state.slotVars[slot]))
		state.currentState.slots[slot] = loc
	}

	// Handle any register clobbering. Call operations, for example,
	// clobber all registers even though they don't explicitly write to
	// them.
	clobbers := uint64(opcodeTable[v.Op].reg.clobbers)
	for {
		if clobbers == 0 {
			break
		}
		reg := uint8(bits.TrailingZeros64(clobbers))
		clobbers &^= 1 << reg

		for _, slot := range locs.registers[reg] {
			if state.loggingEnabled {
				state.logf("at %v: %v clobbered out of %v\n", v, state.slots[slot], &state.registers[reg])
			}

			last := locs.slots[slot]
			if last.absent() {
				state.f.Fatalf("at %v: slot %v in register %v with no location entry", v, state.slots[slot], &state.registers[reg])
				continue
			}
			regs := last.Registers &^ (1 << reg)
			setSlot(slot, VarLoc{regs, last.StackOffset})
		}

		locs.registers[reg] = locs.registers[reg][:0]
	}

	switch {
	case v.Op == OpVarDef, v.Op == OpVarKill:
		n := v.Aux.(*ir.Name)
		if ir.IsSynthetic(n) {
			break
		}

		slotID := state.varParts[n][0]
		var stackOffset StackOffset
		if v.Op == OpVarDef {
			stackOffset = StackOffset(state.stackOffset(state.slots[slotID])<<1 | 1)
		}
		setSlot(slotID, VarLoc{0, stackOffset})
		if state.loggingEnabled {
			if v.Op == OpVarDef {
				state.logf("at %v: stack-only var %v now live\n", v, state.slots[slotID])
			} else {
				state.logf("at %v: stack-only var %v now dead\n", v, state.slots[slotID])
			}
		}

	case v.Op == OpArg:
		home := state.f.getHome(v.ID).(LocalSlot)
		stackOffset := state.stackOffset(home)<<1 | 1
		for _, slot := range vSlots {
			if state.loggingEnabled {
				state.logf("at %v: arg %v now on stack in location %v\n", v, state.slots[slot], home)
				if last := locs.slots[slot]; !last.absent() {
					state.logf("at %v: unexpected arg op on already-live slot %v\n", v, state.slots[slot])
				}
			}

			setSlot(slot, VarLoc{0, StackOffset(stackOffset)})
		}

	case v.Op == OpStoreReg:
		home := state.f.getHome(v.ID).(LocalSlot)
		stackOffset := state.stackOffset(home)<<1 | 1
		for _, slot := range vSlots {
			last := locs.slots[slot]
			if last.absent() {
				if state.loggingEnabled {
					state.logf("at %v: unexpected spill of unnamed register %s\n", v, vReg)
				}
				break
			}

			setSlot(slot, VarLoc{last.Registers, StackOffset(stackOffset)})
			if state.loggingEnabled {
				state.logf("at %v: %v spilled to stack location %v\n", v, state.slots[slot], home)
			}
		}

	case vReg != nil:
		if state.loggingEnabled {
			newSlots := make([]bool, len(state.slots))
			for _, slot := range vSlots {
				newSlots[slot] = true
			}

			for _, slot := range locs.registers[vReg.num] {
				if !newSlots[slot] {
					state.logf("at %v: overwrote %v in register %v\n", v, state.slots[slot], vReg)
				}
			}
		}

		for _, slot := range locs.registers[vReg.num] {
			last := locs.slots[slot]
			setSlot(slot, VarLoc{last.Registers &^ (1 << uint8(vReg.num)), last.StackOffset})
		}
		locs.registers[vReg.num] = locs.registers[vReg.num][:0]
		locs.registers[vReg.num] = append(locs.registers[vReg.num], vSlots...)
		for _, slot := range vSlots {
			if state.loggingEnabled {
				state.logf("at %v: %v now in %s\n", v, state.slots[slot], vReg)
			}

			last := locs.slots[slot]
			setSlot(slot, VarLoc{1<<uint8(vReg.num) | last.Registers, last.StackOffset})
		}
	}
	return changed
}

// varOffset returns the offset of slot within the user variable it was
// decomposed from. This has nothing to do with its stack offset.
func varOffset(slot LocalSlot) int64 {
	offset := slot.Off
	s := &slot
	for ; s.SplitOf != nil; s = s.SplitOf {
		offset += s.SplitOffset
	}
	return offset
}

type partsByVarOffset struct {
	slotIDs []SlotID
	slots   []LocalSlot
}

func (a partsByVarOffset) Len() int { return len(a.slotIDs) }
func (a partsByVarOffset) Less(i, j int) bool {
	return varOffset(a.slots[a.slotIDs[i]]) < varOffset(a.slots[a.slotIDs[j]])
}
func (a partsByVarOffset) Swap(i, j int) { a.slotIDs[i], a.slotIDs[j] = a.slotIDs[j], a.slotIDs[i] }

// A pendingEntry represents the beginning of a location list entry, missing
// only its end coordinate.
type pendingEntry struct {
	present                bool
	startBlock, startValue ID
	// The location of each piece of the variable, in the same order as the
	// SlotIDs in varParts.
	pieces []VarLoc
}

func (e *pendingEntry) clear() {
	e.present = false
	e.startBlock = 0
	e.startValue = 0
	for i := range e.pieces {
		e.pieces[i] = VarLoc{}
	}
}

// canMerge reports whether the location description for new is the same as
// pending.
func canMerge(pending, new VarLoc) bool {
	if pending.absent() && new.absent() {
		return true
	}
	if pending.absent() || new.absent() {
		return false
	}
	if pending.onStack() {
		return pending.StackOffset == new.StackOffset
	}
	if pending.Registers != 0 && new.Registers != 0 {
		return firstReg(pending.Registers) == firstReg(new.Registers)
	}
	return false
}

// firstReg returns the first register in set that is present.
func firstReg(set RegisterSet) uint8 {
	if set == 0 {
		// This is wrong, but there seem to be some situations where we
		// produce locations with no storage.
		return 0
	}
	return uint8(bits.TrailingZeros64(uint64(set)))
}

// buildLocationLists builds location lists for all the user variables in
// state.f, using the information about block state in blockLocs.
// The returned location lists are not fully complete. They are in terms of
// SSA values rather than PCs, and have no base address/end entries. They will
// be finished by PutLocationList.
func (state *debugState) buildLocationLists(blockLocs []*BlockDebug) {
	// Run through the function in program text order, building up location
	// lists as we go. The heavy lifting has mostly already been done.

	var prevBlock *Block
	for _, b := range state.f.Blocks {
		state.mergePredecessors(b, blockLocs, prevBlock)

		if !blockLocs[b.ID].relevant {
			// Handle any differences among predecessor blocks and previous block (perhaps not a predecessor)
			for _, varID := range state.changedVars.contents() {
				state.updateVar(VarID(varID), b, BlockStart)
			}
			continue
		}

		mustBeFirst := func(v *Value) bool {
			return v.Op == OpPhi || v.Op.isLoweredGetClosurePtr() ||
				v.Op == OpArgIntReg || v.Op == OpArgFloatReg
		}

		zeroWidthPending := false
		blockPrologComplete := false // set to true at first non-zero-width op
		apcChangedSize := 0          // size of changedVars for leading Args, Phi, ClosurePtr
		// expect to see values in pattern (apc)* (zerowidth|real)*
		for _, v := range b.Values {
			slots := state.valueNames[v.ID]
			reg, _ := state.f.getHome(v.ID).(*Register)
			changed := state.processValue(v, slots, reg) // changed == added to state.changedVars

			if opcodeTable[v.Op].zeroWidth {
				if changed {
					if mustBeFirst(v) || v.Op == OpArg {
						// These ranges begin at true beginning of block, not after first instruction
						if blockPrologComplete && mustBeFirst(v) {
							panic(fmt.Errorf("Unexpected placement of op '%s' appearing after non-pseudo-op at beginning of block %s in %s\n%s", v.LongString(), b, b.Func.Name, b.Func))
						}
						apcChangedSize = len(state.changedVars.contents())
						// Other zero-width ops must wait on a "real" op.
						zeroWidthPending = true
						continue
					}
				}
				continue
			}

			if !changed && !zeroWidthPending {
				continue
			}
			// Not zero-width; i.e., a "real" instruction.

			zeroWidthPending = false
			blockPrologComplete = true
			for i, varID := range state.changedVars.contents() {
				if i < apcChangedSize { // buffered true start-of-block changes
					state.updateVar(VarID(varID), v.Block, BlockStart)
				} else {
					state.updateVar(VarID(varID), v.Block, v)
				}
			}
			state.changedVars.clear()
			apcChangedSize = 0
		}
		for i, varID := range state.changedVars.contents() {
			if i < apcChangedSize { // buffered true start-of-block changes
				state.updateVar(VarID(varID), b, BlockStart)
			} else {
				state.updateVar(VarID(varID), b, BlockEnd)
			}
		}

		prevBlock = b
	}

	if state.loggingEnabled {
		state.logf("location lists:\n")
	}

	// Flush any leftover entries live at the end of the last block.
	for varID := range state.lists {
		state.writePendingEntry(VarID(varID), state.f.Blocks[len(state.f.Blocks)-1].ID, FuncEnd.ID)
		list := state.lists[varID]
		if state.loggingEnabled {
			if len(list) == 0 {
				state.logf("\t%v : empty list\n", state.vars[varID])
			} else {
				state.logf("\t%v : %q\n", state.vars[varID], hex.EncodeToString(state.lists[varID]))
			}
		}
	}
}

// updateVar updates the pending location list entry for varID to
// reflect the new locations in curLoc, beginning at v in block b.
// v may be one of the special values indicating block start or end.
func (state *debugState) updateVar(varID VarID, b *Block, v *Value) {
	curLoc := state.currentState.slots
	// Assemble the location list entry with whatever's live.
	empty := true
	for _, slotID := range state.varSlots[varID] {
		if !curLoc[slotID].absent() {
			empty = false
			break
		}
	}
	pending := &state.pendingEntries[varID]
	if empty {
		state.writePendingEntry(varID, b.ID, v.ID)
		pending.clear()
		return
	}

	// Extend the previous entry if possible.
	if pending.present {
		merge := true
		for i, slotID := range state.varSlots[varID] {
			if !canMerge(pending.pieces[i], curLoc[slotID]) {
				merge = false
				break
			}
		}
		if merge {
			return
		}
	}

	state.writePendingEntry(varID, b.ID, v.ID)
	pending.present = true
	pending.startBlock = b.ID
	pending.startValue = v.ID
	for i, slot := range state.varSlots[varID] {
		pending.pieces[i] = curLoc[slot]
	}
}

// writePendingEntry writes out the pending entry for varID, if any,
// terminated at endBlock/Value.
func (state *debugState) writePendingEntry(varID VarID, endBlock, endValue ID) {
	pending := state.pendingEntries[varID]
	if !pending.present {
		return
	}

	// Pack the start/end coordinates into the start/end addresses
	// of the entry, for decoding by PutLocationList.
	start, startOK := encodeValue(state.ctxt, pending.startBlock, pending.startValue)
	end, endOK := encodeValue(state.ctxt, endBlock, endValue)
	if !startOK || !endOK {
		// If someone writes a function that uses >65K values,
		// they get incomplete debug info on 32-bit platforms.
		return
	}
	if start == end {
		if state.loggingEnabled {
			// Printf not logf so not gated by GOSSAFUNC; this should fire very rarely.
			fmt.Printf("Skipping empty location list for %v in %s\n", state.vars[varID], state.f.Name)
		}
		return
	}

	list := state.lists[varID]
	list = appendPtr(state.ctxt, list, start)
	list = appendPtr(state.ctxt, list, end)
	// Where to write the length of the location description once
	// we know how big it is.
	sizeIdx := len(list)
	list = list[:len(list)+2]

	if state.loggingEnabled {
		var partStrs []string
		for i, slot := range state.varSlots[varID] {
			partStrs = append(partStrs, fmt.Sprintf("%v@%v", state.slots[slot], state.LocString(pending.pieces[i])))
		}
		state.logf("Add entry for %v: \tb%vv%v-b%vv%v = \t%v\n", state.vars[varID], pending.startBlock, pending.startValue, endBlock, endValue, strings.Join(partStrs, " "))
	}

	for i, slotID := range state.varSlots[varID] {
		loc := pending.pieces[i]
		slot := state.slots[slotID]

		if !loc.absent() {
			if loc.onStack() {
				if loc.stackOffsetValue() == 0 {
					list = append(list, dwarf.DW_OP_call_frame_cfa)
				} else {
					list = append(list, dwarf.DW_OP_fbreg)
					list = dwarf.AppendSleb128(list, int64(loc.stackOffsetValue()))
				}
			} else {
				regnum := state.ctxt.Arch.DWARFRegisters[state.registers[firstReg(loc.Registers)].ObjNum()]
				if regnum < 32 {
					list = append(list, dwarf.DW_OP_reg0+byte(regnum))
				} else {
					list = append(list, dwarf.DW_OP_regx)
					list = dwarf.AppendUleb128(list, uint64(regnum))
				}
			}
		}

		if len(state.varSlots[varID]) > 1 {
			list = append(list, dwarf.DW_OP_piece)
			list = dwarf.AppendUleb128(list, uint64(slot.Type.Size()))
		}
	}
	state.ctxt.Arch.ByteOrder.PutUint16(list[sizeIdx:], uint16(len(list)-sizeIdx-2))
	state.lists[varID] = list
}

// PutLocationList adds list (a location list in its intermediate representation) to listSym.
func (debugInfo *FuncDebug) PutLocationList(list []byte, ctxt *obj.Link, listSym, startPC *obj.LSym) {
	getPC := debugInfo.GetPC

	if ctxt.UseBASEntries {
		listSym.WriteInt(ctxt, listSym.Size, ctxt.Arch.PtrSize, ^0)
		listSym.WriteAddr(ctxt, listSym.Size, ctxt.Arch.PtrSize, startPC, 0)
	}

	// Re-read list, translating its address from block/value ID to PC.
	for i := 0; i < len(list); {
		begin := getPC(decodeValue(ctxt, readPtr(ctxt, list[i:])))
		end := getPC(decodeValue(ctxt, readPtr(ctxt, list[i+ctxt.Arch.PtrSize:])))

		// Horrible hack. If a range contains only zero-width
		// instructions, e.g. an Arg, and it's at the beginning of the
		// function, this would be indistinguishable from an
		// end entry. Fudge it.
		if begin == 0 && end == 0 {
			end = 1
		}

		if ctxt.UseBASEntries {
			listSym.WriteInt(ctxt, listSym.Size, ctxt.Arch.PtrSize, int64(begin))
			listSym.WriteInt(ctxt, listSym.Size, ctxt.Arch.PtrSize, int64(end))
		} else {
			listSym.WriteCURelativeAddr(ctxt, listSym.Size, startPC, int64(begin))
			listSym.WriteCURelativeAddr(ctxt, listSym.Size, startPC, int64(end))
		}

		i += 2 * ctxt.Arch.PtrSize
		datalen := 2 + int(ctxt.Arch.ByteOrder.Uint16(list[i:]))
		listSym.WriteBytes(ctxt, listSym.Size, list[i:i+datalen]) // copy datalen and location encoding
		i += datalen
	}

	// Location list contents, now with real PCs.
	// End entry.
	listSym.WriteInt(ctxt, listSym.Size, ctxt.Arch.PtrSize, 0)
	listSym.WriteInt(ctxt, listSym.Size, ctxt.Arch.PtrSize, 0)
}

// Pack a value and block ID into an address-sized uint, returning encoded
// value and boolean indicating whether the encoding succeeded.  For
// 32-bit architectures the process may fail for very large procedures
// (the theory being that it's ok to have degraded debug quality in
// this case).
func encodeValue(ctxt *obj.Link, b, v ID) (uint64, bool) {
	if ctxt.Arch.PtrSize == 8 {
		result := uint64(b)<<32 | uint64(uint32(v))
		//ctxt.Logf("b %#x (%d) v %#x (%d) -> %#x\n", b, b, v, v, result)
		return result, true
	}
	if ctxt.Arch.PtrSize != 4 {
		panic("unexpected pointer size")
	}
	if ID(int16(b)) != b || ID(int16(v)) != v {
		return 0, false
	}
	return uint64(b)<<16 | uint64(uint16(v)), true
}

// Unpack a value and block ID encoded by encodeValue.
func decodeValue(ctxt *obj.Link, word uint64) (ID, ID) {
	if ctxt.Arch.PtrSize == 8 {
		b, v := ID(word>>32), ID(word)
		//ctxt.Logf("%#x -> b %#x (%d) v %#x (%d)\n", word, b, b, v, v)
		return b, v
	}
	if ctxt.Arch.PtrSize != 4 {
		panic("unexpected pointer size")
	}
	return ID(word >> 16), ID(int16(word))
}

// Append a pointer-sized uint to buf.
func appendPtr(ctxt *obj.Link, buf []byte, word uint64) []byte {
	if cap(buf) < len(buf)+20 {
		b := make([]byte, len(buf), 20+cap(buf)*2)
		copy(b, buf)
		buf = b
	}
	writeAt := len(buf)
	buf = buf[0 : len(buf)+ctxt.Arch.PtrSize]
	writePtr(ctxt, buf[writeAt:], word)
	return buf
}

// Write a pointer-sized uint to the beginning of buf.
func writePtr(ctxt *obj.Link, buf []byte, word uint64) {
	switch ctxt.Arch.PtrSize {
	case 4:
		ctxt.Arch.ByteOrder.PutUint32(buf, uint32(word))
	case 8:
		ctxt.Arch.ByteOrder.PutUint64(buf, word)
	default:
		panic("unexpected pointer size")
	}

}

// Read a pointer-sized uint from the beginning of buf.
func readPtr(ctxt *obj.Link, buf []byte) uint64 {
	switch ctxt.Arch.PtrSize {
	case 4:
		return uint64(ctxt.Arch.ByteOrder.Uint32(buf))
	case 8:
		return ctxt.Arch.ByteOrder.Uint64(buf)
	default:
		panic("unexpected pointer size")
	}

}

// setupLocList creates the initial portion of a location list for a
// user variable. It emits the encoded start/end of the range and a
// placeholder for the size. Return value is the new list plus the
// slot in the list holding the size (to be updated later).
func setupLocList(ctxt *obj.Link, f *Func, list []byte, st, en ID) ([]byte, int) {
	start, startOK := encodeValue(ctxt, f.Entry.ID, st)
	end, endOK := encodeValue(ctxt, f.Entry.ID, en)
	if !startOK || !endOK {
		// This could happen if someone writes a function that uses
		// >65K values on a 32-bit platform. Hopefully a degraded debugging
		// experience is ok in that case.
		return nil, 0
	}
	list = appendPtr(ctxt, list, start)
	list = appendPtr(ctxt, list, end)

	// Where to write the length of the location description once
	// we know how big it is.
	sizeIdx := len(list)
	list = list[:len(list)+2]
	return list, sizeIdx
}

// locatePrologEnd walks the entry block of a function with incoming
// register arguments and locates the last instruction in the prolog
// that spills a register arg. It returns the ID of that instruction
// Example:
//
//   b1:
//       v3 = ArgIntReg <int> {p1+0} [0] : AX
//       ... more arg regs ..
//       v4 = ArgFloatReg <float32> {f1+0} [0] : X0
//       v52 = MOVQstore <mem> {p1} v2 v3 v1
//       ... more stores ...
//       v68 = MOVSSstore <mem> {f4} v2 v67 v66
//       v38 = MOVQstoreconst <mem> {blob} [val=0,off=0] v2 v32
//
// Important: locatePrologEnd is expected to work properly only with
// optimization turned off (e.g. "-N"). If optimization is enabled
// we can't be assured of finding all input arguments spilled in the
// entry block prolog.
func locatePrologEnd(f *Func) ID {

	// returns true if this instruction looks like it moves an ABI
	// register to the stack, along with the value being stored.
	isRegMoveLike := func(v *Value) (bool, ID) {
		n, ok := v.Aux.(*ir.Name)
		var r ID
		if !ok || n.Class != ir.PPARAM {
			return false, r
		}
		regInputs, memInputs, spInputs := 0, 0, 0
		for _, a := range v.Args {
			if a.Op == OpArgIntReg || a.Op == OpArgFloatReg {
				regInputs++
				r = a.ID
			} else if a.Type.IsMemory() {
				memInputs++
			} else if a.Op == OpSP {
				spInputs++
			} else {
				return false, r
			}
		}
		return v.Type.IsMemory() && memInputs == 1 &&
			regInputs == 1 && spInputs == 1, r
	}

	// OpArg*Reg values we've seen so far on our forward walk,
	// for which we have not yet seen a corresponding spill.
	regArgs := make([]ID, 0, 32)

	// removeReg tries to remove a value from regArgs, returning true
	// if found and removed, or false otherwise.
	removeReg := func(r ID) bool {
		for i := 0; i < len(regArgs); i++ {
			if regArgs[i] == r {
				regArgs = append(regArgs[:i], regArgs[i+1:]...)
				return true
			}
		}
		return false
	}

	// Walk forwards through the block. When we see OpArg*Reg, record
	// the value it produces in the regArgs list. When see a store that uses
	// the value, remove the entry. When we hit the last store (use)
	// then we've arrived at the end of the prolog.
	for k, v := range f.Entry.Values {
		if v.Op == OpArgIntReg || v.Op == OpArgFloatReg {
			regArgs = append(regArgs, v.ID)
			continue
		}
		if ok, r := isRegMoveLike(v); ok {
			if removed := removeReg(r); removed {
				if len(regArgs) == 0 {
					// Found our last spill; return the value after
					// it. Note that it is possible that this spill is
					// the last instruction in the block. If so, then
					// return the "end of block" sentinel.
					if k < len(f.Entry.Values)-1 {
						return f.Entry.Values[k+1].ID
					}
					return BlockEnd.ID
				}
			}
		}
		if v.Op.IsCall() {
			// if we hit a call, we've gone too far.
			return v.ID
		}
	}
	// nothing found
	return ID(-1)
}

// isNamedRegParam returns true if the param corresponding to "p"
// is a named, non-blank input parameter assigned to one or more
// registers.
func isNamedRegParam(p abi.ABIParamAssignment) bool {
	if p.Name == nil {
		return false
	}
	n := p.Name.(*ir.Name)
	if n.Sym() == nil || n.Sym().IsBlank() {
		return false
	}
	if len(p.Registers) == 0 {
		return false
	}
	return true
}

// BuildFuncDebugNoOptimized constructs a FuncDebug object with
// entries corresponding to the register-resident input parameters for
// the function "f"; it is used when we are compiling without
// optimization but the register ABI is enabled. For each reg param,
// it constructs a 2-element location list: the first element holds
// the input register, and the second element holds the stack location
// of the param (the assumption being that when optimization is off,
// each input param reg will be spilled in the prolog.
func BuildFuncDebugNoOptimized(ctxt *obj.Link, f *Func, loggingEnabled bool, stackOffset func(LocalSlot) int32) *FuncDebug {
	fd := FuncDebug{}

	pri := f.ABISelf.ABIAnalyzeFuncType(f.Type.FuncType())

	// Look to see if we have any named register-promoted parameters.
	// If there are none, bail early and let the caller sort things
	// out for the remainder of the params/locals.
	numRegParams := 0
	for _, inp := range pri.InParams() {
		if isNamedRegParam(inp) {
			numRegParams++
		}
	}
	if numRegParams == 0 {
		return &fd
	}

	state := debugState{f: f}

	if loggingEnabled {
		state.logf("generating -N reg param loc lists for func %q\n", f.Name)
	}

	// Allocate location lists.
	fd.LocationLists = make([][]byte, numRegParams)

	// Locate the value corresponding to the last spill of
	// an input register.
	afterPrologVal := locatePrologEnd(f)

	// Walk the input params again and process the register-resident elements.
	pidx := 0
	for _, inp := range pri.InParams() {
		if !isNamedRegParam(inp) {
			// will be sorted out elsewhere
			continue
		}

		n := inp.Name.(*ir.Name)
		sl := LocalSlot{N: n, Type: inp.Type, Off: 0}
		fd.Vars = append(fd.Vars, n)
		fd.Slots = append(fd.Slots, sl)
		slid := len(fd.VarSlots)
		fd.VarSlots = append(fd.VarSlots, []SlotID{SlotID(slid)})

		if afterPrologVal == ID(-1) {
			// This can happen for degenerate functions with infinite
			// loops such as that in issue 45948. In such cases, leave
			// the var/slot set up for the param, but don't try to
			// emit a location list.
			if loggingEnabled {
				state.logf("locatePrologEnd failed, skipping %v\n", n)
			}
			pidx++
			continue
		}

		// Param is arriving in one or more registers. We need a 2-element
		// location expression for it. First entry in location list
		// will correspond to lifetime in input registers.
		list, sizeIdx := setupLocList(ctxt, f, fd.LocationLists[pidx],
			BlockStart.ID, afterPrologVal)
		if list == nil {
			pidx++
			continue
		}
		if loggingEnabled {
			state.logf("param %v:\n  [<entry>, %d]:\n", n, afterPrologVal)
		}
		rtypes, _ := inp.RegisterTypesAndOffsets()
		padding := make([]uint64, 0, 32)
		padding = inp.ComputePadding(padding)
		for k, r := range inp.Registers {
			reg := ObjRegForAbiReg(r, f.Config)
			dwreg := ctxt.Arch.DWARFRegisters[reg]
			if dwreg < 32 {
				list = append(list, dwarf.DW_OP_reg0+byte(dwreg))
			} else {
				list = append(list, dwarf.DW_OP_regx)
				list = dwarf.AppendUleb128(list, uint64(dwreg))
			}
			if loggingEnabled {
				state.logf("    piece %d -> dwreg %d", k, dwreg)
			}
			if len(inp.Registers) > 1 {
				list = append(list, dwarf.DW_OP_piece)
				ts := rtypes[k].Width
				list = dwarf.AppendUleb128(list, uint64(ts))
				if padding[k] > 0 {
					if loggingEnabled {
						state.logf(" [pad %d bytes]", padding[k])
					}
					list = append(list, dwarf.DW_OP_piece)
					list = dwarf.AppendUleb128(list, padding[k])
				}
			}
			if loggingEnabled {
				state.logf("\n")
			}
		}
		// fill in length of location expression element
		ctxt.Arch.ByteOrder.PutUint16(list[sizeIdx:], uint16(len(list)-sizeIdx-2))

		// Second entry in the location list will be the stack home
		// of the param, once it has been spilled.  Emit that now.
		list, sizeIdx = setupLocList(ctxt, f, list,
			afterPrologVal, FuncEnd.ID)
		if list == nil {
			pidx++
			continue
		}
		soff := stackOffset(sl)
		if soff == 0 {
			list = append(list, dwarf.DW_OP_call_frame_cfa)
		} else {
			list = append(list, dwarf.DW_OP_fbreg)
			list = dwarf.AppendSleb128(list, int64(soff))
		}
		if loggingEnabled {
			state.logf("  [%d, <end>): stackOffset=%d\n", afterPrologVal, soff)
		}

		// fill in size
		ctxt.Arch.ByteOrder.PutUint16(list[sizeIdx:], uint16(len(list)-sizeIdx-2))

		fd.LocationLists[pidx] = list
		pidx++
	}
	return &fd
}

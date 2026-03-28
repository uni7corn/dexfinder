package dex

import (
	"testing"
)

func TestOpcodeFormat(t *testing.T) {
	tests := []struct {
		op   Opcode
		fmt  InstrFormat
		size uint32
	}{
		{OpNop, Fmt10x, 1},
		{OpMove, Fmt12x, 1},
		{OpConst4, Fmt11n, 1},
		{OpReturnVoid, Fmt10x, 1},
		{OpReturn, Fmt11x, 1},
		{OpGoto, Fmt10t, 1},
		{OpGoto16, Fmt20t, 2},
		{OpConst16, Fmt21s, 2},
		{OpConstString, Fmt21c, 2},
		{OpConstClass, Fmt21c, 2},
		{OpCheckCast, Fmt21c, 2},
		{OpIfEq, Fmt22t, 2},
		{OpIfEqz, Fmt21t, 2},
		{OpIget, Fmt22c, 2},
		{OpSget, Fmt21c, 2},
		{OpIput, Fmt22c, 2},
		{OpSput, Fmt21c, 2},
		{OpGoto32, Fmt30t, 3},
		{OpConst, Fmt31i, 3},
		{OpConstStringJumbo, Fmt31c, 3},
		{OpPackedSwitch, Fmt31t, 3},
		{OpInvokeVirtual, Fmt35c, 3},
		{OpInvokeStatic, Fmt35c, 3},
		{OpInvokeDirect, Fmt35c, 3},
		{OpInvokeVirtualRange, Fmt3rc, 3},
		{OpInvokeStaticRange, Fmt3rc, 3},
		{OpConstWide, Fmt51l, 5},
		{OpInvokePolymorphic, Fmt45cc, 4},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := opcodeFormat[tt.op]; got != tt.fmt {
				t.Errorf("opcode 0x%02X format = %d, want %d", tt.op, got, tt.fmt)
			}
			if got := InstrSizeInCodeUnits(tt.fmt); got != tt.size {
				t.Errorf("format %d size = %d, want %d", tt.fmt, got, tt.size)
			}
		})
	}
}

func TestVRegAccessors_35c(t *testing.T) {
	// invoke-virtual {v0, v1, v2}, method@0x1234
	// Format 35c: [AG|op] [BBBB] [FEDC]
	// A=3 (arg count), G=0 (unused with 3 args)
	// BBBB=0x1234 (method index)
	// C=0, D=1, E=2
	insns := []uint16{
		0x306E, // A=3, op=0x6E (invoke-virtual)
		0x1234, // BBBB = method index
		0x0210, // F=0, E=2, D=1, C=0
	}
	inst := Instruction{Op: OpInvokeVirtual, Raw: insns, DexPC: 0}

	if got := inst.VRegA_35c(); got != 3 {
		t.Errorf("VRegA_35c = %d, want 3", got)
	}
	if got := inst.VRegB_35c(); got != 0x1234 {
		t.Errorf("VRegB_35c = %d, want 0x1234", got)
	}

	args := inst.GetVarArgs_35c()
	if args[0] != 0 || args[1] != 1 || args[2] != 2 {
		t.Errorf("GetVarArgs = %v, want [0 1 2 ...]", args)
	}
}

func TestVRegAccessors_3rc(t *testing.T) {
	// invoke-virtual/range {v5..v7}, method@0x00AB
	// Format 3rc: [AA|op] [BBBB] [CCCC]
	insns := []uint16{
		0x0374, // AA=3 (count), op=0x74 (invoke-virtual/range)
		0x00AB, // BBBB = method index
		0x0005, // CCCC = first register
	}
	inst := Instruction{Op: OpInvokeVirtualRange, Raw: insns, DexPC: 0}

	if got := inst.VRegA_3rc(); got != 3 {
		t.Errorf("VRegA_3rc = %d, want 3", got)
	}
	if got := inst.VRegB_3rc(); got != 0x00AB {
		t.Errorf("VRegB_3rc = %d, want 0x00AB", got)
	}
	if got := inst.VRegC_3rc(); got != 5 {
		t.Errorf("VRegC_3rc = %d, want 5", got)
	}
}

func TestVRegAccessors_21c(t *testing.T) {
	// const-string v3, string@0x0042
	insns := []uint16{
		0x031A, // vAA=3, op=0x1A (const-string)
		0x0042, // BBBB = string index
	}
	inst := Instruction{Op: OpConstString, Raw: insns, DexPC: 0}

	if got := inst.VRegA_21c(); got != 3 {
		t.Errorf("VRegA_21c = %d, want 3", got)
	}
	if got := inst.VRegB_21c(); got != 0x0042 {
		t.Errorf("VRegB_21c = %d, want 0x0042", got)
	}
}

func TestVRegAccessors_22c(t *testing.T) {
	// iget v0, v1, field@0x00FF
	insns := []uint16{
		0x1052, // B=1, A=0, op=0x52 (iget)
		0x00FF, // CCCC = field index
	}
	inst := Instruction{Op: OpIget, Raw: insns, DexPC: 0}

	if got := inst.VRegA_22c(); got != 0 {
		t.Errorf("VRegA_22c = %d, want 0", got)
	}
	if got := inst.VRegB_22c(); got != 1 {
		t.Errorf("VRegB_22c = %d, want 1", got)
	}
	if got := inst.VRegC_22c(); got != 0x00FF {
		t.Errorf("VRegC_22c = %d, want 0x00FF", got)
	}
}

func TestDecodeAll(t *testing.T) {
	// Sequence: nop, const-string v0 string@1, return-void
	insns := []uint16{
		0x0000, // nop (1 unit)
		0x001A, // const-string v0 (2 units)
		0x0001, //   string@1
		0x000E, // return-void (1 unit)
	}

	decoded := DecodeAll(insns)
	if len(decoded) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(decoded))
	}

	if decoded[0].Op != OpNop || decoded[0].DexPC != 0 {
		t.Errorf("inst[0]: op=%02X pc=%d", decoded[0].Op, decoded[0].DexPC)
	}
	if decoded[1].Op != OpConstString || decoded[1].DexPC != 1 {
		t.Errorf("inst[1]: op=%02X pc=%d", decoded[1].Op, decoded[1].DexPC)
	}
	if decoded[1].VRegB_21c() != 1 {
		t.Errorf("inst[1] string index = %d, want 1", decoded[1].VRegB_21c())
	}
	if decoded[2].Op != OpReturnVoid || decoded[2].DexPC != 3 {
		t.Errorf("inst[2]: op=%02X pc=%d", decoded[2].Op, decoded[2].DexPC)
	}
}

func TestOpcodePredicates(t *testing.T) {
	if !OpInvokeVirtual.IsInvoke() {
		t.Error("InvokeVirtual should be invoke")
	}
	if !OpInvokeStaticRange.IsInvoke() {
		t.Error("InvokeStaticRange should be invoke")
	}
	if !OpInvokeStaticRange.IsInvokeRange() {
		t.Error("InvokeStaticRange should be range")
	}
	if OpInvokeVirtual.IsInvokeRange() {
		t.Error("InvokeVirtual should not be range")
	}
	if !OpIget.IsFieldGet() {
		t.Error("Iget should be field get")
	}
	if !OpSput.IsFieldPut() {
		t.Error("Sput should be field put")
	}
	if !OpGoto.IsBranch() {
		t.Error("Goto should be branch")
	}
	if !OpGoto.IsUnconditionalBranch() {
		t.Error("Goto should be unconditional")
	}
	if !OpIfEq.IsBranch() {
		t.Error("IfEq should be branch")
	}
	if OpIfEq.IsUnconditionalBranch() {
		t.Error("IfEq should not be unconditional")
	}
	if !OpReturnVoid.IsReturn() {
		t.Error("ReturnVoid should be return")
	}
	if !OpPackedSwitch.IsSwitch() {
		t.Error("PackedSwitch should be switch")
	}
}

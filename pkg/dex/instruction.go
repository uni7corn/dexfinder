package dex

// Opcode represents a DEX bytecode opcode.
type Opcode uint8

// All DEX opcodes from dex_instruction_list.h
const (
	OpNop                  Opcode = 0x00
	OpMove                 Opcode = 0x01
	OpMoveFrom16           Opcode = 0x02
	OpMove16               Opcode = 0x03
	OpMoveWide             Opcode = 0x04
	OpMoveWideFrom16       Opcode = 0x05
	OpMoveWide16           Opcode = 0x06
	OpMoveObject           Opcode = 0x07
	OpMoveObjectFrom16     Opcode = 0x08
	OpMoveObject16         Opcode = 0x09
	OpMoveResult           Opcode = 0x0A
	OpMoveResultWide       Opcode = 0x0B
	OpMoveResultObject     Opcode = 0x0C
	OpMoveException        Opcode = 0x0D
	OpReturnVoid           Opcode = 0x0E
	OpReturn               Opcode = 0x0F
	OpReturnWide           Opcode = 0x10
	OpReturnObject         Opcode = 0x11
	OpConst4               Opcode = 0x12
	OpConst16              Opcode = 0x13
	OpConst                Opcode = 0x14
	OpConstHigh16          Opcode = 0x15
	OpConstWide16          Opcode = 0x16
	OpConstWide32          Opcode = 0x17
	OpConstWide            Opcode = 0x18
	OpConstWideHigh16      Opcode = 0x19
	OpConstString          Opcode = 0x1A
	OpConstStringJumbo     Opcode = 0x1B
	OpConstClass           Opcode = 0x1C
	OpMonitorEnter         Opcode = 0x1D
	OpMonitorExit          Opcode = 0x1E
	OpCheckCast            Opcode = 0x1F
	OpInstanceOf           Opcode = 0x20
	OpArrayLength          Opcode = 0x21
	OpNewInstance          Opcode = 0x22
	OpNewArray             Opcode = 0x23
	OpFilledNewArray       Opcode = 0x24
	OpFilledNewArrayRange  Opcode = 0x25
	OpFillArrayData        Opcode = 0x26
	OpThrow                Opcode = 0x27
	OpGoto                 Opcode = 0x28
	OpGoto16               Opcode = 0x29
	OpGoto32               Opcode = 0x2A
	OpPackedSwitch         Opcode = 0x2B
	OpSparseSwitch         Opcode = 0x2C
	OpCmplFloat            Opcode = 0x2D
	OpCmpgFloat            Opcode = 0x2E
	OpCmplDouble           Opcode = 0x2F
	OpCmpgDouble           Opcode = 0x30
	OpCmpLong              Opcode = 0x31
	OpIfEq                 Opcode = 0x32
	OpIfNe                 Opcode = 0x33
	OpIfLt                 Opcode = 0x34
	OpIfGe                 Opcode = 0x35
	OpIfGt                 Opcode = 0x36
	OpIfLe                 Opcode = 0x37
	OpIfEqz                Opcode = 0x38
	OpIfNez                Opcode = 0x39
	OpIfLtz                Opcode = 0x3A
	OpIfGez                Opcode = 0x3B
	OpIfGtz                Opcode = 0x3C
	OpIfLez                Opcode = 0x3D
	OpAget                 Opcode = 0x44
	OpAgetWide             Opcode = 0x45
	OpAgetObject           Opcode = 0x46
	OpAgetBoolean          Opcode = 0x47
	OpAgetByte             Opcode = 0x48
	OpAgetChar             Opcode = 0x49
	OpAgetShort            Opcode = 0x4A
	OpAput                 Opcode = 0x4B
	OpAputWide             Opcode = 0x4C
	OpAputObject           Opcode = 0x4D
	OpAputBoolean          Opcode = 0x4E
	OpAputByte             Opcode = 0x4F
	OpAputChar             Opcode = 0x50
	OpAputShort            Opcode = 0x51
	OpIget                 Opcode = 0x52
	OpIgetWide             Opcode = 0x53
	OpIgetObject           Opcode = 0x54
	OpIgetBoolean          Opcode = 0x55
	OpIgetByte             Opcode = 0x56
	OpIgetChar             Opcode = 0x57
	OpIgetShort            Opcode = 0x58
	OpIput                 Opcode = 0x59
	OpIputWide             Opcode = 0x5A
	OpIputObject           Opcode = 0x5B
	OpIputBoolean          Opcode = 0x5C
	OpIputByte             Opcode = 0x5D
	OpIputChar             Opcode = 0x5E
	OpIputShort            Opcode = 0x5F
	OpSget                 Opcode = 0x60
	OpSgetWide             Opcode = 0x61
	OpSgetObject           Opcode = 0x62
	OpSgetBoolean          Opcode = 0x63
	OpSgetByte             Opcode = 0x64
	OpSgetChar             Opcode = 0x65
	OpSgetShort            Opcode = 0x66
	OpSput                 Opcode = 0x67
	OpSputWide             Opcode = 0x68
	OpSputObject           Opcode = 0x69
	OpSputBoolean          Opcode = 0x6A
	OpSputByte             Opcode = 0x6B
	OpSputChar             Opcode = 0x6C
	OpSputShort            Opcode = 0x6D
	OpInvokeVirtual        Opcode = 0x6E
	OpInvokeSuper          Opcode = 0x6F
	OpInvokeDirect         Opcode = 0x70
	OpInvokeStatic         Opcode = 0x71
	OpInvokeInterface      Opcode = 0x72
	OpInvokeVirtualRange   Opcode = 0x74
	OpInvokeSuperRange     Opcode = 0x75
	OpInvokeDirectRange    Opcode = 0x76
	OpInvokeStaticRange    Opcode = 0x77
	OpInvokeInterfaceRange Opcode = 0x78
	OpNegInt               Opcode = 0x7B
	OpNotInt               Opcode = 0x7C
	OpNegLong              Opcode = 0x7D
	OpNotLong              Opcode = 0x7E
	OpNegFloat             Opcode = 0x7F
	OpNegDouble            Opcode = 0x80
	OpIntToLong            Opcode = 0x81
	OpIntToFloat           Opcode = 0x82
	OpIntToDouble          Opcode = 0x83
	OpLongToInt            Opcode = 0x84
	OpLongToFloat          Opcode = 0x85
	OpLongToDouble         Opcode = 0x86
	OpFloatToInt           Opcode = 0x87
	OpFloatToLong          Opcode = 0x88
	OpFloatToDouble        Opcode = 0x89
	OpDoubleToInt          Opcode = 0x8A
	OpDoubleToLong         Opcode = 0x8B
	OpDoubleToFloat        Opcode = 0x8C
	OpIntToByte            Opcode = 0x8D
	OpIntToChar            Opcode = 0x8E
	OpIntToShort           Opcode = 0x8F
	OpAddInt               Opcode = 0x90
	OpSubInt               Opcode = 0x91
	OpMulInt               Opcode = 0x92
	OpDivInt               Opcode = 0x93
	OpRemInt               Opcode = 0x94
	OpAndInt               Opcode = 0x95
	OpOrInt                Opcode = 0x96
	OpXorInt               Opcode = 0x97
	OpShlInt               Opcode = 0x98
	OpShrInt               Opcode = 0x99
	OpUshrInt              Opcode = 0x9A
	OpAddLong              Opcode = 0x9B
	OpSubLong              Opcode = 0x9C
	OpMulLong              Opcode = 0x9D
	OpDivLong              Opcode = 0x9E
	OpRemLong              Opcode = 0x9F
	OpAndLong              Opcode = 0xA0
	OpOrLong               Opcode = 0xA1
	OpXorLong              Opcode = 0xA2
	OpShlLong              Opcode = 0xA3
	OpShrLong              Opcode = 0xA4
	OpUshrLong             Opcode = 0xA5
	OpAddFloat             Opcode = 0xA6
	OpSubFloat             Opcode = 0xA7
	OpMulFloat             Opcode = 0xA8
	OpDivFloat             Opcode = 0xA9
	OpRemFloat             Opcode = 0xAA
	OpAddDouble            Opcode = 0xAB
	OpSubDouble            Opcode = 0xAC
	OpMulDouble            Opcode = 0xAD
	OpDivDouble            Opcode = 0xAE
	OpRemDouble            Opcode = 0xAF
	OpAddInt2Addr          Opcode = 0xB0
	OpSubInt2Addr          Opcode = 0xB1
	OpMulInt2Addr          Opcode = 0xB2
	OpDivInt2Addr          Opcode = 0xB3
	OpRemInt2Addr          Opcode = 0xB4
	OpAndInt2Addr          Opcode = 0xB5
	OpOrInt2Addr           Opcode = 0xB6
	OpXorInt2Addr          Opcode = 0xB7
	OpShlInt2Addr          Opcode = 0xB8
	OpShrInt2Addr          Opcode = 0xB9
	OpUshrInt2Addr         Opcode = 0xBA
	OpAddLong2Addr         Opcode = 0xBB
	OpSubLong2Addr         Opcode = 0xBC
	OpMulLong2Addr         Opcode = 0xBD
	OpDivLong2Addr         Opcode = 0xBE
	OpRemLong2Addr         Opcode = 0xBF
	OpAndLong2Addr         Opcode = 0xC0
	OpOrLong2Addr          Opcode = 0xC1
	OpXorLong2Addr         Opcode = 0xC2
	OpShlLong2Addr         Opcode = 0xC3
	OpShrLong2Addr         Opcode = 0xC4
	OpUshrLong2Addr        Opcode = 0xC5
	OpAddFloat2Addr        Opcode = 0xC6
	OpSubFloat2Addr        Opcode = 0xC7
	OpMulFloat2Addr        Opcode = 0xC8
	OpDivFloat2Addr        Opcode = 0xC9
	OpRemFloat2Addr        Opcode = 0xCA
	OpAddDouble2Addr       Opcode = 0xCB
	OpSubDouble2Addr       Opcode = 0xCC
	OpMulDouble2Addr       Opcode = 0xCD
	OpDivDouble2Addr       Opcode = 0xCE
	OpRemDouble2Addr       Opcode = 0xCF
	OpAddIntLit16          Opcode = 0xD0
	OpRsubInt              Opcode = 0xD1
	OpMulIntLit16          Opcode = 0xD2
	OpDivIntLit16          Opcode = 0xD3
	OpRemIntLit16          Opcode = 0xD4
	OpAndIntLit16          Opcode = 0xD5
	OpOrIntLit16           Opcode = 0xD6
	OpXorIntLit16          Opcode = 0xD7
	OpAddIntLit8           Opcode = 0xD8
	OpRsubIntLit8          Opcode = 0xD9
	OpMulIntLit8           Opcode = 0xDA
	OpDivIntLit8           Opcode = 0xDB
	OpRemIntLit8           Opcode = 0xDC
	OpAndIntLit8           Opcode = 0xDD
	OpOrIntLit8            Opcode = 0xDE
	OpXorIntLit8           Opcode = 0xDF
	OpShlIntLit8           Opcode = 0xE0
	OpShrIntLit8           Opcode = 0xE1
	OpUshrIntLit8          Opcode = 0xE2
	OpInvokePolymorphic      Opcode = 0xFA
	OpInvokePolymorphicRange Opcode = 0xFB
	OpInvokeCustom           Opcode = 0xFC
	OpInvokeCustomRange      Opcode = 0xFD
	OpConstMethodHandle      Opcode = 0xFE
	OpConstMethodType        Opcode = 0xFF
)

// InstrFormat represents the encoding format of an instruction.
type InstrFormat uint8

const (
	FmtUnknown InstrFormat = iota
	Fmt10x     // op
	Fmt12x     // op vA, vB
	Fmt11n     // op vA, #+B
	Fmt11x     // op vAA
	Fmt10t     // op +AA
	Fmt20t     // op +AAAA
	Fmt22x     // op vAA, vBBBB
	Fmt21t     // op vAA, +BBBB
	Fmt21s     // op vAA, #+BBBB
	Fmt21h     // op vAA, #+BBBB0000(00000000)
	Fmt21c     // op vAA, thing@BBBB
	Fmt23x     // op vAA, vBB, vCC
	Fmt22b     // op vAA, vBB, #+CC
	Fmt22t     // op vA, vB, +CCCC
	Fmt22s     // op vA, vB, #+CCCC
	Fmt22c     // op vA, vB, thing@CCCC
	Fmt32x     // op vAAAA, vBBBB
	Fmt30t     // op +AAAAAAAA
	Fmt31t     // op vAA, +BBBBBBBB
	Fmt31i     // op vAA, #+BBBBBBBB
	Fmt31c     // op vAA, string@BBBBBBBB
	Fmt35c     // op {vC,vD,vE,vF,vG}, thing@BBBB
	Fmt3rc     // op {vCCCC..v(CCCC+AA)}, thing@BBBB
	Fmt45cc    // op {vC,vD,vE,vF,vG}, meth@BBBB, proto@HHHH
	Fmt4rcc    // op {vCCCC..v(CCCC+AA)}, meth@BBBB, proto@HHHH
	Fmt51l     // op vAA, #+BBBBBBBBBBBBBBBB
)

// opcodeFormat maps each opcode to its instruction format.
var opcodeFormat [256]InstrFormat

func init() {
	// Default: all unknown
	for i := range opcodeFormat {
		opcodeFormat[i] = Fmt10x // unused opcodes are treated as 1 code unit
	}

	// k10x: 1 unit
	for _, op := range []Opcode{OpNop, OpReturnVoid} {
		opcodeFormat[op] = Fmt10x
	}

	// k12x: 1 unit
	for op := OpMove; op <= OpMove; op++ {
		opcodeFormat[op] = Fmt12x
	}
	opcodeFormat[OpMoveWide] = Fmt12x
	opcodeFormat[OpMoveObject] = Fmt12x
	opcodeFormat[OpArrayLength] = Fmt12x
	for op := OpNegInt; op <= OpIntToShort; op++ {
		opcodeFormat[op] = Fmt12x
	}
	for op := OpAddInt2Addr; op <= OpRemDouble2Addr; op++ {
		opcodeFormat[op] = Fmt12x
	}

	// k11n: 1 unit
	opcodeFormat[OpConst4] = Fmt11n

	// k11x: 1 unit
	for _, op := range []Opcode{OpMoveResult, OpMoveResultWide, OpMoveResultObject, OpMoveException,
		OpReturn, OpReturnWide, OpReturnObject, OpMonitorEnter, OpMonitorExit, OpThrow} {
		opcodeFormat[op] = Fmt11x
	}

	// k10t: 1 unit
	opcodeFormat[OpGoto] = Fmt10t

	// k20t: 2 units
	opcodeFormat[OpGoto16] = Fmt20t

	// k22x: 2 units
	opcodeFormat[OpMoveFrom16] = Fmt22x
	opcodeFormat[OpMoveWideFrom16] = Fmt22x
	opcodeFormat[OpMoveObjectFrom16] = Fmt22x

	// k21t: 2 units
	for op := OpIfEqz; op <= OpIfLez; op++ {
		opcodeFormat[op] = Fmt21t
	}

	// k21s: 2 units
	opcodeFormat[OpConst16] = Fmt21s
	opcodeFormat[OpConstWide16] = Fmt21s

	// k21h: 2 units
	opcodeFormat[OpConstHigh16] = Fmt21h
	opcodeFormat[OpConstWideHigh16] = Fmt21h

	// k21c: 2 units
	opcodeFormat[OpConstString] = Fmt21c
	opcodeFormat[OpConstClass] = Fmt21c
	opcodeFormat[OpCheckCast] = Fmt21c
	opcodeFormat[OpNewInstance] = Fmt21c
	opcodeFormat[OpConstMethodHandle] = Fmt21c
	opcodeFormat[OpConstMethodType] = Fmt21c
	// sget/sput
	for op := OpSget; op <= OpSputShort; op++ {
		opcodeFormat[op] = Fmt21c
	}

	// k23x: 2 units
	for op := OpCmplFloat; op <= OpCmpLong; op++ {
		opcodeFormat[op] = Fmt23x
	}
	for op := OpAget; op <= OpAputShort; op++ {
		opcodeFormat[op] = Fmt23x
	}
	for op := OpAddInt; op <= OpRemDouble; op++ {
		opcodeFormat[op] = Fmt23x
	}

	// k22b: 2 units
	for op := OpAddIntLit8; op <= OpUshrIntLit8; op++ {
		opcodeFormat[op] = Fmt22b
	}

	// k22t: 2 units
	for op := OpIfEq; op <= OpIfLe; op++ {
		opcodeFormat[op] = Fmt22t
	}

	// k22s: 2 units
	for op := OpAddIntLit16; op <= OpXorIntLit16; op++ {
		opcodeFormat[op] = Fmt22s
	}

	// k22c: 2 units
	opcodeFormat[OpInstanceOf] = Fmt22c
	opcodeFormat[OpNewArray] = Fmt22c
	for op := OpIget; op <= OpIputShort; op++ {
		opcodeFormat[op] = Fmt22c
	}

	// k32x: 3 units
	opcodeFormat[OpMove16] = Fmt32x
	opcodeFormat[OpMoveWide16] = Fmt32x
	opcodeFormat[OpMoveObject16] = Fmt32x

	// k30t: 3 units
	opcodeFormat[OpGoto32] = Fmt30t

	// k31t: 3 units
	opcodeFormat[OpPackedSwitch] = Fmt31t
	opcodeFormat[OpSparseSwitch] = Fmt31t
	opcodeFormat[OpFillArrayData] = Fmt31t

	// k31i: 3 units
	opcodeFormat[OpConst] = Fmt31i
	opcodeFormat[OpConstWide32] = Fmt31i

	// k31c: 3 units
	opcodeFormat[OpConstStringJumbo] = Fmt31c

	// k35c: 3 units
	opcodeFormat[OpFilledNewArray] = Fmt35c
	opcodeFormat[OpInvokeVirtual] = Fmt35c
	opcodeFormat[OpInvokeSuper] = Fmt35c
	opcodeFormat[OpInvokeDirect] = Fmt35c
	opcodeFormat[OpInvokeStatic] = Fmt35c
	opcodeFormat[OpInvokeInterface] = Fmt35c
	opcodeFormat[OpInvokeCustom] = Fmt35c

	// k3rc: 3 units
	opcodeFormat[OpFilledNewArrayRange] = Fmt3rc
	opcodeFormat[OpInvokeVirtualRange] = Fmt3rc
	opcodeFormat[OpInvokeSuperRange] = Fmt3rc
	opcodeFormat[OpInvokeDirectRange] = Fmt3rc
	opcodeFormat[OpInvokeStaticRange] = Fmt3rc
	opcodeFormat[OpInvokeInterfaceRange] = Fmt3rc
	opcodeFormat[OpInvokeCustomRange] = Fmt3rc

	// k45cc: 4 units
	opcodeFormat[OpInvokePolymorphic] = Fmt45cc

	// k4rcc: 4 units
	opcodeFormat[OpInvokePolymorphicRange] = Fmt4rcc

	// k51l: 5 units
	opcodeFormat[OpConstWide] = Fmt51l
}

// InstrSizeInCodeUnits returns the size of an instruction format in 16-bit code units.
func InstrSizeInCodeUnits(fmt InstrFormat) uint32 {
	switch fmt {
	case Fmt10x, Fmt12x, Fmt11n, Fmt11x, Fmt10t:
		return 1
	case Fmt20t, Fmt22x, Fmt21t, Fmt21s, Fmt21h, Fmt21c, Fmt23x, Fmt22b, Fmt22t, Fmt22s, Fmt22c:
		return 2
	case Fmt32x, Fmt30t, Fmt31t, Fmt31i, Fmt31c, Fmt35c, Fmt3rc:
		return 3
	case Fmt45cc, Fmt4rcc:
		return 4
	case Fmt51l:
		return 5
	default:
		return 1
	}
}

// Instruction represents a decoded DEX instruction.
type Instruction struct {
	Op    Opcode
	Raw   []uint16 // raw code units
	DexPC uint32   // position in code units
}

// SizeInCodeUnits returns the size of this instruction.
func (inst *Instruction) SizeInCodeUnits() uint32 {
	op := inst.Op
	// NOP can be a spacer (pseudo-opcode for switch/array data)
	if op == OpNop && len(inst.Raw) > 0 {
		ident := inst.Raw[0] >> 8
		if ident == 0x01 { // packed-switch-payload
			if len(inst.Raw) >= 2 {
				size := uint32(inst.Raw[1])
				return (size*2 + 4) // in code units
			}
		} else if ident == 0x02 { // sparse-switch-payload
			if len(inst.Raw) >= 2 {
				size := uint32(inst.Raw[1])
				return (size*4 + 2) // in code units
			}
		} else if ident == 0x03 { // fill-array-data-payload
			if len(inst.Raw) >= 4 {
				elemWidth := uint32(inst.Raw[1])
				size := uint32(inst.Raw[2]) | uint32(inst.Raw[3])<<16
				totalBytes := size * elemWidth
				units := (totalBytes + 1) / 2
				return units + 4
			}
		}
	}
	return InstrSizeInCodeUnits(opcodeFormat[op])
}

// Format returns the instruction format.
func (inst *Instruction) Format() InstrFormat {
	return opcodeFormat[inst.Op]
}

// --- Register accessors matching C++ VReg patterns ---

// VRegA_10t: signed 8-bit offset (goto)
func (inst *Instruction) VRegA_10t() int8 {
	return int8(inst.Raw[0] >> 8)
}

// VRegA_11n: 4-bit register
func (inst *Instruction) VRegA_11n() uint32 {
	return uint32((inst.Raw[0] >> 8) & 0x0F)
}

// VRegB_11n: 4-bit signed literal
func (inst *Instruction) VRegB_11n() int32 {
	return int32(int8(inst.Raw[0]&0xFF00>>8)) >> 4
}

// VRegA_11x: 8-bit register
func (inst *Instruction) VRegA_11x() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegA_12x: 4-bit register A
func (inst *Instruction) VRegA_12x() uint32 {
	return uint32((inst.Raw[0] >> 8) & 0x0F)
}

// VRegB_12x: 4-bit register B
func (inst *Instruction) VRegB_12x() uint32 {
	return uint32(inst.Raw[0] >> 12)
}

// VRegA_20t: (unused upper byte)
// VRegA_20t offset is in inst.Raw[1] as signed 16-bit
func (inst *Instruction) VRegA_20t() int16 {
	return int16(inst.Raw[1])
}

// VRegA_21c: 8-bit register
func (inst *Instruction) VRegA_21c() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_21c: 16-bit index
func (inst *Instruction) VRegB_21c() uint32 {
	return uint32(inst.Raw[1])
}

// VRegA_21s: 8-bit register
func (inst *Instruction) VRegA_21s() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_21s: signed 16-bit literal
func (inst *Instruction) VRegB_21s() int32 {
	return int32(int16(inst.Raw[1]))
}

// VRegA_21t: 8-bit register
func (inst *Instruction) VRegA_21t() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_21t: signed 16-bit branch offset
func (inst *Instruction) VRegB_21t() int16 {
	return int16(inst.Raw[1])
}

// VRegA_21h: 8-bit register
func (inst *Instruction) VRegA_21h() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_21h: 16-bit high literal (shifted left by 16)
func (inst *Instruction) VRegB_21h() int32 {
	return int32(int16(inst.Raw[1])) << 16
}

// VRegA_22x: 8-bit register
func (inst *Instruction) VRegA_22x() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_22x: 16-bit register
func (inst *Instruction) VRegB_22x() uint32 {
	return uint32(inst.Raw[1])
}

// VRegA_22c: 4-bit register A
func (inst *Instruction) VRegA_22c() uint32 {
	return uint32((inst.Raw[0] >> 8) & 0x0F)
}

// VRegB_22c: 4-bit register B
func (inst *Instruction) VRegB_22c() uint32 {
	return uint32(inst.Raw[0] >> 12)
}

// VRegC_22c: 16-bit index
func (inst *Instruction) VRegC_22c() uint32 {
	return uint32(inst.Raw[1])
}

// VRegA_22t: 4-bit register A
func (inst *Instruction) VRegA_22t() uint32 {
	return uint32((inst.Raw[0] >> 8) & 0x0F)
}

// VRegB_22t: 4-bit register B
func (inst *Instruction) VRegB_22t() uint32 {
	return uint32(inst.Raw[0] >> 12)
}

// VRegC_22t: signed 16-bit branch offset
func (inst *Instruction) VRegC_22t() int16 {
	return int16(inst.Raw[1])
}

// VRegA_22s: 4-bit register A
func (inst *Instruction) VRegA_22s() uint32 {
	return uint32((inst.Raw[0] >> 8) & 0x0F)
}

// VRegB_22s: 4-bit register B
func (inst *Instruction) VRegB_22s() uint32 {
	return uint32(inst.Raw[0] >> 12)
}

// VRegC_22s: signed 16-bit literal
func (inst *Instruction) VRegC_22s() int32 {
	return int32(int16(inst.Raw[1]))
}

// VRegA_22b: 8-bit register A
func (inst *Instruction) VRegA_22b() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_22b: 8-bit register B
func (inst *Instruction) VRegB_22b() uint32 {
	return uint32(inst.Raw[1] & 0xFF)
}

// VRegC_22b: signed 8-bit literal
func (inst *Instruction) VRegC_22b() int32 {
	return int32(int8(inst.Raw[1] >> 8))
}

// VRegA_23x: 8-bit register A
func (inst *Instruction) VRegA_23x() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_23x: 8-bit register B
func (inst *Instruction) VRegB_23x() uint32 {
	return uint32(inst.Raw[1] & 0xFF)
}

// VRegC_23x: 8-bit register C
func (inst *Instruction) VRegC_23x() uint32 {
	return uint32(inst.Raw[1] >> 8)
}

// VRegA_30t: signed 32-bit offset (goto/32)
func (inst *Instruction) VRegA_30t() int32 {
	return int32(uint32(inst.Raw[1]) | uint32(inst.Raw[2])<<16)
}

// VRegA_31t: 8-bit register
func (inst *Instruction) VRegA_31t() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_31t: signed 32-bit offset
func (inst *Instruction) VRegB_31t() int32 {
	return int32(uint32(inst.Raw[1]) | uint32(inst.Raw[2])<<16)
}

// VRegA_31i: 8-bit register
func (inst *Instruction) VRegA_31i() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_31i: 32-bit literal
func (inst *Instruction) VRegB_31i() int32 {
	return int32(uint32(inst.Raw[1]) | uint32(inst.Raw[2])<<16)
}

// VRegA_31c: 8-bit register
func (inst *Instruction) VRegA_31c() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_31c: 32-bit string index
func (inst *Instruction) VRegB_31c() uint32 {
	return uint32(inst.Raw[1]) | uint32(inst.Raw[2])<<16
}

// VRegA_32x: 16-bit register A
func (inst *Instruction) VRegA_32x() uint32 {
	return uint32(inst.Raw[1])
}

// VRegB_32x: 16-bit register B
func (inst *Instruction) VRegB_32x() uint32 {
	return uint32(inst.Raw[2])
}

// --- 35c format: op {vC,vD,vE,vF,vG}, kind@BBBB ---

// VRegA_35c: argument count (4-bit)
func (inst *Instruction) VRegA_35c() uint32 {
	return uint32((inst.Raw[0] >> 12) & 0x0F)
}

// VRegB_35c: 16-bit method/type index
func (inst *Instruction) VRegB_35c() uint32 {
	return uint32(inst.Raw[1])
}

// VRegC_35c: first register (4-bit)
func (inst *Instruction) VRegC_35c() uint32 {
	return uint32(inst.Raw[2] & 0x0F)
}

// GetVarArgs_35c returns up to 5 register arguments.
func (inst *Instruction) GetVarArgs_35c() [5]uint32 {
	var args [5]uint32
	args[0] = uint32(inst.Raw[2] & 0x0F)
	args[1] = uint32((inst.Raw[2] >> 4) & 0x0F)
	args[2] = uint32((inst.Raw[2] >> 8) & 0x0F)
	args[3] = uint32((inst.Raw[2] >> 12) & 0x0F)
	args[4] = uint32((inst.Raw[0] >> 8) & 0x0F)
	return args
}

// --- 3rc format: op {vCCCC..v(CCCC+AA)}, kind@BBBB ---

// VRegA_3rc: 8-bit argument count
func (inst *Instruction) VRegA_3rc() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_3rc: 16-bit method/type index
func (inst *Instruction) VRegB_3rc() uint32 {
	return uint32(inst.Raw[1])
}

// VRegC_3rc: 16-bit first register
func (inst *Instruction) VRegC_3rc() uint32 {
	return uint32(inst.Raw[2])
}

// --- 51l format ---

// VRegA_51l: 8-bit register
func (inst *Instruction) VRegA_51l() uint32 {
	return uint32(inst.Raw[0] >> 8)
}

// VRegB_51l: 64-bit literal
func (inst *Instruction) VRegB_51l() int64 {
	return int64(uint64(inst.Raw[1]) | uint64(inst.Raw[2])<<16 |
		uint64(inst.Raw[3])<<32 | uint64(inst.Raw[4])<<48)
}

// IsInvoke returns true if the opcode is any invoke instruction.
func (op Opcode) IsInvoke() bool {
	switch op {
	case OpInvokeVirtual, OpInvokeSuper, OpInvokeDirect, OpInvokeStatic, OpInvokeInterface,
		OpInvokeVirtualRange, OpInvokeSuperRange, OpInvokeDirectRange, OpInvokeStaticRange, OpInvokeInterfaceRange,
		OpInvokePolymorphic, OpInvokePolymorphicRange, OpInvokeCustom, OpInvokeCustomRange:
		return true
	}
	return false
}

// IsInvokeRange returns true if the opcode is a range invoke.
func (op Opcode) IsInvokeRange() bool {
	switch op {
	case OpInvokeVirtualRange, OpInvokeSuperRange, OpInvokeDirectRange, OpInvokeStaticRange, OpInvokeInterfaceRange,
		OpInvokePolymorphicRange, OpInvokeCustomRange:
		return true
	}
	return false
}

// IsFieldGet returns true for all iget/sget opcodes.
func (op Opcode) IsFieldGet() bool {
	return (op >= OpIget && op <= OpIgetShort) || (op >= OpSget && op <= OpSgetShort)
}

// IsFieldPut returns true for all iput/sput opcodes.
func (op Opcode) IsFieldPut() bool {
	return (op >= OpIput && op <= OpIputShort) || (op >= OpSput && op <= OpSputShort)
}

// IsInstanceField returns true for iget/iput opcodes.
func (op Opcode) IsInstanceField() bool {
	return op >= OpIget && op <= OpIputShort
}

// IsStaticField returns true for sget/sput opcodes.
func (op Opcode) IsStaticField() bool {
	return op >= OpSget && op <= OpSputShort
}

// IsBranch returns true for branch instructions.
func (op Opcode) IsBranch() bool {
	switch op {
	case OpGoto, OpGoto16, OpGoto32:
		return true
	case OpIfEq, OpIfNe, OpIfLt, OpIfGe, OpIfGt, OpIfLe:
		return true
	case OpIfEqz, OpIfNez, OpIfLtz, OpIfGez, OpIfGtz, OpIfLez:
		return true
	}
	return false
}

// IsUnconditionalBranch returns true for goto instructions.
func (op Opcode) IsUnconditionalBranch() bool {
	return op == OpGoto || op == OpGoto16 || op == OpGoto32
}

// IsSwitch returns true for switch instructions.
func (op Opcode) IsSwitch() bool {
	return op == OpPackedSwitch || op == OpSparseSwitch
}

// IsReturn returns true for return instructions.
func (op Opcode) IsReturn() bool {
	return op >= OpReturnVoid && op <= OpReturnObject
}

// DecodeAll decodes all instructions from a code_item's insns array.
func DecodeAll(insns []uint16) []Instruction {
	var result []Instruction
	pc := uint32(0)
	for pc < uint32(len(insns)) {
		op := Opcode(insns[pc] & 0xFF)
		fmt := opcodeFormat[op]
		size := InstrSizeInCodeUnits(fmt)

		// Handle pseudo-opcodes (switch/array data payloads)
		if op == OpNop && pc < uint32(len(insns)) {
			ident := insns[pc] >> 8
			switch ident {
			case 0x01: // packed-switch
				if pc+1 < uint32(len(insns)) {
					sz := uint32(insns[pc+1])
					size = sz*2 + 4
				}
			case 0x02: // sparse-switch
				if pc+1 < uint32(len(insns)) {
					sz := uint32(insns[pc+1])
					size = sz*4 + 2
				}
			case 0x03: // fill-array-data
				if pc+3 < uint32(len(insns)) {
					elemWidth := uint32(insns[pc+1])
					elemCount := uint32(insns[pc+2]) | uint32(insns[pc+3])<<16
					totalBytes := elemCount * elemWidth
					size = (totalBytes+1)/2 + 4
				}
			}
		}

		// Clamp to available data
		end := pc + size
		if end > uint32(len(insns)) {
			end = uint32(len(insns))
		}

		inst := Instruction{
			Op:    op,
			Raw:   insns[pc:end],
			DexPC: pc,
		}
		result = append(result, inst)
		pc += size
	}
	return result
}

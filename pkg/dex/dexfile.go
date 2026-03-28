package dex

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// DEX file magic: "dex\n035\0" or "dex\n039\0" etc.
var dexMagicPrefix = []byte("dex\n")

// Header represents the DEX file header.
type Header struct {
	Magic        [8]byte
	Checksum     uint32
	Signature    [20]byte
	FileSize     uint32
	HeaderSize   uint32
	EndianTag    uint32
	LinkSize     uint32
	LinkOff      uint32
	MapOff       uint32
	StringIdsSize uint32
	StringIdsOff  uint32
	TypeIdsSize   uint32
	TypeIdsOff    uint32
	ProtoIdsSize  uint32
	ProtoIdsOff   uint32
	FieldIdsSize  uint32
	FieldIdsOff   uint32
	MethodIdsSize uint32
	MethodIdsOff  uint32
	ClassDefsSize uint32
	ClassDefsOff  uint32
	DataSize      uint32
	DataOff       uint32
}

// StringID is an entry in the string_ids table.
type StringID struct {
	DataOff uint32
}

// TypeID is an entry in the type_ids table.
type TypeID struct {
	DescriptorIdx uint32
}

// ProtoID is an entry in the proto_ids table.
type ProtoID struct {
	ShortyIdx     uint32
	ReturnTypeIdx uint32
	ParametersOff uint32
}

// FieldID is an entry in the field_ids table.
type FieldID struct {
	ClassIdx uint16
	TypeIdx  uint16
	NameIdx  uint32
}

// MethodID is an entry in the method_ids table.
type MethodID struct {
	ClassIdx uint16
	ProtoIdx uint16
	NameIdx  uint32
}

// ClassDef is an entry in the class_defs table.
type ClassDef struct {
	ClassIdx        uint32
	AccessFlags     uint32
	SuperclassIdx   uint32 // NO_INDEX = 0xFFFFFFFF
	InterfacesOff   uint32
	SourceFileIdx   uint32
	AnnotationsOff  uint32
	ClassDataOff    uint32
	StaticValuesOff uint32
}

// TypeList is a list of type indices.
type TypeList struct {
	Size  uint32
	Items []uint16
}

// CodeItem contains method bytecode and metadata.
type CodeItem struct {
	RegistersSize uint16
	InsSize       uint16
	OutsSize      uint16
	TriesSize     uint16
	DebugInfoOff  uint32
	InsnsSize     uint32 // in 16-bit code units
	Insns         []uint16
	Tries         []TryItem
	Handlers      []EncodedCatchHandler
}

// TryItem describes an exception handler range.
type TryItem struct {
	StartAddr  uint32
	InsnCount  uint16
	HandlerOff uint16
}

// EncodedCatchHandler describes catch handlers for a try block.
type EncodedCatchHandler struct {
	Handlers   []EncodedTypeAddrPair
	CatchAllAddr int32 // -1 if no catch-all
}

// EncodedTypeAddrPair maps a type to a handler address.
type EncodedTypeAddrPair struct {
	TypeIdx uint32
	Addr    uint32
}

// NoIndex is the sentinel value for "no index".
const NoIndex = 0xFFFFFFFF

// DexFile represents a parsed DEX file.
type DexFile struct {
	Data      []byte
	Header    Header
	StringIDs []StringID
	TypeIDs   []TypeID
	ProtoIDs  []ProtoID
	FieldIDs  []FieldID
	MethodIDs []MethodID
	ClassDefs []ClassDef

	// Cached string table
	strings []string
}

// Parse parses a DEX file from raw bytes.
func Parse(data []byte) (*DexFile, error) {
	if len(data) < 112 {
		return nil, fmt.Errorf("data too short for DEX header: %d bytes", len(data))
	}

	// Validate magic
	if string(data[:4]) != string(dexMagicPrefix) {
		return nil, fmt.Errorf("invalid DEX magic: %x", data[:4])
	}

	f := &DexFile{Data: data}

	// Parse header
	f.Header = Header{
		Checksum:      le32(data, 8),
		FileSize:      le32(data, 32),
		HeaderSize:    le32(data, 36),
		EndianTag:     le32(data, 40),
		LinkSize:      le32(data, 44),
		LinkOff:       le32(data, 48),
		MapOff:        le32(data, 52),
		StringIdsSize: le32(data, 56),
		StringIdsOff:  le32(data, 60),
		TypeIdsSize:   le32(data, 64),
		TypeIdsOff:    le32(data, 68),
		ProtoIdsSize:  le32(data, 72),
		ProtoIdsOff:   le32(data, 76),
		FieldIdsSize:  le32(data, 80),
		FieldIdsOff:   le32(data, 84),
		MethodIdsSize: le32(data, 88),
		MethodIdsOff:  le32(data, 92),
		ClassDefsSize: le32(data, 96),
		ClassDefsOff:  le32(data, 100),
		DataSize:      le32(data, 104),
		DataOff:       le32(data, 108),
	}
	copy(f.Header.Magic[:], data[:8])
	copy(f.Header.Signature[:], data[12:32])

	// Parse string IDs
	f.StringIDs = make([]StringID, f.Header.StringIdsSize)
	off := int(f.Header.StringIdsOff)
	for i := range f.StringIDs {
		f.StringIDs[i].DataOff = le32(data, off)
		off += 4
	}

	// Parse type IDs
	f.TypeIDs = make([]TypeID, f.Header.TypeIdsSize)
	off = int(f.Header.TypeIdsOff)
	for i := range f.TypeIDs {
		f.TypeIDs[i].DescriptorIdx = le32(data, off)
		off += 4
	}

	// Parse proto IDs
	f.ProtoIDs = make([]ProtoID, f.Header.ProtoIdsSize)
	off = int(f.Header.ProtoIdsOff)
	for i := range f.ProtoIDs {
		f.ProtoIDs[i].ShortyIdx = le32(data, off)
		f.ProtoIDs[i].ReturnTypeIdx = le32(data, off+4)
		f.ProtoIDs[i].ParametersOff = le32(data, off+8)
		off += 12
	}

	// Parse field IDs
	f.FieldIDs = make([]FieldID, f.Header.FieldIdsSize)
	off = int(f.Header.FieldIdsOff)
	for i := range f.FieldIDs {
		f.FieldIDs[i].ClassIdx = le16(data, off)
		f.FieldIDs[i].TypeIdx = le16(data, off+2)
		f.FieldIDs[i].NameIdx = le32(data, off+4)
		off += 8
	}

	// Parse method IDs
	f.MethodIDs = make([]MethodID, f.Header.MethodIdsSize)
	off = int(f.Header.MethodIdsOff)
	for i := range f.MethodIDs {
		f.MethodIDs[i].ClassIdx = le16(data, off)
		f.MethodIDs[i].ProtoIdx = le16(data, off+2)
		f.MethodIDs[i].NameIdx = le32(data, off+4)
		off += 8
	}

	// Parse class defs
	f.ClassDefs = make([]ClassDef, f.Header.ClassDefsSize)
	off = int(f.Header.ClassDefsOff)
	for i := range f.ClassDefs {
		f.ClassDefs[i].ClassIdx = le32(data, off)
		f.ClassDefs[i].AccessFlags = le32(data, off+4)
		f.ClassDefs[i].SuperclassIdx = le32(data, off+8)
		f.ClassDefs[i].InterfacesOff = le32(data, off+12)
		f.ClassDefs[i].SourceFileIdx = le32(data, off+16)
		f.ClassDefs[i].AnnotationsOff = le32(data, off+20)
		f.ClassDefs[i].ClassDataOff = le32(data, off+24)
		f.ClassDefs[i].StaticValuesOff = le32(data, off+28)
		off += 32
	}

	// Pre-cache all strings
	f.strings = make([]string, len(f.StringIDs))
	for i, sid := range f.StringIDs {
		f.strings[i] = f.readMUTF8(int(sid.DataOff))
	}

	return f, nil
}

// GetString returns the string at the given index.
func (f *DexFile) GetString(idx uint32) string {
	if idx >= uint32(len(f.strings)) {
		return ""
	}
	return f.strings[idx]
}

// GetTypeDescriptor returns the type descriptor string for the given type index.
func (f *DexFile) GetTypeDescriptor(typeIdx uint32) string {
	if typeIdx >= uint32(len(f.TypeIDs)) {
		return ""
	}
	return f.GetString(f.TypeIDs[typeIdx].DescriptorIdx)
}

// GetMethodName returns the method name for the given method index.
func (f *DexFile) GetMethodName(methodIdx uint32) string {
	if methodIdx >= uint32(len(f.MethodIDs)) {
		return ""
	}
	return f.GetString(f.MethodIDs[methodIdx].NameIdx)
}

// GetFieldName returns the field name for the given field index.
func (f *DexFile) GetFieldName(fieldIdx uint32) string {
	if fieldIdx >= uint32(len(f.FieldIDs)) {
		return ""
	}
	return f.GetString(f.FieldIDs[fieldIdx].NameIdx)
}

// GetFieldTypeDescriptor returns the type descriptor of a field.
func (f *DexFile) GetFieldTypeDescriptor(fieldIdx uint32) string {
	if fieldIdx >= uint32(len(f.FieldIDs)) {
		return ""
	}
	return f.GetTypeDescriptor(uint32(f.FieldIDs[fieldIdx].TypeIdx))
}

// GetFieldClassDescriptor returns the class descriptor of a field.
func (f *DexFile) GetFieldClassDescriptor(fieldIdx uint32) string {
	if fieldIdx >= uint32(len(f.FieldIDs)) {
		return ""
	}
	return f.GetTypeDescriptor(uint32(f.FieldIDs[fieldIdx].ClassIdx))
}

// GetMethodClassDescriptor returns the class descriptor for a method.
func (f *DexFile) GetMethodClassDescriptor(methodIdx uint32) string {
	if methodIdx >= uint32(len(f.MethodIDs)) {
		return ""
	}
	return f.GetTypeDescriptor(uint32(f.MethodIDs[methodIdx].ClassIdx))
}

// GetMethodSignature returns the full method signature string like "(I)V".
func (f *DexFile) GetMethodSignature(methodIdx uint32) string {
	if methodIdx >= uint32(len(f.MethodIDs)) {
		return ""
	}
	protoIdx := f.MethodIDs[methodIdx].ProtoIdx
	if uint32(protoIdx) >= uint32(len(f.ProtoIDs)) {
		return ""
	}
	proto := f.ProtoIDs[protoIdx]

	var sb strings.Builder
	sb.WriteByte('(')

	// Parameters
	if proto.ParametersOff != 0 {
		params := f.readTypeList(int(proto.ParametersOff))
		for _, typeIdx := range params.Items {
			sb.WriteString(f.GetTypeDescriptor(uint32(typeIdx)))
		}
	}

	sb.WriteByte(')')
	sb.WriteString(f.GetTypeDescriptor(proto.ReturnTypeIdx))
	return sb.String()
}

// GetApiMethodName returns the full API name like "Lclass;->method(params)RetType".
func (f *DexFile) GetApiMethodName(methodIdx uint32) string {
	cls := f.GetMethodClassDescriptor(methodIdx)
	name := f.GetMethodName(methodIdx)
	sig := f.GetMethodSignature(methodIdx)
	return cls + "->" + name + sig
}

// GetApiFieldName returns the full API name like "Lclass;->field:Type".
func (f *DexFile) GetApiFieldName(fieldIdx uint32) string {
	cls := f.GetFieldClassDescriptor(fieldIdx)
	name := f.GetFieldName(fieldIdx)
	typ := f.GetFieldTypeDescriptor(fieldIdx)
	return cls + "->" + name + ":" + typ
}

// GetInterfacesList returns the interfaces for a class def.
func (f *DexFile) GetInterfacesList(cd *ClassDef) *TypeList {
	if cd.InterfacesOff == 0 {
		return nil
	}
	tl := f.readTypeList(int(cd.InterfacesOff))
	return &tl
}

// GetClassData parses the class_data_item for a class def.
func (f *DexFile) GetClassData(cd *ClassDef) *ClassData {
	if cd.ClassDataOff == 0 {
		return nil
	}
	return f.readClassData(int(cd.ClassDataOff))
}

// GetCodeItem parses the code_item for a method.
func (f *DexFile) GetCodeItem(codeOff uint32) *CodeItem {
	if codeOff == 0 {
		return nil
	}
	return f.readCodeItem(int(codeOff))
}

// NumStringIDs returns the number of string IDs.
func (f *DexFile) NumStringIDs() uint32 { return f.Header.StringIdsSize }

// NumTypeIDs returns the number of type IDs.
func (f *DexFile) NumTypeIDs() uint32 { return f.Header.TypeIdsSize }

// NumMethodIDs returns the number of method IDs.
func (f *DexFile) NumMethodIDs() uint32 { return f.Header.MethodIdsSize }

// NumFieldIDs returns the number of field IDs.
func (f *DexFile) NumFieldIDs() uint32 { return f.Header.FieldIdsSize }

// --- Internal parsing helpers ---

func (f *DexFile) readTypeList(off int) TypeList {
	size := le32(f.Data, off)
	items := make([]uint16, size)
	for i := uint32(0); i < size; i++ {
		items[i] = le16(f.Data, off+4+int(i)*2)
	}
	return TypeList{Size: size, Items: items}
}

// readMUTF8 reads a MUTF-8 string at the given offset.
// Format: uleb128 length (in UTF-16 code units), then MUTF-8 bytes, then \0.
func (f *DexFile) readMUTF8(off int) string {
	// Skip the uleb128 utf16_size
	_, n := decodeULEB128(f.Data[off:])
	off += n

	// Read bytes until null terminator
	var buf []byte
	for off < len(f.Data) && f.Data[off] != 0 {
		b := f.Data[off]
		if b < 0x80 {
			buf = append(buf, b)
			off++
		} else if b&0xE0 == 0xC0 {
			// 2-byte sequence
			if off+1 < len(f.Data) {
				if b == 0xC0 && f.Data[off+1] == 0x80 {
					// Encoded null
					buf = append(buf, 0)
				} else {
					buf = append(buf, b, f.Data[off+1])
				}
			}
			off += 2
		} else if b&0xF0 == 0xE0 {
			// 3-byte sequence
			if off+2 < len(f.Data) {
				buf = append(buf, b, f.Data[off+1], f.Data[off+2])
			}
			off += 3
		} else {
			buf = append(buf, b)
			off++
		}
	}
	return string(buf)
}

func (f *DexFile) readCodeItem(off int) *CodeItem {
	ci := &CodeItem{
		RegistersSize: le16(f.Data, off),
		InsSize:       le16(f.Data, off+2),
		OutsSize:      le16(f.Data, off+4),
		TriesSize:     le16(f.Data, off+6),
		DebugInfoOff:  le32(f.Data, off+8),
		InsnsSize:     le32(f.Data, off+12),
	}

	// Read instructions
	insnsOff := off + 16
	ci.Insns = make([]uint16, ci.InsnsSize)
	for i := uint32(0); i < ci.InsnsSize; i++ {
		ci.Insns[i] = le16(f.Data, insnsOff+int(i)*2)
	}

	// Read tries and handlers
	if ci.TriesSize > 0 {
		triesOff := insnsOff + int(ci.InsnsSize)*2
		// Align to 4 bytes if insns_size is odd
		if ci.InsnsSize%2 != 0 {
			triesOff += 2
		}

		ci.Tries = make([]TryItem, ci.TriesSize)
		for i := uint16(0); i < ci.TriesSize; i++ {
			ci.Tries[i].StartAddr = le32(f.Data, triesOff)
			ci.Tries[i].InsnCount = le16(f.Data, triesOff+4)
			ci.Tries[i].HandlerOff = le16(f.Data, triesOff+6)
			triesOff += 8
		}

		// Parse encoded_catch_handler_list
		handlersOff := triesOff
		handlerListSize, n := decodeULEB128(f.Data[handlersOff:])
		handlersOff += n

		ci.Handlers = make([]EncodedCatchHandler, handlerListSize)
		for i := uint32(0); i < handlerListSize; i++ {
			size, n := decodeSLEB128(f.Data[handlersOff:])
			handlersOff += n

			catchAll := int32(-1)
			count := size
			if count <= 0 {
				count = -count
			}

			handler := EncodedCatchHandler{CatchAllAddr: -1}
			handler.Handlers = make([]EncodedTypeAddrPair, count)
			for j := int32(0); j < count; j++ {
				typeIdx, n := decodeULEB128(f.Data[handlersOff:])
				handlersOff += n
				addr, n := decodeULEB128(f.Data[handlersOff:])
				handlersOff += n
				handler.Handlers[j] = EncodedTypeAddrPair{TypeIdx: typeIdx, Addr: addr}
			}

			if size <= 0 {
				catchAllVal, n := decodeULEB128(f.Data[handlersOff:])
				handlersOff += n
				catchAll = int32(catchAllVal)
			}
			handler.CatchAllAddr = catchAll
			ci.Handlers[i] = handler
		}
	}

	return ci
}

// --- Little-endian helpers ---

func le16(data []byte, off int) uint16 {
	return binary.LittleEndian.Uint16(data[off:])
}

func le32(data []byte, off int) uint32 {
	return binary.LittleEndian.Uint32(data[off:])
}

// --- LEB128 decoding ---

func decodeULEB128(data []byte) (uint32, int) {
	var result uint32
	var shift uint
	for i := 0; i < len(data) && i < 5; i++ {
		b := data[i]
		result |= uint32(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, i + 1
		}
		shift += 7
	}
	return result, 5
}

func decodeSLEB128(data []byte) (int32, int) {
	var result int32
	var shift uint
	var i int
	for i = 0; i < len(data) && i < 5; i++ {
		b := data[i]
		result |= int32(b&0x7F) << shift
		shift += 7
		if b&0x80 == 0 {
			if shift < 32 && b&0x40 != 0 {
				result |= -(1 << shift)
			}
			return result, i + 1
		}
	}
	return result, i
}

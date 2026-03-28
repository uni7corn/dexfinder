package dex

// ClassData represents a decoded class_data_item.
type ClassData struct {
	StaticFields   []EncodedField
	InstanceFields []EncodedField
	DirectMethods  []EncodedMethod
	VirtualMethods []EncodedMethod
}

// EncodedField represents a field in class_data_item.
type EncodedField struct {
	FieldIdx    uint32
	AccessFlags uint32
}

// EncodedMethod represents a method in class_data_item.
type EncodedMethod struct {
	MethodIdx   uint32
	AccessFlags uint32
	CodeOff     uint32
}

// readClassData parses class_data_item at the given offset.
func (f *DexFile) readClassData(off int) *ClassData {
	data := f.Data[off:]
	pos := 0

	staticFieldsSize, n := decodeULEB128(data[pos:])
	pos += n
	instanceFieldsSize, n := decodeULEB128(data[pos:])
	pos += n
	directMethodsSize, n := decodeULEB128(data[pos:])
	pos += n
	virtualMethodsSize, n := decodeULEB128(data[pos:])
	pos += n

	cd := &ClassData{}

	// Static fields
	cd.StaticFields = make([]EncodedField, staticFieldsSize)
	var fieldIdx uint32
	for i := uint32(0); i < staticFieldsSize; i++ {
		diff, n := decodeULEB128(data[pos:])
		pos += n
		flags, n := decodeULEB128(data[pos:])
		pos += n
		fieldIdx += diff
		cd.StaticFields[i] = EncodedField{FieldIdx: fieldIdx, AccessFlags: flags}
	}

	// Instance fields
	cd.InstanceFields = make([]EncodedField, instanceFieldsSize)
	fieldIdx = 0
	for i := uint32(0); i < instanceFieldsSize; i++ {
		diff, n := decodeULEB128(data[pos:])
		pos += n
		flags, n := decodeULEB128(data[pos:])
		pos += n
		fieldIdx += diff
		cd.InstanceFields[i] = EncodedField{FieldIdx: fieldIdx, AccessFlags: flags}
	}

	// Direct methods
	cd.DirectMethods = make([]EncodedMethod, directMethodsSize)
	var methodIdx uint32
	for i := uint32(0); i < directMethodsSize; i++ {
		diff, n := decodeULEB128(data[pos:])
		pos += n
		flags, n := decodeULEB128(data[pos:])
		pos += n
		codeOff, n := decodeULEB128(data[pos:])
		pos += n
		methodIdx += diff
		cd.DirectMethods[i] = EncodedMethod{MethodIdx: methodIdx, AccessFlags: flags, CodeOff: codeOff}
	}

	// Virtual methods
	cd.VirtualMethods = make([]EncodedMethod, virtualMethodsSize)
	methodIdx = 0
	for i := uint32(0); i < virtualMethodsSize; i++ {
		diff, n := decodeULEB128(data[pos:])
		pos += n
		flags, n := decodeULEB128(data[pos:])
		pos += n
		codeOff, n := decodeULEB128(data[pos:])
		pos += n
		methodIdx += diff
		cd.VirtualMethods[i] = EncodedMethod{MethodIdx: methodIdx, AccessFlags: flags, CodeOff: codeOff}
	}

	return cd
}

// AllFields returns both static and instance fields.
func (cd *ClassData) AllFields() []EncodedField {
	result := make([]EncodedField, 0, len(cd.StaticFields)+len(cd.InstanceFields))
	result = append(result, cd.StaticFields...)
	result = append(result, cd.InstanceFields...)
	return result
}

// AllMethods returns both direct and virtual methods.
func (cd *ClassData) AllMethods() []EncodedMethod {
	result := make([]EncodedMethod, 0, len(cd.DirectMethods)+len(cd.VirtualMethods))
	result = append(result, cd.DirectMethods...)
	result = append(result, cd.VirtualMethods...)
	return result
}

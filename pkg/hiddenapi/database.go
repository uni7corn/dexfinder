package hiddenapi

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// SignatureSource indicates where a class was defined.
type SignatureSource uint8

const (
	SourceUnknown SignatureSource = iota
	SourceBoot
	SourceApp
)

// Database stores the hidden API flags and signature sources.
type Database struct {
	apiList map[string]ApiList
	source  map[string]SignatureSource
	filter  *ApiListFilter
	// classMembers: class descriptor → set of member short names (method/field names without signatures)
	// Built lazily on first call to GetMembersOfClass.
	classMembers map[string]map[string]bool
}

// NewDatabase creates an empty database with the given filter.
func NewDatabase(filter *ApiListFilter) *Database {
	return &Database{
		apiList: make(map[string]ApiList),
		source:  make(map[string]SignatureSource),
		filter:  filter,
	}
}

// LoadFromFile loads hiddenapi-flags.csv into the database.
// Each line: signature,flag1,flag2,...
func (db *Database) LoadFromFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open flags file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		signature := parts[0]
		membership, ok := ApiListFromNames(parts[1:])
		if !ok || !membership.IsValid() {
			// Skip entries with unknown flags rather than failing
			// (newer CSV versions may introduce flags we don't know yet)
			continue
		}

		db.addSignature(signature, membership)

		// Also index partial signatures for prefix matching
		if pos := strings.Index(signature, "->"); pos != -1 {
			className := signature[:pos]
			// Add class name
			db.addSignature(className, membership)
			// Mark class as BOOT (all classes in CSV come from boot classpath)
			db.AddSignatureSource(className, SourceBoot)

			// Add class->method (without param signature)
			if paren := strings.Index(signature, "("); paren != -1 {
				db.addSignature(signature[:paren], membership)
			}

			// Add class->field (without type)
			if colon := strings.Index(signature, ":"); colon != -1 {
				db.addSignature(signature[:colon], membership)
			}
		}
	}

	return scanner.Err()
}

func (db *Database) addSignature(signature string, membership ApiList) {
	existing, ok := db.apiList[signature]
	if !ok {
		db.apiList[signature] = membership
	} else if membership.GetMaxAllowedSdkVersion() < existing.GetMaxAllowedSdkVersion() {
		// More restrictive wins
		db.apiList[signature] = membership
	}
}

// GetApiList returns the API list for a signature.
func (db *Database) GetApiList(signature string) ApiList {
	if v, ok := db.apiList[signature]; ok {
		return v
	}
	return Invalid
}

// ShouldReport returns true if the signature should be reported.
func (db *Database) ShouldReport(signature string) bool {
	return db.filter.Matches(db.GetApiList(signature))
}

// AddSignatureSource records whether a class descriptor comes from BOOT or APP.
func (db *Database) AddSignatureSource(signature string, source SignatureSource) {
	cls := getApiClassName(signature)
	existing, ok := db.source[cls]
	if !ok || existing == SourceUnknown {
		db.source[cls] = source
	} else if existing != source && source == SourceBoot {
		// Boot takes precedence
		db.source[cls] = source
	}
}

// GetSignatureSource returns the source of a signature's class.
func (db *Database) GetSignatureSource(signature string) SignatureSource {
	cls := getApiClassName(signature)
	if v, ok := db.source[cls]; ok {
		return v
	}
	return SourceUnknown
}

// IsInBoot returns true if the signature's class is from the boot classpath.
func (db *Database) IsInBoot(signature string) bool {
	return db.GetSignatureSource(signature) == SourceBoot
}

// Size returns the number of entries in the API list.
func (db *Database) Size() int {
	return len(db.apiList)
}

// GetMembersOfClass returns all known member names (without signatures) for a class.
// Returns nil if the class is not in the database.
func (db *Database) GetMembersOfClass(classDesc string) map[string]bool {
	if db.classMembers == nil {
		db.buildClassMembers()
	}
	return db.classMembers[classDesc]
}

func (db *Database) buildClassMembers() {
	db.classMembers = make(map[string]map[string]bool)
	for sig := range db.apiList {
		cls := getApiClassName(sig)
		member := getApiMemberName(sig)
		if member == "" {
			continue
		}
		members, ok := db.classMembers[cls]
		if !ok {
			members = make(map[string]bool)
			db.classMembers[cls] = members
		}
		members[member] = true
	}
}

// getApiClassName extracts the class descriptor from a full signature.
// "Lcom/foo/Bar;->method(I)V" → "Lcom/foo/Bar;"
func getApiClassName(signature string) string {
	if pos := strings.Index(signature, "->"); pos != -1 {
		return signature[:pos]
	}
	return signature
}

// getApiMemberName extracts the member name (without params) from a full signature.
// "Lcom/foo/Bar;->method(I)V" → "method"
// "Lcom/foo/Bar;->field:I" → "field"
func getApiMemberName(signature string) string {
	arrowPos := strings.Index(signature, "->")
	if arrowPos == -1 {
		return ""
	}
	member := signature[arrowPos+2:]
	// Strip params: "method(I)V" → "method"
	if parenPos := strings.Index(member, "("); parenPos != -1 {
		return member[:parenPos]
	}
	// Strip field type: "field:I" → "field"
	if colonPos := strings.Index(member, ":"); colonPos != -1 {
		return member[:colonPos]
	}
	return member
}

// ToInternalName converts a Java class name to internal format.
// "com.foo.Bar" → "Lcom/foo/Bar;"
func ToInternalName(name string) string {
	return "L" + strings.ReplaceAll(name, ".", "/") + ";"
}

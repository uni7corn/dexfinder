package hiddenapi

import "fmt"

// ApiList represents the restriction level of a hidden API.
type ApiList uint8

const (
	Sdk            ApiList = iota // Public SDK API
	Unsupported                  // Unsupported (greylist)
	MaxTargetO                   // max-target-o (greylist-max-o)
	MaxTargetP                   // max-target-p (greylist-max-p)
	MaxTargetQ                   // max-target-q (greylist-max-q)
	MaxTargetR                   // max-target-r (greylist-max-r)
	MaxTargetS                   // max-target-s
	Blocked                      // Blocked (blacklist)
	NumApiLists                  // sentinel: count of valid values
	Invalid    ApiList = 0xFF    // not found in any list
)

// String returns the human-readable name.
func (a ApiList) String() string {
	switch a {
	case Sdk:
		return "sdk"
	case Unsupported:
		return "unsupported"
	case MaxTargetO:
		return "max-target-o"
	case MaxTargetP:
		return "max-target-p"
	case MaxTargetQ:
		return "max-target-q"
	case MaxTargetR:
		return "max-target-r"
	case MaxTargetS:
		return "max-target-s"
	case Blocked:
		return "blocked"
	case Invalid:
		return "invalid"
	default:
		return fmt.Sprintf("unknown(%d)", a)
	}
}

// GetMaxAllowedSdkVersion returns the maximum target SDK version that can use this API.
func (a ApiList) GetMaxAllowedSdkVersion() int {
	switch a {
	case Sdk:
		return 9999
	case Unsupported:
		return 9999 // allowed but unsupported
	case MaxTargetO:
		return 26
	case MaxTargetP:
		return 28
	case MaxTargetQ:
		return 29
	case MaxTargetR:
		return 30
	case MaxTargetS:
		return 31
	case Blocked:
		return 0
	default:
		return 0
	}
}

// IsValid returns true if this is a valid API list value.
func (a ApiList) IsValid() bool {
	return a < NumApiLists
}

// nameToApiList maps CSV flag names to ApiList values.
var nameToApiList = map[string]ApiList{
	"sdk":             Sdk,
	"whitelist":       Sdk,
	"unsupported":     Unsupported,
	"greylist":        Unsupported,
	"max-target-o":    MaxTargetO,
	"greylist-max-o":  MaxTargetO,
	"max-target-p":    MaxTargetP,
	"greylist-max-p":  MaxTargetP,
	"max-target-q":    MaxTargetQ,
	"greylist-max-q":  MaxTargetQ,
	"max-target-r":    MaxTargetR,
	"greylist-max-r":  MaxTargetR,
	"max-target-s":    MaxTargetS,
	"greylist-max-s":  MaxTargetS,
	"blocked":         Blocked,
	"blacklist":       Blocked,
	// These flags are informational, not restriction levels.
	// They may appear alongside a restriction flag.
	"public-api":        Sdk,
	"system-api":        Sdk,
	"test-api":          Sdk,
	"lo-prio":           Unsupported,
	"core-platform-api": Sdk,
}

// ApiListFromNames parses flag names from CSV and returns the most restrictive ApiList.
func ApiListFromNames(names []string) (ApiList, bool) {
	result := Invalid
	for _, name := range names {
		name = trimSpace(name)
		if name == "" {
			continue
		}
		val, ok := nameToApiList[name]
		if !ok {
			return Invalid, false
		}
		if result == Invalid || val.GetMaxAllowedSdkVersion() < result.GetMaxAllowedSdkVersion() {
			result = val
		}
	}
	if result == Invalid {
		return Invalid, false
	}
	return result, true
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\r' || s[j-1] == '\n') {
		j--
	}
	return s[i:j]
}

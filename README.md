# dexfinder

Cross-platform APK/DEX method & field reference finder with call chain tracing, ProGuard/R8 deobfuscation, and Android hidden API detection.

Inspired by Android's [veridex](https://android.googlesource.com/platform/art/+/refs/heads/master/tools/veridex/) tool, reimplemented in Go with enhanced capabilities.

## Features

- **APK/DEX/JAR scanning** — Parse DEX bytecode, extract all method/field/string references
- **Multi-format query** — Search by Java name, DEX/JNI signature, or simple keyword
- **Call chain tracing** — Trace callers up to N levels deep, with cycle detection
- **ProGuard/R8 deobfuscation** — Load mapping.txt, display original names
- **Hidden API detection** — Load hiddenapi-flags.csv, detect blocked/unsupported APIs
- **Reflection detection** — Cross-match classes × strings to find reflection-based hidden API usage (veridex-compatible)
- **Multiple output formats** — text, json, stacktrace (Java crash style), model (structured JSON for IDE/CI)
- **Zero external dependencies** — Pure Go, self-contained DEX parser
- **Cross-platform** — macOS (Intel/Apple Silicon), Linux (amd64/arm64), Windows

## Install

### Homebrew (macOS/Linux)

```bash
brew tap JuneLeGency/tap
brew install dexfinder
```

### Script

```bash
curl -sSL https://raw.githubusercontent.com/JuneLeGency/dexfinder/main/install.sh | bash
```

### Go

```bash
go install dex_method_finder/cmd/dexfinder@latest
```

### Binary

Download from [Releases](https://github.com/JuneLeGency/dexfinder/releases).

## Quick Start

```bash
# Scan APK and show stats
dexfinder --dex-file app.apk --stats

# Find all calls to a method
dexfinder --dex-file app.apk --query "requestLocationUpdates"

# Trace call chains
dexfinder --dex-file app.apk --query "requestLocationUpdates" --trace --depth 8

# Java crash stacktrace style
dexfinder --dex-file app.apk --query "requestLocationUpdates" --trace --format stacktrace
```

## Usage

```
dexfinder --dex-file <path> [options]
```

### Query Formats

| Format | Example | Behavior |
|---|---|---|
| Simple name | `requestLocationUpdates` | Fuzzy substring match |
| Java class | `android.location.LocationManager` | Match all methods of class |
| Java class#method | `...LocationManager#requestLocationUpdates` | Match all overloads |
| Java full sig | `...#requestLocationUpdates(java.lang.String, long, float, ...)` | Exact match |
| DEX/JNI sig | `Landroid/location/LocationManager;->requestLocationUpdates(...)V` | Exact match |

### Output Formats (`--format`)

| Format | Description |
|---|---|
| `text` | Plain text with `[METHOD]`/`[FIELD]`/`[STRING]` tags (default) |
| `json` | Simple JSON |
| `stacktrace` | Java crash stacktrace style (best with `--trace`) |
| `model` | Structured JSON with full type info (for IDE/CI) |

### Search Scope (`--scope`)

| Scope | Searches |
|---|---|
| `all` | Methods + fields + code strings (default) |
| `callee` | Only target API signatures |
| `caller` | Only calling method signatures |
| `string` | Only string constants in code |
| `string-table` | Code strings + full DEX string table |
| `everything` | All of the above |

### Deobfuscation (`--mapping`)

```bash
# Show deobfuscated names
dexfinder --dex-file app.apk --query "requestLocationUpdates" --mapping mapping.txt

# Show both obfuscated and original names
dexfinder --dex-file app.apk --query "requestLocationUpdates" --mapping mapping.txt --show-obf
```

### Hidden API Detection (`--api-flags`)

```bash
# Download the CSV (one-time)
curl -o hiddenapi-flags.csv https://dl.google.com/developers/android/baklava/non-sdk/hiddenapi-flags.csv

# Detect hidden API usage (linking + reflection)
dexfinder --dex-file app.apk --api-flags hiddenapi-flags.csv
```

### Structured Output (`--format model`)

```bash
# Full structured output for CI/IDE integration
dexfinder --dex-file app.apk --query "LocationManager" --trace --format model | jq '.call_chains'
```

Output includes:
- `metadata` — file info, query, options
- `method_calls` — target method + caller locations
- `field_accesses` — target field + accessor locations
- `string_refs` — string constants + usage locations
- `hidden_apis` — hidden API findings with restriction level
- `reflection_findings` — potential reflection-based access
- `call_chains` — traced call chains with cycle detection
- `summary` — aggregate counts

## Examples

### Find all location API usage in DingTalk

```bash
dexfinder --dex-file dingtalk.apk \
  --query "android.location.LocationManager#requestLocationUpdates" \
  --trace --format stacktrace
```

Output:
```
--- Call chain #1 for android.location.LocationManager.requestLocationUpdates(String, long, float, LocationListener, Looper) ---
    at com.amap.api.col.3sl.d$a.handleMessage(d.java)
    at com.amap.api.col.3sl.g.a(g.java)
    at com.amap.api.col.3sl.g.g(g.java)
    at com.alibaba.wireless.security.aopsdk.replace.android.location.LocationManager.requestLocationUpdates(LocationManager.java)
    at android.location.LocationManager.requestLocationUpdates(LocationManager.java)
```

### Detect hidden API via reflection

```bash
dexfinder --dex-file dingtalk.apk --api-flags hiddenapi-flags.csv 2>/dev/null | grep ILocationManager
```

Output:
```
#1769: Reflection blocked Landroid/location/ILocationManager;->getLastLocation potential use(s):
```

### Search content:// URIs

```bash
dexfinder --dex-file app.apk --query "content://com.android.contacts" --scope everything
```

## Performance

| APK | Size | DEX | Classes | Methods | Time |
|---|---|---|---|---|---|
| AppSearch | 1.5MB | 1 | 1,708 | 17,652 | **24ms** |
| Tinker | 8.7MB | 2 | 24,937 | 99,359 | **335ms** |
| DingTalk | 285MB | 31 | 183,913 | 1,250,566 | **3.9s** (scan) / **5.4s** (hidden API) |

## Building from Source

```bash
git clone https://github.com/JuneLeGency/dexfinder.git
cd dexfinder
go build -o dexfinder ./cmd/dexfinder/
go test ./...
```

## License

Apache License 2.0

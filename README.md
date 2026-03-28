# dexfinder

Cross-platform APK/DEX method & field reference finder with call chain tracing, ProGuard/R8 deobfuscation, and Android hidden API detection.

Inspired by Android's [veridex](https://android.googlesource.com/platform/art/+/refs/heads/master/tools/veridex/) tool, reimplemented in Go with enhanced capabilities.

## Features

- **APK/DEX/JAR scanning** — Parse DEX bytecode, extract all method/field/string references
- **Multi-format query** — Search by Java name, DEX/JNI signature, or simple keyword
- **Call chain tracing** — Trace callers up to N levels deep, merged tree or flat list, with cycle detection
- **ProGuard/R8 deobfuscation** — Load mapping.txt, display original names
- **Hidden API detection** — Load hiddenapi-flags.csv, detect blocked/unsupported APIs
- **Reflection detection** — Cross-match classes × strings to find reflection-based hidden API usage (veridex-compatible)
- **Flexible output** — text / json / model (structured), tree / list layout, java / dex name style
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
go install github.com/JuneLeGency/dexfinder/cmd/dexfinder@latest
```

### Binary

Download from [Releases](https://github.com/JuneLeGency/dexfinder/releases).

## Quick Start

```bash
# Scan APK and show stats
dexfinder --dex-file app.apk --stats

# Find all calls to a method
dexfinder --dex-file app.apk --query "getDeviceId"

# Trace call chains (merged tree, Java readable)
dexfinder --dex-file app.apk --query "getDeviceId" --trace --depth 5

# Trace as flat list
dexfinder --dex-file app.apk --query "getDeviceId" --trace --layout list

# Exact JNI signature query
dexfinder --dex-file app.apk \
  --query "Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;" \
  --trace --depth 5
```

## Usage

```
dexfinder --dex-file <path> [options]
```

### Query Formats (`--query`)

| Format | Example | Behavior |
|---|---|---|
| Simple name | `getDeviceId` | Fuzzy substring match |
| Java class | `android.telephony.TelephonyManager` | Match all methods of class |
| Java class#method | `android.telephony.TelephonyManager#getDeviceId` | Match all overloads |
| Java full sig | `...#getDeviceId()` | Exact match + overload fallback |
| DEX/JNI sig | `Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;` | Exact match |

### Output Format (`--format`)

| Format | Description |
|---|---|
| `text` | Plain text (default) |
| `json` | JSON output (supports `--trace` with tree/list layout) |
| `model` | Structured JSON with full type info, for IDE/CI integration |

### Trace Layout (`--layout`)

Controls how call chains are rendered when using `--trace`:

| Layout | Description |
|---|---|
| `tree` | Merged tree — shared paths collapsed (default) |
| `list` | Flat list — each chain shown independently |

### Name Style (`--style`)

Controls how method/class names are displayed:

| Style | Example |
|---|---|
| `java` | `com.example.Foo.method(Foo.java)` (default) |
| `dex` | `Foo.method(Ljava/lang/String;)V` |

`--format`, `--layout`, `--style` are orthogonal — any combination works:

```
--format=text/json/model  ×  --layout=tree/list  ×  --style=java/dex
```

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
dexfinder --dex-file app.apk --query "getDeviceId" --trace --mapping mapping.txt

# Show both obfuscated and original names
dexfinder --dex-file app.apk --query "getDeviceId" --trace --mapping mapping.txt --show-obf
```

### Hidden API Detection (`--api-flags`)

```bash
# Download the CSV (one-time)
curl -o hiddenapi-flags.csv \
  https://dl.google.com/developers/android/baklava/non-sdk/hiddenapi-flags.csv

# Detect hidden API usage (linking + reflection)
dexfinder --dex-file app.apk --api-flags hiddenapi-flags.csv
```

## Examples

### Trace call chains as merged tree

```bash
dexfinder --dex-file app.apk \
  --query "Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;" \
  --trace --depth 5
```

```
android.telephony.TelephonyManager.getDeviceId()
└── ...aopsdk...TelephonyManager.getDeviceId(TelephonyManager.java)
    ├── PhoneInfo.getImei(PhoneInfo.java)
    ├── ClientIdHelper.initClientId(ClientIdHelper.java)
    │   └── ContextInfo.<init>(ContextInfo.java)
    │       ├── LogStrategyManager.getInstance(...)
    │       └── LogContextImpl.<init>(...)
    ├── DeviceInfo.k(DeviceInfo.java)
    │   └── DeviceInfo.<init>() → getInstance()
    │       ├── TidHelper.getIMEI()
    │       └── TidHelper.getIMSI()
    └── ...
```

### Trace as flat list (Java crash style)

```bash
dexfinder --dex-file app.apk --query "getDeviceId" --trace --layout list
```

```
--- Call chain #1 for android.telephony.TelephonyManager.getDeviceId() ---
	at com.example.PhoneInfo.getImei(PhoneInfo.java)
	at com.example.TelephonyManager.getDeviceId(TelephonyManager.java)
	at android.telephony.TelephonyManager.getDeviceId(TelephonyManager.java)
```

### JSON tree output

```bash
dexfinder --dex-file app.apk --query "getDeviceId" --trace --format json
```

```json
{
  "targets": [{
    "api": "android.telephony.TelephonyManager.getDeviceId()",
    "tree": {
      "method": "...TelephonyManager.getDeviceId(TelephonyManager.java)",
      "callers": [
        { "method": "PhoneInfo.getImei(PhoneInfo.java)" },
        { "method": "ClientIdHelper.initClientId(ClientIdHelper.java)",
          "callers": [{ "method": "ContextInfo.<init>(...)" }] }
      ]
    }
  }]
}
```

### JSON list output

```bash
dexfinder --dex-file app.apk --query "getDeviceId" --trace --format json --layout list
```

```json
{
  "targets": [{
    "api": "android.telephony.TelephonyManager.getDeviceId()",
    "chains": [
      ["PhoneInfo.getImei(...)", "...TelephonyManager.getDeviceId(...)", "TelephonyManager.getDeviceId(...)"],
      ["ClientIdHelper.initClientId(...)", "...TelephonyManager.getDeviceId(...)", "TelephonyManager.getDeviceId(...)"]
    ]
  }]
}
```

### Search content:// URIs (including optimized-out strings)

```bash
dexfinder --dex-file app.apk --query "content://com.android.contacts" --scope everything
```

### Detect hidden API via reflection

```bash
dexfinder --dex-file app.apk --api-flags hiddenapi-flags.csv | grep ILocationManager
```

```
#135: Reflection blocked Landroid/location/ILocationManager;->getCurrentLocation potential use(s):
```

## Performance

Benchmarked on real-world APKs (Apple M-series, single thread):

| APK Size | DEX Files | Classes | Method Refs | Scan Time | Hidden API |
|---|---|---|---|---|---|
| ~1MB | 1 | ~2K | ~18K | **24ms** | — |
| ~10MB | 2 | ~25K | ~100K | **335ms** | — |
| ~300MB | 30+ | ~180K | ~1.2M | **3.9s** | **5.4s** |

## All Options

```
--dex-file        APK/DEX/JAR file to analyze (required)
--query           Search keyword
--trace           Trace call chains (requires --query)
--depth           Max call chain depth (default 5)
--layout          Trace layout: tree or list (default tree)
--style           Name style: java or dex (default java)
--format          Output format: text, json, model (default text)
--mapping         ProGuard/R8 mapping.txt for deobfuscation
--show-obf        Show obfuscated names alongside deobfuscated
--api-flags       Path to hiddenapi-flags.csv
--class-filter    Comma-separated class descriptor prefixes
--exclude-api-lists  API lists to exclude from reporting
--scope           Search scope: all, callee, caller, string, string-table, everything
--stats           Show summary statistics only
--version         Show version
```

## Building from Source

```bash
git clone https://github.com/JuneLeGency/dexfinder.git
cd dexfinder
go build -o dexfinder ./cmd/dexfinder/
go test ./...
```

## License

Apache License 2.0

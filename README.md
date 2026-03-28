# dexfinder

[English](#english) | [中文](#中文)

---

<a name="english"></a>

Cross-platform APK/DEX method & field reference finder with call chain tracing, ProGuard/R8 deobfuscation, and Android hidden API detection.

Inspired by Android's [veridex](https://android.googlesource.com/platform/art/+/refs/heads/master/tools/veridex/) tool, reimplemented in Go with enhanced capabilities: faster reflection detection, call chain tracing (veridex only shows one level), and flexible output formats.

## Features

- **APK/DEX/JAR scanning** — Parse DEX bytecode, extract all method/field/string references
- **Multi-format query** — Search by Java name, DEX/JNI signature, or simple keyword
- **Call chain tracing** — Trace callers up to N levels deep, merged tree or flat list, with cycle detection
- **ProGuard/R8 deobfuscation** — Load mapping.txt, display original names alongside obfuscated
- **Hidden API detection** — Load hiddenapi-flags.csv, detect blocked/unsupported APIs
- **Reflection detection** — Cross-match classes × strings to find reflection-based hidden API usage
- **Flexible output** — text / json / model, tree / list layout, java / dex name style — all orthogonal
- **Zero external dependencies** — Pure Go, self-contained DEX parser
- **Cross-platform** — macOS (Intel / Apple Silicon), Linux (amd64 / arm64), Windows

## Install

**Homebrew** (macOS / Linux):
```bash
brew tap JuneLeGency/tap
brew install dexfinder
```

**Script** (auto-detects OS/arch):
```bash
curl -sSL https://raw.githubusercontent.com/JuneLeGency/dexfinder/main/install.sh | bash
```

**Go install**:
```bash
go install github.com/JuneLeGency/dexfinder/cmd/dexfinder@latest
```

**Binary**: download from [Releases](https://github.com/JuneLeGency/dexfinder/releases).

## Quick Start

```bash
# Show APK overview
dexfinder --dex-file app.apk --stats

# Find all calls to getDeviceId (IMEI)
dexfinder --dex-file app.apk --query "getDeviceId"

# Trace call chains as merged tree
dexfinder --dex-file app.apk --query "getDeviceId" --trace

# Trace as flat call stacks (Java crash style)
dexfinder --dex-file app.apk --query "getDeviceId" --trace --layout list

# Exact JNI signature query
dexfinder --dex-file app.apk \
  --query "Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;" \
  --trace --depth 8

# Hidden API detection
dexfinder --dex-file app.apk --api-flags hiddenapi-flags.csv
```

## Query Formats

The `--query` flag accepts multiple input styles. dexfinder auto-detects and converts between them.

| Format | Example | Behavior |
|---|---|---|
| Simple name | `getDeviceId` | Fuzzy substring match across all APIs |
| Java class | `android.telephony.TelephonyManager` | All methods/fields of that class |
| Java class#method | `android.telephony.TelephonyManager#getDeviceId` | All overloads of that method |
| Java full signature | `...TelephonyManager#getDeviceId()` | Exact + overload fallback |
| DEX/JNI signature | `Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;` | Exact match only |

```bash
# All equivalent — find requestLocationUpdates in LocationManager:
dexfinder --dex-file app.apk --query "requestLocationUpdates"
dexfinder --dex-file app.apk --query "android.location.LocationManager#requestLocationUpdates"
dexfinder --dex-file app.apk --query "Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V"
```

## Output Control

Three independent axes, freely combinable:

```
--format  (text / json / model)    what to output
--layout  (tree / list)            how to arrange traces
--style   (java / dex)             how to display names
```

### `--format`

| Value | Description |
|---|---|
| `text` | Plain text output (default) |
| `json` | JSON — scan results or trace with tree/list layout |
| `model` | Structured JSON with full MethodInfo/FieldInfo types (for IDE/CI) |

### `--layout` (used with `--trace`)

| Value | Description |
|---|---|
| `tree` | Merged tree — shared call paths collapsed into one tree (default) |
| `list` | Flat list — each unique call chain shown as independent stack |

### `--style`

| Value | Example | Use case |
|---|---|---|
| `java` | `com.example.Foo.method(Foo.java)` | Human-readable (default) |
| `dex` | `Foo.method(Ljava/lang/String;)V` | Precise signature analysis |

### `--scope` (search scope)

Controls **what kind of references** the query matches against. This is critical for understanding results.

| Value | What it searches | Question it answers | Output tag |
|---|---|---|---|
| `all` | Callee APIs + fields + code strings | "Who calls this API?" (default) | `[METHOD]` `[FIELD]` `[STRING]` |
| `callee` | Only target API signatures in `invoke-*` / `get/put` instructions | "Who calls this specific method/field?" | `[METHOD]` `[FIELD]` |
| `caller` | Only the calling method's signature | "What does this method call internally?" | `[CALLER→]` |
| `string` | String constants in `const-string` instructions | "Where is this string used in code?" | `[STRING]` |
| `string-table` | Code strings + full DEX string table | "Does this string exist anywhere in DEX?" (includes annotations, dead code) | `[STRING]` `[STRING_TABLE]` |
| `everything` | All of the above combined | Full picture | all tags |

**Understanding callee vs caller:**

```
scope=callee: "Who calls finish()?"
    onCreate ──calls──→ finish()     ← these callers are shown
    onResume ──calls──→ finish()

scope=caller: "What does finish() call internally?"
    finish() ──calls──→ Log.i()      ← these callees are shown
    finish() ──calls──→ super.finish()
```

`--scope=all` (default) = `callee` + `string`. The `caller` direction is intentionally excluded from default because it answers a fundamentally different question. Use `--scope=caller` or `--scope=everything` explicitly when you need it.

**Understanding output tags:**

| Tag | Meaning |
|---|---|
| `[METHOD]` | A method **being called** matches your query (callee match). Indented lines are the callers. |
| `[FIELD]` | A field **being accessed** matches your query. Indented lines are the accessors. |
| `[CALLER→]` | A **calling method** matches your query. The indented line shows what API it's calling. |
| `[STRING]` | A string constant in code matches your query. Indented lines are where it's used. |
| `[STRING_TABLE]` | String exists in DEX string table but has no `const-string` reference in code (may be in annotations, optimized out by R8, etc.) |

## Examples

### 1. Scan APK statistics

```bash
dexfinder --dex-file app.apk --stats
```
```
Loaded 31 DEX file(s): 183913 classes, 1250566 method refs
Method references: 680610
Field references:  625572
String constants:  654353
Referenced types:  192586
Time: 3.9s
```

### 2. Find all location tracking calls

```bash
dexfinder --dex-file app.apk --query "requestLocationUpdates"
```
```
[METHOD] Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V (3 ref)
       Lcom/example/TestEntry;->init(Landroid/content/Context;)V (2 occurrences)
       Lcom/example/service/LocationService;->onStartCommand(Landroid/content/Intent;II)I
```

### 3. Trace call chains — tree view

```bash
dexfinder --dex-file app.apk \
  --query "Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;" \
  --trace --depth 5
```
```
android.telephony.TelephonyManager.getDeviceId()
└── com.example.aopsdk.TelephonyManager.getDeviceId(TelephonyManager.java)
    ├── com.example.session.PhoneInfo.getImei(PhoneInfo.java)
    ├── com.example.logging.ClientIdHelper.initClientId(ClientIdHelper.java)
    │   └── com.example.logging.ContextInfo.<init>(ContextInfo.java)
    │       ├── com.example.logging.LogStrategyManager.getInstance(LogStrategyManager.java)
    │       └── com.example.logging.LogContextImpl.<init>(LogContextImpl.java)
    ├── com.example.msp.DeviceInfo.k(DeviceInfo.java)
    │   └── com.example.msp.DeviceInfo.<init>(DeviceInfo.java)
    │       └── com.example.msp.DeviceInfo.getInstance(DeviceInfo.java)
    │           ├── com.example.msp.TidHelper.getIMEI(TidHelper.java)
    │           ├── com.example.msp.TidHelper.getIMSI(TidHelper.java)
    │           └── com.example.msp.DeviceCollector.collectData(DeviceCollector.java)
    └── com.example.weex.WXEnvironment.getDevId(WXEnvironment.java)
        └── com.example.weex.WXEnvironment.<clinit>(WXEnvironment.java)
```

### 4. Trace call chains — list view (Java crash style)

```bash
dexfinder --dex-file app.apk \
  --query "Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;" \
  --trace --depth 5 --layout list
```
```
--- Call chain #1 for android.telephony.TelephonyManager.getDeviceId() ---
	at com.example.session.PhoneInfo.getImei(PhoneInfo.java)
	at com.example.aopsdk.TelephonyManager.getDeviceId(TelephonyManager.java)
	at android.telephony.TelephonyManager.getDeviceId(TelephonyManager.java)

--- Call chain #2 for android.telephony.TelephonyManager.getDeviceId() ---
	at com.example.logging.LogStrategyManager.getInstance(LogStrategyManager.java)
	at com.example.logging.ContextInfo.<init>(ContextInfo.java)
	at com.example.logging.ClientIdHelper.initClientId(ClientIdHelper.java)
	at com.example.aopsdk.TelephonyManager.getDeviceId(TelephonyManager.java)
	at android.telephony.TelephonyManager.getDeviceId(TelephonyManager.java)
```

### 5. Trace with DEX signature style

```bash
dexfinder --dex-file app.apk --query "getDeviceId" --trace --depth 3 --style dex
```
```
Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;
└── TelephonyManager.getDeviceId(Landroid/telephony/TelephonyManager;)Ljava/lang/String;
    ├── PhoneInfo.getImei(Landroid/content/Context;)Ljava/lang/String;
    ├── ClientIdHelper.initClientId(Landroid/content/Context;)Ljava/lang/String;
    └── DeviceInfo.k(Landroid/content/Context;)V
```

### 6. JSON output — tree

```bash
dexfinder --dex-file app.apk --query "getDeviceId" --trace --depth 2 --format json
```
```json
{
  "targets": [{
    "api": "android.telephony.TelephonyManager.getDeviceId()",
    "tree": {
      "method": "android.telephony.TelephonyManager.getDeviceId(TelephonyManager.java)",
      "callers": [
        { "method": "com.example.aopsdk.TelephonyManager.getDeviceId(TelephonyManager.java)",
          "callers": [
            { "method": "com.example.session.PhoneInfo.getImei(PhoneInfo.java)" },
            { "method": "com.example.logging.ClientIdHelper.initClientId(ClientIdHelper.java)" }
          ]}
      ]
    }
  }]
}
```

### 7. JSON output — list

```bash
dexfinder --dex-file app.apk --query "getDeviceId" --trace --depth 2 --format json --layout list
```
```json
{
  "targets": [{
    "api": "android.telephony.TelephonyManager.getDeviceId()",
    "chains": [
      ["com.example.session.PhoneInfo.getImei(PhoneInfo.java)",
       "com.example.aopsdk.TelephonyManager.getDeviceId(TelephonyManager.java)",
       "android.telephony.TelephonyManager.getDeviceId(TelephonyManager.java)"],
      ["com.example.logging.ClientIdHelper.initClientId(ClientIdHelper.java)",
       "com.example.aopsdk.TelephonyManager.getDeviceId(TelephonyManager.java)",
       "android.telephony.TelephonyManager.getDeviceId(TelephonyManager.java)"]
    ]
  }]
}
```

### 8. Structured model output (for CI/IDE)

```bash
dexfinder --dex-file app.apk --query "getDeviceId" --trace --format model | jq '.call_chains[0]'
```
```json
{
  "target": "Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;",
  "chain": [
    { "method": { "dex_signature": "...", "class": "...", "name": "getImei",
                   "param_types": ["Landroid/content/Context;"], "return_type": "Ljava/lang/String;",
                   "java_readable": "com.example.session.PhoneInfo.getImei(...)" }},
    { "method": { "dex_signature": "...", "java_readable": "...TelephonyManager.getDeviceId(...)" }},
    { "method": { "dex_signature": "...", "java_readable": "...TelephonyManager.getDeviceId(...)" }}
  ],
  "depth": 2
}
```

### 9. ProGuard/R8 mapping — query and display

With `--mapping`, both **input** and **output** support original (unobfuscated) names.

**Query by original name → auto-converts to obfuscated name for DEX search:**

```bash
# Query with original simple class name (mapping converts "KotlinCases" → "LJ7;" internally)
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt

# Query with original Java full name
dexfinder --dex-file app.apk --query "com.example.app.utils.Helper" --mapping mapping.txt

# Query with obfuscated name still works
dexfinder --dex-file app.apk --query "LJ7;" --mapping mapping.txt
```

**Output deobfuscated names in trace:**

```bash
# Tree trace with deobfuscated names
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt --trace --depth 3
```
```
com.example.kotlin.KotlinCases$$ExternalSyntheticLambda1.<init>(int)
└── com.example.TestEntry.runAllTests(TestEntry.java)
    └── com.example.MainActivity.onCreate(MainActivity.java)
```

**Show both obfuscated and original names:**

```bash
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt --show-obf --trace
```
```
com.example.kotlin.KotlinCases.fetchLocationAsync(KotlinCases.java)
└── com.example.kotlin.KotlinCases$testCoroutines$3.invokeSuspend(KotlinCases.java)  [obf: G7.e]
    └── com.example.kotlin.KotlinCases$testCoroutines$3.create(KotlinCases.java)  [obf: G7.b]
```

**All combinations with other flags:**

```bash
# Original name + trace as flat list
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt --trace --layout list

# Original name + DEX signature style
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt --trace --style dex

# Original name + JSON tree + show-obf
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt --show-obf --trace --format json

# Original name + reverse direction (what does this class call?)
dexfinder --dex-file app.apk --query "com.example.kotlin.KotlinCases" --mapping mapping.txt --scope caller
```

**Input × Output matrix:**

| Query input | No mapping | `--mapping` | `--mapping --show-obf` |
|---|---|---|---|
| Obfuscated: `LJ7;` | ✓ obfuscated output | ✓ deobfuscated output | ✓ both names |
| Original simple: `KotlinCases` | ✗ not found | ✓ auto-converts, deobf output | ✓ auto-converts, both names |
| Original full: `com.example...KotlinCases` | ✗ not found | ✓ auto-converts, deobf output | ✓ auto-converts, both names |

### 10. Hidden API detection

```bash
# Download CSV (one-time)
curl -o hiddenapi-flags.csv \
  https://dl.google.com/developers/android/baklava/non-sdk/hiddenapi-flags.csv

# Full scan — linking + reflection detection
dexfinder --dex-file app.apk --api-flags hiddenapi-flags.csv
```
```
#1: Linking unsupported Lsun/misc/Unsafe;->allocateInstance(Ljava/lang/Class;)Ljava/lang/Object; use(s):
       Lcom/google/gson/internal/UnsafeAllocator;->create()Lcom/google/gson/internal/UnsafeAllocator;

#2: Reflection blocked Landroid/location/ILocationManager;->getCurrentLocation potential use(s):
       Lcom/example/monitor/LocationMonitor;->hookSystemLocationManager(Landroid/content/Context;)V
```

### 11. Search string constants (content:// URIs, API keys, etc.)

```bash
# Find content:// URIs in code
dexfinder --dex-file app.apk --query "content://com.android.contacts" --scope string

# Include strings only in DEX table (optimized out by R8, annotations, etc.)
dexfinder --dex-file app.apk --query "content://com.android.contacts" --scope everything
```
```
[STRING] "content://com.android.contacts/" (1 ref)
       Lcom/example/imageloader/BaseImageDownloader;->getStreamFromContent(Ljava/lang/String;)Ljava/io/InputStream;
[STRING_TABLE] "content://com.android.contacts" (in DEX string table, no code reference found)
```

### 12. Filter by class prefix

```bash
# Only scan classes in your own package
dexfinder --dex-file app.apk --query "getDeviceId" --class-filter "Lcom/mycompany/"

# Scan multiple packages
dexfinder --dex-file app.apk --query "getDeviceId" --class-filter "Lcom/mycompany/,Lcom/mylib/"
```

### 13. Combine everything

```bash
# Deobfuscated JSON tree of location API usage, filtered to your code
dexfinder --dex-file app.apk \
  --query "android.location.LocationManager#requestLocationUpdates" \
  --trace --depth 8 \
  --format json --layout tree --style java \
  --mapping mapping.txt --show-obf \
  --class-filter "Lcom/mycompany/"
```

## Performance

Benchmarked on Apple M-series, single thread:

| APK Size | DEX Files | Classes | Method Refs | Scan | Hidden API |
|---|---|---|---|---|---|
| ~1 MB | 1 | ~2K | ~18K | **24ms** | — |
| ~10 MB | 2 | ~25K | ~100K | **335ms** | — |
| ~300 MB | 30+ | ~180K | ~1.2M | **3.9s** | **5.4s** |

Compared to veridex (C++, imprecise mode) on the same ~300MB APK:
- veridex precise: **27s** (no reflection via Binder/AIDL)
- veridex imprecise: **>32 min** (killed, cartesian product explosion)
- **dexfinder: 5.4s** (reverse-index optimization)

## All Options

| Flag | Description | Default |
|---|---|---|
| `--dex-file` | APK/DEX/JAR file to analyze **(required)** | — |
| `--query` | Search keyword (Java, DEX/JNI, or simple name) | — |
| `--trace` | Enable call chain tracing (requires `--query`) | `false` |
| `--depth` | Max call chain depth | `5` |
| `--layout` | Trace layout: `tree` or `list` | `tree` |
| `--style` | Name style: `java` or `dex` | `java` |
| `--format` | Output format: `text`, `json`, `model` | `text` |
| `--mapping` | ProGuard/R8 mapping.txt path | — |
| `--show-obf` | Show obfuscated names alongside deobfuscated | `false` |
| `--api-flags` | Path to hiddenapi-flags.csv | — |
| `--class-filter` | Comma-separated class descriptor prefixes | — |
| `--exclude-api-lists` | API lists to exclude from reporting | — |
| `--scope` | Search scope: `all`, `callee`, `caller`, `string`, `string-table`, `everything` | `all` |
| `--stats` | Show summary statistics only | `false` |
| `--version` | Show version | `false` |

## Building from Source

```bash
git clone https://github.com/JuneLeGency/dexfinder.git
cd dexfinder
go build -o dexfinder ./cmd/dexfinder/
go test ./...
```

## License

Apache License 2.0

---

<a name="中文"></a>

# dexfinder

跨平台 APK/DEX 方法与字段引用查找器，支持调用链追踪、ProGuard/R8 反混淆、Android Hidden API 检测。

基于 Android [veridex](https://android.googlesource.com/platform/art/+/refs/heads/master/tools/veridex/) 原理，用 Go 重新实现并增强：更快的反射检测、多层调用链追踪（veridex 仅一层）、灵活的输出格式。

## 特性

- **APK/DEX/JAR 扫描** — 解析 DEX 字节码，提取所有方法/字段/字符串引用
- **多格式查询** — 支持 Java 类名、DEX/JNI 签名、简单关键字
- **调用链追踪** — 向上追溯 N 层调用者，合并树或展开列表，自动检测递归环
- **ProGuard/R8 反混淆** — 加载 mapping.txt，显示原始名称
- **Hidden API 检测** — 加载 hiddenapi-flags.csv，检测 blocked/unsupported API
- **反射检测** — 类名×字符串交叉匹配，发现反射调用的隐藏 API（兼容 veridex）
- **灵活输出** — text / json / model 格式，tree / list 布局，java / dex 命名风格——正交组合
- **零外部依赖** — 纯 Go 实现，自包含 DEX 解析器
- **跨平台** — macOS (Intel / Apple Silicon)、Linux (amd64 / arm64)、Windows

## 安装

**Homebrew** (macOS / Linux):
```bash
brew tap JuneLeGency/tap
brew install dexfinder
```

**脚本安装** (自动检测系统):
```bash
curl -sSL https://raw.githubusercontent.com/JuneLeGency/dexfinder/main/install.sh | bash
```

**Go 安装**:
```bash
go install github.com/JuneLeGency/dexfinder/cmd/dexfinder@latest
```

**二进制下载**: [Releases](https://github.com/JuneLeGency/dexfinder/releases)

## 快速开始

```bash
# 查看 APK 概况
dexfinder --dex-file app.apk --stats

# 查找所有 getDeviceId 调用（获取 IMEI）
dexfinder --dex-file app.apk --query "getDeviceId"

# 追踪调用链（合并树形视图）
dexfinder --dex-file app.apk --query "getDeviceId" --trace

# 追踪调用链（展开为独立调用栈）
dexfinder --dex-file app.apk --query "getDeviceId" --trace --layout list

# 用精确 JNI 签名查询
dexfinder --dex-file app.apk \
  --query "Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;" \
  --trace --depth 8
```

## 查询格式 (`--query`)

| 格式 | 示例 | 行为 |
|---|---|---|
| 简单名称 | `getDeviceId` | 模糊子串匹配 |
| Java 类名 | `android.telephony.TelephonyManager` | 匹配该类所有方法 |
| Java 类名#方法 | `...TelephonyManager#getDeviceId` | 匹配该方法所有重载 |
| Java 完整签名 | `...#getDeviceId()` | 精确匹配 + 重载回退 |
| DEX/JNI 签名 | `Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;` | 精确匹配 |

## 输出控制

三个独立维度，自由组合：

```
--format  (text / json / model)     输出什么
--layout  (tree / list)             怎么排列调用链
--style   (java / dex)              怎么显示名称
```

### `--layout` 对比（配合 `--trace`）

**tree** — 合并共同路径，一棵树展示全貌：
```
android.telephony.TelephonyManager.getDeviceId()
└── ...aopsdk...TelephonyManager.getDeviceId(TelephonyManager.java)
    ├── PhoneInfo.getImei(PhoneInfo.java)
    ├── ClientIdHelper.initClientId(ClientIdHelper.java)
    │   └── ContextInfo.<init>(ContextInfo.java)
    └── DeviceInfo.k(DeviceInfo.java)
        └── DeviceInfo.getInstance(DeviceInfo.java)
            ├── TidHelper.getIMEI(TidHelper.java)
            └── DeviceCollector.collectData(DeviceCollector.java)
```

**list** — 每条链独立展示（Java crash 风格）：
```
--- Call chain #1 ---
    at PhoneInfo.getImei(PhoneInfo.java)
    at ...aopsdk...TelephonyManager.getDeviceId(TelephonyManager.java)
    at android.telephony.TelephonyManager.getDeviceId(TelephonyManager.java)

--- Call chain #2 ---
    at ContextInfo.<init>(ContextInfo.java)
    at ClientIdHelper.initClientId(ClientIdHelper.java)
    at ...aopsdk...TelephonyManager.getDeviceId(TelephonyManager.java)
    at android.telephony.TelephonyManager.getDeviceId(TelephonyManager.java)
```

### `--style` 对比

**java** (默认): `com.example.Foo.method(Foo.java)`
**dex**: `Foo.method(Ljava/lang/String;)V`

### JSON 输出

```bash
# JSON 树
dexfinder --dex-file app.apk --query "getDeviceId" --trace --format json

# JSON 列表
dexfinder --dex-file app.apk --query "getDeviceId" --trace --format json --layout list
```

### `--scope` 搜索范围

控制查询匹配**哪种引用类型**。理解这个参数对正确解读结果至关重要。

| 值 | 搜索内容 | 回答的问题 | 输出标签 |
|---|---|---|---|
| `all` | 被调 API + 字段 + 代码字符串 | "谁调了这个方法？"（默认） | `[METHOD]` `[FIELD]` `[STRING]` |
| `callee` | 仅 `invoke-*` / `get/put` 指令中的目标签名 | "谁调了这个具体方法/字段？" | `[METHOD]` `[FIELD]` |
| `caller` | 仅调用方法的签名 | "这个方法内部调了什么？" | `[CALLER→]` |
| `string` | `const-string` 指令中的字符串常量 | "这个字符串在代码哪里使用了？" | `[STRING]` |
| `string-table` | 代码字符串 + DEX 完整字符串表 | "这个字符串是否存在于 DEX 中？"（含注解、死代码） | `[STRING]` `[STRING_TABLE]` |
| `everything` | 以上全部 | 完整视图 | 全部标签 |

**callee vs caller 的区别：**

```
scope=callee: "谁调了 finish()？"
    onCreate ──调用──→ finish()     ← 显示这些调用者
    onResume ──调用──→ finish()

scope=caller: "finish() 内部调了什么？"
    finish() ──调用──→ Log.i()      ← 显示这些被调用者
    finish() ──调用──→ super.finish()
```

`--scope=all`（默认）= `callee` + `string`。`caller` 方向被故意排除在默认之外，因为它回答的是完全不同的问题。需要时用 `--scope=caller` 或 `--scope=everything` 显式启用。

**输出标签含义：**

| 标签 | 含义 |
|---|---|
| `[METHOD]` | 你搜的方法**被别人调用了**。缩进行是调用者。 |
| `[FIELD]` | 你搜的字段**被别人访问了**。缩进行是访问者。 |
| `[CALLER→]` | 你搜的方法名出现在某个**调用方**中，缩进行显示它调了什么 API。 |
| `[STRING]` | 代码中的字符串常量匹配。缩进行是使用该字符串的方法。 |
| `[STRING_TABLE]` | 字符串仅存在于 DEX 字符串表中，代码里没有 `const-string` 引用（可能在注解中、被 R8 优化掉等）。 |

## 更多用法

### 反混淆（--mapping）

加载 `--mapping` 后，**输入和输出**都支持原始（未混淆）名称。

**用原始名查询 → 自动转换为混淆名搜索 DEX：**

```bash
# 用原始简短类名查（mapping 内部将 "KotlinCases" 转为 "LJ7;"）
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt

# 用原始 Java 全名查
dexfinder --dex-file app.apk --query "com.example.app.utils.Helper" --mapping mapping.txt

# 用混淆名查也正常工作
dexfinder --dex-file app.apk --query "LJ7;" --mapping mapping.txt
```

**输出反混淆名称：**

```bash
# trace 树形 + 反混淆
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt --trace
```

**同时显示混淆名和原始名：**

```bash
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt --show-obf --trace
```
```
com.example.KotlinCases.fetchLocationAsync(KotlinCases.java)
└── com.example.KotlinCases$testCoroutines$3.invokeSuspend(KotlinCases.java)  [obf: G7.e]
```

**与其他参数自由组合：**

```bash
# 原始名 + 展开列表
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt --trace --layout list

# 原始名 + DEX 签名风格
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt --trace --style dex

# 原始名 + JSON 树 + 显示混淆名
dexfinder --dex-file app.apk --query "KotlinCases" --mapping mapping.txt --show-obf --trace --format json

# 原始名 + 反向查看（这个类内部调了什么）
dexfinder --dex-file app.apk --query "com.example.KotlinCases" --mapping mapping.txt --scope caller
```

**输入×输出矩阵：**

| 查询输入 | 无 mapping | `--mapping` | `--mapping --show-obf` |
|---|---|---|---|
| 混淆名 `LJ7;` | ✓ 混淆输出 | ✓ 反混淆输出 | ✓ 两者并列 |
| 原始简名 `KotlinCases` | ✗ 找不到 | ✓ 自动转换 + 反混淆输出 | ✓ 自动转换 + 两者并列 |
| 原始全名 `com.example...` | ✗ 找不到 | ✓ 自动转换 + 反混淆输出 | ✓ 自动转换 + 两者并列 |

### Hidden API 检测

```bash
# 下载 CSV（一次性）
curl -o hiddenapi-flags.csv \
  https://dl.google.com/developers/android/baklava/non-sdk/hiddenapi-flags.csv

# 全量检测（直接链接 + 反射检测）
dexfinder --dex-file app.apk --api-flags hiddenapi-flags.csv
```

### 字符串搜索

```bash
# 搜索代码中的 content:// URI
dexfinder --dex-file app.apk --query "content://com.android.contacts" --scope string

# 包含被 R8 优化掉的字符串（注解、死代码等）
dexfinder --dex-file app.apk --query "content://com.android.contacts" --scope everything
```

### 按包名过滤

```bash
# 只扫描自己的代码
dexfinder --dex-file app.apk --query "getDeviceId" --class-filter "Lcom/mycompany/"
```

### 组合使用

```bash
# 反混淆 + JSON 树形输出 + 定位 API 调用 + 过滤自己的代码
dexfinder --dex-file app.apk \
  --query "android.location.LocationManager#requestLocationUpdates" \
  --trace --depth 8 \
  --format json --layout tree --style java \
  --mapping mapping.txt --show-obf \
  --class-filter "Lcom/mycompany/"
```

## 性能

Apple M 系列芯片，单线程：

| APK 大小 | DEX 数 | 类数 | 方法引用 | 扫描 | Hidden API |
|---|---|---|---|---|---|
| ~1 MB | 1 | ~2K | ~18K | **24ms** | — |
| ~10 MB | 2 | ~25K | ~100K | **335ms** | — |
| ~300 MB | 30+ | ~180K | ~1.2M | **3.9s** | **5.4s** |

与 veridex (C++) 在同一 ~300MB APK 上对比：
- veridex precise: **27s**（无法追踪 Binder/AIDL 反射）
- veridex imprecise: **>32 分钟**（笛卡尔积爆炸，被 kill）
- **dexfinder: 5.4s**（反向索引优化）

## 全部参数

| 参数 | 说明 | 默认值 |
|---|---|---|
| `--dex-file` | APK/DEX/JAR 文件路径 **（必需）** | — |
| `--query` | 搜索关键字（Java / DEX/JNI / 简单名称） | — |
| `--trace` | 启用调用链追踪（需配合 `--query`） | `false` |
| `--depth` | 调用链最大深度 | `5` |
| `--layout` | 追踪布局: `tree`（合并树）或 `list`（展开列表） | `tree` |
| `--style` | 命名风格: `java`（可读）或 `dex`（JNI 签名） | `java` |
| `--format` | 输出格式: `text`、`json`、`model` | `text` |
| `--mapping` | ProGuard/R8 mapping.txt 路径 | — |
| `--show-obf` | 同时显示混淆名和反混淆名 | `false` |
| `--api-flags` | hiddenapi-flags.csv 路径 | — |
| `--class-filter` | 类描述符前缀过滤（逗号分隔） | — |
| `--exclude-api-lists` | 排除的 API 级别 | — |
| `--scope` | 搜索范围: `all`、`callee`、`caller`、`string`、`string-table`、`everything` | `all` |
| `--stats` | 仅显示统计摘要 | `false` |
| `--version` | 显示版本号 | `false` |

## 从源码构建

```bash
git clone https://github.com/JuneLeGency/dexfinder.git
cd dexfinder
go build -o dexfinder ./cmd/dexfinder/
go test ./...
```

## 许可证

Apache License 2.0

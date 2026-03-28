# 为什么 dexfinder 能做到 5 秒？

[English](#english) | [中文](#中文)

---

<a name="english"></a>

## Why dexfinder Can Do It in 5 Seconds

On a 248MB WeChat APK (16 DEX, 225K classes, 1M method refs), a full hidden API scan including reflection detection completes in **5.2 seconds**. Google's veridex takes **26.8 seconds** in precise mode and **>32 minutes** (killed) in imprecise mode on the same input.

This document explains the five key architectural decisions behind this performance.

### Phase-by-Phase Breakdown

Benchmarked on Apple M-series, single thread:

| Phase | dexfinder | veridex | Why |
|---|---|---|---|
| DEX parsing | **1.0s** | ~3s | No boot classpath loading |
| Bytecode scan | **2.6s** | ~5s | Single-pass, 3 instruction types only |
| CSV loading | **0.36s** | ~1s | `bufio.Scanner` + lenient parsing |
| Linking filter | **0.07s** | ~3s | String concat + map lookup vs JLS resolution |
| Reflection detection | **0.27s** | ~15s | **Reverse index vs cartesian product** |
| Type resolution | **skipped** | ~5s | Not needed for our approach |
| Call graph | **0.8s** | N/A | veridex doesn't trace call chains |
| **Total** | **5.2s** | **26.8s** | **5.2x faster** |

### Decision 1: Skip Boot Classpath (saves ~4s)

veridex must load `system-stubs.zip` (Android framework stub DEX files), parse all class/method/field definitions, build a TypeMap, and set up well-known methods (`Class.forName`, `getMethod`, etc.).

We skip this entirely. The DEX `method_ids` table already contains the full class name + method name + signature for every referenced method. To check against the CSV, we just concatenate strings:

```
veridex:  load stubs → parse classes → build TypeMap → resolve each ref → lookup CSV
dexfinder: read method_id → concat signature string → lookup CSV
```

**Trade-off**: We can't distinguish between an app-defined method with the same signature as a framework method. In practice, this is a non-issue — apps rarely shadow framework API signatures.

### Decision 2: Skip Type Resolution (saves ~5s)

veridex's `Resolver` does JLS method lookup for each method_id: search the target class → superclass → interfaces. This requires the full class inheritance chain.

We don't need it. `method_ids[i]` directly gives us `class_idx + name_idx + proto_idx`, which assembles into the exact signature format used in the CSV. We don't need to know which ancestor class actually defined the method.

```
veridex:  method_id → resolve class → walk super chain → find definition → build signature
dexfinder: method_id → class_idx→type_ids→string = "Lcom/foo/Bar;"
                      + name_idx→string = "method"
                      + proto_idx→string = "(I)V"
                      = "Lcom/foo/Bar;->method(I)V"   ← done, lookup CSV
```

### Decision 3: Reverse Index for Reflection (saves ~15s — the key insight)

This is the single biggest optimization. veridex's imprecise reflection detection:

```python
# veridex: O(classes × strings) cartesian product
for cls in all_referenced_classes:      # 225,000
    for string in all_const_strings:    # 414,000
        if csv.lookup(cls + "->" + string):
            report(cls, string)
# = 93,150,000,000 map lookups
```

Our approach — **reverse the lookup direction**:

```python
# dexfinder: O(boot_classes × avg_members_per_class)
for cls in boot_classes_in_dex:         # ~2,000 (pre-filtered)
    for member in csv.members_of(cls):  # ~20 avg (pre-indexed)
        if member in string_const_set:  # O(1) hash lookup
            report(cls, member)
# = ~40,000 lookups + one-time index build
```

**2,000,000x fewer lookups**. This is why veridex's reflection phase takes 37 seconds and ours takes 0.27 seconds.

The trick: instead of asking "does this random class+string combination exist in the CSV?", we ask "for each class that IS in the CSV, which of its known members appear as string constants?" We pre-build a `class → set<member_name>` index from the CSV, turning the inner loop from 414K iterations to ~20.

### Decision 4: Single-Pass Scan + Lazy Construction

```go
// One pass collects all three reference types simultaneously
for each instruction in code_item.insns:
    switch opcode:
        case INVOKE_*:      → MethodRefs[api] = append(callerInfo)
        case IGET/SGET/...: → FieldRefs[api] = append(callerInfo)
        case CONST_STRING:  → StringRefs[str] = append(callerInfo)
```

Everything else is built **on demand**:
- No `--trace`? Call graph is never built (saves 0.8s)
- No `--api-flags`? Reflection matching is never run (saves 0.6s)
- Has `--query`? Only matching results are formatted (saves I/O)

veridex computes everything upfront regardless of what you need.

### Decision 5: String Pre-caching + Efficient Memory Layout

```go
// Decode all MUTF-8 strings once during DEX parse
f.strings = make([]string, len(f.StringIDs))
for i, sid := range f.StringIDs {
    f.strings[i] = f.readMUTF8(int(sid.DataOff))
}
// All subsequent GetString(idx) calls are O(1) array index
```

Go strings are `(pointer, length)` — 16 bytes, no copy on read. C++ `std::string` has SSO (small string optimization) but copies on assignment. For 414K strings accessed millions of times during scanning, this adds up.

### What We Give Up

| Capability dropped | Impact | Worth it? |
|---|---|---|
| Boot classpath type resolution | Can't distinguish app vs framework same-name methods | Yes — 99% cases don't need it |
| JLS method inheritance lookup | May miss calls through parent class | Yes — CSV signatures are already fully qualified |
| Precise data flow analysis | Can't trace `Class.forName(variable)` | Acceptable — imprecise covers most cases |
| Full DEX verification | No checksum/signature validation | Yes — we're an analysis tool, not a runtime |

### Core Philosophy

**An analysis tool doesn't need to simulate the runtime.**

veridex inherits ART's full `libdexfile` capabilities because it lives inside the ART codebase. Much of that computation — type resolution, inheritance chains, data flow analysis — is essential for a runtime but unnecessary for answering "who calls this API?".

We keep only the necessary path: parse instructions → extract references → match against database.

---

<a name="中文"></a>

## 为什么 dexfinder 能做到 5 秒

在 248MB 微信 APK（16 个 DEX，22.5 万类，100 万方法引用）上，包含反射检测的完整 Hidden API 扫描只需 **5.2 秒**。Google 的 veridex 在同一输入上 precise 模式需要 **26.8 秒**，imprecise 模式跑了 **32 分钟被 kill**。

### 逐阶段耗时拆解

Apple M 系列芯片，单线程：

| 阶段 | dexfinder | veridex | 差异原因 |
|---|---|---|---|
| DEX 解析 | **1.0s** | ~3s | 不加载 boot classpath |
| 字节码扫描 | **2.6s** | ~5s | 单遍扫描，只提取 3 类指令 |
| CSV 加载 | **0.36s** | ~1s | `bufio.Scanner` + 宽容解析 |
| 直接链接过滤 | **0.07s** | ~3s | 字符串拼接 + map 查找 vs JLS 解析 |
| 反射检测 | **0.27s** | ~15s | **反向索引 vs 笛卡尔积** |
| 类型解析 | **跳过** | ~5s | 我们的方案不需要 |
| 调用图构建 | **0.8s** | 不支持 | veridex 没有调用链追踪 |
| **总计** | **5.2s** | **26.8s** | **快 5.2 倍** |

### 决策一：跳过 Boot Classpath（省 ~4 秒）

veridex 必须加载 `system-stubs.zip`（Android framework 的 stub DEX），解析其中所有类/方法/字段定义，构建 TypeMap，设置 well-known 方法（`Class.forName`、`getMethod` 等）。

我们**完全跳过**。DEX 的 `method_ids` 表已经包含每个被引用方法的完整类名+方法名+签名，查 CSV 只需拼字符串：

```
veridex:  加载 stubs → 解析类 → 构建 TypeMap → resolve 每个引用 → 查 CSV
dexfinder: 读 method_id → 拼签名字符串 → 查 CSV
```

**代价**：无法区分 app 自定义的同名方法 vs framework 方法。实际上这种情况极罕见。

### 决策二：跳过类型解析（省 ~5 秒）

veridex 的 `Resolver` 对每个 method_id 做 JLS 方法查找：在目标类中找 → 父类 → 接口。这需要完整的类继承链。

我们不需要。`method_ids[i]` 直接给出 `class_idx + name_idx + proto_idx`，拼出来就是 CSV 中的签名格式：

```
veridex:  method_id → 解析类 → 遍历继承链 → 找到定义 → 构建签名
dexfinder: method_id → 拼 "Lcom/foo/Bar;->method(I)V" → 查 CSV  ← 完成
```

### 决策三：反向索引替代笛卡尔积（省 ~15 秒，最关键）

这是**最大的单点优化**。

veridex imprecise 反射检测：

```python
# veridex: O(类数 × 字符串数) 笛卡尔积
for cls in 所有引用的类:              # 225,000
    for string in 所有代码字符串:      # 414,000
        if csv.查找(cls + "->" + string):
            报告(cls, string)
# = 931 亿次 map 查找
```

我们的做法——**反转查找方向**：

```python
# dexfinder: O(boot类数 × 每类平均成员数)
for cls in DEX中引用的boot类:         # ~2,000（预过滤）
    for member in csv.该类的成员():     # ~20 个（预索引）
        if member in 字符串常量集合:    # O(1) hash 查找
            报告(cls, member)
# = ~40,000 次查找 + 一次索引构建
```

**查找次数减少 200 万倍**。这就是 veridex 反射阶段 37 秒 → 我们 0.27 秒的原因。

核心思路：不问"这个随机的类+字符串组合在 CSV 中存在吗？"，而是问"对于 CSV 中已知的每个类，它的哪些成员名出现在了字符串常量中？"。预构建 `class → set<member_name>` 倒排索引，内层循环从 41.4 万次降到 ~20 次。

### 决策四：单遍扫描 + 按需构建

```go
// 一遍扫描同时收集三类信息
for each instruction in code_item.insns:
    switch opcode:
        case INVOKE_*:      → MethodRefs[api] = append(callerInfo)
        case IGET/SGET/...: → FieldRefs[api] = append(callerInfo)
        case CONST_STRING:  → StringRefs[str] = append(callerInfo)
```

其余一切**按需构建**：
- 不 `--trace`？调用图不构建（省 0.8s）
- 不 `--api-flags`？反射匹配不运行（省 0.6s）
- 有 `--query`？只格式化匹配的结果（省 I/O）

veridex 无论你关心什么都**全量计算后再输出**。

### 决策五：字符串预缓存

```go
// DEX 解析时一次性解码全部 MUTF-8 字符串
f.strings = make([]string, len(f.StringIDs))
for i, sid := range f.StringIDs {
    f.strings[i] = f.readMUTF8(int(sid.DataOff))
}
// 后续所有 GetString(idx) 都是 O(1) 数组索引
```

Go 的 string 是 `(指针, 长度)` —— 16 字节，读取零拷贝。C++ 的 `std::string` 有 SSO 优化但赋值时仍有拷贝。41.4 万个字符串在扫描时被访问数百万次，差异累积很可观。

### 我们放弃了什么

| 放弃的能力 | 影响 | 值不值 |
|---|---|---|
| Boot classpath 类型解析 | 无法区分 app 同名方法 vs framework 方法 | 值——99% 场景不需要 |
| JLS 方法继承链查找 | 可能漏掉通过父类调用的场景 | 值——CSV 签名已包含完整类名 |
| Precise 数据流分析 | 不能追踪 `Class.forName(变量)` 中变量的值 | 可接受——imprecise 覆盖大部分场景 |
| 完整 DEX 校验 | 不验证 checksum/signature | 值——我们是分析工具不是运行时 |

### 核心哲学

**做分析工具不需要模拟运行时。**

veridex 继承了 ART 的完整 `libdexfile` 能力，因为它本身就在 ART 代码库里。其中很多计算——类型解析、继承链构建、数据流分析——对运行时至关重要，但对于回答"谁调了这个 API"这个问题是多余的。

我们只保留必要路径：**解析指令 → 提取引用 → 匹配数据库**。

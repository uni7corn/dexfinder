# dex_method_finder: Go 跨平台 veridex 实现方案

## Context

Android veridex 是 AOSP 中基于 C++ 的静态分析工具，用于扫描 APK/DEX 文件中的非 SDK (hidden) API 调用。原版深度依赖 AOSP 的 `libdexfile`，无法独立跨平台编译。本项目用 Go 重新实现其完整功能，包括**直接链接检测**和**反射检测（数据流分析）**，实现 macOS/Linux/Windows 跨平台运行。

---

## 目录结构

```
dex_method_finder/
├── go.mod
├── cmd/
│   └── dexfinder/
│       └── main.go                 # CLI 入口，参数解析，流程编排
├── pkg/
│   ├── dex/
│   │   ├── dexfile.go              # DexFile 结构体，头部解析，各表索引访问
│   │   ├── instruction.go          # Opcode 枚举，指令解码（~25种格式）
│   │   ├── class_accessor.go       # 遍历 class_defs / fields / methods / code_items
│   │   └── signature.go            # 方法签名解析与比较
│   ├── apk/
│   │   └── apk.go                  # ZIP 解压，提取 classes*.dex
│   ├── hiddenapi/
│   │   ├── apilist.go              # ApiList 类型（blocked/unsupported/max-target-*）
│   │   ├── database.go             # 加载 hiddenapi-flags.csv，构建 signature→ApiList 映射
│   │   ├── filter.go               # ApiListFilter（排除列表）
│   │   ├── signature.go            # GetApiMethodName, GetApiFieldName, ToInternalName
│   │   └── source.go              # SignatureSource 枚举（BOOT/APP/UNKNOWN）
│   ├── resolver/
│   │   ├── class.go                # VeriClass 类型（Kind/Dimensions/ClassDefIndex）
│   │   ├── typemap.go              # TypeMap: descriptor string → *VeriClass
│   │   ├── resolver.go             # Resolver: Run(), GetVeriClass, GetMethod, GetField
│   │   └── lookup.go              # JLS 方法/字段查找（declared → super → interfaces）
│   ├── analysis/
│   │   ├── register.go             # RegisterSource, RegisterValue
│   │   ├── reflect.go              # ReflectAccessInfo
│   │   ├── flow.go                 # FlowAnalysis 基础引擎（FindBranches, AnalyzeCode, ProcessDexInstruction）
│   │   ├── collector.go            # FlowAnalysisCollector（第一轮：收集反射使用）
│   │   └── substitutor.go          # FlowAnalysisSubstitutor（第二轮：不动点参数替换）
│   ├── finder/
│   │   ├── direct.go               # HiddenApiFinder: 直接链接检测（invoke/field 指令扫描）
│   │   ├── precise.go              # PreciseHiddenApiFinder: 反射检测（流分析 + 不动点迭代）
│   │   └── classfilter.go          # ClassFilter（前缀匹配）
│   └── report/
│       ├── text.go                 # 文本输出（兼容原版 veridex 格式）
│       ├── json.go                 # JSON 结构化输出
│       └── stats.go                # HiddenApiStats 统计
└── testdata/                       # 测试用 DEX/APK 文件
```

## 核心数据类型

### DEX 指令解码 (`pkg/dex/instruction.go`)

自行实现字节码解码器（~300行），覆盖 veridex 用到的所有指令格式（10x, 12x, 21c, 22c, 35c, 3rc 等）：

```go
type Opcode uint16
type Instruction struct {
    Opcode Opcode
    Raw    []uint16
    DexPC  uint32
}
// VRegA(), VRegB_21c(), VRegC_22c(), VRegB_35c(), VRegB_3rc() 等访问器
```

### 类型系统 (`pkg/resolver/class.go`)

```go
type VeriClass struct {
    Kind        PrimitiveType  // PrimNot = 引用类型
    Dimensions  uint8          // 0 = 非数组
    DexFileIdx  int            // 所在 dex 文件索引
    ClassDefIdx uint32         // class_defs 内索引
}
```

C++ 版用原始 `uint8_t*` 指针作方法/字段唯一标识。Go 版改用 `(DexFileIdx, MethodIdx)` 元组，解析器列表用 `[]*Resolver` 索引。

### 寄存器值追踪 (`pkg/analysis/register.go`)

```go
type RegisterSource uint8  // None/Parameter/Field/Method/Class/String/Constant
type RegisterValue struct {
    Source     RegisterSource
    Value      uint32         // 参数索引或常量值
    DexFileIdx int
    RefIndex   uint32         // string/type/method/field 索引
    Type       *VeriClass
}
type ReflectAccessInfo struct {
    Cls      RegisterValue
    Name     RegisterValue
    IsMethod bool
}
```

### 流分析引擎 (`pkg/analysis/flow.go`)

用 Go interface 替代 C++ 虚函数：

```go
type InvokeAnalyzer interface {
    AnalyzeInvoke(ctx *FlowContext, inst *Instruction, isRange bool) RegisterValue
    AnalyzeFieldSet(ctx *FlowContext, inst *Instruction)
    GetUses() []ReflectAccessInfo
}
```

`FlowAnalysisCollector` 和 `FlowAnalysisSubstitutor` 分别实现此接口。

## 算法流程

### 直接链接检测 (imprecise 模式)
```
遍历 app DEX 每条指令:
  INVOKE_* → 提取 method_id → 拼 "Lclass;->method(sig)RetType" → 查 CSV
  IGET/SGET/IPUT/SPUT → 提取 field_id → 拼 "Lclass;->field:Type" → 查 CSV
  CONST_STRING → 收集字符串，与所有 boot 类做交叉匹配
```

### 精确反射检测 (precise 模式)
```
第一轮 FlowAnalysisCollector:
  对每个 app 方法做前向数据流分析（worklist 算法）:
    1. FindBranches() → 标记分支目标（goto/if/switch/异常处理器）
    2. AnalyzeCode() → 逐指令处理，更新寄存器状态
    3. AnalyzeInvoke() → 识别反射模式:
       - Class.forName(string) → 类引用
       - getMethod/getDeclaredMethod(string,...) → ReflectAccessInfo(cls, name, isMethod=true)
       - getField/getDeclaredField(string) → ReflectAccessInfo(cls, name, isMethod=false)
    4. 分类: concrete（cls+name 已知）vs abstract（含参数引用）

不动点迭代（最多 10 轮）FlowAnalysisSubstitutor:
  while abstract_uses 非空 && iterations < 10:
    对每个方法重新分析:
      遇到被调方法有 abstract_uses 时，用调用者实参替换形参
      新解析的 concrete 加入结果，新 abstract 加入下轮 worklist
```

## 构建 vs 使用库

| 组件 | 决策 | 理由 |
|---|---|---|
| DEX 文件解析 | 自行实现 | 需要完整控制 code_item、try/catch、switch 表访问 |
| 字节码指令解码 | 自行实现 | veridex 核心，~300 行，需精确控制寄存器提取 |
| APK/ZIP 读取 | Go 标准库 `archive/zip` | 够用 |
| CSV 解析 | Go 标准库行分割 | 格式极简 |
| 类型解析/流分析 | 自行实现 | 核心逻辑，无现成 Go 库 |

外部依赖：仅 Go 标准库。DEX 解析完全自行实现以获得最大控制力。

## 分阶段实施

### Phase 1: 项目骨架 + DEX 解析器
- 初始化 Go module
- 实现 `pkg/dex/dexfile.go`: DEX header / string_ids / type_ids / proto_ids / field_ids / method_ids / class_defs 解析
- 实现 `pkg/dex/instruction.go`: 字节码指令解码器（所有格式的 SizeInCodeUnits + VReg 访问器）
- 实现 `pkg/dex/class_accessor.go`: 遍历 class_data_item（LEB128 解码）
- 实现 `pkg/apk/apk.go`: ZIP 解压提取 classes*.dex

### Phase 2: Hidden API 数据库 + 类型系统
- 实现 `pkg/hiddenapi/`: CSV 加载、ApiList 枚举、签名构建、过滤器
- 实现 `pkg/resolver/class.go` + `typemap.go`: VeriClass / 基本类型单例 / TypeMap

### Phase 3: 解析器 (Resolver)
- 实现 `pkg/resolver/resolver.go`: Resolver.Run() — 遍历 class_defs 填充 TypeInfos/MethodInfos/FieldInfos
- 实现 `pkg/resolver/lookup.go`: JLS 方法/字段查找（declared → super → interfaces）

### Phase 4: 直接链接检测 + CLI
- 实现 `pkg/finder/direct.go` + `classfilter.go`
- 实现 `pkg/report/text.go` + `stats.go`
- 实现 `cmd/dexfinder/main.go`
- **里程碑: imprecise 模式端到端可用**

### Phase 5: 流分析引擎
- 实现 `pkg/analysis/register.go` + `reflect.go`
- 实现 `pkg/analysis/flow.go`: FindBranches / AnalyzeCode / ProcessDexInstruction / MergeRegisterValues
- 实现 `pkg/analysis/collector.go`: 反射模式识别

### Phase 6: 精确反射检测
- 实现 `pkg/analysis/substitutor.go`: 参数替换
- 实现 `pkg/finder/precise.go`: 两阶段检测 + 不动点迭代
- 接入 CLI precise 模式
- **里程碑: precise 模式端到端可用**

### Phase 7: JSON 输出 + 优化
- 实现 `pkg/report/json.go`
- 并发优化：方法级并行分析（`sync.WaitGroup` + goroutine pool）
- 跨平台测试

## 关键设计决策

### 方法/字段标识
C++ 用 `uint8_t*` 指针做唯一标识，Go 用 `(DexFileIdx, MethodIdx)` 元组。已知方法（forName/getField 等）通过指针相等性比较。

### 跨 DEX 解析器查找
C++ 用 `map<uintptr_t, Resolver*>` 按 mmap 地址查找。Go 在 VeriClass 中存 DexFileIdx，用 `[]*Resolver` 切片索引。

### 分支目标合并
C++ 的 MergeRegisterValues 使用"首次访问获胜"语义（非 lattice 合并）。Go 版复刻此行为以保证结果一致。

### 并发
C++ 版单线程。Go 版在 Resolver.Run() 后只读，流分析按方法独立，天然支持 goroutine 并行。

## 验证方式

1. **单元测试**: 每个 pkg 编写 `_test.go`
2. **Golden 测试**: 原版 C++ veridex 输出 vs Go 版输出 diff
3. **测试 fixture**: `testdata/` 包含 simple_link.dex / reflection.dex / indirect_reflection.dex / multi_dex.apk
4. **性能基准**: BenchmarkParseAPK, BenchmarkFlowAnalysis

## 参考源码

| 文件 | 说明 |
|---|---|
| `art/tools/veridex/flow_analysis.cc` | 流分析引擎（800行），最复杂组件 |
| `art/tools/veridex/hidden_api_finder.cc` | 直接链接检测 + 非精确反射匹配 |
| `art/tools/veridex/resolver.cc` | JLS 方法/字段解析，跨 DEX 类型解析 |
| `art/tools/veridex/precise_hidden_api_finder.cc` | 两阶段反射检测 + 不动点迭代 |
| `art/tools/veridex/veridex.cc` | 整体编排逻辑 |

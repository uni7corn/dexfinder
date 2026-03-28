# 我用 Go 重写了 Android veridex，5 秒扫完 300MB APK 的全部调用链

> 你有没有遇到过这样的场景：线上 crash 堆栈只有一个混淆后的 `a.b.c()`，你需要知道到底是谁调用了 `TelephonyManager.getDeviceId()`？或者合规审查要求你在 48 小时内列出一个 300MB 的超级 App 里所有获取 IMEI、定位、通讯录的调用点？

这篇文章讲的就是怎么解决这个问题。

## 痛点

做过大型 Android 项目的人一定深有体会：

### 1. "这个 API 到底谁在调？"

你负责隐私合规，需要排查 APK 里所有调用 `getDeviceId()` 的地方。APK 有 30 多个 DEX，18 万个类，120 万个方法引用。你打开 Android Studio 的 APK Analyzer，点进 classes.dex... classes2.dex... classes31.dex... 一个个翻？

### 2. "混淆后的堆栈看不懂"

线上 crash：
```
at a0.f(a0.java)
at a0.a(a0.java)
at a0.g(a0.java)
```
你有 mapping.txt，但要把整条链手动反混淆，再搞清楚 `a0` 是 `WukongAuthManager`、`f` 是 `getDeviceToken`...

### 3. "veridex 跑了 32 分钟还没出结果"

Google 官方的 veridex 工具能检测 Hidden API，但它的 imprecise 模式在大 APK 上做类名×字符串的笛卡尔积匹配，**300MB 的 APK 跑了 32 分钟被我 kill 了**。

### 4. "多模块项目，CI 里怎么自动卡点？"

你们团队 30 个人，十几个模块，怎么在 MR 阶段自动检测有没有人偷偷调了 `getDeviceId()`、`requestLocationUpdates()` 这些敏感 API？

## 解法：dexfinder

我用 Go 从零实现了一个 DEX 字节码分析器，核心能力：

- 解析 APK 中所有 DEX 文件的字节码指令
- 提取每一条 `invoke-*`（方法调用）和 `iget/sget/iput/sput`（字段访问）指令
- 构建完整的方法调用图（Call Graph）
- 支持 ProGuard/R8 mapping.txt 反混淆
- 支持 Android Hidden API（hiddenapi-flags.csv）检测
- 支持反射检测（`Class.forName` + `getMethod` 模式）

**零外部依赖**，纯 Go 标准库，交叉编译出 macOS/Linux/Windows 二进制。

## 5 秒能做什么？

一个真实的 300MB 超级 App（31 个 DEX，18 万类，120 万方法引用）：

```bash
$ dexfinder --dex-file app.apk --stats
Loaded 31 DEX file(s): 183913 classes, 1250566 method refs
Method references: 680610
Field references:  625572
String constants:  654353
Time: 3.9s
```

**3.9 秒**扫完全部引用。加上 Hidden API 反射检测？**5.4 秒**。

对比 veridex：

| 工具 | 耗时 | 结果 |
|---|---|---|
| veridex precise | 27s | 找不到 Binder/AIDL 反射 |
| veridex imprecise | >32 min (killed) | 笛卡尔积爆炸 |
| **dexfinder** | **5.4s** | 全部找到 |

## 实战：查找 IMEI 获取的完整调用链

```bash
dexfinder --dex-file app.apk \
  --query "Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;" \
  --trace --depth 5
```

输出一棵**合并的调用树**，共享路径被折叠：

```
android.telephony.TelephonyManager.getDeviceId()
└── com.example.aopsdk.TelephonyManager.getDeviceId(TelephonyManager.java)
    ├── PhoneInfo.getImei(PhoneInfo.java)
    ├── ClientIdHelper.initClientId(ClientIdHelper.java)
    │   └── ContextInfo.<init>(ContextInfo.java)
    │       ├── LogStrategyManager.getInstance(...)
    │       └── LogContextImpl.<init>(...)
    ├── DeviceInfo.k(DeviceInfo.java)
    │   └── DeviceInfo.<init>() → getInstance()
    │       ├── TidHelper.getIMEI()
    │       ├── TidHelper.getIMSI()
    ���       └── DeviceCollector.collectData()
    ├── WXEnvironment.getDevId()
    │   └── WXEnvironment.<clinit>()
    └── ...共 30 条调用链
```

一眼看清：**所有调用都经过安全 AOP 拦截层**，背后的业务方包括日志系统、邮箱加密、设备信息插件、Weex 环境、网商银行 SDK...

不想看树？换成 Java crash 风格：

```bash
dexfinder --dex-file app.apk --query "getDeviceId" --trace --layout list
```

```
--- Call chain #1 ---
    at com.example.session.PhoneInfo.getImei(PhoneInfo.java)
    at com.example.aopsdk.TelephonyManager.getDeviceId(TelephonyManager.java)
    at android.telephony.TelephonyManager.getDeviceId(TelephonyManager.java)

--- Call chain #2 ---
    at com.example.logging.LogStrategyManager.getInstance(LogStrategyManager.java)
    at com.example.logging.ContextInfo.<init>(ContextInfo.java)
    at com.example.logging.ClientIdHelper.initClientId(ClientIdHelper.java)
    at com.example.aopsdk.TelephonyManager.getDeviceId(TelephonyManager.java)
    at android.telephony.TelephonyManager.getDeviceId(TelephonyManager.java)
```

直接贴到 issue 里，人人能看懂。

## 查询有多灵活？

你不需要记住 DEX 签名格式，这些写法**都能工作**：

```bash
# 简单名字
--query "getDeviceId"

# Java 全限定名
--query "android.telephony.TelephonyManager"

# Java 类名#方法名
--query "android.telephony.TelephonyManager#getDeviceId"

# Java 完整签名（自动转换为 DEX 格式）
--query "android.telephony.TelephonyManager#getDeviceId()"

# DEX/JNI 精确签名
--query "Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;"
```

dexfinder 自动检测输入格式并转换。带参数的精确签名只匹配对应重载，不带参数的匹配所有重载。

## 输出三轴正���

```
--format  (text / json / model)     输出什么格式
--layout  (tree / list)             调用链怎么排列
--style   (java / dex)              名称怎么显示
```

随意组合：
- `--format json --layout tree` → JSON 嵌套树
- `--format json --layout list` → JSON 平铺数组
- `--format text --layout tree --style dex` → 文本树 + JNI 签名
- `--format model --trace` → 结构化 JSON，包含完整 MethodInfo（类、方法名、参数类型、返回值）

## 大工程多模块场景

### CI 卡点：敏感 API 自动检测

```bash
# 在 CI pipeline 中
dexfinder --dex-file app-release.apk \
  --query "getDeviceId" \
  --format json --layout list | jq '.targets[].chains | length'

# 如果调用链数 > 0，fail 这个 MR
```

### 模块定责：谁引入了这个调用？

```bash
# 只扫描 payment 模块的类
dexfinder --dex-file app.apk \
  --query "getDeviceId" \
  --trace --depth 8 \
  --class-filter "Lcom/mycompany/payment/"
```

### 合规审计：导出 JSON 给安全团队

```bash
dexfinder --dex-file app.apk \
  --query "getDeviceId" \
  --trace --format model \
  > audit_report.json
```

安全团队拿到的是结构化数据，包含 `dex_signature`、`class`、`name`、`param_types`、`return_type`、`java_readable`，直接导入他们的合规系统。

## 隐藏 API 检测：比 veridex 更快更准

```bash
curl -o hiddenapi-flags.csv \
  https://dl.google.com/developers/android/baklava/non-sdk/hiddenapi-flags.csv

dexfinder --dex-file app.apk --api-flags hiddenapi-flags.csv
```

dexfinder 实现了 veridex 的两种检测模式：

1. **直接链接检测**：扫描 `invoke-*` / `iget` / `sget` 指令引用的方法/字段
2. **反射检测**：从代码中的 `const-string` 提取类名（如 `"android.location.ILocationManager"`），转换为 DEX 描述符，与 CSV 中的成员做反向索引匹配

关键优化：veridex 做 `O(classes × strings)` 的笛卡尔积（3 亿次 map 查找），我们用**反向索引**——对每个 boot class 只查它在 CSV 中已知的成员名，复杂度降到 `O(classes × avg_members)`，从 37 秒降到 **0.8 秒**。

## 字符串搜索��R8 优化后也能找到

R8 会把 `static final String URI = "content://contacts"` 内联到调用处。内联后原字段消失，但字符串变成了 `const-string` 指令——dexfinder 两种都能抓住：

```bash
# 代码中的字符串引用（有调用者信息）
dexfinder --dex-file app.apk --query "content://com.android.contacts" --scope string

# 包含 DEX 字符串表中被优化掉的（注解、死代码等）
dexfinder --dex-file app.apk --query "content://com.android.contacts" --scope everything
```

## 技术实现

整个项目零外部依赖，核心模块：

| 模块 | 作用 |
|---|---|
| `pkg/dex` | DEX 文件解析器（header、string/type/method/field IDs、class_data、code_item） |
| `pkg/dex/instruction.go` | 全量 DEX 字节码解码器（256 个 opcode，25 种指令格式） |
| `pkg/finder` | 方法/字段引用扫描、调用图构建、反射检测 |
| `pkg/mapping` | ProGuard/R8 mapping.txt 双向映射解析 |
| `pkg/hiddenapi` | hiddenapi-flags.csv 加载、ApiList 过滤 |
| `pkg/model` | 结构化输出模型（MethodInfo、FieldInfo、CallChainInfo） |

DEX 解析器参考了 AOSP `libdexfile` 的指令格式定义（`dex_instruction_list.h`），但完全独立实现——Google 自己都写了 4 套 DEX 解析器（libdexfile C++、dexlib2 Java、D8/R8 Java、dx Java），每套适配不同场景。我们的 Go 实现只覆盖 veridex 需要的子集，保持精简。

## 安装

```bash
# macOS / Linux
brew tap JuneLeGency/tap && brew install dexfinder

# 或者一键脚本
curl -sSL https://raw.githubusercontent.com/JuneLeGency/dexfinder/main/install.sh | bash

# 或者直接下载
# https://github.com/JuneLeGency/dexfinder/releases
```

## 开源

GitHub: [github.com/JuneLeGency/dexfinder](https://github.com/JuneLeGency/dexfinder)

Apache 2.0 协议，欢迎 Star / Issue / PR。

---

*如果你也在做大型 Android 项目的合规审计、敏感 API 排查、线上问题定位，试试 dexfinder——它可能帮你把几天的人工排查缩短到几秒钟。*

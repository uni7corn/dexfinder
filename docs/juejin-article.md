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
    │       └── DeviceCollector.collectData()
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

## 输出三轴正交

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

## 字符串搜索：R8 优化后也能找到

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

## 实战案例：5 秒扫出微信的定位隐私全景

### 背景

假设你是隐私合规审计人员，需要回答一个问题：**微信在哪些场景下获取了用户的地理位置？** 这是一个 248MB 的 APK，16 个 DEX 文件，22.5 万个类，100 万个方法引用。

手工排查？不可能。反编译看 smali？大海捞针。

一行命令：

```bash
dexfinder --dex-file weixin.apk \
  --query "android.location.LocationManager" \
  --trace --depth 6
```

3.7 秒扫描完毕，5.4 秒输出完整调用链树。

### 发现了什么

微信对 `LocationManager` 的调用分为 5 大类，覆盖了 7 个系统 API：

#### 1. 持续定位：`requestLocationUpdates`

```
android.location.LocationManager.requestLocationUpdates(String, long, float, LocationListener, Looper)
└── com.tencent.magicar.MagicAR.startLocationListen(MagicAR.java)
```

只有一个直接调用点——**MagicAR**（微信的 AR 功能）。微信的主业务定位**没有直接调用系统 API**，而是走腾讯地图 SDK 的封装。这说明微信在定位架构上做了良好的收口管理。

#### 2. 最近位置：`getLastKnownLocation`

```
android.location.LocationManager.getLastKnownLocation(String)
└── androidx.appcompat.app.t0.a(t0.java)
    └── ...g0.a(g0.java)
        ├── AppCompatActivity.onConfigurationChanged(...)
        │   ├── WxaLiteAppSheetUI.onConfigurationChanged(...)
        │   ├── UIComponentActivity.onConfigurationChanged(...)
        │   │   └── MMFragmentActivity → MMActivity → VASLauncher → BizChatConversationUI
        │   └── WxaLiteAppTransparentUI.onConfigurationChanged(...)
        └── AppCompatActivity.onStart(...)
            ├── LauncherUI.onStart(...)
            ├── AppBrandUI.onStart(...)
            └── WxaFlutterActivity.onStart(...)
```

这个调用来自 AndroidX 的 **TwilightManager**（日夜模式切换），用位置判断日出日落时间。触发者包括微信主界面 `LauncherUI`、小程序 `AppBrandUI`、Flutter 容器 `WxaFlutterActivity` 等——**几乎每个 Activity 的 `onStart` 和 `onConfigurationChanged` 都会间接触发**。

这是个有意思的发现：很多用户不知道的是，你每次打开微信、切换小程序、旋转屏幕时，AndroidX 都在静默获取一次粗略位置来判断是否该切换深色模式。

#### 3. 定位可用性检查：`isProviderEnabled`（40+ 调用点）

这是调用量最大的一类，遍布微信的各个业务模块：

```
android.location.LocationManager.isProviderEnabled(String)
├── l2.a(l2.java)  ← 核心工具类，被以下场景调用：
│   ├── 聊天发位置：chatting.component.biz.a/c/d.onClick
│   ├── 摇一摇：ShakeReportUI$1.onGetLocation
│   ├── 附近的人：NearbyFriendsUI.onSceneEnd / RadarViewController
│   ├── 卡券门店：CardDetailUI.onResume / CardShopUI.onCreate
│   ├── 视频号直播 POI：finder.live.widget.ko/u7.onClick
│   ├── 视频号发现 POI：finder.activity.poi.ui.q0/q.onGetLocation
│   ├── 扫一扫：scanner.ui.o0.onGetLocation
│   ├── 运动轨迹：traceroute.ui.e.onGetLocation
│   ├── 小程序定位：appbrand.jsapi.lbs.l1 / appbrand.i3.k
│   ├── 外接设备：ExdeviceAddDataSourceUI / ExdeviceBindDeviceGuideUI
│   ├── 附近小程序：nearlife.ui.j.onGetLocation
│   ├── H5 定位：webview.ui.tools.jsapi.h5.A6 / z0.Q4
│   └── 钱包地址：WalletAddAddressUI.onCreate
├── l2.b(l2.java)  ← 另一个重载，同样被大量业务调用
└── j91.e0.onReceive(...)  ← 广播接收器监听定位状态变化
```

`l2` 是微信的定位工具类（`com.tencent.mm.sdk.platformtools.l2`），是微信所有定位功能的**统一入口**。40 多个业务模块都经过它来检查 GPS/Network Provider 是否可用。

#### 4. GPS 支持检查：`getAllProviders`

```
android.location.LocationManager.getAllProviders()
└── TencentLocationUtils.isSupportGps(...)
```

腾讯地图 SDK 的内部调用，检查设备是否支持 GPS。

#### 5. GNSS 状态监听：`registerGnssStatusCallback`

```
android.location.LocationManager.registerGnssStatusCallback(...)
└── MagicAR.startLocationListen(...)
```

同样只在 AR 功能中使用。

### 结合大模型深度分析

dexfinder 输出结构化 JSON（`--format model`），可以直接喂给大模型做深度分析：

```bash
dexfinder --dex-file weixin.apk \
  --query "android.location.LocationManager" \
  --trace --depth 6 --format model \
  > weixin_location_report.json
```

然后把 JSON 交给 Claude/GPT 分析：

> **Prompt**: 以下是微信 APK 中所有 LocationManager 系统 API 的调用链数据（JSON 格式）。请从隐私合规角度分析：1) 哪些是用户主动触发的定位？2) 哪些是后台静默获取的？3) 是否有过度收集的风险？4) 给出合规改进建议。

大模型可以基于结构化数据给出精准分析：

---

**1. 用户主动触发的定位（合理）**

| 场景 | 触发方式 | 调用链入口 |
|---|---|---|
| 聊天发位置 | 用户点击"位置"按钮 | `chatting.component.biz.c.onClick` |
| 摇一摇 | 用户摇动手机 | `ShakeReportUI$1.onGetLocation` |
| 附近的人 | 用户主动打开 | `NearbyFriendsUI.onSceneEnd` |
| 扫一扫 | 用户点击扫码 | `scanner.ui.o0.onGetLocation` |
| 视频号 POI | 用户选择位置 | `finder.activity.poi.ui.q.onGetLocation` |
| 小程序定位 | 小程序调用 wx.getLocation | `appbrand.jsapi.lbs.l1` |

这些都是**用户明确触发**的场景，符合"最小必要"原则。

**2. 非用户直接触发但合理的定位**

| 场景 | 触发时机 | 分析 |
|---|---|---|
| TwilightManager | 每个 Activity 启动 | AndroidX 框架自动行为，用于日夜模式判断，只读 `getLastKnownLocation`（缓存值），不主动请求新定位 |
| 卡券门店 | `CardDetailUI.onResume` | 打开卡券时自动定位附近门店，属于功能需要 |

**3. 需要关注的潜在风险**

| 风险 | 说明 |
|---|---|
| `j91.e0.onReceive` | 广播接收器监听定位开关状态变化，可能在后台触发 |
| `isProviderEnabled` 高频调用 | 40+ 个场景都在检查定位状态，虽不获取坐标，但反映了对定位的强依赖 |
| 外接设备相关 | `ExdeviceBindDeviceGuideUI` 等在设备绑定时获取位置，需告知用户 |

**4. 架构亮点**

微信的定位架构设计值得参考：
- **统一收口**：所有业务通过 `l2` 工具类和腾讯地图 SDK，不直接调系统 API
- **只有 MagicAR 直接调 `requestLocationUpdates`**，其余业务都经过 SDK 封装
- **权限与功能解耦**：先 `isProviderEnabled` 检查，再决定是否请求定位

---

### 从人工审计到 AI 辅助审计

传统做法：安全工程师用 jadx 反编译 → 搜索 `LocationManager` → 人肉跟踪调用链 → 写报告。一个 248MB 的 APK 至少需要 **2-3 天**。

现在：`dexfinder` 5 秒出数据 → 结构化 JSON 喂给大模型 → 大模型输出分析报告。整个过程 **5 分钟**。

这不是取代安全工程师，而是让他们从"找数据"的苦力中解放出来，专注于"做判断"。

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

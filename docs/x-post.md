300MB 的 APK，18 万个类，想知道谁调了 getDeviceId()？

veridex 跑了 32 分钟我 kill 了。
用 Go 重写了一个，5 秒出结果，还能输出完整调用链树。

一行命令：
dexfinder --dex-file app.apk --query "getDeviceId" --trace

大工程合规审计 / 敏感 API 排查 / 线上混淆堆栈定位，从人肉翻 DEX 变成秒级查询。

开源：github.com/JuneLeGency/dexfinder

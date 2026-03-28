# Aggressive obfuscation for testing
-optimizationpasses 5
-allowaccessmodification
-repackageclasses ''

# Keep entry point for Android
-keep public class * extends android.app.Activity
-keep public class * extends android.app.Service

# Keep the TestEntry class name but obfuscate everything it calls
-keep class com.test.dexfinder.TestEntry {
    public static void runAllTests(android.content.Context);
}

# Let everything else be fully obfuscated
-dontwarn kotlinx.**
-dontwarn org.jetbrains.**

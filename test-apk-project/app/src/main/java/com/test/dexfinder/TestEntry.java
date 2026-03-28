package com.test.dexfinder;

import android.content.Context;
import android.content.Intent;

import com.test.dexfinder.cases.*;
import com.test.dexfinder.kotlin.KotlinCases;

/**
 * Entry point that exercises all test scenarios.
 * This class is kept by ProGuard, everything else gets obfuscated.
 */
public class TestEntry {
    public static void runAllTests(Context context) {
        // Case 2: Reflection
        ReflectionCases.testClassForName();
        ReflectionCases.testGetMethod();
        ReflectionCases.testGetDeclaredField();
        ReflectionCases.testIndirectReflection("android.app.ActivityThread", "currentActivityThread");

        // Case 3: Generics
        GenericCases.testGenericClass();
        GenericCases.testGenericMethod();
        GenericCases.testWildcard();
        GenericCases.testBoundedType();

        // Case 4: Primitive types
        PrimitiveTypeCases.testAllPrimitives(true, (byte) 1, 'c', (short) 2, 3, 4L, 5.0f, 6.0);
        PrimitiveTypeCases.testArrays(new int[]{1, 2}, new String[]{"a"}, new byte[][]{});
        PrimitiveTypeCases.testVarargs("a", "b", "c");

        // Case 5: Threading
        ThreadingCases.testThread();
        ThreadingCases.testHandler(context);
        ThreadingCases.testExecutorService();
        ThreadingCases.testAsyncCallback(result -> {});

        // Case 6: Location (target API for search)
        LocationCases.requestGpsLocation(context);
        LocationCases.requestNetworkLocation(context);
        LocationCases.requestWithCriteria(context);

        // Case 7: Nested/deep call chains
        DeepCallChain.entryPoint(context);

        // Case 8: Static methods & constants
        StaticCases.getInstance().doWork();
        String val = StaticCases.CONSTANT_VALUE;

        // Case 9: Exception handling
        ExceptionCases.testTryCatch();
        ExceptionCases.testMultiCatch();
        ExceptionCases.testFinally();

        // Case 10: Enum
        EnumCases.testEnum();

        // Case 11: Kotlin scenarios
        KotlinCases.INSTANCE.runAll(context);

        // Case 12: Service start (cross-component)
        context.startService(new Intent(context, com.test.dexfinder.service.BackgroundLocationService.class));
    }
}

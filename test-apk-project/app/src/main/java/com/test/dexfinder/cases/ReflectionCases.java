package com.test.dexfinder.cases;

import java.lang.reflect.Field;
import java.lang.reflect.Method;

/**
 * Case 2: Reflection — all patterns veridex tracks
 */
public class ReflectionCases {

    /** Direct Class.forName */
    public static void testClassForName() {
        try {
            Class<?> cls = Class.forName("android.app.ActivityThread");
            Object instance = cls.getMethod("currentActivityThread").invoke(null);
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    /** getMethod with string constant */
    public static void testGetMethod() {
        try {
            Class<?> cls = Class.forName("android.os.ServiceManager");
            Method m = cls.getMethod("getService", String.class);
            Object result = m.invoke(null, "package");
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    /** getDeclaredField */
    public static void testGetDeclaredField() {
        try {
            Class<?> cls = Class.forName("android.app.Activity");
            Field f = cls.getDeclaredField("mCalled");
            f.setAccessible(true);
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    /** Indirect: class and method name come from parameters (abstract reflection) */
    public static void testIndirectReflection(String className, String methodName) {
        try {
            Class<?> cls = Class.forName(className);
            Method m = cls.getMethod(methodName);
            m.invoke(null);
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    /** ClassLoader.loadClass pattern */
    public static void testLoadClass(ClassLoader loader) {
        try {
            Class<?> cls = loader.loadClass("android.app.Application");
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    /** Object.getClass + getMethod pattern */
    public static void testGetClassReflection(Object obj) {
        try {
            Method m = obj.getClass().getMethod("toString");
            m.invoke(obj);
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}

package com.test.dexfinder.cases;

/**
 * Case 4: Primitive types — all JNI type signatures
 */
public class PrimitiveTypeCases {

    /** All primitive parameter types */
    public static String testAllPrimitives(boolean z, byte b, char c, short s, int i, long j, float f, double d) {
        return "" + z + b + c + s + i + j + f + d;
    }

    /** Array types: 1D, Object array, 2D */
    public static void testArrays(int[] intArr, String[] strArr, byte[][] byteArr2d) {
        int len = intArr.length + strArr.length + byteArr2d.length;
    }

    /** Varargs (becomes array in bytecode) */
    public static String testVarargs(String... args) {
        StringBuilder sb = new StringBuilder();
        for (String arg : args) {
            sb.append(arg);
        }
        return sb.toString();
    }

    /** Return type coverage */
    public static boolean returnBoolean() { return true; }
    public static byte returnByte() { return 0; }
    public static char returnChar() { return 'a'; }
    public static short returnShort() { return 0; }
    public static int returnInt() { return 0; }
    public static long returnLong() { return 0L; }
    public static float returnFloat() { return 0.0f; }
    public static double returnDouble() { return 0.0; }
    public static int[] returnIntArray() { return new int[0]; }
    public static String[] returnStringArray() { return new String[0]; }
    public static void returnVoid() {}
}

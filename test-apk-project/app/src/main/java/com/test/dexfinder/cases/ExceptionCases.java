package com.test.dexfinder.cases;

import java.io.FileNotFoundException;
import java.io.IOException;

/**
 * Case 9: Exception handling — try/catch/finally, multi-catch, custom exceptions
 */
public class ExceptionCases {

    /** Custom exception hierarchy */
    public static class AppException extends Exception {
        private final int code;
        public AppException(String message, int code) {
            super(message);
            this.code = code;
        }
        public int getCode() { return code; }
    }

    public static class NetworkException extends AppException {
        public NetworkException(String url) {
            super("Network error: " + url, 500);
        }
    }

    public static void testTryCatch() {
        try {
            riskyOperation();
        } catch (AppException e) {
            handleError(e.getCode(), e.getMessage());
        }
    }

    public static void testMultiCatch() {
        try {
            multiRiskyOperation();
        } catch (IOException | NumberFormatException e) {
            e.printStackTrace();
        } catch (AppException e) {
            handleError(e.getCode(), e.getMessage());
        }
    }

    public static void testFinally() {
        Object resource = null;
        try {
            resource = acquireResource();
            useResource(resource);
        } catch (Exception e) {
            e.printStackTrace();
        } finally {
            releaseResource(resource);
        }
    }

    private static void riskyOperation() throws AppException {
        throw new NetworkException("https://example.com");
    }

    private static void multiRiskyOperation() throws IOException, AppException {
        if (System.currentTimeMillis() % 2 == 0) {
            throw new FileNotFoundException("not found");
        }
        throw new AppException("general", 400);
    }

    private static Object acquireResource() { return new Object(); }
    private static void useResource(Object r) {}
    private static void releaseResource(Object r) {}
    private static void handleError(int code, String message) {}
}

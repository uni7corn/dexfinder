package com.test.dexfinder.cases;

/**
 * Case 8: Static methods, singleton pattern, constants
 */
public class StaticCases {

    public static final String CONSTANT_VALUE = "static_constant";
    private static final int MAX_RETRY = 3;
    private static volatile StaticCases sInstance;

    private int counter = 0;

    private StaticCases() {}

    /** Double-checked locking singleton */
    public static StaticCases getInstance() {
        if (sInstance == null) {
            synchronized (StaticCases.class) {
                if (sInstance == null) {
                    sInstance = new StaticCases();
                }
            }
        }
        return sInstance;
    }

    public void doWork() {
        counter++;
        staticHelper(counter);
    }

    private static void staticHelper(int value) {
        if (value > MAX_RETRY) {
            return;
        }
    }

    /** Static factory method */
    public static StaticCases create() {
        return new StaticCases();
    }

    /** Static inner class with static method */
    public static class Config {
        public static final String KEY = "config_key";

        public static Config defaultConfig() {
            return new Config();
        }
    }
}

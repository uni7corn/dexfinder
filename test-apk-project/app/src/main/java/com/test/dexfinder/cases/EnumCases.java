package com.test.dexfinder.cases;

/**
 * Case 10: Enum — with fields, methods, abstract methods
 */
public class EnumCases {

    public enum Priority {
        LOW(1) {
            @Override
            public String label() { return "Low Priority"; }
        },
        MEDIUM(2) {
            @Override
            public String label() { return "Medium Priority"; }
        },
        HIGH(3) {
            @Override
            public String label() { return "High Priority"; }
        };

        private final int value;

        Priority(int value) {
            this.value = value;
        }

        public int getValue() { return value; }

        public abstract String label();

        public static Priority fromValue(int v) {
            for (Priority p : values()) {
                if (p.value == v) return p;
            }
            return MEDIUM;
        }
    }

    /** Enum in switch */
    public static void testEnum() {
        Priority p = Priority.fromValue(2);
        switch (p) {
            case LOW:
                handleLow();
                break;
            case MEDIUM:
                handleMedium();
                break;
            case HIGH:
                handleHigh();
                break;
        }
        String label = p.label();
    }

    private static void handleLow() {}
    private static void handleMedium() {}
    private static void handleHigh() {}
}

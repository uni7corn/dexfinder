package com.test.dexfinder.cases;

import android.content.Context;
import android.location.LocationManager;

/**
 * Case 7: Deep nested call chain A→B→C→D→...→target
 * Tests call graph traversal at depth
 */
public class DeepCallChain {

    public static void entryPoint(Context context) {
        level1(context);
    }

    private static void level1(Context context) {
        level2(context);
    }

    private static void level2(Context context) {
        level3(context);
    }

    private static void level3(Context context) {
        level4(context);
    }

    private static void level4(Context context) {
        level5(context);
    }

    private static void level5(Context context) {
        // Deep target: LocationManager call at depth 6
        LocationManager lm = (LocationManager) context.getSystemService(Context.LOCATION_SERVICE);
        try {
            lm.requestLocationUpdates(LocationManager.GPS_PROVIDER, 60000L, 100.0f,
                    location -> {
                        // Lambda listener at depth 6
                        double lat = location.getLatitude();
                    });
        } catch (SecurityException e) {
            e.printStackTrace();
        }
    }

    /** Cross-class call chain */
    public static class Orchestrator {
        private final Processor processor;

        public Orchestrator(Context context) {
            processor = new Processor(context);
        }

        public void execute() {
            processor.process();
        }
    }

    public static class Processor {
        private final Worker worker;

        public Processor(Context context) {
            worker = new Worker(context);
        }

        public void process() {
            worker.doWork();
        }
    }

    public static class Worker {
        private final Context context;

        public Worker(Context context) {
            this.context = context;
        }

        public void doWork() {
            LocationCases.requestGpsLocation(context);
        }
    }
}

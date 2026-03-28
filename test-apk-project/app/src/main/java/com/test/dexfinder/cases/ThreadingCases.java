package com.test.dexfinder.cases;

import android.content.Context;
import android.os.Handler;
import android.os.HandlerThread;
import android.os.Looper;
import android.os.Message;

import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

/**
 * Case 5: Threading — Thread, Handler, Executor, callbacks
 */
public class ThreadingCases {

    /** Callback interface for async results */
    public interface AsyncCallback<T> {
        void onResult(T result);
    }

    /** Direct Thread creation */
    public static void testThread() {
        // Anonymous Runnable in Thread
        new Thread(new Runnable() {
            @Override
            public void run() {
                doHeavyWork();
            }
        }, "worker-thread").start();

        // Lambda thread
        new Thread(() -> doHeavyWork()).start();
    }

    /** Handler + Looper pattern */
    public static void testHandler(Context context) {
        // Main thread handler
        Handler mainHandler = new Handler(Looper.getMainLooper()) {
            @Override
            public void handleMessage(Message msg) {
                // Case: anonymous class extending Handler
                String data = (String) msg.obj;
            }
        };

        mainHandler.post(() -> {
            // Lambda posted to handler
        });

        mainHandler.sendMessageDelayed(Message.obtain(mainHandler, 1, "test"), 1000);

        // HandlerThread pattern
        HandlerThread ht = new HandlerThread("bg-handler");
        ht.start();
        Handler bgHandler = new Handler(ht.getLooper());
        bgHandler.post(() -> doHeavyWork());
    }

    /** ExecutorService patterns */
    public static void testExecutorService() {
        ExecutorService executor = Executors.newFixedThreadPool(4);

        // Submit Runnable
        executor.submit(() -> doHeavyWork());

        // Submit Callable with Future
        Future<String> future = executor.submit(new Callable<String>() {
            @Override
            public String call() {
                return doHeavyWork();
            }
        });

        // ScheduledExecutorService
        ScheduledExecutorService scheduler = Executors.newScheduledThreadPool(1);
        scheduler.scheduleAtFixedRate(() -> doHeavyWork(), 0, 1, TimeUnit.SECONDS);

        executor.shutdown();
        scheduler.shutdown();
    }

    /** Async callback pattern */
    public static <T> void testAsyncCallback(AsyncCallback<T> callback) {
        new Thread(() -> {
            try {
                Thread.sleep(100);
                // Callback from background thread
                callback.onResult(null);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }).start();
    }

    private static String doHeavyWork() {
        try { Thread.sleep(10); } catch (InterruptedException e) {}
        return "done";
    }
}

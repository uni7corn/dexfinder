package com.test.dexfinder;

import android.app.Activity;
import android.os.Bundle;

/**
 * Case 1: Java Activity — basic lifecycle, inner class, anonymous class
 */
public class MainActivity extends Activity {

    private String mTitle;

    // Case: inner class
    public static class ViewHolder {
        public int position;
        public String label;

        public void bind(String text) {
            this.label = text;
        }
    }

    // Case: interface callback
    public interface OnItemClickListener {
        void onItemClick(int position, String data);
        default void onItemLongClick(int position) {
            // Case: default method
        }
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        // Case: anonymous inner class
        Runnable runnable = new Runnable() {
            @Override
            public void run() {
                updateTitle("from anonymous class");
            }
        };
        runnable.run();

        // Case: lambda (Java)
        OnItemClickListener listener = (pos, data) -> {
            mTitle = data + pos;
        };
        listener.onItemClick(0, "test");

        // Case: method reference
        new Thread(this::backgroundWork).start();

        // Case: nested call chain
        TestEntry.runAllTests(this);
    }

    private void updateTitle(String title) {
        mTitle = title;
    }

    private void backgroundWork() {
        updateTitle("from background thread");
    }
}

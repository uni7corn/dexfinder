package com.test.dexfinder.service;

import android.app.Service;
import android.content.Intent;
import android.os.IBinder;

import com.test.dexfinder.cases.LocationCases;

/**
 * Case 12: Service — cross-component call, tests startService → onCreate → location chain
 */
public class BackgroundLocationService extends Service {

    private LocationCases.LocationWrapper locationWrapper;

    @Override
    public void onCreate() {
        super.onCreate();
        locationWrapper = new LocationCases.LocationWrapper(this);
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        // Deep chain: Service → LocationWrapper → LocationCases → LocationManager
        locationWrapper.startTracking("gps");
        return START_STICKY;
    }

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    @Override
    public void onDestroy() {
        super.onDestroy();
    }
}

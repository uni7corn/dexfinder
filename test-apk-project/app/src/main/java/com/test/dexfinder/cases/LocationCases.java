package com.test.dexfinder.cases;

import android.content.Context;
import android.location.Criteria;
import android.location.Location;
import android.location.LocationListener;
import android.location.LocationManager;
import android.os.Bundle;
import android.os.Looper;

/**
 * Case 6: Location API — the primary search target
 * Covers all overloads of requestLocationUpdates
 */
public class LocationCases {

    /** requestLocationUpdates(String, long, float, LocationListener) */
    public static void requestGpsLocation(Context context) {
        LocationManager lm = (LocationManager) context.getSystemService(Context.LOCATION_SERVICE);
        try {
            lm.requestLocationUpdates(
                    LocationManager.GPS_PROVIDER,
                    1000L,
                    10.0f,
                    new SimpleLocationListener("gps")
            );
        } catch (SecurityException e) {
            e.printStackTrace();
        }
    }

    /** requestLocationUpdates(String, long, float, LocationListener, Looper) */
    public static void requestNetworkLocation(Context context) {
        LocationManager lm = (LocationManager) context.getSystemService(Context.LOCATION_SERVICE);
        try {
            lm.requestLocationUpdates(
                    LocationManager.NETWORK_PROVIDER,
                    5000L,
                    50.0f,
                    new SimpleLocationListener("network"),
                    Looper.getMainLooper()
            );
        } catch (SecurityException e) {
            e.printStackTrace();
        }
    }

    /** requestLocationUpdates(long, float, Criteria, LocationListener, Looper) */
    public static void requestWithCriteria(Context context) {
        LocationManager lm = (LocationManager) context.getSystemService(Context.LOCATION_SERVICE);
        Criteria criteria = new Criteria();
        criteria.setAccuracy(Criteria.ACCURACY_FINE);
        criteria.setPowerRequirement(Criteria.POWER_LOW);
        try {
            lm.requestLocationUpdates(3000L, 20.0f, criteria, new SimpleLocationListener("criteria"), Looper.getMainLooper());
        } catch (SecurityException e) {
            e.printStackTrace();
        }
    }

    /** Inner class implementing LocationListener */
    private static class SimpleLocationListener implements LocationListener {
        private final String tag;

        SimpleLocationListener(String tag) {
            this.tag = tag;
        }

        @Override
        public void onLocationChanged(Location location) {
            double lat = location.getLatitude();
            double lng = location.getLongitude();
        }

        @Override
        public void onStatusChanged(String provider, int status, Bundle extras) {}

        @Override
        public void onProviderEnabled(String provider) {}

        @Override
        public void onProviderDisabled(String provider) {}
    }

    /** Wrapper that delegates — tests deep call chain through location */
    public static class LocationWrapper {
        private final LocationManager lm;

        public LocationWrapper(Context context) {
            lm = (LocationManager) context.getSystemService(Context.LOCATION_SERVICE);
        }

        public void startTracking(String provider) {
            doRequestLocation(provider, 1000, 10.0f);
        }

        private void doRequestLocation(String provider, long minTime, float minDist) {
            try {
                lm.requestLocationUpdates(provider, minTime, minDist, new SimpleLocationListener("wrapper"));
            } catch (SecurityException e) {
                e.printStackTrace();
            }
        }
    }
}

package com.test.dexfinder.kotlin

import android.content.Context
import android.location.Location
import android.location.LocationManager
import android.os.Handler
import android.os.Looper
import kotlinx.coroutines.*

/**
 * Case 11: Kotlin-specific scenarios
 */
object KotlinCases {

    fun runAll(context: Context) {
        testDataClass()
        testSealedClass()
        testCompanionObject()
        testExtensionFunction(context)
        testHigherOrderFunction()
        testCoroutines(context)
        testInlineFunction()
        testLazyAndDelegates()
        testScopeFunctions()
        testNullSafety(context)
        testSequence()
        testDestructuring()
    }

    // --- Case 11a: Data class ---
    data class LocationData(
        val latitude: Double,
        val longitude: Double,
        val provider: String,
        val timestamp: Long = System.currentTimeMillis()
    )

    private fun testDataClass() {
        val loc = LocationData(39.9, 116.4, "gps")
        val copy = loc.copy(provider = "network")
        val (lat, lng, provider) = loc  // destructuring
        val hash = loc.hashCode()
        val str = loc.toString()
        val eq = loc == copy
    }

    // --- Case 11b: Sealed class ---
    sealed class LocationResult {
        data class Success(val location: LocationData) : LocationResult()
        data class Error(val message: String, val code: Int) : LocationResult()
        object Loading : LocationResult()
        object Idle : LocationResult()
    }

    private fun testSealedClass() {
        val result: LocationResult = LocationResult.Success(LocationData(0.0, 0.0, "test"))
        when (result) {
            is LocationResult.Success -> result.location.latitude
            is LocationResult.Error -> result.message
            LocationResult.Loading -> Unit
            LocationResult.Idle -> Unit
        }
    }

    // --- Case 11c: Companion object ---
    class LocationConfig {
        companion object {
            const val DEFAULT_INTERVAL = 1000L
            const val DEFAULT_DISTANCE = 10.0f
            private var instance: LocationConfig? = null

            @JvmStatic
            fun getDefault(): LocationConfig {
                return instance ?: LocationConfig().also { instance = it }
            }
        }

        var interval: Long = DEFAULT_INTERVAL
        var minDistance: Float = DEFAULT_DISTANCE
    }

    private fun testCompanionObject() {
        val config = LocationConfig.getDefault()
        config.interval = 5000L
        val interval = LocationConfig.DEFAULT_INTERVAL
    }

    // --- Case 11d: Extension functions ---
    private fun Location.toLocationData(): LocationData {
        return LocationData(latitude, longitude, provider ?: "unknown", time)
    }

    private fun Context.getLocationManager(): LocationManager {
        return getSystemService(Context.LOCATION_SERVICE) as LocationManager
    }

    private fun String.isValidProvider(): Boolean {
        return this == LocationManager.GPS_PROVIDER || this == LocationManager.NETWORK_PROVIDER
    }

    private fun testExtensionFunction(context: Context) {
        val lm = context.getLocationManager()
        val valid = "gps".isValidProvider()
    }

    // --- Case 11e: Higher-order functions & lambdas ---
    private fun <T> withRetry(times: Int, block: () -> T): T? {
        repeat(times) {
            try {
                return block()
            } catch (e: Exception) {
                if (it == times - 1) throw e
            }
        }
        return null
    }

    private inline fun measureTime(tag: String, block: () -> Unit) {
        val start = System.nanoTime()
        block()
        val elapsed = System.nanoTime() - start
    }

    private fun testHigherOrderFunction() {
        val list = listOf(1, 2, 3, 4, 5)

        // map/filter/reduce chain
        val result = list
            .filter { it > 2 }
            .map { it * 2 }
            .fold(0) { acc, v -> acc + v }

        // Function as parameter
        withRetry(3) { "success" }

        // Lambda with receiver
        val sb = StringBuilder().apply {
            append("hello")
            append(" ")
            append("world")
        }

        // let/also chain
        val processed = "test".let { it.uppercase() }.also { println(it) }
    }

    // --- Case 11f: Coroutines ---
    private fun testCoroutines(context: Context) {
        val scope = CoroutineScope(Dispatchers.Main + SupervisorJob())

        // launch
        scope.launch {
            val data = fetchLocationAsync()
            withContext(Dispatchers.Main) {
                // UI update
            }
        }

        // async/await
        scope.launch {
            val deferred1 = async(Dispatchers.IO) { fetchLocationAsync() }
            val deferred2 = async(Dispatchers.IO) { fetchLocationAsync() }
            val loc1 = deferred1.await()
            val loc2 = deferred2.await()
        }

        // withTimeout
        scope.launch {
            try {
                withTimeout(5000) {
                    fetchLocationAsync()
                }
            } catch (e: TimeoutCancellationException) {
                // timeout
            }
        }

        scope.cancel()
    }

    private suspend fun fetchLocationAsync(): LocationData {
        return withContext(Dispatchers.IO) {
            delay(100)
            LocationData(39.9, 116.4, "async")
        }
    }

    // --- Case 11g: Inline / reified ---
    private inline fun <reified T> isType(value: Any): Boolean {
        return value is T
    }

    private fun testInlineFunction() {
        val isString = isType<String>("hello")
        val isInt = isType<Int>(42)

        measureTime("test") {
            Thread.sleep(1)
        }
    }

    // --- Case 11h: Lazy & property delegates ---
    private val lazyValue: String by lazy {
        "computed once"
    }

    class ObservableProperty {
        var name: String = ""
            set(value) {
                val old = field
                field = value
                onChanged(old, value)
            }

        private fun onChanged(old: String, new: String) {}
    }

    private fun testLazyAndDelegates() {
        val v = lazyValue
        val obs = ObservableProperty()
        obs.name = "test"
    }

    // --- Case 11i: Scope functions ---
    private fun testScopeFunctions() {
        val list = mutableListOf<String>()

        // apply
        list.apply {
            add("a")
            add("b")
        }

        // run
        val size = list.run {
            add("c")
            size
        }

        // with
        val joined = with(list) {
            joinToString(",")
        }

        // takeIf / takeUnless
        val nonEmpty = list.takeIf { it.isNotEmpty() }
        val empty = list.takeUnless { it.isEmpty() }
    }

    // --- Case 11j: Null safety ---
    private fun testNullSafety(context: Context?) {
        // Safe call
        val name = context?.packageName

        // Elvis operator
        val safeName = context?.packageName ?: "unknown"

        // Let with null check
        context?.let {
            val lm = it.getSystemService(Context.LOCATION_SERVICE)
        }

        // Not-null assertion (dangerous but valid case)
        try {
            val forced = context!!.packageName
        } catch (e: NullPointerException) {
            // expected
        }
    }

    // --- Case 11k: Sequence (lazy evaluation) ---
    private fun testSequence() {
        val result = generateSequence(1) { it + 1 }
            .filter { it % 2 == 0 }
            .map { it * it }
            .take(10)
            .toList()
    }

    // --- Case 11l: Destructuring ---
    private fun testDestructuring() {
        val pair = Pair("key", 42)
        val (key, value) = pair

        val map = mapOf("a" to 1, "b" to 2)
        for ((k, v) in map) {
            println("$k=$v")
        }

        val triple = Triple("x", 1, true)
        val (a, b, c) = triple
    }
}

package com.test.dexfinder.cases;

import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Case 3: Generics — type erasure, bounded types, wildcards
 */
public class GenericCases {

    /** Generic class with type parameter */
    public static class Container<T> {
        private T value;
        private List<T> items = new ArrayList<>();

        public void set(T value) { this.value = value; }
        public T get() { return value; }
        public void addItem(T item) { items.add(item); }
        public List<T> getItems() { return items; }
    }

    /** Generic method with bounded type */
    public static <T extends Comparable<T>> T findMax(List<T> list) {
        return Collections.max(list);
    }

    /** Multiple type parameters */
    public static class Pair<K, V> {
        private K key;
        private V value;
        public Pair(K key, V value) { this.key = key; this.value = value; }
        public K getKey() { return key; }
        public V getValue() { return value; }
    }

    /** Generic interface */
    public interface Transformer<I, O> {
        O transform(I input);
    }

    public static void testGenericClass() {
        Container<String> strContainer = new Container<>();
        strContainer.set("hello");
        strContainer.addItem("world");
        String s = strContainer.get();

        Container<Integer> intContainer = new Container<>();
        intContainer.set(42);
        int val = intContainer.get();

        // Nested generics
        Container<List<String>> nested = new Container<>();
        nested.set(new ArrayList<>());

        // Map with generics
        Map<String, Container<Integer>> map = new HashMap<>();
        map.put("key", intContainer);
    }

    public static void testGenericMethod() {
        List<Integer> nums = new ArrayList<>();
        nums.add(3); nums.add(1); nums.add(2);
        Integer max = findMax(nums);

        // Generic method with lambda
        Transformer<String, Integer> parser = Integer::parseInt;
        int result = parser.transform("123");

        // Comparator (generic interface from stdlib)
        Comparator<String> comp = Comparator.comparingInt(String::length);
        Collections.sort(new ArrayList<String>(), comp);
    }

    public static void testWildcard() {
        List<? extends Number> numbers = new ArrayList<Integer>();
        List<? super Integer> superInts = new ArrayList<Number>();

        // Wildcard method
        printAll(numbers);
    }

    private static void printAll(List<? extends Number> list) {
        for (Number n : list) {
            n.intValue();
        }
    }

    public static void testBoundedType() {
        Pair<String, Integer> pair = new Pair<>("age", 30);
        String key = pair.getKey();
        Integer value = pair.getValue();
    }
}

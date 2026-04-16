Output dexfinder-demo.gif

Set Theme "Dracula"
Set Width 1200
Set Height 700
Set FontSize 14
Set WindowBar Colorful
Set TypingSpeed 30ms
Set Padding 20

# ── Scene 1: Stats ──
Type "# APK overview"
Enter
Sleep 300ms
Type "./dexfinder --dex-file testdata/test.apk --stats"
Enter
Sleep 2s

# ── Scene 2: Query with color ──
Type "# Find all getDeviceId calls (color output)"
Enter
Sleep 300ms
Type "./dexfinder --dex-file testdata/test.apk --query getDeviceId --color always"
Enter
Sleep 3s

# ── Scene 3: Call chain trace (tree) ──
Type "# Trace call chains — tree view"
Enter
Sleep 300ms
Type "./dexfinder --dex-file testdata/test.apk --query requestLocationUpdates --trace --depth 3"
Enter
Sleep 3s

# ── Scene 4: Deobfuscation with mapping ──
Type "# Deobfuscated trace with ProGuard mapping"
Enter
Sleep 300ms
Type "./dexfinder --dex-file testdata/test.apk --query KotlinCases --mapping testdata/test_mapping.txt --trace --depth 2"
Enter
Sleep 3s

# ── Scene 5: HTML report ──
Type "# Generate interactive HTML report"
Enter
Sleep 300ms
Type "./dexfinder --dex-file testdata/test.apk --query requestLocationUpdates --trace --format html --output report.html"
Enter
Sleep 2s
Type "ls -lh report.html"
Enter
Sleep 1500ms

# ── Scene 6: SARIF output ──
Type "# SARIF output for GitHub Code Scanning"
Enter
Sleep 300ms
Type "./dexfinder --dex-file testdata/test.apk --query getDeviceId --format sarif | head -20"
Enter
Sleep 3s

# ── Scene 7: Diff mode ──
Type "# APK diff (same file = no changes)"
Enter
Sleep 300ms
Type "./dexfinder --dex-file testdata/test.apk --diff testdata/test.apk --query requestLocationUpdates"
Enter
Sleep 2s

# ── Scene 8: JSON trace ──
Type "# JSON tree output"
Enter
Sleep 300ms
Type "./dexfinder --dex-file testdata/test.apk --query getDeviceId --trace --depth 2 --format json | head -25"
Enter
Sleep 3s

# Clean up
Sleep 1s
Type "rm -f report.html"
Enter
Sleep 500ms

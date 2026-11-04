# Keep the JNI entry points and the result type constructed from native code.
-keepclasseswithmembernames class * {
    native <methods>;
}
-keep class dev.filmprint.lab.NativeResult { *; }
-keep class dev.filmprint.lab.FilmEngine { *; }

package dev.filmprint.lab

/**
 * The outcome of one native render. Constructed from JNI; the field order and
 * types must match the constructor signature used in jni_bridge.cpp
 * (II[ILjava/lang/String;Ljava/lang/String;).
 */
class NativeResult(
    @JvmField val width: Int,
    @JvmField val height: Int,
    @JvmField val argb: IntArray?,
    @JvmField val error: String?,
    @JvmField val openReply: String?,
)

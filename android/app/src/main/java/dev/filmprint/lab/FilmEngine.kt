package dev.filmprint.lab

/**
 * Thin Kotlin facade over the SpektraLab C ABI (engine/include/spektrafilm/
 * spk_engine.h), loaded as libfilmprint_engine.so.
 *
 * The engine is created once against a resources directory (the staged
 * engine assets extracted to app storage) and reused for every render.
 */
object FilmEngine {
    init {
        System.loadLibrary("filmprint_engine")
    }

    private external fun nativeCreate(resourcesDir: String): Long
    private external fun nativeDestroy(handle: Long)
    private external fun nativeLastError(handle: Long): String
    private external fun nativeCapabilities(handle: Long): String
    private external fun nativeBuildInfo(): String
    private external fun nativeRender(
        handle: Long,
        pixels: FloatArray,
        width: Int,
        height: Int,
        paramsJson: String,
        tier: String,
    ): NativeResult

    @Volatile
    private var handle: Long = 0L

    val isCreated: Boolean get() = handle != 0L

    fun buildInfo(): String = nativeBuildInfo()

    /** Create the engine against [resourcesDir]; returns null on success or an error message. */
    @Synchronized
    fun create(resourcesDir: String): String? {
        if (handle != 0L) return null
        handle = nativeCreate(resourcesDir)
        if (handle == 0L) {
            val err = nativeLastError(0L)
            return if (err.isNotBlank()) err else "spk_engine_create returned null"
        }
        return null
    }

    @Synchronized
    fun destroy() {
        if (handle != 0L) {
            nativeDestroy(handle)
            handle = 0L
        }
    }

    fun capabilities(): String = if (handle != 0L) nativeCapabilities(handle) else ""

    /**
     * Render one frame. [pixels] is tightly packed RGB (sRGB-encoded 0..1,
     * top row first), length width*height*3.
     */
    @Synchronized
    fun render(
        pixels: FloatArray,
        width: Int,
        height: Int,
        paramsJson: String,
        tier: String = "full",
    ): NativeResult {
        if (handle == 0L) return NativeResult(0, 0, null, "engine is not created", null)
        return nativeRender(handle, pixels, width, height, paramsJson, tier)
    }
}

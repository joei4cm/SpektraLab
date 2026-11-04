package dev.filmprint.lab

import android.content.Context
import java.io.File

/**
 * The engine's baked resources (spektrafilm_constants.bin, the profile JSONs,
 * the compiled SPIR-V kernels, print LUTs) ship inside the APK under
 * assets/engine. The native engine needs them as real files on disk, so we
 * copy the subtree into app storage once and hand spk_engine_create that path.
 */
object EngineAssets {
    private const val ASSET_ROOT = "engine"
    private const val STAMP = "assets.version"

    fun ensureExtracted(context: Context): File {
        val dest = File(context.filesDir, ASSET_ROOT)
        val stampFile = File(dest, STAMP)
        val version = versionOf(context)
        if (stampFile.exists() && stampFile.readText() == version) return dest

        dest.deleteRecursively()
        dest.mkdirs()
        copyDir(context, ASSET_ROOT, dest)
        stampFile.writeText(version)
        return dest
    }

    private fun versionOf(context: Context): String = try {
        val info = context.packageManager.getPackageInfo(context.packageName, 0)
        "${info.versionName}-${info.longVersionCodeCompat()}"
    } catch (_: Exception) {
        "unknown"
    }

    private fun copyDir(context: Context, assetPath: String, destDir: File) {
        val entries = context.assets.list(assetPath) ?: emptyArray()
        if (entries.isEmpty()) {
            // A leaf (file). Copy it.
            copyFile(context, assetPath, destDir)
            return
        }
        destDir.mkdirs()
        for (entry in entries) {
            val childAsset = "$assetPath/$entry"
            val children = context.assets.list(childAsset) ?: emptyArray()
            if (children.isEmpty()) {
                copyFile(context, childAsset, File(destDir, entry))
            } else {
                copyDir(context, childAsset, File(destDir, entry))
            }
        }
    }

    private fun copyFile(context: Context, assetPath: String, destFile: File) {
        destFile.parentFile?.mkdirs()
        context.assets.open(assetPath).use { input ->
            destFile.outputStream().use { output -> input.copyTo(output) }
        }
    }
}

private fun android.content.pm.PackageInfo.longVersionCodeCompat(): Long =
    if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.P) longVersionCode
    else @Suppress("DEPRECATION") versionCode.toLong()

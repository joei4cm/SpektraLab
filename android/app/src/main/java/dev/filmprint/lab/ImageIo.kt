package dev.filmprint.lab

import android.content.ContentValues
import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.net.Uri
import android.os.Build
import android.provider.MediaStore
import java.io.InputStream

/** A decoded source frame: sRGB bitmap already downscaled for rendering. */
class SourceImage(val bitmap: Bitmap, val width: Int, val height: Int)

object ImageIo {
    /** Decode a picked JPEG/PNG and downscale so the long edge is <= [maxEdge]. */
    fun decode(context: Context, uri: Uri, maxEdge: Int = 2048): SourceImage? {
        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        context.contentResolver.openInputStream(uri)?.use {
            BitmapFactory.decodeStream(it, null, bounds)
        } ?: return null
        if (bounds.outWidth <= 0 || bounds.outHeight <= 0) return null

        var sample = 1
        val longEdge = maxOf(bounds.outWidth, bounds.outHeight)
        while (longEdge / sample > maxEdge) sample *= 2

        val opts = BitmapFactory.Options().apply {
            inSampleSize = sample
            inPreferredConfig = Bitmap.Config.ARGB_8888
        }
        val decoded = context.contentResolver.openInputStream(uri)?.use { input: InputStream ->
            BitmapFactory.decodeStream(input, null, opts)
        } ?: return null
        return SourceImage(decoded, decoded.width, decoded.height)
    }

    /**
     * ARGB_8888 (sRGB) -> tightly packed RGB floats in [0,1], top row first,
     * which is exactly what spk_open expects with input_color_space "sRGB".
     */
    fun toLinearFloats(bitmap: Bitmap): FloatArray {
        val w = bitmap.width
        val h = bitmap.height
        val pixels = IntArray(w * h)
        bitmap.getPixels(pixels, 0, w, 0, 0, w, h)
        val out = FloatArray(w * h * 3)
        val inv = 1.0f / 255.0f
        for (i in pixels.indices) {
            val p = pixels[i]
            out[i * 3] = ((p shr 16) and 0xFF) * inv
            out[i * 3 + 1] = ((p shr 8) and 0xFF) * inv
            out[i * 3 + 2] = (p and 0xFF) * inv
        }
        return out
    }

    fun bitmapFromArgb(argb: IntArray, width: Int, height: Int): Bitmap {
        val bmp = Bitmap.createBitmap(width, height, Bitmap.Config.ARGB_8888)
        bmp.setPixels(argb, 0, width, 0, 0, width, height)
        return bmp
    }

    /** Save [bitmap] as a PNG into the gallery (Pictures/FilmPrint Lab). */
    fun saveToGallery(context: Context, bitmap: Bitmap, displayName: String): Uri? {
        val values = ContentValues().apply {
            put(MediaStore.Images.Media.DISPLAY_NAME, displayName)
            put(MediaStore.Images.Media.MIME_TYPE, "image/png")
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                put(MediaStore.Images.Media.RELATIVE_PATH, "Pictures/FilmPrint Lab")
                put(MediaStore.Images.Media.IS_PENDING, 1)
            }
        }
        val resolver = context.contentResolver
        val collection = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q)
            MediaStore.Images.Media.getContentUri(MediaStore.VOLUME_EXTERNAL_PRIMARY)
        else MediaStore.Images.Media.EXTERNAL_CONTENT_URI
        val uri = resolver.insert(collection, values) ?: return null
        resolver.openOutputStream(uri)?.use { out ->
            bitmap.compress(Bitmap.CompressFormat.PNG, 100, out)
        } ?: return null
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            values.clear()
            values.put(MediaStore.Images.Media.IS_PENDING, 0)
            resolver.update(uri, values, null, null)
        }
        return uri
    }
}

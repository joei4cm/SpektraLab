// FilmPrint Lab -- JNI bridge to the SpektraLab C ABI (spk_engine.h).
//
// The boundary stays exactly as narrow as the C ABI: Kotlin hands down a
// tightly packed float RGB frame (sRGB-encoded 0..1, top row first) plus a
// JSON params delta, and gets back an ARGB_8888 int[] it wraps in a Bitmap.
// Nothing but the public spk_* surface is used; no engine C++ header crosses.
#include <jni.h>

#include <cstdint>
#include <cstring>
#include <string>
#include <vector>

#include "spektrafilm/spk_engine.h"

namespace {

std::string jstr(JNIEnv* env, jstring s) {
    if (!s) return {};
    const char* chars = env->GetStringUTFChars(s, nullptr);
    std::string out(chars ? chars : "");
    if (chars) env->ReleaseStringUTFChars(s, chars);
    return out;
}

jobject make_result(JNIEnv* env, jint width, jint height, jintArray argb,
                    const char* error, const char* open_reply) {
    jclass cls = env->FindClass("dev/filmprint/lab/NativeResult");
    if (!cls) return nullptr;
    jmethodID ctor = env->GetMethodID(
        cls, "<init>", "(II[ILjava/lang/String;Ljava/lang/String;)V");
    if (!ctor) return nullptr;
    jstring jerr = error ? env->NewStringUTF(error) : nullptr;
    jstring jreply = open_reply ? env->NewStringUTF(open_reply) : nullptr;
    return env->NewObject(cls, ctor, width, height, argb, jerr, jreply);
}

}  // namespace

extern "C" {

JNIEXPORT jlong JNICALL
Java_dev_filmprint_lab_FilmEngine_nativeCreate(JNIEnv* env, jclass, jstring resources_dir) {
    const std::string dir = jstr(env, resources_dir);
    spk_engine* engine = spk_engine_create(dir.c_str(), nullptr);
    return reinterpret_cast<jlong>(engine);
}

JNIEXPORT void JNICALL
Java_dev_filmprint_lab_FilmEngine_nativeDestroy(JNIEnv*, jclass, jlong handle) {
    if (handle) spk_engine_destroy(reinterpret_cast<spk_engine*>(handle));
}

JNIEXPORT jstring JNICALL
Java_dev_filmprint_lab_FilmEngine_nativeLastError(JNIEnv* env, jclass, jlong handle) {
    const char* msg = spk_last_error(reinterpret_cast<spk_engine*>(handle));
    return env->NewStringUTF(msg ? msg : "");
}

JNIEXPORT jstring JNICALL
Java_dev_filmprint_lab_FilmEngine_nativeCapabilities(JNIEnv* env, jclass, jlong handle) {
    if (!handle) return env->NewStringUTF("");
    const char* caps = spk_capabilities(reinterpret_cast<spk_engine*>(handle));
    return env->NewStringUTF(caps ? caps : "");
}

JNIEXPORT jstring JNICALL
Java_dev_filmprint_lab_FilmEngine_nativeBuildInfo(JNIEnv* env, jclass) {
    const char* info = spk_build_info();
    return env->NewStringUTF(info ? info : "");
}

// Open one frame, render the requested tier, and pack the result to
// ARGB_8888. The engine copies the pixels at spk_open, so the caller's
// float[] can be released immediately after.
JNIEXPORT jobject JNICALL
Java_dev_filmprint_lab_FilmEngine_nativeRender(
    JNIEnv* env, jclass, jlong handle, jfloatArray pixels,
    jint width, jint height, jstring params_json, jstring tier_str) {
    spk_engine* engine = reinterpret_cast<spk_engine*>(handle);
    if (!engine) return make_result(env, 0, 0, nullptr, "engine is not created", nullptr);

    const jsize count = env->GetArrayLength(pixels);
    if (count != static_cast<jsize>(width) * height * 3)
        return make_result(env, 0, 0, nullptr, "pixel array length must be width*height*3", nullptr);

    jfloat* data = env->GetFloatArrayElements(pixels, nullptr);
    if (!data) return make_result(env, 0, 0, nullptr, "cannot read pixel array", nullptr);

    const std::string params = jstr(env, params_json);
    const std::string tier = tier_str ? jstr(env, tier_str) : std::string("full");

    spk_image image{reinterpret_cast<const float*>(data),
                    static_cast<uint32_t>(width), static_cast<uint32_t>(height), 3};
    char* open_reply = nullptr;
    spk_session* session = spk_open(engine, &image, params.empty() ? nullptr : params.c_str(), &open_reply);
    env->ReleaseFloatArrayElements(pixels, data, JNI_ABORT);

    if (!session) {
        const char* err = spk_last_error(engine);
        if (open_reply) spk_string_free(open_reply);
        return make_result(env, 0, 0, nullptr, err && *err ? err : "spk_open failed", nullptr);
    }
    std::string reply = open_reply ? open_reply : "";
    if (open_reply) spk_string_free(open_reply);

    spk_result result{};
    spk_status status = spk_render(session, tier.c_str(), &result);
    if (status != SPK_OK) {
        const char* err = spk_last_error(engine);
        spk_result_free(&result);
        spk_session_release(session);
        return make_result(env, 0, 0, nullptr, err && *err ? err : "spk_render failed", reply.c_str());
    }

    const uint32_t ow = result.width, oh = result.height, stride = result.row_stride_px;
    jintArray argb = env->NewIntArray(static_cast<jsize>(ow) * oh);
    if (!argb || !result.rgba16) {
        spk_result_free(&result);
        spk_session_release(session);
        return make_result(env, 0, 0, nullptr, "engine returned no pixels", reply.c_str());
    }

    std::vector<jint> buffer(static_cast<size_t>(ow) * oh);
    for (uint32_t y = 0; y < oh; ++y) {
        const uint16_t* row = result.rgba16 + static_cast<size_t>(y) * stride * 4;
        for (uint32_t x = 0; x < ow; ++x) {
            const uint16_t* px = row + static_cast<size_t>(x) * 4;
            // RGBA16 (already in the sRGB output space) -> ARGB_8888.
            const uint32_t r = px[0] >> 8, g = px[1] >> 8, b = px[2] >> 8;
            buffer[static_cast<size_t>(y) * ow + x] =
                static_cast<jint>((0xFFu << 24) | (r << 16) | (g << 8) | b);
        }
    }
    env->SetIntArrayRegion(argb, 0, static_cast<jsize>(buffer.size()), buffer.data());

    spk_result_free(&result);
    spk_session_release(session);
    return make_result(env, static_cast<jint>(ow), static_cast<jint>(oh), argb, nullptr, reply.c_str());
}

}  // extern "C"

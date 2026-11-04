# FilmPrint Lab -- Android (NDK) native build of the SpektraLab C++/Vulkan
# render engine plus a thin JNI bridge.
#
# This reuses the portable engine sources verbatim (engine/src/core,
# engine/src/pipeline, engine/src/gpu/vulkan_gpu.cpp) with the same
# SPK_VULKAN_BACKEND path the Linux/Windows builds take. The Vulkan loader is
# never linked: engine/third_party/volk opens libvulkan.so at run time.
#
# Shaders are NOT compiled here; the Gradle `stageEngineAssets` task compiles
# the kernels to SPIR-V on the host and stages them (with the baked resources)
# into the APK assets. See android/README.md.
cmake_minimum_required(VERSION 3.22.1)
project(filmprint_engine LANGUAGES C CXX)

# android/app/src/main/cpp -> repo root is five levels up.
get_filename_component(SPK_REPO_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../../../../.." ABSOLUTE)
set(SPK_ENGINE "${SPK_REPO_ROOT}/engine")

if(NOT EXISTS "${SPK_ENGINE}/include/spektrafilm/spk_engine.h")
    message(FATAL_ERROR "Cannot find the engine tree at ${SPK_ENGINE}. "
                        "The Android module must live at <repo>/android.")
endif()

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

add_compile_definitions(SPK_VULKAN_BACKEND=1)

# --- the pinned imprint fonts, embedded exactly as engine/cmake/ImprintFonts.cmake does ---
set(imprint_font_dir "${SPK_ENGINE}/resources_product/fonts")
file(READ "${imprint_font_dir}/LiberationSans-Regular.ttf" imprint_regular HEX)
file(READ "${imprint_font_dir}/LiberationSans-Bold.ttf" imprint_bold HEX)
file(READ "${imprint_font_dir}/DejaVuSansMono-Bold.ttf" imprint_mono_bold HEX)
string(REGEX REPLACE "(..)" "0x\\1," imprint_regular "${imprint_regular}")
string(REGEX REPLACE "(..)" "0x\\1," imprint_bold "${imprint_bold}")
string(REGEX REPLACE "(..)" "0x\\1," imprint_mono_bold "${imprint_mono_bold}")
file(GENERATE OUTPUT "${CMAKE_CURRENT_BINARY_DIR}/generated/spk_imprint_fonts.hpp"
    CONTENT "#pragma once\nnamespace spk::imprint::font_data {\nconst unsigned char regular[] = {${imprint_regular}};\nconst unsigned char bold[] = {${imprint_bold}};\nconst unsigned char mono_bold[] = {${imprint_mono_bold}};\n}\n")
unset(imprint_regular)
unset(imprint_bold)
unset(imprint_mono_bold)

# --- volk (C), the run-time Vulkan loader ---
add_library(spk_volk STATIC "${SPK_ENGINE}/third_party/volk/volk.c")
set_target_properties(spk_volk PROPERTIES LINKER_LANGUAGE C)
target_include_directories(spk_volk PUBLIC "${SPK_ENGINE}/third_party/volk")
# NDK ships the Vulkan headers in the sysroot, so no Khronos tree is needed.

# --- the engine: core + pipeline + the Vulkan backend ---
set(SPK_CORE_SOURCES
    src/core/blob.cpp
    src/core/cam16.cpp
    src/core/colour.cpp
    src/core/curves.cpp
    src/core/digital_intermediate.cpp
    src/core/hanatos.cpp
    src/core/json.cpp
    src/core/latitude_fit.cpp
    src/core/numeric.cpp
    src/core/params.cpp
    src/core/print_lut.cpp
    src/core/printing.cpp
    src/core/profile.cpp
    src/core/setup_cache.cpp
    src/core/spectral.cpp)

set(SPK_PIPELINE_SOURCES
    src/pipeline/blur.cpp
    src/pipeline/engine.cpp
    src/pipeline/pipeline.cpp
    src/pipeline/scene_latitude.cpp
    src/pipeline/windows_unported.cpp
    src/pipeline/view_process.cpp
    src/pipeline/overscan.cpp
    src/pipeline/imprint_raster.cpp)

set(SPK_GPU_SOURCES src/gpu/vulkan_gpu.cpp)

list(TRANSFORM SPK_CORE_SOURCES PREPEND "${SPK_ENGINE}/")
list(TRANSFORM SPK_PIPELINE_SOURCES PREPEND "${SPK_ENGINE}/")
list(TRANSFORM SPK_GPU_SOURCES PREPEND "${SPK_ENGINE}/")

add_library(filmprint_engine SHARED
    ${SPK_CORE_SOURCES}
    ${SPK_PIPELINE_SOURCES}
    ${SPK_GPU_SOURCES}
    jni_bridge.cpp)

set_target_properties(filmprint_engine PROPERTIES OUTPUT_NAME "filmprint_engine")

target_include_directories(filmprint_engine PRIVATE
    "${SPK_ENGINE}/include"
    "${SPK_ENGINE}/src"
    "${SPK_ENGINE}/src/core"
    "${CMAKE_CURRENT_BINARY_DIR}/generated")

target_link_libraries(filmprint_engine PRIVATE spk_volk android log ${CMAKE_DL_LIBS})

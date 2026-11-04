import java.io.File

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

// The engine tree lives one level above the Android Gradle root (repo/engine).
val engineRoot: File = rootProject.projectDir.parentFile.resolve("engine")
val stagedAssetsDir: File = layout.buildDirectory.dir("generated/engineAssets").get().asFile

android {
    namespace = "dev.filmprint.lab"
    compileSdk = 34

    defaultConfig {
        applicationId = "dev.filmprint.lab"
        minSdk = 26
        targetSdk = 34
        versionCode = 1
        versionName = "0.1.0-mvp"

        ndk {
            // arm64 for modern devices; x86_64 so it also runs on an emulator.
            abiFilters += listOf("arm64-v8a", "x86_64")
        }
        externalNativeBuild {
            cmake {
                arguments += "-DANDROID_STL=c++_shared"
                cppFlags += "-std=c++20"
            }
        }
    }

    externalNativeBuild {
        cmake {
            path = file("src/main/cpp/CMakeLists.txt")
            version = "3.22.1"
        }
    }

    // The staged engine resources (constants, profiles, compiled SPIR-V) are
    // produced by stageEngineAssets into a generated dir and packed as assets.
    sourceSets.getByName("main") {
        assets.srcDir(stagedAssetsDir)
    }

    buildTypes {
        getByName("debug") {
            isMinifyEnabled = false
        }
        getByName("release") {
            isMinifyEnabled = false
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        compose = true
    }
    composeOptions {
        kotlinCompilerExtensionVersion = "1.5.14"
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
    }
}

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2024.06.00")
    implementation(composeBom)
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.activity:activity-compose:1.9.0")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-graphics")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.2")
}

// ---------------------------------------------------------------------------
// stageEngineAssets: compile the Vulkan kernels to SPIR-V on the host and
// stage them with the baked engine resources into the APK assets. Keeps the
// engine tree the single source of truth -- nothing is duplicated in git.
// ---------------------------------------------------------------------------
fun findGlslang(): String {
    System.getenv("GLSLANG_VALIDATOR")?.let { if (File(it).canExecute()) return it }
    System.getenv("VULKAN_SDK")?.let {
        val candidate = File(it, "bin/glslangValidator")
        if (candidate.canExecute()) return candidate.absolutePath
    }
    val path = System.getenv("PATH")?.split(File.pathSeparator).orEmpty()
    for (dir in path) {
        val candidate = File(dir, "glslangValidator")
        if (candidate.canExecute()) return candidate.absolutePath
    }
    throw GradleException(
        "glslangValidator not found. Install it (apt install glslang-tools, or the " +
            "Vulkan SDK) and ensure it is on PATH / VULKAN_SDK / GLSLANG_VALIDATOR."
    )
}

val stageEngineAssets by tasks.registering {
    val resources = engineRoot.resolve("resources")
    val product = engineRoot.resolve("resources_product")
    val shaderSrc = engineRoot.resolve("src/shaders/vulkan")
    val stbLicense = engineRoot.resolve("third_party/stb/LICENSE")
    val outEngine = File(stagedAssetsDir, "engine")

    inputs.dir(resources)
    inputs.dir(product)
    inputs.dir(shaderSrc)
    outputs.dir(stagedAssetsDir)

    doLast {
        delete(stagedAssetsDir)
        outEngine.mkdirs()

        // 1. Baked engine resources (constants.bin, profiles, print LUTs, io/).
        //    DI/*.cube is intentionally omitted: the MVP does not use the
        //    Digital Intermediate path and it is ~15 MB.
        copy {
            from(resources)
            into(outEngine)
        }
        // 2. Product profile overlay (B&W films + Multigrade paper).
        copy {
            from(product.resolve("profiles"))
            into(File(outEngine, "profiles"))
        }
        // 3. Product tables + attribution/licence texts (CC BY-SA, OFL, stb).
        copy {
            from(product.resolve("paper_grades.json"))
            from(product.resolve("neutral_print_filters.json")) { rename { "product-neutral-print-filters.json" } }
            from(product.resolve("NOTICE.md")) { rename { "Product-profiles-NOTICE.md" } }
            from(product.resolve("fonts/LICENSE")) { rename { "Imprint-fonts-LICENSE.txt" } }
            from(product.resolve("fonts/DejaVu-LICENSE")) { rename { "Imprint-mono-font-DejaVu-LICENSE.txt" } }
            if (stbLicense.exists()) from(stbLicense) { rename { "stb-LICENSE.txt" } }
            into(outEngine)
        }

        // 4. Compile every kernel to SPIR-V (the engine loads them by name).
        val glslang = findGlslang()
        val spvDir = File(outEngine, "vulkan")
        spvDir.mkdirs()
        val comps = shaderSrc.listFiles { f -> f.extension == "comp" }
            ?: throw GradleException("no .comp kernels found in $shaderSrc")
        comps.sortedBy { it.name }.forEach { comp ->
            val spv = File(spvDir, comp.nameWithoutExtension + ".spv")
            exec {
                commandLine(
                    glslang, "-V", "--target-env", "vulkan1.0",
                    "-I$shaderSrc", "-o", spv.absolutePath, comp.absolutePath,
                )
            }
        }
        logger.lifecycle("stageEngineAssets: staged ${comps.size} SPIR-V kernels + resources into $outEngine")
    }
}

tasks.named("preBuild").configure { dependsOn(stageEngineAssets) }

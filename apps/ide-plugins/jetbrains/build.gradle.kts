// Crucible JetBrains plugin.
// Targets IntelliJ Platform 2024.3+; cross-IDE (IDEA / GoLand / PyCharm / etc.)
// via since-build = 243.

import org.jetbrains.intellij.platform.gradle.TestFrameworkType

plugins {
    id("java")
    id("org.jetbrains.kotlin.jvm") version "2.1.0"
    id("org.jetbrains.intellij.platform") version "2.2.1"
}

group = "dev.crucible"
version = "2026.06.0-phase7"

repositories {
    mavenCentral()
    intellijPlatform { defaultRepositories() }
}

dependencies {
    intellijPlatform {
        intellijIdeaCommunity("2024.3")
        bundledPlugin("com.intellij.java")
        testFramework(TestFrameworkType.Platform)
    }
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.10.1")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")
    testImplementation("junit:junit:4.13.2")
}

intellijPlatform {
    pluginConfiguration {
        ideaVersion {
            sinceBuild = "243"
        }
        changeNotes = """
            Phase 7 — Initial public release.
            • Plan-approval tool window
            • Status-bar budget viewer
            • Attestation chain explorer
            • Junie-style tool window integration
        """.trimIndent()
    }
}

tasks {
    withType<org.jetbrains.kotlin.gradle.tasks.KotlinCompile> {
        kotlinOptions {
            jvmTarget = "21"
            freeCompilerArgs = listOf("-Xjsr305=strict")
        }
    }
}

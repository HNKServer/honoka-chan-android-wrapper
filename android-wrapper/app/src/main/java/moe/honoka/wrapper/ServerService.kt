package moe.honoka.wrapper

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.os.Build
import android.os.Handler
import android.os.IBinder
import android.os.Looper
import android.os.PowerManager
import android.system.Os
import android.system.OsConstants
import android.util.Log
import java.io.File
import java.util.Collections
import java.util.zip.ZipInputStream

class ServerService : Service() {
    companion object {
        const val ACTION_START = "moe.honoka.wrapper.START"
        const val ACTION_STOP = "moe.honoka.wrapper.STOP"
        const val ACTION_RESTART = "moe.honoka.wrapper.RESTART"
    }

    @Volatile
    private var process: Process? = null

    @Volatile
    private var workerRunning = false

    private val lock = Any()
    private val handler = Handler(Looper.getMainLooper())
    private var wakeLock: PowerManager.WakeLock? = null

    private val prefs by lazy {
        getSharedPreferences("honoka", MODE_PRIVATE)
    }

    private val goLogBuffer = Collections.synchronizedList(mutableListOf<String>())

    private val watchdog = object : Runnable {
        override fun run() {
            val shouldRun = prefs.getBoolean("desired_running", false)

            if (shouldRun && !workerRunning && !isProcessAlive(process)) {
                Log.w("HonokaServer", "Go process is not alive; watchdog will restart")
                startWorker(restart = false, reason = "watchdog 自动重启")
            }

            handler.postDelayed(this, 5000)
        }
    }

    override fun onCreate() {
        super.onCreate()

        createNotificationChannel()

        if (ServerState.state(this) == ServerState.RUNNING) {
            ServerState.set(this, ServerState.STOPPED, "服务进程刚创建，尚未启动 Go 子进程")
        }

        handler.post(watchdog)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        return when (intent?.action) {
            ACTION_STOP -> {
                prefs.edit().putBoolean("desired_running", false).apply()
                stopServerProcess(markStopped = true)

                if (Build.VERSION.SDK_INT >= 24) {
                    stopForeground(STOP_FOREGROUND_REMOVE)
                } else {
                    @Suppress("DEPRECATION")
                    stopForeground(true)
                }

                stopSelf()
                START_NOT_STICKY
            }

            ACTION_RESTART -> {
                prefs.edit().putBoolean("desired_running", true).apply()
                startForeground(1, buildNotification("正在重启"))
                startWorker(restart = true, reason = "用户手动重启")
                START_STICKY
            }

            else -> {
                prefs.edit().putBoolean("desired_running", true).apply()
                startForeground(1, buildNotification("正在启动"))

                if (!isProcessAlive(process)) {
                    startWorker(restart = false, reason = "用户手动启动")
                } else {
                    ServerState.set(this, ServerState.RUNNING, "Go 子进程已经在运行")
                    updateNotification("正在运行")
                }

                START_STICKY
            }
        }
    }

    private fun startWorker(restart: Boolean, reason: String) {
        synchronized(lock) {
            if (workerRunning) {
                return
            }
            workerRunning = true
        }

        Thread {
            try {
                ServerState.set(this, ServerState.STARTING, reason)
                updateNotification("正在启动")

                if (restart) {
                    stopServerProcess(markStopped = false)
                }

                startServerProcessInternal()

                val p = process

                if (isProcessAlive(p)) {
                    ServerState.set(
                        this,
                        ServerState.RUNNING,
                        "Go 子进程已启动，端口 ${ConfigUtil.getLastRunningPort(this)}"
                    )

                    updateNotification("正在运行：${ConfigUtil.localBaseUrl(this)}")

                    if (p != null) {
                        watchProcessExit(p)
                    }
                } else {
                    ServerState.set(
                        this,
                        ServerState.FAILED,
                        "Go 子进程启动后立即退出",
                        "process is not alive\n\nGo 输出：\n${lastGoLog()}"
                    )

                    updateNotification("启动失败")
                }
            } catch (e: Throwable) {
                Log.e("HonokaServer", "start failed", e)
                process = null

                ServerState.set(
                    this,
                    ServerState.FAILED,
                    "启动失败",
                    e.stackTraceToString()
                )

                updateNotification("启动失败：${e.message ?: e.javaClass.simpleName}")
                releaseWakeLock()
            } finally {
                workerRunning = false
            }
        }.start()
    }

    private fun watchProcessExit(p: Process) {
        Thread {
            try {
                val code = p.waitFor()

                if (process === p) {
                    process = null
                }

                releaseWakeLock()

                val desired = prefs.getBoolean("desired_running", false)

                if (desired) {
                    ServerState.set(
                        this,
                        ServerState.FAILED,
                        "Go 子进程退出，watchdog 将尝试重启",
                        "exit code=$code\n\nGo 输出：\n${lastGoLog()}"
                    )

                    updateNotification("进程退出，等待重启")
                } else {
                    ServerState.set(this, ServerState.STOPPED, "Go 子进程已停止")
                    updateNotification("已停止")
                }
            } catch (e: Throwable) {
                Log.e("HonokaServer", "waitFor failed", e)
            }
        }.start()
    }

    private fun startServerProcessInternal() {
        val serverDir = PathUtils.serverDir(this)

        clearGoLog()

        prepareServerBase(serverDir)
        verifyRequiredServerFiles(serverDir)
        prepareRuntimeDirs(serverDir)
        ensureDefaultConfigExists()
        linkArchives(serverDir)
        acquireWakeLock()

        val port = ConfigUtil.getPortFromConfigFile(this, "8080")
        ConfigUtil.setLastRunningPort(this, port)

        val exe = File(applicationInfo.nativeLibraryDir, "libhonoka.so")

        if (!exe.exists()) {
            throw IllegalStateException(
                "Go binary not found: ${exe.absolutePath}. " +
                    "请先执行 scripts/build_go_android.ps1 生成 app/src/main/jniLibs/arm64-v8a/libhonoka.so"
            )
        }

        val p = ProcessBuilder(exe.absolutePath)
            .directory(serverDir)
            .redirectErrorStream(true)
            .apply {
                environment()["HOME"] = serverDir.absolutePath
                environment()["TMPDIR"] = cacheDir.absolutePath
            }
            .start()

        process = p

        Thread {
            try {
                p.inputStream.bufferedReader().forEachLine {
                    appendGoLog(it)
                    Log.i("HonokaServer", it)
                }
            } catch (e: Throwable) {
                Log.e("HonokaServer", "log reader failed", e)
            }
        }.start()

        Thread.sleep(700)

        if (!isProcessAlive(p)) {
            val exit = try {
                p.exitValue()
            } catch (_: Throwable) {
                -999
            }

            throw IllegalStateException(
                "Go process exited immediately, exit code=$exit.\n\nGo 输出：\n${lastGoLog()}"
            )
        }
    }

    private fun stopServerProcess(markStopped: Boolean) {
        try {
            process?.destroy()
        } catch (_: Throwable) {
        }

        process = null
        releaseWakeLock()

        if (markStopped) {
            ServerState.set(this, ServerState.STOPPED, "用户已停止服务端")
            updateNotification("已停止")
        }
    }

    private fun prepareRuntimeDirs(serverDir: File) {
        File(serverDir, "data").mkdirs()
        File(serverDir, "assets/userdata").mkdirs()
        File(serverDir, "static/Android").mkdirs()
        File(serverDir, "static/Android/extracted").mkdirs()
    }

    private fun ensureDefaultConfigExists() {
        val f = PathUtils.configFile(this)

        if (!f.exists()) {
            f.parentFile?.mkdirs()
            f.writeText(ConfigUtil.defaultConfigText())
        }
    }

    private fun requiredServerFiles(serverDir: File): List<File> = listOf(
        File(serverDir, "config.json"),
        File(serverDir, "assets/main.db"),
        File(serverDir, "assets/certs/privatekey.pem"),
        File(serverDir, "assets/certs/publickey.pem"),
        File(serverDir, "assets/certs/certificate.crt"),
        File(serverDir, "assets/certs/server.crt")
    )

    private fun hasRequiredServerFiles(serverDir: File): Boolean {
        return requiredServerFiles(serverDir).all { it.exists() }
    }

    private fun verifyRequiredServerFiles(serverDir: File) {
        val missing = requiredServerFiles(serverDir).filter { !it.exists() }

        if (missing.isNotEmpty()) {
            throw IllegalStateException(
                "基础资源缺失：\n" +
                    missing.joinToString("\n") { it.absolutePath } +
                    "\n\n请确认 APK assets/server-base.zip 内包含这些文件；" +
                    "如果确认存在，请清空 APP 数据后重新启动。"
            )
        }
    }

    private fun clearGoLog() {
        goLogBuffer.clear()
        prefs.edit().putString("go_last_log", "").apply()
    }

    private fun appendGoLog(line: String) {
        synchronized(goLogBuffer) {
            goLogBuffer.add(line)

            while (goLogBuffer.size > 80) {
                goLogBuffer.removeAt(0)
            }

            prefs.edit()
                .putString("go_last_log", goLogBuffer.joinToString("\n"))
                .apply()
        }
    }

    private fun lastGoLog(): String {
        val inMemory = synchronized(goLogBuffer) {
            goLogBuffer.joinToString("\n")
        }

        if (inMemory.isNotBlank()) {
            return inMemory
        }

        return prefs.getString("go_last_log", "") ?: ""
    }

    /** API 23 compatible replacement for Process.isAlive, which requires Android API 26. */
    private fun isProcessAlive(p: Process?): Boolean {
        if (p == null) {
            return false
        }

        return try {
            p.exitValue()
            false
        } catch (_: IllegalThreadStateException) {
            true
        } catch (_: Throwable) {
            false
        }
    }

    private fun bundledServerBaseVersion(): String {
        return try {
            assets.open("server-base.sha256").bufferedReader().use { it.readText().trim() }
                .ifBlank { "legacy" }
        } catch (_: Throwable) {
            "legacy"
        }
    }

    private fun prepareServerBase(serverDir: File) {
        val marker = File(serverDir, ".base_extracted")
        val bundledVersion = bundledServerBaseVersion()
        val installedVersion = runCatching { marker.readText().trim() }.getOrDefault("")

        if (
            marker.exists() &&
            installedVersion == bundledVersion &&
            hasRequiredServerFiles(serverDir)
        ) {
            return
        }

        if (marker.exists()) {
            Log.w(
                "HonokaServer",
                "server-base changed or required files are missing; re-extracting. " +
                    "installed=$installedVersion bundled=$bundledVersion"
            )
            marker.delete()
        }

        serverDir.mkdirs()

        try {
            assets.open("server-base.zip").use { input ->
                ZipInputStream(input).use { zis ->
                    while (true) {
                        val entry = zis.nextEntry ?: break

                        /*
                         * Windows/PowerShell generated zip entries may be assets\main.db.
                         * Android/Linux treats backslash as a normal filename character, not a path separator.
                         */
                        val normalizedName = entry.name
                            .replace('\\', '/')
                            .removePrefix("/")

                        if (
                            normalizedName.isBlank() ||
                            normalizedName.contains("../") ||
                            normalizedName.startsWith("..")
                        ) {
                            Log.w("HonokaServer", "skipping unsafe zip entry: ${entry.name}")
                            zis.closeEntry()
                            continue
                        }

                        val outFile = File(serverDir, normalizedName)

                        if (entry.isDirectory || normalizedName.endsWith("/")) {
                            outFile.mkdirs()
                        } else {
                            outFile.parentFile?.mkdirs()

                            when {
                                normalizedName == "config.json" && outFile.exists() -> {
                                    Log.i(
                                        "HonokaServer",
                                        "skip existing user config: ${outFile.absolutePath}"
                                    )
                                }

                                normalizedName == "assets/data.db" && outFile.exists() -> {
                                    Log.i(
                                        "HonokaServer",
                                        "skip existing user database: ${outFile.absolutePath}"
                                    )
                                }

                                else -> {
                                    outFile.outputStream().use { output ->
                                        zis.copyTo(output)
                                    }
                                }
                            }
                        }

                        Log.i(
                            "HonokaServer",
                            "extract ${entry.name} -> ${outFile.absolutePath}"
                        )

                        zis.closeEntry()
                    }
                }
            }

            marker.writeText(bundledVersion)
        } catch (e: java.io.FileNotFoundException) {
            throw IllegalStateException(
                "APK assets 中没有 server-base.zip。请先执行 scripts/prepare_server_base_zip.ps1，然后重新编译 APK。",
                e
            )
        }
    }

    private fun linkArchives(serverDir: File) {
        val targetPath = PathUtils.getStoredArchivesDir(this)
        val target = File(targetPath)

        if (!target.exists() || !target.isDirectory) {
            throw IllegalStateException("archives directory not found: $targetPath")
        }

        if (Build.VERSION.SDK_INT >= 30 && !PathUtils.hasAllFilesAccess()) {
            throw IllegalStateException(
                "Android 11+ 尚未授予全部文件访问权限，Go 子进程无法直接读取公共目录：$targetPath"
            )
        }

        val link = File(serverDir, "static/Android/archives")
        link.parentFile?.mkdirs()

        removeSymlinkOrEmptyDir(link)

        if (link.exists()) {
            throw IllegalStateException(
                "Cannot replace non-empty archives directory: ${link.absolutePath}. " +
                    "如果这里不是空目录，请手动清空 APP 数据后重试。"
            )
        }

        Os.symlink(target.absolutePath, link.absolutePath)

        Log.i(
            "HonokaServer",
            "linked ${link.absolutePath} -> ${target.absolutePath}"
        )
    }

    private fun removeSymlinkOrEmptyDir(path: File) {
        try {
            val st = Os.lstat(path.absolutePath)
            val type = st.st_mode and OsConstants.S_IFMT

            if (type == OsConstants.S_IFLNK) {
                path.delete()
                return
            }
        } catch (_: Throwable) {
        }

        if (path.exists() && path.isDirectory) {
            val children = path.list()

            if (children == null || children.isEmpty()) {
                path.delete()
            }
        }
    }

    private fun acquireWakeLock() {
        if (wakeLock?.isHeld == true) {
            return
        }

        val pm = getSystemService(POWER_SERVICE) as PowerManager

        wakeLock = pm.newWakeLock(
            PowerManager.PARTIAL_WAKE_LOCK,
            "HonokaServer:WakeLock"
        ).apply {
            setReferenceCounted(false)
            acquire()
        }
    }

    private fun releaseWakeLock() {
        if (wakeLock?.isHeld == true) {
            wakeLock?.release()
        }

        wakeLock = null
    }

    override fun onDestroy() {
        handler.removeCallbacks(watchdog)
        stopServerProcess(markStopped = false)
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= 26) {
            val channel = NotificationChannel(
                "server",
                "Honoka Server",
                NotificationManager.IMPORTANCE_LOW
            )

            getSystemService(NotificationManager::class.java)
                .createNotificationChannel(channel)
        }
    }

    private fun updateNotification(text: String) {
        try {
            val nm = getSystemService(NotificationManager::class.java)
            nm.notify(1, buildNotification(text))
        } catch (_: Throwable) {
        }
    }

    private fun buildNotification(text: String = ConfigUtil.localBaseUrl(this)): Notification {
        val builder = if (Build.VERSION.SDK_INT >= 26) {
            Notification.Builder(this, "server")
        } else {
            @Suppress("DEPRECATION")
            Notification.Builder(this)
        }

        return builder
            .setContentTitle("Honoka Server")
            .setContentText(text)
            .setSmallIcon(android.R.drawable.stat_sys_upload_done)
            .setOngoing(true)
            .build()
    }
}

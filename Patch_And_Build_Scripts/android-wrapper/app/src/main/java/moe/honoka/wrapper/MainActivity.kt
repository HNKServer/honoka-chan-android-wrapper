package moe.honoka.wrapper

import android.Manifest
import android.app.Activity
import android.app.AlertDialog
import android.content.Intent
import android.database.sqlite.SQLiteDatabase
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.Toast
import com.google.android.material.button.MaterialButton
import com.google.android.material.textfield.TextInputEditText
import java.io.File
import java.net.HttpURLConnection
import java.net.URL
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

class MainActivity : Activity() {
    private lateinit var archivesEdit: TextInputEditText
    private lateinit var statusCardContent: LinearLayout
    private val uiHandler = android.os.Handler(android.os.Looper.getMainLooper())
    private var lastHealthText: String = "检测中"
    private var pendingImportTarget: String? = null
    private var pendingExportTarget: String? = null
    private var renderedFightTitleVisible: Boolean? = null

    companion object {
        private const val REQ_PICK_REAL_DIR = 3002
        private const val REQ_EXPORT_DB = 4101
        private const val REQ_IMPORT_DB = 4102
        private const val DB_MAIN = "main"
        private const val DB_USER = "user"
    }

    private val statusTicker = object : Runnable {
        override fun run() {
            checkHealthAsync()
            refreshStatusCard()
            uiHandler.postDelayed(this, 2000)
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        if (Build.VERSION.SDK_INT >= 33) {
            requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 100)
        } else if (Build.VERSION.SDK_INT < 30) {
            requestPermissions(arrayOf(Manifest.permission.READ_EXTERNAL_STORAGE, Manifest.permission.WRITE_EXTERNAL_STORAGE), 101)
        }
        buildUi()
    }

    override fun onResume() {
        super.onResume()

        val currentFightTitleVisible = Ui.showFightTitle(this)

        if (renderedFightTitleVisible != null && renderedFightTitleVisible != currentFightTitleVisible) {
            buildUi()
        } else {
            if (::archivesEdit.isInitialized) {
                archivesEdit.setText(PathUtils.getStoredArchivesDir(this))
            }
        }

        uiHandler.removeCallbacks(statusTicker)
        uiHandler.post(statusTicker)
    }

    override fun onPause() {
        uiHandler.removeCallbacks(statusTicker)
        super.onPause()
    }

    private fun buildUi() {
        renderedFightTitleVisible = Ui.showFightTitle(this)

        val root = Ui.contentRoot(this)

        val wide = Ui.isWide(this)
        val left = if (wide) Ui.column(this) else root
        val right = if (wide) Ui.column(this).apply { (layoutParams as LinearLayout.LayoutParams).marginEnd = 0 } else root

        val statusCard = buildStatusCard()
        val dataCard = buildDataCard()
        val serviceCard = buildServiceCard()
        val toolsCard = buildToolsCard()
        val backupCard = buildBackupCard()
        val permissionCard = buildPermissionCard()

        if (wide) {
            left.addView(statusCard)
            left.addView(dataCard)
            left.addView(permissionCard)
            right.addView(serviceCard)
            right.addView(toolsCard)
            right.addView(backupCard)
            val panes = Ui.twoPane(this)
            panes.addView(left)
            panes.addView(right)
            root.addView(panes)
        } else {
            root.addView(statusCard)
            root.addView(dataCard)
            root.addView(serviceCard)
            root.addView(toolsCard)
            root.addView(backupCard)
            root.addView(permissionCard)
        }

        setContentView(
            Ui.screenShell(
                this,
                "Honoka Server",
                "本地 Go 服务端控制台 · 路径选择 / 保活 / 配置热加载",
                root,
                onBack = null,
                onToggleTitle = {
                    Ui.toggleFightTitle(this)
                    buildUi()
                }
            )
        )
        refreshStatusCard()
    }

    private fun buildStatusCard(): com.google.android.material.card.MaterialCardView {
        val statusCard = Ui.card(this)
        statusCardContent = Ui.cardContent(this)
        statusCard.addView(statusCardContent)
        return statusCard
    }

    private fun buildDataCard(): com.google.android.material.card.MaterialCardView {
        val dataCard = Ui.card(this)
        val dataContent = Ui.cardContent(this)
        dataContent.addView(Ui.sectionTitle(this, "数据包位置"))
        dataContent.addView(Ui.body(this, "请选择真正包含 zip 数据包的 archives 文件夹。选择器使用真实文件路径，方便 Go 子进程通过 symlink 读取。"))
        val input = Ui.input(this, "archives 目录", "例如：${PathUtils.DEFAULT_ARCHIVES_DIR}", minLines = 2)
        archivesEdit = input.second
        archivesEdit.setText(PathUtils.getStoredArchivesDir(this))
        dataContent.addView(input.first)
        val row = Ui.row(this)
        row.addView(Ui.tonalButton(this, "选择目录").apply {
            layoutParams = Ui.rowButtonParams(this@MainActivity)
            setOnClickListener { pickArchivesDir() }
        })
        row.addView(Ui.filledButton(this, "保存位置").apply {
            layoutParams = Ui.rowButtonParams(this@MainActivity).apply { marginEnd = 0 }
            setOnClickListener { saveArchivesPath(showToast = true) }
        })
        dataContent.addView(row)
        dataCard.addView(dataContent)
        return dataCard
    }

    private fun buildServiceCard(): com.google.android.material.card.MaterialCardView {
        val serviceCard = Ui.card(this)
        val serviceContent = Ui.cardContent(this)
        serviceContent.addView(Ui.sectionTitle(this, "服务端控制"))
        serviceContent.addView(Ui.body(this, "启动后以前台服务运行，并由 watchdog 监控 Go 子进程。"))
        val startButton = Ui.filledButton(this, "启动服务端")
        startButton.setOnClickListener { startServer() }
        serviceContent.addView(startButton)
        val row = Ui.row(this)
        row.addView(MaterialButton(this).apply {
            text = "停止"
            cornerRadius = Ui.dp(this@MainActivity, 20)
            setTextColor(Ui.ORANGE_DARK_TEXT)
            setBackgroundColor(Ui.ORANGE_LIGHT)
            layoutParams = Ui.rowButtonParams(this@MainActivity)
            setOnClickListener { stopServer() }
        })
        row.addView(MaterialButton(this).apply {
            text = "重启"
            cornerRadius = Ui.dp(this@MainActivity, 20)
            setTextColor(Ui.ORANGE_DARK_TEXT)
            setBackgroundColor(Ui.ORANGE_LIGHT)
            layoutParams = Ui.rowButtonParams(this@MainActivity).apply { marginEnd = 0 }
            setOnClickListener { restartServer() }
        })
        serviceContent.addView(row)
        serviceContent.addView(Ui.tonalButton(this, "打开 WebUI").apply {
            setOnClickListener {
                if (lastHealthText != "在线") Toast.makeText(this@MainActivity, "服务端当前不是在线状态，请先看上方错误信息", Toast.LENGTH_LONG).show()
                startActivity(Intent(Intent.ACTION_VIEW, Uri.parse("${ConfigUtil.localBaseUrl(this@MainActivity)}/admin/index")))
            }
        })
        serviceCard.addView(serviceContent)
        return serviceCard
    }

    private fun buildToolsCard(): com.google.android.material.card.MaterialCardView {
        val card = Ui.card(this)
        val c = Ui.cardContent(this)
        c.addView(Ui.sectionTitle(this, "配置与快捷功能"))
        c.addView(Ui.body(this, "“解锁全部日替”不是单纯的 UI 配置项，它需要 Go 后端在返回日替列表时忽略 weekday 过滤。开启后会写入 config.json 并尝试热加载。"))
        c.addView(Ui.tonalButton(this, "编辑 config.json").apply {
            setOnClickListener { startActivity(Intent(this@MainActivity, ConfigActivity::class.java)) }
        })
        c.addView(Ui.filledButton(this, unlockButtonText()).apply {
            setOnClickListener { toggleUnlockAllSpecialRotation() }
        })
        card.addView(c)
        return card
    }

    private fun buildBackupCard(): com.google.android.material.card.MaterialCardView {
        val card = Ui.card(this)
        val c = Ui.cardContent(this)
        c.addView(Ui.sectionTitle(this, "数据库备份"))
        c.addView(Ui.body(this, "导入/导出对象是运行目录里的 assets/main.db 与 assets/data.db。为避免 SQLite 锁和 WAL 未落盘，操作前请先停止服务端。"))
        val mainRow = Ui.row(this)
        mainRow.addView(Ui.tonalButton(this, "导出 Main DB").apply {
            layoutParams = Ui.rowButtonParams(this@MainActivity)
            setOnClickListener { openExportDb(DB_MAIN) }
        })
        mainRow.addView(Ui.tonalButton(this, "导入 Main DB").apply {
            layoutParams = Ui.rowButtonParams(this@MainActivity).apply { marginEnd = 0 }
            setOnClickListener { openImportDb(DB_MAIN) }
        })
        c.addView(mainRow)
        val userRow = Ui.row(this)
        userRow.addView(Ui.tonalButton(this, "导出 User DB").apply {
            layoutParams = Ui.rowButtonParams(this@MainActivity)
            setOnClickListener { openExportDb(DB_USER) }
        })
        userRow.addView(Ui.tonalButton(this, "导入 User DB").apply {
            layoutParams = Ui.rowButtonParams(this@MainActivity).apply { marginEnd = 0 }
            setOnClickListener { openImportDb(DB_USER) }
        })
        c.addView(userRow)
        card.addView(c)
        return card
    }

    private fun buildPermissionCard(): com.google.android.material.card.MaterialCardView {
        val permissionCard = Ui.card(this)
        val permissionContent = Ui.cardContent(this)
        permissionContent.addView(Ui.sectionTitle(this, "权限与保活"))
        permissionContent.addView(Ui.body(this, "Android 11+ 如果要让 Go 子进程直接读取公共目录的大数据包，通常需要全部文件访问权限。息屏后要继续服务，则还建议忽略电池优化。"))
        permissionContent.addView(Ui.tonalButton(this, "授予全部文件访问权限").apply { setOnClickListener { openAllFilesAccessSettings() } })
        permissionContent.addView(Ui.tonalButton(this, "请求忽略电池优化").apply { setOnClickListener { requestIgnoreBatteryOptimization() } })
        permissionCard.addView(permissionContent)
        return permissionCard
    }

    private fun refreshStatusCard() {
        if (!::statusCardContent.isInitialized) return
        val desired = getSharedPreferences("honoka", MODE_PRIVATE).getBoolean("desired_running", false)
        val state = ServerState.state(this)
        val detail = ServerState.detail(this)
        val error = ServerState.lastError(this)
        val updated = ServerState.updatedAt(this)
        val updatedText = if (updated > 0) SimpleDateFormat("HH:mm:ss", Locale.getDefault()).format(Date(updated)) else "从未更新"
        statusCardContent.removeAllViews()
        statusCardContent.addView(Ui.sectionTitle(this, "运行状态"))
        statusCardContent.addView(Ui.body(this, """
            期望状态：${if (desired) "应保持运行" else "已请求停止"}
            服务状态：$state
            健康检查：$lastHealthText
            WebUI：${ConfigUtil.localBaseUrl(this)}/admin/index
            运行目录：${PathUtils.serverDir(this).absolutePath}
            数据包目录：${PathUtils.getStoredArchivesDir(this)}
            全部文件访问权限：${if (PathUtils.hasAllFilesAccess()) "已授权/不需要" else "未授权"}
            解锁全部日替：${if (ConfigUtil.getUnlockAllSpecialRotation(this)) "已开启" else "关闭"}
            更新时间：$updatedText
        """.trimIndent()))
        if (detail.isNotBlank()) statusCardContent.addView(Ui.body(this, "\n详情：$detail"))
        if (error.isNotBlank()) statusCardContent.addView(Ui.body(this, "\n错误：$error"))
        statusCardContent.addView(Ui.tonalButton(this, "刷新状态").apply {
            setOnClickListener { checkHealthAsync(); refreshStatusCard() }
        })
    }

    private fun checkHealthAsync() {
        Thread {
            val result = try {
                val url = URL("${ConfigUtil.localBaseUrl(this)}/__android/health")
                val conn = url.openConnection() as HttpURLConnection
                conn.connectTimeout = 800
                conn.readTimeout = 800
                conn.requestMethod = "GET"
                val code = conn.responseCode
                conn.disconnect()
                if (code in 200..299) "在线" else "HTTP $code"
            } catch (_: Throwable) {
                "离线/拒绝连接"
            }
            runOnUiThread { lastHealthText = result; refreshStatusCard() }
        }.start()
    }

    private fun openAllFilesAccessSettings() {
        if (Build.VERSION.SDK_INT >= 30) {
            try { startActivity(Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION, Uri.parse("package:$packageName"))) }
            catch (_: Throwable) { startActivity(Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION)) }
        } else {
            Toast.makeText(this, "Android 10 及以下通常不需要全部文件访问权限", Toast.LENGTH_SHORT).show()
        }
    }

    private fun pickArchivesDir() {
        val start = File(archivesEdit.text?.toString()?.trim().orEmpty()).takeIf { it.exists() }?.absolutePath ?: PathUtils.DEFAULT_ARCHIVES_DIR
        startActivityForResult(Intent(this, DirectoryPickerActivity::class.java).putExtra("start", start), REQ_PICK_REAL_DIR)
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (resultCode != RESULT_OK) return
        when (requestCode) {
            REQ_PICK_REAL_DIR -> {
                val path = data?.getStringExtra("path") ?: return
                archivesEdit.setText(path)
                saveArchivesPath(showToast = true)
            }
            REQ_EXPORT_DB -> data?.data?.let { exportDbToUri(it, pendingExportTarget ?: DB_USER) }
            REQ_IMPORT_DB -> data?.data?.let { confirmImportDb(it, pendingImportTarget ?: DB_USER) }
        }
    }

    private fun saveArchivesPath(showToast: Boolean): Boolean {
        val raw = archivesEdit.text?.toString()?.trim().orEmpty()
        if (raw.isEmpty()) { Toast.makeText(this, "路径为空", Toast.LENGTH_LONG).show(); return false }
        val normalized = PathUtils.normalizeArchivesDir(File(raw))
        if (!normalized.exists() || !normalized.isDirectory) {
            Toast.makeText(this, "目录不存在：${normalized.absolutePath}", Toast.LENGTH_LONG).show()
            return false
        }
        PathUtils.setStoredArchivesDir(this, normalized.absolutePath)
        archivesEdit.setText(normalized.absolutePath)
        if (showToast) Toast.makeText(this, "已保存数据包位置", Toast.LENGTH_SHORT).show()
        refreshStatusCard()
        return true
    }

    private fun startServer() {
        if (!saveArchivesPath(showToast = false)) return
        ServerState.set(this, ServerState.STARTING, "已发送启动请求")
        refreshStatusCard()
        getSharedPreferences("honoka", MODE_PRIVATE).edit().putBoolean("desired_running", true).apply()
        val intent = Intent(this, ServerService::class.java).setAction(ServerService.ACTION_START)
        if (Build.VERSION.SDK_INT >= 26) startForegroundService(intent) else startService(intent)
        Toast.makeText(this, "正在启动服务端", Toast.LENGTH_SHORT).show()
    }

    private fun stopServer() {
        getSharedPreferences("honoka", MODE_PRIVATE).edit().putBoolean("desired_running", false).apply()
        val intent = Intent(this, ServerService::class.java).setAction(ServerService.ACTION_STOP)
        startService(intent)
        ServerState.set(this, ServerState.STOPPED, "已发送停止请求")
        refreshStatusCard()
        Toast.makeText(this, "正在停止服务端", Toast.LENGTH_SHORT).show()
    }

    private fun restartServer() {
        if (!saveArchivesPath(showToast = false)) return
        ServerState.set(this, ServerState.STARTING, "已发送重启请求")
        refreshStatusCard()
        getSharedPreferences("honoka", MODE_PRIVATE).edit().putBoolean("desired_running", true).apply()
        val intent = Intent(this, ServerService::class.java).setAction(ServerService.ACTION_RESTART)
        if (Build.VERSION.SDK_INT >= 26) startForegroundService(intent) else startService(intent)
        Toast.makeText(this, "正在重启服务端", Toast.LENGTH_SHORT).show()
    }

    private fun unlockButtonText(): String = if (ConfigUtil.getUnlockAllSpecialRotation(this)) "关闭解锁全部日替" else "一键解锁全部日替"

    private fun toggleUnlockAllSpecialRotation() {
        val next = !ConfigUtil.getUnlockAllSpecialRotation(this)
        val text = ConfigUtil.setUnlockAllSpecialRotation(this, next)
        Toast.makeText(this, if (next) "已开启解锁全部日替" else "已关闭解锁全部日替", Toast.LENGTH_SHORT).show()
        reloadRuntimeConfig(text)
        refreshStatusCard()
        buildUi()
    }

    private fun reloadRuntimeConfig(configText: String) {
        Thread {
            for (port in ConfigUtil.candidatePorts(this, configText)) {
                try {
                    val conn = URL("http://127.0.0.1:$port/__android/config/reload").openConnection() as HttpURLConnection
                    conn.requestMethod = "POST"
                    conn.connectTimeout = 1500
                    conn.readTimeout = 1500
                    val code = conn.responseCode
                    conn.disconnect()
                    if (code in 200..299) return@Thread
                } catch (_: Throwable) {
                }
            }
        }.start()
    }

    private fun requestIgnoreBatteryOptimization() {
        if (Build.VERSION.SDK_INT >= 23) {
            try { startActivity(Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS, Uri.parse("package:$packageName"))) }
            catch (e: Throwable) { Toast.makeText(this, "无法打开电池优化设置：${e.message}", Toast.LENGTH_LONG).show() }
        }
    }

    private fun ensureDbOperationAllowed(): Boolean {
        val state = ServerState.state(this)
        val desired = getSharedPreferences("honoka", MODE_PRIVATE).getBoolean("desired_running", false)
        if (desired || state == ServerState.RUNNING || state == ServerState.STARTING || lastHealthText == "在线") {
            Toast.makeText(this, "请先停止服务端再导入或导出数据库备份", Toast.LENGTH_LONG).show()
            return false
        }
        return true
    }

    private fun dbFile(target: String): File {
        val name = if (target == DB_MAIN) "main.db" else "data.db"
        return File(PathUtils.serverDir(this), "assets/$name")
    }

    private fun openExportDb(target: String) {
        if (!ensureDbOperationAllowed()) return
        val source = dbFile(target)
        if (!source.exists()) {
            Toast.makeText(this, "数据库不存在：${source.absolutePath}\n请先启动一次服务端生成运行目录。", Toast.LENGTH_LONG).show()
            return
        }
        pendingExportTarget = target
        val title = if (target == DB_MAIN) "honoka-main.db" else "honoka-user-data.db"
        val intent = Intent(Intent.ACTION_CREATE_DOCUMENT)
            .addCategory(Intent.CATEGORY_OPENABLE)
            .setType("application/octet-stream")
            .putExtra(Intent.EXTRA_TITLE, title)
        startActivityForResult(intent, REQ_EXPORT_DB)
    }

    private fun openImportDb(target: String) {
        if (!ensureDbOperationAllowed()) return
        pendingImportTarget = target
        val intent = Intent(Intent.ACTION_OPEN_DOCUMENT)
            .addCategory(Intent.CATEGORY_OPENABLE)
            .setType("*/*")
        startActivityForResult(intent, REQ_IMPORT_DB)
    }

    private fun exportDbToUri(uri: Uri, target: String) {
        Thread {
            runCatching {
                val source = dbFile(target)
                if (target == DB_USER) checkpointDatabase(source)
                contentResolver.openOutputStream(uri)?.use { output ->
                    source.inputStream().use { input -> input.copyTo(output) }
                } ?: error("无法打开导出目标文件")
            }.onSuccess {
                runOnUiThread { Toast.makeText(this, "数据库已导出", Toast.LENGTH_SHORT).show() }
            }.onFailure {
                runOnUiThread { Toast.makeText(this, "导出失败：${it.message}", Toast.LENGTH_LONG).show() }
            }
        }.start()
    }

    private fun confirmImportDb(uri: Uri, target: String) {
        val label = if (target == DB_MAIN) "Main DB" else "User DB"
        AlertDialog.Builder(this)
            .setTitle("导入 $label")
            .setMessage("导入会覆盖当前运行目录里的 ${dbFile(target).name}，并删除同名 WAL/SHM 文件。是否继续？")
            .setNegativeButton("取消", null)
            .setPositiveButton("继续") { _, _ -> importDbFromUri(uri, target) }
            .show()
    }

    private fun importDbFromUri(uri: Uri, target: String) {
        Thread {
            runCatching {
                val dest = dbFile(target)
                dest.parentFile?.mkdirs()
                val temp = File(dest.parentFile, "${dest.name}.importing")
                contentResolver.openInputStream(uri)?.use { input ->
                    temp.outputStream().use { output -> input.copyTo(output) }
                } ?: error("无法打开备份文件")
                deleteSidecars(dest)
                if (dest.exists()) dest.delete()
                if (!temp.renameTo(dest)) {
                    temp.inputStream().use { input -> dest.outputStream().use { output -> input.copyTo(output) } }
                    temp.delete()
                }
                deleteSidecars(dest)
            }.onSuccess {
                runOnUiThread { Toast.makeText(this, "数据库已导入，下次启动服务端时生效", Toast.LENGTH_LONG).show() }
            }.onFailure {
                runOnUiThread { Toast.makeText(this, "导入失败：${it.message}", Toast.LENGTH_LONG).show() }
            }
        }.start()
    }

    private fun checkpointDatabase(file: File) {
        if (!file.exists()) return
        val db = SQLiteDatabase.openDatabase(file.absolutePath, null, SQLiteDatabase.OPEN_READWRITE)
        db.use {
            it.rawQuery("PRAGMA wal_checkpoint(TRUNCATE)", null).use { cursor -> while (cursor.moveToNext()) {} }
        }
    }

    private fun deleteSidecars(file: File) {
        File(file.parentFile, "${file.name}-wal").delete()
        File(file.parentFile, "${file.name}-shm").delete()
    }
}

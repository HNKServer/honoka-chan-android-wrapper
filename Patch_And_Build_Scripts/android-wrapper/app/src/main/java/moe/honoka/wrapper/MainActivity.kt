package moe.honoka.wrapper

import android.Manifest
import android.app.Activity
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.provider.Settings
import android.view.ViewGroup
import android.widget.ScrollView
import android.widget.Toast
import com.google.android.material.button.MaterialButton
import com.google.android.material.color.DynamicColors
import com.google.android.material.textfield.TextInputEditText
import java.io.File
import java.net.HttpURLConnection
import java.net.URL
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

class MainActivity : Activity() {
    private lateinit var archivesEdit: TextInputEditText
    private lateinit var statusCardContent: android.widget.LinearLayout
    private val uiHandler = Handler(Looper.getMainLooper())
    private var lastHealthText: String = "检测中"

    companion object {
        private const val REQ_PICK_REAL_DIR = 3002
    }

    private val statusTicker = object : Runnable {
        override fun run() {
            checkHealthAsync()
            refreshStatusCard()
            uiHandler.postDelayed(this, 2000)
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        DynamicColors.applyToActivityIfAvailable(this)
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
        if (::archivesEdit.isInitialized) {
            archivesEdit.setText(PathUtils.getStoredArchivesDir(this))
        }
        uiHandler.post(statusTicker)
    }

    override fun onPause() {
        uiHandler.removeCallbacks(statusTicker)
        super.onPause()
    }

    private fun buildUi() {
        val root = Ui.verticalRoot(this)
        root.addView(Ui.title(this, "Honoka Server"))
        root.addView(Ui.subtitle(this, "本地 Go 服务端控制台 · 路径选择 / 保活 / 配置热加载"))

        val statusCard = Ui.card(this)
        statusCardContent = Ui.cardContent(this)
        statusCard.addView(statusCardContent)
        root.addView(statusCard)
        refreshStatusCard()

        val dataCard = Ui.card(this)
        val dataContent = Ui.cardContent(this)
        dataContent.addView(Ui.sectionTitle(this, "数据包位置"))
        dataContent.addView(Ui.body(this, "请选择真正包含 zip 数据包的 archives 文件夹。现在的“选择 archives 目录”是 APP 内置的真实路径选择器，不再使用 Android 的 content:// 目录选择器。"))

        val input = Ui.input(
            this,
            hint = "archives 目录",
            helper = "例如：${PathUtils.DEFAULT_ARCHIVES_DIR}",
            minLines = 2
        )
        archivesEdit = input.second
        archivesEdit.setText(PathUtils.getStoredArchivesDir(this))
        dataContent.addView(input.first)

        val pickButton = Ui.tonalButton(this, "选择 archives 目录")
        pickButton.setOnClickListener { pickArchivesDir() }
        dataContent.addView(pickButton)

        val saveButton = Ui.filledButton(this, "保存数据包位置")
        saveButton.setOnClickListener { saveArchivesPath(showToast = true) }
        dataContent.addView(saveButton)

        dataCard.addView(dataContent)
        root.addView(dataCard)

        val serviceCard = Ui.card(this)
        val serviceContent = Ui.cardContent(this)
        serviceContent.addView(Ui.sectionTitle(this, "服务端控制"))
        serviceContent.addView(Ui.body(this, "启动后会以前台服务方式运行，并由 watchdog 监控 Go 子进程。启动失败时，错误会显示在上面的状态卡片里。"))

        val startButton = Ui.filledButton(this, "启动服务端")
        startButton.setOnClickListener { startServer() }
        serviceContent.addView(startButton)

        val row = Ui.row(this)
        val stopButton = MaterialButton(this).apply {
            text = "停止"
            cornerRadius = Ui.dp(this@MainActivity, 20)
            layoutParams = Ui.rowButtonParams(this@MainActivity)
            setOnClickListener { stopServer() }
        }
        val restartButton = MaterialButton(this).apply {
            text = "重启"
            cornerRadius = Ui.dp(this@MainActivity, 20)
            layoutParams = Ui.rowButtonParams(this@MainActivity).apply { marginEnd = 0 }
            setOnClickListener { restartServer() }
        }
        row.addView(stopButton)
        row.addView(restartButton)
        serviceContent.addView(row)

        val webButton = Ui.tonalButton(this, "打开 WebUI")
        webButton.setOnClickListener {
            if (lastHealthText != "在线") {
                Toast.makeText(this, "服务端当前不是在线状态，请先看上方错误信息", Toast.LENGTH_LONG).show()
            }
            startActivity(Intent(Intent.ACTION_VIEW, Uri.parse("${ConfigUtil.localBaseUrl(this)}/admin/index")))
        }
        serviceContent.addView(webButton)

        val configButton = Ui.tonalButton(this, "编辑 config.json")
        configButton.setOnClickListener { startActivity(Intent(this, ConfigActivity::class.java)) }
        serviceContent.addView(configButton)

        serviceCard.addView(serviceContent)
        root.addView(serviceCard)

        val permissionCard = Ui.card(this)
        val permissionContent = Ui.cardContent(this)
        permissionContent.addView(Ui.sectionTitle(this, "权限与保活"))
        permissionContent.addView(Ui.body(this, "Android 11+ 如果要让 Go 子进程直接读取公共目录的大数据包，通常需要全部文件访问权限。息屏后要继续服务，则还建议忽略电池优化。"))

        val grantButton = Ui.tonalButton(this, "授予全部文件访问权限")
        grantButton.setOnClickListener { openAllFilesAccessSettings() }
        permissionContent.addView(grantButton)

        val batteryButton = Ui.tonalButton(this, "请求忽略电池优化")
        batteryButton.setOnClickListener { requestIgnoreBatteryOptimization() }
        permissionContent.addView(batteryButton)

        permissionCard.addView(permissionContent)
        root.addView(permissionCard)

        val scroll = ScrollView(this).apply {
            addView(root, ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT))
        }
        setContentView(scroll)
    }

    private fun refreshStatusCard() {
        if (!::statusCardContent.isInitialized) return
        val desired = getSharedPreferences("honoka", MODE_PRIVATE).getBoolean("desired_running", false)
        val state = ServerState.state(this)
        val detail = ServerState.detail(this)
        val error = ServerState.lastError(this)
        val updated = ServerState.updatedAt(this)
        val updatedText = if (updated > 0) {
            SimpleDateFormat("HH:mm:ss", Locale.getDefault()).format(Date(updated))
        } else {
            "从未更新"
        }

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
            更新时间：$updatedText
        """.trimIndent()))

        if (detail.isNotBlank()) {
            statusCardContent.addView(Ui.body(this, "\n详情：$detail"))
        }
        if (error.isNotBlank()) {
            statusCardContent.addView(Ui.body(this, "\n错误：$error"))
        }

        val refresh = Ui.tonalButton(this, "刷新状态")
        refresh.setOnClickListener {
            checkHealthAsync()
            refreshStatusCard()
        }
        statusCardContent.addView(refresh)
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
            runOnUiThread {
                lastHealthText = result
                refreshStatusCard()
            }
        }.start()
    }

    private fun openAllFilesAccessSettings() {
        if (Build.VERSION.SDK_INT >= 30) {
            try {
                startActivity(Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION, Uri.parse("package:$packageName")))
            } catch (_: Throwable) {
                startActivity(Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION))
            }
        } else {
            Toast.makeText(this, "Android 10 及以下通常不需要全部文件访问权限", Toast.LENGTH_SHORT).show()
        }
    }

    private fun pickArchivesDir() {
        val start = File(archivesEdit.text?.toString()?.trim().orEmpty()).takeIf { it.exists() }?.absolutePath
            ?: PathUtils.DEFAULT_ARCHIVES_DIR
        val intent = Intent(this, DirectoryPickerActivity::class.java).putExtra("start", start)
        startActivityForResult(intent, REQ_PICK_REAL_DIR)
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode != REQ_PICK_REAL_DIR || resultCode != RESULT_OK) return
        val path = data?.getStringExtra("path") ?: return
        archivesEdit.setText(path)
        saveArchivesPath(showToast = true)
    }

    private fun saveArchivesPath(showToast: Boolean): Boolean {
        val raw = archivesEdit.text?.toString()?.trim().orEmpty()
        if (raw.isEmpty()) {
            Toast.makeText(this, "路径为空", Toast.LENGTH_LONG).show()
            return false
        }

        val normalized = PathUtils.normalizeArchivesDir(File(raw))
        if (!normalized.exists() || !normalized.isDirectory) {
            Toast.makeText(this, "目录不存在：${normalized.absolutePath}", Toast.LENGTH_LONG).show()
            return false
        }

        if (Build.VERSION.SDK_INT >= 30 && !PathUtils.hasAllFilesAccess()) {
            Toast.makeText(this, "尚未授予全部文件访问权限，Go 子进程可能无法读取公共目录", Toast.LENGTH_LONG).show()
        }

        archivesEdit.setText(normalized.absolutePath)
        PathUtils.setStoredArchivesDir(this, normalized.absolutePath)
        refreshStatusCard()
        if (showToast) Toast.makeText(this, "已保存数据包位置：${normalized.absolutePath}", Toast.LENGTH_SHORT).show()
        return true
    }

    private fun startServer() {
        if (!saveArchivesPath(showToast = false)) return
        ServerState.set(this, ServerState.STARTING, "已发送启动请求")
        refreshStatusCard()
        getSharedPreferences("honoka", MODE_PRIVATE).edit().putBoolean("desired_running", true).apply()
        val intent = Intent(this, ServerService::class.java).setAction(ServerService.ACTION_START)
        if (Build.VERSION.SDK_INT >= 26) startForegroundService(intent) else startService(intent)
        Toast.makeText(this, "正在启动服务端，稍等 1-3 秒后看状态卡片", Toast.LENGTH_SHORT).show()
    }

    private fun stopServer() {
        getSharedPreferences("honoka", MODE_PRIVATE).edit().putBoolean("desired_running", false).apply()
        startService(Intent(this, ServerService::class.java).setAction(ServerService.ACTION_STOP))
        Toast.makeText(this, "已发送停止请求", Toast.LENGTH_SHORT).show()
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

    private fun requestIgnoreBatteryOptimization() {
        if (Build.VERSION.SDK_INT >= 23) {
            try {
                startActivity(Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS, Uri.parse("package:$packageName")))
            } catch (e: Throwable) {
                Toast.makeText(this, "无法打开电池优化设置：${e.message}", Toast.LENGTH_LONG).show()
            }
        }
    }
}

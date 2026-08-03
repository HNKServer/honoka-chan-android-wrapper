package moe.honoka.wrapper

import android.app.Activity
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import android.view.ViewGroup
import android.widget.ScrollView
import android.widget.TextView
import android.widget.Toast
import com.google.android.material.button.MaterialButton
import com.google.android.material.card.MaterialCardView
import java.io.File
import java.util.Locale

class DirectoryPickerActivity : Activity() {
    private var currentDir: File = File(PathUtils.STORAGE_ROOT)
    private lateinit var root: android.widget.LinearLayout

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        currentDir = File(intent.getStringExtra("start") ?: PathUtils.STORAGE_ROOT)
        if (!currentDir.exists() || !currentDir.isDirectory) currentDir = File(PathUtils.STORAGE_ROOT)
        render()
    }

    private fun render() {
        root = Ui.verticalRoot(this)
        root.addView(Ui.header(this, "选择 archives 目录", "这是给 Go 子进程用的真实文件路径选择器，不使用 content://。"))

        val infoCard = Ui.card(this)
        val info = Ui.cardContent(this)
        info.addView(Ui.sectionTitle(this, "当前位置"))
        info.addView(Ui.body(this, currentDir.absolutePath))
        if (Build.VERSION.SDK_INT >= 30 && !PathUtils.hasAllFilesAccess()) {
            info.addView(Ui.body(this, "\n当前没有“全部文件访问权限”，可能无法读取公共目录。请先授权。"))
            val grant = Ui.tonalButton(this, "打开全部文件访问权限设置")
            grant.setOnClickListener { openAllFilesAccessSettings() }
            info.addView(grant)
        }
        infoCard.addView(info)
        root.addView(infoCard)

        val actionCard = Ui.card(this)
        val actions = Ui.cardContent(this)
        actions.addView(Ui.sectionTitle(this, "操作"))

        val select = Ui.filledButton(this, "使用此文件夹")
        select.setOnClickListener { selectCurrentDir() }
        actions.addView(select)

        val recommended = Ui.tonalButton(this, "跳到推荐目录")
        recommended.setOnClickListener {
            val f = File(PathUtils.DEFAULT_ARCHIVES_DIR)
            currentDir = if (f.exists()) f else File("/storage/emulated/0/Download")
            render()
        }
        actions.addView(recommended)

        val parent = Ui.tonalButton(this, "返回上一级")
        parent.setOnClickListener {
            currentDir.parentFile?.let {
                if (it.absolutePath.startsWith("/storage")) currentDir = it
            }
            render()
        }
        actions.addView(parent)

        actionCard.addView(actions)
        root.addView(actionCard)

        val listCard = Ui.card(this)
        val list = Ui.cardContent(this)
        list.addView(Ui.sectionTitle(this, "子文件夹"))

        val dirs = currentDir.listFiles()
            ?.filter { it.isDirectory && !it.name.startsWith(".") }
            ?.sortedWith(compareBy<File> { !likelyImportant(it.name) }.thenBy { it.name.lowercase(Locale.ROOT) })
            ?: emptyList()

        if (dirs.isEmpty()) {
            list.addView(Ui.body(this, "没有可进入的子文件夹，或当前目录无读取权限。"))
        } else {
            dirs.take(160).forEach { dir ->
                val b = MaterialButton(this).apply {
                    text = if (PathUtils.looksLikeArchivesDir(dir)) "📦 ${dir.name}" else "📁 ${dir.name}"
                    cornerRadius = Ui.dp(this@DirectoryPickerActivity, 18)
                    layoutParams = android.widget.LinearLayout.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        Ui.dp(this@DirectoryPickerActivity, 48)
                    ).apply { topMargin = Ui.dp(this@DirectoryPickerActivity, 6) }
                    setOnClickListener {
                        currentDir = dir
                        render()
                    }
                }
                list.addView(b)
            }
            if (dirs.size > 160) {
                list.addView(Ui.body(this, "\n只显示前 160 个文件夹，请进入更具体的目录。"))
            }
        }

        listCard.addView(list)
        root.addView(listCard)

        setContentView(Ui.safeScroll(this, root))
    }

    private fun likelyImportant(name: String): Boolean {
        val n = name.lowercase(Locale.ROOT)
        return n == "download" || n == "downloads" || n == "honokadata" || n == "static" || n == "android" || n == "archives" || n == "documents"
    }

    private fun selectCurrentDir() {
        val normalized = PathUtils.normalizeArchivesDir(currentDir)
        if (!normalized.exists() || !normalized.isDirectory) {
            Toast.makeText(this, "目录不存在：${normalized.absolutePath}", Toast.LENGTH_LONG).show()
            return
        }
        val out = Intent().putExtra("path", normalized.absolutePath)
        setResult(RESULT_OK, out)
        finish()
    }

    private fun openAllFilesAccessSettings() {
        if (Build.VERSION.SDK_INT >= 30) {
            try {
                startActivity(Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION, Uri.parse("package:$packageName")))
            } catch (_: Throwable) {
                startActivity(Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION))
            }
        }
    }
}

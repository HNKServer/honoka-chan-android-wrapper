package moe.honoka.wrapper

import android.app.Activity
import android.content.Intent
import android.os.Build
import android.os.Bundle
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.Toast
import com.google.android.material.textfield.TextInputEditText
import org.json.JSONArray
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

class ConfigActivity : Activity() {
    private lateinit var formRoot: LinearLayout
    private lateinit var jsonRoot: JSONObject
    private val entries = mutableListOf<ConfigEntry>()

    private data class ConfigEntry(
        val path: String,
        val parent: Any,
        val key: Any,
        val originalValue: Any?,
        val edit: TextInputEditText
    )

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        buildUi()
    }

    private fun buildUi() {
        val root = Ui.contentRoot(this)

        val wide = Ui.isWide(this)
        val mainContainer = if (wide) Ui.twoPane(this) else root
        val left = if (wide) Ui.column(this) else root
        val right = if (wide) Ui.column(this).apply { (layoutParams as LinearLayout.LayoutParams).marginEnd = 0 } else root

        val formCard = Ui.card(this)
        val formContent = Ui.cardContent(this)
        formContent.addView(Ui.sectionTitle(this, "配置项"))
        formContent.addView(Ui.body(this, "布尔值请填 true / false；数字请填数字；字符串会按文本保存。listen_port 修改后需要重启服务端。"))
        formRoot = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL }
        formContent.addView(formRoot)
        formCard.addView(formContent)
        left.addView(formCard)

        val actionCard = Ui.card(this)
        val actionContent = Ui.cardContent(this)
        actionContent.addView(Ui.sectionTitle(this, "操作"))

        val reloadFields = Ui.tonalButton(this, "重新读取 config.json")
        reloadFields.setOnClickListener { rebuildForm() }
        actionContent.addView(reloadFields)

        val save = Ui.tonalButton(this, "保存 config.json")
        save.setOnClickListener { saveConfigOnly() }
        actionContent.addView(save)

        val saveApply = Ui.filledButton(this, "保存并实时应用到内存")
        saveApply.setOnClickListener {
            if (saveConfigOnly()) reloadRuntimeConfig(currentConfigText())
        }
        actionContent.addView(saveApply)

        val saveRestart = Ui.tonalButton(this, "保存并重启服务端")
        saveRestart.setOnClickListener {
            if (saveConfigOnly()) restartServer()
        }
        actionContent.addView(saveRestart)

        val unlock = Ui.tonalButton(this, "一键解锁全部日替")
        unlock.setOnClickListener {
            ConfigUtil.setUnlockAllSpecialRotation(this, true)
            rebuildForm()
            reloadRuntimeConfig(ConfigUtil.readConfigText(this))
            Toast.makeText(this, "已写入 unlock_all_special_rotation=true", Toast.LENGTH_SHORT).show()
        }
        actionContent.addView(unlock)


        actionCard.addView(actionContent)
        right.addView(actionCard)

        val hintCard = Ui.card(this)
        val hintContent = Ui.cardContent(this)
        hintContent.addView(Ui.sectionTitle(this, "说明"))
        hintContent.addView(Ui.body(this, "这个页面不再是一个大文本框，而是根据 config.json 的实际字段动态生成。保存时只更新叶子字段，不会把 settings/user_prefs 这种层级拍扁。"))
        hintCard.addView(hintContent)
        right.addView(hintCard)

        if (wide) {
            mainContainer.addView(left)
            mainContainer.addView(right)
            root.addView(mainContainer)
        }

        setContentView(
            Ui.screenShell(
                this,
                "编辑 config.json",
                "按配置项动态生成单行编辑框，保存时保持原 JSON 层级",
                root,
                onBack = { finish() },
                onToggleTitle = {
                    Ui.toggleFightTitle(this)
                    buildUi()
                }
            )
        )
        rebuildForm()
    }

    private fun rebuildForm() {
        entries.clear()
        formRoot.removeAllViews()
        jsonRoot = try {
            JSONObject(ConfigUtil.readConfigText(this))
        } catch (e: Throwable) {
            Toast.makeText(this, "config.json 解析失败：${e.message}", Toast.LENGTH_LONG).show()
            JSONObject(ConfigUtil.defaultConfigText())
        }
        renderObject("", jsonRoot, formRoot)
    }

    private fun renderObject(prefix: String, obj: JSONObject, container: LinearLayout) {
        val keys = obj.keys().asSequence().toList()
        for (key in keys) {
            val value = obj.opt(key)
            val path = if (prefix.isBlank()) key else "$prefix.$key"
            when (value) {
                is JSONObject -> {
                    container.addView(Ui.sectionTitle(this, path))
                    renderObject(path, value, container)
                }
                is JSONArray -> {
                    if (value.length() == 0) {
                        addField(container, path, obj, key, value.toString())
                    } else {
                        container.addView(Ui.sectionTitle(this, path))
                        for (i in 0 until value.length()) {
                            val item = value.opt(i)
                            val childPath = "$path[$i]"
                            if (item is JSONObject) renderObject(childPath, item, container)
                            else addField(container, childPath, value, i, item)
                        }
                    }
                }
                else -> addField(container, path, obj, key, value)
            }
        }
    }

    private fun addField(container: LinearLayout, path: String, parent: Any, key: Any, value: Any?) {
        val input = Ui.input(
            this,
            hint = path,
            helper = typeHelper(value),
            minLines = 1,
            singleLine = true,
            monospace = false
        )
        val edit = input.second
        edit.setText(if (value == JSONObject.NULL || value == null) "" else value.toString())
        entries.add(ConfigEntry(path, parent, key, value, edit))
        container.addView(input.first)
    }

    private fun typeHelper(value: Any?): String {
        return when (value) {
            is Boolean -> "Boolean: true / false"
            is Int, is Long -> "Integer"
            is Number -> "Number"
            JSONObject.NULL, null -> "null / string"
            else -> "String"
        }
    }

    private fun currentConfigText(): String {
        applyEntriesToJson()
        return jsonRoot.toString(2) + "\n"
    }

    private fun applyEntriesToJson() {
        for (entry in entries) {
            val text = entry.edit.text?.toString().orEmpty()
            val value = parseValue(text, entry.originalValue, entry.path)
            when (entry.parent) {
                is JSONObject -> entry.parent.put(entry.key as String, value)
                is JSONArray -> entry.parent.put(entry.key as Int, value)
            }
        }
    }

    private fun parseValue(text: String, original: Any?, path: String): Any {
        if (original == JSONObject.NULL || original == null) {
            return if (text.equals("null", ignoreCase = true)) JSONObject.NULL else text
        }
        return when (original) {
            is Boolean -> when (text.lowercase()) {
                "true" -> true
                "false" -> false
                else -> throw IllegalArgumentException("$path 需要 true 或 false")
            }
            is Int -> text.toIntOrNull() ?: throw IllegalArgumentException("$path 需要整数")
            is Long -> text.toLongOrNull() ?: throw IllegalArgumentException("$path 需要整数")
            is Double, is Float -> text.toDoubleOrNull() ?: throw IllegalArgumentException("$path 需要数字")
            is Number -> text.toDoubleOrNull() ?: throw IllegalArgumentException("$path 需要数字")
            else -> text
        }
    }

    private fun saveConfigOnly(): Boolean {
        return try {
            val text = currentConfigText()
            JSONObject(text)
            ConfigUtil.writeConfigText(this, text)
            Toast.makeText(this, "已保存 config.json", Toast.LENGTH_SHORT).show()
            true
        } catch (e: Throwable) {
            Toast.makeText(this, "保存失败：${e.message}", Toast.LENGTH_LONG).show()
            false
        }
    }

    private fun reloadRuntimeConfig(configText: String) {
        Thread {
            var lastError: Throwable? = null
            for (port in ConfigUtil.candidatePorts(this, configText)) {
                try {
                    val url = URL("http://127.0.0.1:$port/__android/config/reload")
                    val conn = url.openConnection() as HttpURLConnection
                    conn.requestMethod = "POST"
                    conn.connectTimeout = 2000
                    conn.readTimeout = 2000
                    val code = conn.responseCode
                    val body = if (code in 200..299) conn.inputStream.bufferedReader().readText()
                    else conn.errorStream?.bufferedReader()?.readText().orEmpty()
                    conn.disconnect()
                    if (code in 200..299) {
                        runOnUiThread { Toast.makeText(this, "运行时配置已更新", Toast.LENGTH_SHORT).show() }
                        return@Thread
                    } else {
                        throw IllegalStateException("HTTP $code $body")
                    }
                } catch (e: Throwable) {
                    lastError = e
                }
            }
            runOnUiThread {
                Toast.makeText(this, "热加载失败：服务端可能没启动，或端口修改后需要重启。${lastError?.message}", Toast.LENGTH_LONG).show()
            }
        }.start()
    }

    private fun restartServer() {
        getSharedPreferences("honoka", MODE_PRIVATE).edit().putBoolean("desired_running", true).apply()
        val intent = Intent(this, ServerService::class.java).setAction(ServerService.ACTION_RESTART)
        if (Build.VERSION.SDK_INT >= 26) startForegroundService(intent) else startService(intent)
        Toast.makeText(this, "已请求重启服务端", Toast.LENGTH_SHORT).show()
    }
}

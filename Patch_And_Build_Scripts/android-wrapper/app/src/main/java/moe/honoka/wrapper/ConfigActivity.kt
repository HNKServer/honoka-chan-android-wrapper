package moe.honoka.wrapper

import android.app.Activity
import android.content.Intent
import android.graphics.Typeface
import android.os.Build
import android.os.Bundle
import android.util.TypedValue
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.Toast
import com.google.android.material.button.MaterialButton
import com.google.android.material.color.DynamicColors
import com.google.android.material.textfield.TextInputEditText
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

class ConfigActivity : Activity() {
    private lateinit var editor: TextInputEditText

    override fun onCreate(savedInstanceState: Bundle?) {
        DynamicColors.applyToActivityIfAvailable(this)
        super.onCreate(savedInstanceState)

        val root = Ui.verticalRoot(this)
        root.addView(Ui.title(this, "编辑 config.json"))
        root.addView(Ui.subtitle(this, "直接修改原始 JSON，不新增字段，不改变结构"))

        val hintCard = Ui.card(this)
        val hintContent = Ui.cardContent(this)
        hintContent.addView(Ui.sectionTitle(this, "说明"))
        hintContent.addView(Ui.body(this, "修改 user_prefs / sif_cdn_server / as_cdn_server 后可以实时应用到内存；修改 server_port 后请保存并重启服务端。"))
        hintCard.addView(hintContent)
        root.addView(hintCard)

        val editorCard = Ui.card(this)
        val editorContent = Ui.cardContent(this)
        editorContent.addView(Ui.sectionTitle(this, "原始配置"))
        val input = Ui.input(
            this,
            hint = "config.json",
            helper = "保存前只做 JSON 合法性检查，不会格式化或重排字段。",
            minLines = 22,
            monospace = true
        )
        editor = input.second
        editor.setText(ConfigUtil.readConfigText(this))
        editor.setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
        editor.typeface = Typeface.MONOSPACE
        editorContent.addView(input.first)
        editorCard.addView(editorContent)
        root.addView(editorCard)

        val actionCard = Ui.card(this)
        val actionContent = Ui.cardContent(this)
        actionContent.addView(Ui.sectionTitle(this, "操作"))

        val save = Ui.tonalButton(this, "保存 config.json")
        save.setOnClickListener { saveConfigOnly() }
        actionContent.addView(save)

        val saveApply = Ui.filledButton(this, "保存并实时应用到内存")
        saveApply.setOnClickListener {
            if (saveConfigOnly()) reloadRuntimeConfig()
        }
        actionContent.addView(saveApply)

        val saveRestart = MaterialButton(this).apply {
            text = "保存并重启服务端"
            cornerRadius = Ui.dp(this@ConfigActivity, 20)
            layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                Ui.dp(this@ConfigActivity, 52)
            ).apply { topMargin = Ui.dp(this@ConfigActivity, 10) }
            setOnClickListener {
                if (saveConfigOnly()) restartServer()
            }
        }
        actionContent.addView(saveRestart)

        val back = Ui.tonalButton(this, "返回")
        back.setOnClickListener { finish() }
        actionContent.addView(back)

        actionCard.addView(actionContent)
        root.addView(actionCard)

        val scroll = ScrollView(this).apply {
            addView(
                root,
                ViewGroup.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT,
                    ViewGroup.LayoutParams.WRAP_CONTENT
                )
            )
        }
        setContentView(scroll)
    }

    private fun saveConfigOnly(): Boolean {
        return try {
            val raw = editor.text?.toString().orEmpty()
            // Validate only. Do not remap, add fields, or restructure the JSON.
            JSONObject(raw)
            val f = PathUtils.configFile(this)
            f.parentFile?.mkdirs()
            f.writeText(raw)
            Toast.makeText(this, "已保存 config.json", Toast.LENGTH_SHORT).show()
            true
        } catch (e: Throwable) {
            Toast.makeText(this, "JSON 格式错误：${e.message}", Toast.LENGTH_LONG).show()
            false
        }
    }

    private fun candidatePorts(): List<String> {
        val editedPort = ConfigUtil.getPortFromConfigText(editor.text?.toString().orEmpty(), "8080")
        val lastPort = ConfigUtil.getLastRunningPort(this)
        return listOf(lastPort, editedPort, "8080").filter { it.isNotBlank() }.distinct()
    }

    private fun reloadRuntimeConfig() {
        Thread {
            var lastError: Throwable? = null
            for (port in candidatePorts()) {
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
                        runOnUiThread {
                            Toast.makeText(this, "运行时配置已更新", Toast.LENGTH_SHORT).show()
                        }
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

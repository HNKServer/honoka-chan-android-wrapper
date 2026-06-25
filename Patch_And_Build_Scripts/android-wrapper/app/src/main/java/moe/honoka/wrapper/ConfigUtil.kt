package moe.honoka.wrapper

import android.content.Context
import org.json.JSONObject

object ConfigUtil {
    fun readConfigText(context: Context): String {
        val f = PathUtils.configFile(context)
        return if (f.exists()) f.readText() else defaultConfigText()
    }

    fun writeConfigText(context: Context, text: String) {
        val f = PathUtils.configFile(context)
        f.parentFile?.mkdirs()
        f.writeText(text)
    }

    fun getPortFromConfigText(text: String, fallback: String = "8080"): String {
        return try {
            JSONObject(text).getJSONObject("settings").optString("server_port", fallback).ifBlank { fallback }
        } catch (_: Throwable) {
            fallback
        }
    }

    fun getPortFromConfigFile(context: Context, fallback: String = "8080"): String {
        return getPortFromConfigText(readConfigText(context), fallback)
    }

    fun getLastRunningPort(context: Context): String {
        return context.getSharedPreferences("honoka", Context.MODE_PRIVATE)
            .getString("last_server_port", null)
            ?: getPortFromConfigFile(context)
    }

    fun setLastRunningPort(context: Context, port: String) {
        context.getSharedPreferences("honoka", Context.MODE_PRIVATE)
            .edit()
            .putString("last_server_port", port)
            .apply()
    }

    fun localBaseUrl(context: Context): String = "http://127.0.0.1:${getLastRunningPort(context)}"

    fun candidatePorts(context: Context, editedText: String? = null): List<String> {
        val editedPort = editedText?.let { getPortFromConfigText(it, "8080") } ?: getPortFromConfigFile(context)
        val lastPort = getLastRunningPort(context)
        return listOf(lastPort, editedPort, "8080").filter { it.isNotBlank() }.distinct()
    }

    fun getUnlockAllSpecialRotation(context: Context): Boolean {
        return try {
            JSONObject(readConfigText(context))
                .optJSONObject("settings")
                ?.optBoolean("unlock_all_special_rotation", false)
                ?: false
        } catch (_: Throwable) {
            false
        }
    }

    fun setUnlockAllSpecialRotation(context: Context, enabled: Boolean): String {
        val json = try {
            JSONObject(readConfigText(context))
        } catch (_: Throwable) {
            JSONObject(defaultConfigText())
        }
        val settings = json.optJSONObject("settings") ?: JSONObject()
        settings.put("unlock_all_special_rotation", enabled)
        json.put("settings", settings)
        val text = json.toString(2) + "\n"
        writeConfigText(context, text)
        return text
    }

    fun defaultConfigText(): String = """
        {
          "app_name": "honoka-chan",
          "settings": {
            "server_port": "8080",
            "sif_cdn_server": "http://127.0.0.1:8080/static",
            "as_cdn_server": "http://127.0.0.1:8080/static",
            "unlock_all_special_rotation": false
          },
          "user_prefs": {
            "name": "梦路 @bilibili",
            "level": 1028,
            "exp_numerator": 1089696,
            "exp_denominator": 1207185,
            "game_coin": 112124104,
            "sns_coin": 0,
            "energy_max": 417,
            "over_max_energy": 0,
            "invite_code": "377385143"
          }
        }
    """.trimIndent()
}

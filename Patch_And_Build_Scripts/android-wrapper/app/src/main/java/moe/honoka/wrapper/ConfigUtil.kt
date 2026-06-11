package moe.honoka.wrapper

import android.content.Context
import org.json.JSONObject

object ConfigUtil {
    fun readConfigText(context: Context): String {
        val f = PathUtils.configFile(context)
        return if (f.exists()) f.readText() else defaultConfigText()
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

    fun defaultConfigText(): String = """
        {
          "app_name": "honoka-chan",
          "settings": {
            "server_port": "8080",
            "sif_cdn_server": "http://127.0.0.1:8080/static",
            "as_cdn_server": "http://127.0.0.1:8080/static"
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

package moe.honoka.wrapper

import android.content.Context

object ServerState {
    const val STOPPED = "STOPPED"
    const val STARTING = "STARTING"
    const val RUNNING = "RUNNING"
    const val FAILED = "FAILED"

    private const val PREFS = "honoka"
    private const val KEY_STATE = "server_state"
    private const val KEY_DETAIL = "server_state_detail"
    private const val KEY_ERROR = "server_last_error"
    private const val KEY_UPDATED_AT = "server_state_updated_at"

    fun set(context: Context, state: String, detail: String = "", error: String = "") {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putString(KEY_STATE, state)
            .putString(KEY_DETAIL, detail)
            .putString(KEY_ERROR, error)
            .putLong(KEY_UPDATED_AT, System.currentTimeMillis())
            .apply()
    }

    fun state(context: Context): String = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        .getString(KEY_STATE, STOPPED) ?: STOPPED

    fun detail(context: Context): String = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        .getString(KEY_DETAIL, "") ?: ""

    fun lastError(context: Context): String = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        .getString(KEY_ERROR, "") ?: ""

    fun updatedAt(context: Context): Long = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        .getLong(KEY_UPDATED_AT, 0L)
}

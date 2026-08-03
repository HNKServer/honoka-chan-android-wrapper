package moe.honoka.wrapper

import android.content.Context
import android.os.Build
import android.os.Environment
import java.io.File

object PathUtils {
    const val DEFAULT_ARCHIVES_DIR = "/storage/emulated/0/Download/HonokaData/static/Android/archives"
    const val STORAGE_ROOT = "/storage/emulated/0"

    fun serverDir(context: Context): File = File(context.filesDir, "server")

    fun configFile(context: Context): File = File(serverDir(context), "config.json")

    fun getStoredArchivesDir(context: Context): String {
        return context.getSharedPreferences("honoka", Context.MODE_PRIVATE)
            .getString("archives_dir", DEFAULT_ARCHIVES_DIR)
            ?: DEFAULT_ARCHIVES_DIR
    }

    fun setStoredArchivesDir(context: Context, path: String) {
        context.getSharedPreferences("honoka", Context.MODE_PRIVATE)
            .edit()
            .putString("archives_dir", path)
            .apply()
    }

    fun hasAllFilesAccess(): Boolean {
        return if (Build.VERSION.SDK_INT >= 30) Environment.isExternalStorageManager() else true
    }

    /**
     * The user may pick HonokaData, static, Android, or archives. Normalize it to the actual archives dir when possible.
     */
    fun normalizeArchivesDir(input: File): File {
        val dir = input.absoluteFile
        if (dir.name == "archives") return dir

        val direct = File(dir, "static/Android/archives")
        if (direct.isDirectory) return direct

        val androidArchives = File(dir, "Android/archives")
        if (androidArchives.isDirectory) return androidArchives

        val childArchives = File(dir, "archives")
        if (childArchives.isDirectory) return childArchives

        return dir
    }

    fun looksLikeArchivesDir(dir: File): Boolean {
        if (!dir.exists() || !dir.isDirectory) return false
        if (dir.name == "archives") return true
        val entries = dir.listFiles() ?: return false
        return entries.any { it.isFile && it.extension.equals("zip", ignoreCase = true) }
    }
}

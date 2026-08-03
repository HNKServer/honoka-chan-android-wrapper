package moe.honoka.wrapper

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.os.Build

class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent?) {
        if (intent?.action != Intent.ACTION_BOOT_COMPLETED) return
        val prefs = context.getSharedPreferences("honoka", Context.MODE_PRIVATE)
        if (!prefs.getBoolean("desired_running", false)) return

        val serviceIntent = Intent(context, ServerService::class.java).setAction(ServerService.ACTION_START)
        if (Build.VERSION.SDK_INT >= 26) context.startForegroundService(serviceIntent) else context.startService(serviceIntent)
    }
}

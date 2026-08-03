package moe.honoka.wrapper

import android.content.Context
import android.content.res.Configuration
import android.graphics.Color
import android.graphics.Typeface
import android.os.Build
import android.text.InputType
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.view.WindowInsets
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.Space
import android.widget.TextView
import com.google.android.material.button.MaterialButton
import com.google.android.material.card.MaterialCardView
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout

object Ui {
    const val ORANGE_HEX = "#f38500"
    const val PREF_SHOW_FIGHT_TITLE = "show_fight_title"

    val ORANGE: Int = Color.parseColor(ORANGE_HEX)
    val ORANGE_LIGHT: Int = Color.parseColor("#FFE2C2")
    val ORANGE_DARK_TEXT: Int = Color.parseColor("#5E3300")

    fun dp(context: Context, value: Int): Int =
        TypedValue.applyDimension(
            TypedValue.COMPLEX_UNIT_DIP,
            value.toFloat(),
            context.resources.displayMetrics
        ).toInt()

    fun colorAttr(context: Context, attr: Int, fallback: Int = 0): Int {
        val out = TypedValue()
        return if (context.theme.resolveAttribute(attr, out, true)) out.data else fallback
    }

    fun isWide(context: Context): Boolean {
        val c = context.resources.configuration
        return c.screenWidthDp >= 700 ||
            (c.orientation == Configuration.ORIENTATION_LANDSCAPE && c.screenWidthDp >= 560)
    }

    fun showFightTitle(context: Context): Boolean =
        context.getSharedPreferences("honoka", Context.MODE_PRIVATE)
            .getBoolean(PREF_SHOW_FIGHT_TITLE, true)

    fun setShowFightTitle(context: Context, show: Boolean) {
        context.getSharedPreferences("honoka", Context.MODE_PRIVATE)
            .edit()
            .putBoolean(PREF_SHOW_FIGHT_TITLE, show)
            .apply()
    }

    fun toggleFightTitle(context: Context): Boolean {
        val next = !showFightTitle(context)
        setShowFightTitle(context, next)
        return next
    }

    /**
     * Content root used inside the fixed screen shell. It intentionally has no
     * system-bar inset padding; screenShell() owns top/bottom inset handling.
     */
    fun contentRoot(context: Context): LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(context, 20), dp(context, 16), dp(context, 20), dp(context, 16))
        setBackgroundColor(colorAttr(context, android.R.attr.colorBackground))
        layoutParams = ViewGroup.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        )
    }

    /** Backward-compatible alias for older files that still call verticalRoot(). */
    fun verticalRoot(context: Context): LinearLayout = contentRoot(context)

    fun plainScroll(context: Context, content: View): ScrollView = ScrollView(context).apply {
        clipToPadding = false
        isFillViewport = false
        setBackgroundColor(colorAttr(context, android.R.attr.colorBackground))
        addView(
            content,
            ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            )
        )
    }

    /** Backward-compatible safeScroll. New Activity pages should use screenShell(). */
    fun safeScroll(context: Context, content: View): ScrollView = plainScroll(context, content).apply {
        setOnApplyWindowInsetsListener { view, insets ->
            val left: Int
            val top: Int
            val right: Int
            val bottom: Int
            if (Build.VERSION.SDK_INT >= 30) {
                val bars = insets.getInsets(WindowInsets.Type.systemBars() or WindowInsets.Type.displayCutout())
                left = bars.left
                top = bars.top
                right = bars.right
                bottom = bars.bottom
            } else {
                @Suppress("DEPRECATION")
                left = insets.systemWindowInsetLeft
                @Suppress("DEPRECATION")
                top = insets.systemWindowInsetTop
                @Suppress("DEPRECATION")
                right = insets.systemWindowInsetRight
                @Suppress("DEPRECATION")
                bottom = insets.systemWindowInsetBottom
            }
            view.setPadding(left, top, right, bottom)
            insets
        }
        requestApplyInsets()
    }

    /**
     * Screen shell with one fixed top strip:
     * - The system-bar/cutout/taskbar inset is always reserved by the fixed top strip.
     * - Only Fightだよ！ lives in the fixed top strip.
     * - Page title/subtitle rows stay inside the scroll content.
     * - No fixed bottom bar; the scroll view itself owns bottom inset padding.
     */
    fun screenShell(
        context: Context,
        pageTitle: String,
        subtitle: String,
        content: View,
        onBack: (() -> Unit)? = null,
        onToggleTitle: () -> Unit
    ): LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        setBackgroundColor(colorAttr(context, android.R.attr.colorBackground))
        layoutParams = ViewGroup.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.MATCH_PARENT
        )

        val topStrip = fixedFightTopStrip(context)

        val scrollContent = LinearLayout(context).apply {
            orientation = LinearLayout.VERTICAL
            setBackgroundColor(colorAttr(context, android.R.attr.colorBackground))
            layoutParams = ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            )
            addView(pageHeader(context, pageTitle, subtitle, onBack, onToggleTitle))
            addView(content)
        }

        val scroll = plainScroll(context, scrollContent).apply {
            layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                0,
                1f
            )
            setOnApplyWindowInsetsListener { view, insets ->
                val left: Int
                val right: Int
                val bottom: Int
                if (Build.VERSION.SDK_INT >= 30) {
                    val bars = insets.getInsets(WindowInsets.Type.systemBars() or WindowInsets.Type.displayCutout())
                    left = bars.left
                    right = bars.right
                    bottom = bars.bottom
                } else {
                    @Suppress("DEPRECATION")
                    left = insets.systemWindowInsetLeft
                    @Suppress("DEPRECATION")
                    right = insets.systemWindowInsetRight
                    @Suppress("DEPRECATION")
                    bottom = insets.systemWindowInsetBottom
                }
                view.setPadding(left, 0, right, bottom)
                insets
            }
            requestApplyInsets()
        }

        addView(topStrip)
        addView(scroll)
    }

    private fun fixedFightTopStrip(context: Context): LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        gravity = Gravity.CENTER
        setBackgroundColor(colorAttr(context, android.R.attr.colorBackground))
        elevation = dp(context, 2).toFloat()
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        )

        if (showFightTitle(context)) {
            addView(fightTitle(context).apply {
                layoutParams = LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT,
                    ViewGroup.LayoutParams.WRAP_CONTENT
                )
            })
        }

        setOnApplyWindowInsetsListener { view, insets ->
            val left: Int
            val top: Int
            val right: Int
            if (Build.VERSION.SDK_INT >= 30) {
                val bars = insets.getInsets(WindowInsets.Type.systemBars() or WindowInsets.Type.displayCutout())
                left = bars.left
                top = bars.top
                right = bars.right
            } else {
                @Suppress("DEPRECATION")
                left = insets.systemWindowInsetLeft
                @Suppress("DEPRECATION")
                top = insets.systemWindowInsetTop
                @Suppress("DEPRECATION")
                right = insets.systemWindowInsetRight
            }

            val bottomPadding = if (showFightTitle(context)) dp(context, 10) else 0
            view.setPadding(
                left + dp(context, 20),
                top + if (showFightTitle(context)) dp(context, 10) else 0,
                right + dp(context, 20),
                bottomPadding
            )
            insets
        }
        requestApplyInsets()
    }

    fun pageHeader(
        context: Context,
        pageTitle: String,
        subtitle: String,
        onBack: (() -> Unit)?,
        onToggleTitle: () -> Unit
    ): LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(context, 20), dp(context, 18), dp(context, 20), dp(context, 8))
        setBackgroundColor(colorAttr(context, android.R.attr.colorBackground))
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        )

        val titleRow = LinearLayout(context).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            )
        }

        if (onBack != null) {
            titleRow.addView(MaterialButton(context).apply {
                text = "‹ 返回"
                cornerRadius = dp(context, 18)
                minHeight = dp(context, 40)
                insetTop = 0
                insetBottom = 0
                setTextColor(ORANGE_DARK_TEXT)
                setBackgroundColor(ORANGE_LIGHT)
                layoutParams = LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.WRAP_CONTENT,
                    dp(context, 42)
                ).apply {
                    marginEnd = dp(context, 10)
                }
                setOnClickListener { onBack() }
            })
        }

        titleRow.addView(title(context, pageTitle).apply {
            layoutParams = LinearLayout.LayoutParams(
                0,
                ViewGroup.LayoutParams.WRAP_CONTENT,
                1f
            )
        })

        titleRow.addView(titleToggleTriangleButton(context, onToggleTitle))
        addView(titleRow)

        if (subtitle.isNotBlank()) {
            addView(subtitle(context, subtitle).apply {
                layoutParams = LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT,
                    ViewGroup.LayoutParams.WRAP_CONTENT
                ).apply {
                    topMargin = dp(context, 4)
                }
            })
        }
    }

    private fun titleToggleTriangleButton(context: Context, onToggleTitle: () -> Unit): MaterialButton =
        MaterialButton(context).apply {
            text = if (showFightTitle(context)) "▲" else "▼"
            contentDescription = if (showFightTitle(context)) "隐藏 Fightだよ！标题" else "显示 Fightだよ！标题"
            cornerRadius = dp(context, 20)
            minWidth = 0
            minimumWidth = 0
            minHeight = dp(context, 40)
            insetTop = 0
            insetBottom = 0
            setPaddingRelative(0, 0, 0, 0)
            setTextColor(ORANGE_DARK_TEXT)
            setBackgroundColor(ORANGE_LIGHT)
            layoutParams = LinearLayout.LayoutParams(
                dp(context, 46),
                dp(context, 42)
            ).apply {
                marginStart = dp(context, 10)
            }
            setOnClickListener { onToggleTitle() }
        }

    fun header(context: Context, pageTitle: String, subtitle: String): LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        if (showFightTitle(context)) addView(fightTitle(context))
        addView(title(context, pageTitle))
        addView(subtitle(context, subtitle))
    }

    fun fightTitle(context: Context): TextView = TextView(context).apply {
        text = "Fightだよ！"
        gravity = Gravity.CENTER
        setTextSize(TypedValue.COMPLEX_UNIT_SP, if (isWide(context)) 42f else 38f)
        typeface = Typeface.DEFAULT_BOLD
        setTextColor(ORANGE)
        includeFontPadding = false
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        ).apply {
            bottomMargin = dp(context, 10)
        }
    }

    fun title(context: Context, text: String): TextView = TextView(context).apply {
        this.text = text
        setTextSize(TypedValue.COMPLEX_UNIT_SP, if (isWide(context)) 23f else 25f)
        typeface = Typeface.DEFAULT_BOLD
        setTextColor(colorAttr(context, com.google.android.material.R.attr.colorOnSurface))
        includeFontPadding = false
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        ).apply {
            bottomMargin = dp(context, 6)
        }
    }

    fun subtitle(context: Context, text: String): TextView = TextView(context).apply {
        this.text = text
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
        setTextColor(colorAttr(context, com.google.android.material.R.attr.colorOnSurfaceVariant))
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        )
    }

    fun sectionTitle(context: Context, text: String): TextView = TextView(context).apply {
        this.text = text
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
        typeface = Typeface.DEFAULT_BOLD
        setTextColor(colorAttr(context, com.google.android.material.R.attr.colorOnSurface))
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        ).apply {
            topMargin = dp(context, 4)
            bottomMargin = dp(context, 10)
        }
    }

    fun body(context: Context, text: String): TextView = TextView(context).apply {
        this.text = text
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
        setTextColor(colorAttr(context, com.google.android.material.R.attr.colorOnSurfaceVariant))
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        )
    }

    fun card(context: Context): MaterialCardView = MaterialCardView(context).apply {
        radius = dp(context, 28).toFloat()
        cardElevation = 0f
        strokeWidth = 1
        setStrokeColor(colorAttr(context, com.google.android.material.R.attr.colorOutlineVariant))
        setCardBackgroundColor(colorAttr(context, com.google.android.material.R.attr.colorSurface))
        useCompatPadding = false
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        ).apply {
            bottomMargin = dp(context, 16)
        }
    }

    fun cardContent(context: Context): LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(context, 20), dp(context, 18), dp(context, 20), dp(context, 20))
    }

    fun column(context: Context): LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f).apply {
            marginEnd = dp(context, 8)
        }
    }

    fun twoPane(context: Context): LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.HORIZONTAL
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        )
    }

    fun filledButton(context: Context, text: String): MaterialButton = MaterialButton(context).apply {
        this.text = text
        cornerRadius = dp(context, 20)
        minHeight = dp(context, 48)
        insetTop = 0
        insetBottom = 0
        setTextColor(Color.WHITE)
        setBackgroundColor(ORANGE)
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            dp(context, 52)
        ).apply {
            topMargin = dp(context, 10)
        }
    }

    fun tonalButton(context: Context, text: String): MaterialButton = MaterialButton(context).apply {
        this.text = text
        cornerRadius = dp(context, 20)
        minHeight = dp(context, 48)
        insetTop = 0
        insetBottom = 0
        setTextColor(ORANGE_DARK_TEXT)
        setBackgroundColor(ORANGE_LIGHT)
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            dp(context, 52)
        ).apply {
            topMargin = dp(context, 10)
        }
    }

    fun input(
        context: Context,
        hint: String,
        helper: String? = null,
        minLines: Int = 1,
        singleLine: Boolean = false,
        monospace: Boolean = false
    ): Pair<TextInputLayout, TextInputEditText> {
        val edit = TextInputEditText(context).apply {
            this.hint = hint
            setSingleLine(singleLine)
            this.minLines = minLines
            inputType = if (singleLine) {
                InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_FLAG_NO_SUGGESTIONS
            } else {
                InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_FLAG_MULTI_LINE or InputType.TYPE_TEXT_FLAG_NO_SUGGESTIONS
            }
            if (monospace) typeface = Typeface.MONOSPACE
        }
        val layout = TextInputLayout(context).apply {
            this.hint = hint
            boxBackgroundMode = TextInputLayout.BOX_BACKGROUND_OUTLINE
            this.helperText = helper
            boxStrokeColor = ORANGE
            hintTextColor = android.content.res.ColorStateList.valueOf(ORANGE)
            addView(edit)
            layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            ).apply {
                topMargin = dp(context, 10)
                bottomMargin = dp(context, 4)
            }
        }
        return layout to edit
    }

    fun row(context: Context): LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.HORIZONTAL
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        )
    }

    fun rowButtonParams(context: Context, weight: Float = 1f): LinearLayout.LayoutParams =
        LinearLayout.LayoutParams(0, dp(context, 52), weight).apply {
            topMargin = dp(context, 10)
            marginEnd = dp(context, 8)
        }

    fun spacer(context: Context, heightDp: Int): View = Space(context).apply {
        layoutParams = LinearLayout.LayoutParams(1, dp(context, heightDp))
    }
}

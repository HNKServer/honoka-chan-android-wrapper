package moe.honoka.wrapper

import android.content.Context
import android.graphics.Typeface
import android.text.InputType
import android.util.TypedValue
import android.view.View
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.Space
import android.widget.TextView
import com.google.android.material.button.MaterialButton
import com.google.android.material.card.MaterialCardView
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout

object Ui {
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

    fun verticalRoot(context: Context): LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(context, 20), dp(context, 18), dp(context, 20), dp(context, 24))
        setBackgroundColor(colorAttr(context, android.R.attr.colorBackground))
        layoutParams = ViewGroup.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.MATCH_PARENT
        )
    }

    fun title(context: Context, text: String): TextView = TextView(context).apply {
        this.text = text
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 28f)
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
        ).apply {
            bottomMargin = dp(context, 18)
        }
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

    fun filledButton(context: Context, text: String): MaterialButton = MaterialButton(context).apply {
        this.text = text
        cornerRadius = dp(context, 20)
        minHeight = dp(context, 48)
        insetTop = 0
        insetBottom = 0
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
        setTextColor(colorAttr(context, com.google.android.material.R.attr.colorOnSecondaryContainer))
        setBackgroundColor(colorAttr(context, com.google.android.material.R.attr.colorSecondaryContainer))
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
        monospace: Boolean = false
    ): Pair<TextInputLayout, TextInputEditText> {
        val edit = TextInputEditText(context).apply {
            this.hint = hint
            setSingleLine(false)
            this.minLines = minLines
            inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_FLAG_MULTI_LINE or InputType.TYPE_TEXT_FLAG_NO_SUGGESTIONS
            if (monospace) typeface = Typeface.MONOSPACE
        }
        val layout = TextInputLayout(context).apply {
            this.hint = hint
            boxBackgroundMode = TextInputLayout.BOX_BACKGROUND_OUTLINE
            this.helperText = helper
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

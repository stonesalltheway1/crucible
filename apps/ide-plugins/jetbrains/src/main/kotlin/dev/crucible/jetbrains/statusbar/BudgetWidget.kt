package dev.crucible.jetbrains.statusbar

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.StatusBar
import com.intellij.openapi.wm.StatusBarWidget
import com.intellij.openapi.wm.StatusBarWidgetFactory
import com.intellij.openapi.wm.impl.status.widget.StatusBarEditorBasedWidgetFactory
import com.intellij.util.Consumer
import dev.crucible.jetbrains.client.CrucibleClient
import java.awt.event.MouseEvent
import javax.swing.Timer

class BudgetWidgetFactory : StatusBarWidgetFactory {
    override fun getId() = "crucibleBudget"
    override fun getDisplayName() = "Crucible Budget"
    override fun isAvailable(project: Project) = true
    override fun createWidget(project: Project): StatusBarWidget = BudgetWidget(project)
    override fun disposeWidget(widget: StatusBarWidget) {}
    override fun canBeEnabledOn(statusBar: StatusBar) = true
}

class BudgetWidget(private val project: Project) : StatusBarWidget, StatusBarWidget.TextPresentation {
    private var label = "Crucible · —"
    private var bar: StatusBar? = null
    private val timer = Timer(30_000) { refresh() }

    override fun ID() = "crucibleBudget"
    override fun getPresentation() = this
    override fun getText() = label
    override fun getAlignment(): Float = 0.0f
    override fun getTooltipText() = "Crucible — open the web console"
    override fun getClickConsumer(): Consumer<MouseEvent> = Consumer {
        com.intellij.ide.BrowserUtil.browse("https://app.crucible.dev")
    }

    override fun install(statusBar: StatusBar) {
        bar = statusBar
        refresh()
        timer.start()
    }

    override fun dispose() {
        timer.stop()
    }

    private fun refresh() {
        ApplicationManager.getApplication().executeOnPooledThread {
            label = try {
                val s = CrucibleClient.getInstance().budgetSnapshot()
                val pct = (s.spent_today_usd / s.cap_today_usd.coerceAtLeast(0.01) * 100).toInt()
                "Crucible · \$%.2f / \$%.0f (%d%%)".format(s.spent_today_usd, s.cap_today_usd, pct)
            } catch (_: Throwable) {
                "Crucible · offline"
            }
            ApplicationManager.getApplication().invokeLater { bar?.updateWidget("crucibleBudget") }
        }
    }
}

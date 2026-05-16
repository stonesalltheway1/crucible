package dev.crucible.jetbrains.toolwindow

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.ui.JBColor
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBList
import com.intellij.ui.components.JBPanel
import com.intellij.ui.components.JBScrollPane
import com.intellij.util.ui.JBUI
import dev.crucible.jetbrains.client.CrucibleClient
import java.awt.BorderLayout
import java.awt.GridBagConstraints
import java.awt.GridBagLayout
import javax.swing.*

/** Junie-style tool window with two tabs: tasks + attestations. */
class CrucibleToolWindowFactory : ToolWindowFactory {
    override fun createToolWindowContent(project: Project, toolWindow: ToolWindow) {
        val panel = JBPanel<Nothing>(BorderLayout()).apply {
            border = JBUI.Borders.empty(8)
            val tabs = JTabbedPane().apply {
                addTab("Tasks", TaskListPane(project))
                addTab("Attestations", AttestationListPane())
                addTab("Plan", PlanPane(project))
            }
            add(tabs, BorderLayout.CENTER)
        }
        val content = toolWindow.contentManager.factory.createContent(panel, "", false)
        toolWindow.contentManager.addContent(content)
    }
}

private class TaskListPane(private val project: Project) : JBPanel<Nothing>(BorderLayout()) {
    private val model = DefaultListModel<CrucibleClient.TaskSummary>()
    private val list = JBList(model).apply {
        cellRenderer = TaskCellRenderer()
    }

    init {
        border = JBUI.Borders.empty(4)
        val header = JBPanel<Nothing>(BorderLayout()).apply {
            add(JBLabel("Tasks").apply { font = font.deriveFont(java.awt.Font.BOLD) }, BorderLayout.WEST)
            val refresh = JButton("Refresh").apply { addActionListener { refresh() } }
            add(refresh, BorderLayout.EAST)
        }
        add(header, BorderLayout.NORTH)
        add(JBScrollPane(list), BorderLayout.CENTER)
        refresh()
    }

    private fun refresh() {
        ApplicationManager.getApplication().executeOnPooledThread {
            try {
                val tasks = CrucibleClient.getInstance().listTasks()
                ApplicationManager.getApplication().invokeLater {
                    model.clear()
                    tasks.forEach(model::addElement)
                }
            } catch (_: Throwable) {
                // Soft-fail; the user already sees the panel state.
            }
        }
    }
}

private class TaskCellRenderer : ListCellRenderer<CrucibleClient.TaskSummary> {
    private val main = JBLabel()
    private val sub = JBLabel().apply { foreground = JBColor.GRAY; font = font.deriveFont(11f) }
    private val container = JBPanel<Nothing>(GridBagLayout()).apply { border = JBUI.Borders.empty(4) }

    override fun getListCellRendererComponent(
        list: JList<out CrucibleClient.TaskSummary>,
        value: CrucibleClient.TaskSummary,
        index: Int,
        selected: Boolean,
        focus: Boolean,
    ): java.awt.Component {
        container.removeAll()
        main.text = "${glyph(value.status)}  ${value.description.take(80)}"
        sub.text = "${value.status} · ${"$%.2f".format(value.cost_usd)} · ${value.id}"
        val gbc = GridBagConstraints().apply { gridx = 0; weightx = 1.0; fill = GridBagConstraints.HORIZONTAL }
        gbc.gridy = 0; container.add(main, gbc)
        gbc.gridy = 1; container.add(sub, gbc)
        container.background = if (selected) list.selectionBackground else list.background
        main.foreground = if (selected) list.selectionForeground else list.foreground
        return container
    }

    private fun glyph(s: String) = when (s) {
        "plan_pending_approval" -> "●"
        "executing", "verifying" -> "…"
        "verified", "completed", "promoted" -> "✓"
        "failed", "verification_failed" -> "✗"
        else -> "·"
    }
}

private class AttestationListPane : JBPanel<Nothing>(BorderLayout()) {
    init {
        add(JBLabel("Recent attestations from the relay appear here.").apply { border = JBUI.Borders.empty(8) }, BorderLayout.NORTH)
    }
}

private class PlanPane(private val project: Project) : JBPanel<Nothing>(BorderLayout()) {
    init {
        val empty = JBLabel("<html>Select a task with a pending plan to see the<br>cost preview, top risks, and approval controls.</html>")
            .apply { border = JBUI.Borders.empty(12) }
        add(empty, BorderLayout.CENTER)
    }
}

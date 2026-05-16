package dev.crucible.jetbrains.actions

import com.intellij.ide.BrowserUtil
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.ui.Messages
import dev.crucible.jetbrains.client.CrucibleClient

class NewTaskAction : AnAction("Crucible: New Task") {
    override fun actionPerformed(e: AnActionEvent) {
        val description = Messages.showInputDialog(
            e.project, "Describe the task", "Crucible · New Task", Messages.getQuestionIcon(),
        ) ?: return
        ApplicationManager.getApplication().executeOnPooledThread {
            try {
                val repo = e.project?.basePath?.let { p ->
                    "github.com/${p.substringAfterLast('/').ifEmpty { "unknown" }}"
                } ?: "unknown/unknown"
                val t = CrucibleClient.getInstance().submitTask(description, repo)
                ApplicationManager.getApplication().invokeLater {
                    Messages.showInfoMessage(e.project, "Task submitted: ${t.id}", "Crucible")
                }
            } catch (ex: Throwable) {
                ApplicationManager.getApplication().invokeLater {
                    Messages.showErrorDialog(e.project, ex.message ?: "failed", "Crucible")
                }
            }
        }
    }
}

class ApprovePlanAction : AnAction("Crucible: Approve Plan") {
    override fun actionPerformed(e: AnActionEvent) {
        val id = Messages.showInputDialog(e.project, "Task id", "Crucible · Approve Plan", null) ?: return
        if (Messages.showYesNoDialog(
                e.project,
                "Approve plan? The agent will sign and execute.",
                "Crucible",
                Messages.getQuestionIcon(),
            ) != Messages.YES
        ) return
        ApplicationManager.getApplication().executeOnPooledThread {
            try {
                CrucibleClient.getInstance().approvePlan(id)
            } catch (ex: Throwable) {
                ApplicationManager.getApplication().invokeLater {
                    Messages.showErrorDialog(e.project, ex.message ?: "failed", "Crucible")
                }
            }
        }
    }
}

class InterruptTaskAction : AnAction("Crucible: Halt at Next Checkpoint") {
    override fun actionPerformed(e: AnActionEvent) {
        val id = Messages.showInputDialog(e.project, "Task id", "Crucible · Halt", null) ?: return
        ApplicationManager.getApplication().executeOnPooledThread {
            try {
                CrucibleClient.getInstance().interrupt(id, "user halt from JetBrains IDE")
            } catch (_: Throwable) {}
        }
    }
}

class OpenWebConsoleAction : AnAction("Crucible: Open Web Console") {
    override fun actionPerformed(e: AnActionEvent) = BrowserUtil.browse("https://app.crucible.dev")
}

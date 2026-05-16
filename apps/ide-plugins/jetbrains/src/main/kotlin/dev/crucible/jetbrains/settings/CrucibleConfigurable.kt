package dev.crucible.jetbrains.settings

import com.intellij.openapi.options.Configurable
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBPanel
import com.intellij.ui.components.JBPasswordField
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.JBUI
import dev.crucible.jetbrains.client.CrucibleSettings
import javax.swing.JComponent

class CrucibleConfigurable : Configurable {
    private val endpoint = JBTextField()
    private val token = JBPasswordField()
    private val tenantId = JBTextField()
    private var panel: JComponent? = null

    override fun getDisplayName() = "Crucible"

    override fun createComponent(): JComponent {
        val p = JBPanel<Nothing>().apply {
            border = JBUI.Borders.empty(12)
            layout = java.awt.GridLayout(0, 2, 8, 8)
            add(JBLabel("API endpoint"))
            add(endpoint)
            add(JBLabel("Bearer token"))
            add(token)
            add(JBLabel("Tenant id"))
            add(tenantId)
        }
        panel = p
        reset()
        return p
    }

    override fun isModified(): Boolean {
        val s = CrucibleSettings.getInstance().state
        return s.endpoint != endpoint.text || s.token != String(token.password) || s.tenantId != tenantId.text
    }

    override fun apply() {
        val s = CrucibleSettings.getInstance().state
        s.endpoint = endpoint.text.trim()
        s.token = String(token.password)
        s.tenantId = tenantId.text.trim()
    }

    override fun reset() {
        val s = CrucibleSettings.getInstance().state
        endpoint.text = s.endpoint
        token.text = s.token
        tenantId.text = s.tenantId
    }
}

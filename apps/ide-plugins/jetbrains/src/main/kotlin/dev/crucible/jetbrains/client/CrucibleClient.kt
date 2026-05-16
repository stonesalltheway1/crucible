package dev.crucible.jetbrains.client

import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage
import com.intellij.openapi.components.service
import kotlinx.serialization.Serializable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.time.Duration

/** Persistent settings (token, endpoint, tenant id). */
@State(name = "CrucibleSettings", storages = [Storage("crucible.xml")])
class CrucibleSettings : PersistentStateComponent<CrucibleSettings.S> {
    data class S(var endpoint: String = "https://api.crucible.dev",
                 var token: String = "",
                 var tenantId: String = "ten_demo")
    private var state = S()
    override fun getState() = state
    override fun loadState(s: S) { state = s }
    companion object {
        fun getInstance(): CrucibleSettings = service()
    }
}

/** Thin Crucible API client. Coroutine-friendly via blocking I/O on a background thread. */
@Service(Service.Level.APP)
class CrucibleClient {
    private val http = OkHttpClient.Builder()
        .connectTimeout(Duration.ofSeconds(10))
        .readTimeout(Duration.ofSeconds(30))
        .build()
    private val json = Json { ignoreUnknownKeys = true }

    @Serializable
    data class TaskSummary(
        val id: String,
        val description: String,
        val status: String,
        val repo: String,
        val cost_usd: Double,
        val submitted_at: String,
    )

    @Serializable
    data class TaskListResp(val tasks: List<TaskSummary>)

    @Serializable
    data class PlanRisk(val description: String, val impact: String)

    @Serializable
    data class ExternalEffect(val service: String, val endpoints: List<String>, val live: Boolean)

    @Serializable
    data class Plan(
        val description: String,
        val estimated_cost_usd: Double,
        val estimated_duration_min: Int,
        val files_to_touch: List<String>,
        val db_migrations: Int,
        val external_effects: List<ExternalEffect> = emptyList(),
        val top_risks: List<PlanRisk> = emptyList(),
        val retry_budget_per_step: Int,
        val wall_clock_budget_min: Int,
        val hard_cap_usd: Double,
    )

    @Serializable
    data class TaskDetail(
        val id: String,
        val description: String,
        val status: String,
        val repo: String,
        val cost_usd: Double = 0.0,
        val submitted_at: String,
        val plan: Plan? = null,
    )

    @Serializable
    data class BudgetSnapshot(val spent_today_usd: Double, val cap_today_usd: Double, val tasks_today: Int)

    @Serializable
    data class SubmitTaskReq(val description: String, val repo: String, val submitted_from: String = "jetbrains")

    private fun s() = CrucibleSettings.getInstance().state

    private fun get(path: String): String {
        val req = Request.Builder()
            .url("${s().endpoint}$path")
            .header("Accept", "application/json")
            .header("Authorization", "Bearer ${s().token}")
            .build()
        http.newCall(req).execute().use { r ->
            if (!r.isSuccessful) error("GET $path: ${r.code} ${r.body?.string()}")
            return r.body!!.string()
        }
    }

    private fun post(path: String, body: String): String {
        val req = Request.Builder()
            .url("${s().endpoint}$path")
            .header("Accept", "application/json")
            .header("Authorization", "Bearer ${s().token}")
            .post(body.toRequestBody("application/json".toMediaType()))
            .build()
        http.newCall(req).execute().use { r ->
            if (!r.isSuccessful) error("POST $path: ${r.code} ${r.body?.string()}")
            return r.body!!.string()
        }
    }

    fun listTasks(): List<TaskSummary> =
        json.decodeFromString<TaskListResp>(get("/v1/tenants/${s().tenantId}/tasks?limit=50")).tasks

    fun getTask(id: String): TaskDetail =
        json.decodeFromString(get("/v1/tenants/${s().tenantId}/tasks/$id"))

    fun submitTask(description: String, repo: String): TaskSummary =
        json.decodeFromString(post(
            "/v1/tenants/${s().tenantId}/tasks",
            json.encodeToString(SubmitTaskReq.serializer(), SubmitTaskReq(description, repo)),
        ))

    fun approvePlan(id: String) = post("/v1/tenants/${s().tenantId}/tasks/$id/plan/approve", """{"edits":null}""")
    fun rejectPlan(id: String, reason: String) =
        post("/v1/tenants/${s().tenantId}/tasks/$id/plan/reject", """{"reason":${json.encodeToString(String.serializer(), reason)}}""")
    fun interrupt(id: String, reason: String) =
        post("/v1/tenants/${s().tenantId}/tasks/$id/interrupt", """{"reason":${json.encodeToString(String.serializer(), reason)}}""")

    fun budgetSnapshot(): BudgetSnapshot =
        json.decodeFromString(get("/v1/tenants/${s().tenantId}/budget/snapshot"))

    companion object {
        fun getInstance(): CrucibleClient = service()
    }
}

private fun kotlinx.serialization.builtins.serializer() = kotlinx.serialization.builtins.serializer()
private fun String.Companion.serializer() = kotlinx.serialization.builtins.serializer()

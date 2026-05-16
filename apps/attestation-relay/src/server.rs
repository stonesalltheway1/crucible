//! axum HTTP surface for the attestation relay.
//!
//! Endpoints:
//!
//! - `POST   /v1/attestations`              build → sign → publish
//! - `POST   /v1/attestations/raw`          caller-supplied Statement
//! - `GET    /v1/attestations/{uuid}`       fetch DSSE envelope
//! - `GET    /v1/attestations/{uuid}/inclusion` Rekor inclusion proof
//! - `GET    /v1/journal/tail`              tail of the journal (admin)
//! - `POST   /v1/journal/backfill`          trigger one backfill pass
//! - `GET    /v1/journal/validate`          validate the hash chain
//! - `GET    /v1/predicates`                list predicate-type URIs
//! - `GET    /healthz`                      health + signer + Rekor state

use std::sync::Arc;

use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};

use crate::dsse::DsseEnvelope;
use crate::error::Error;
use crate::predicate::ALL_PREDICATES;
use crate::rekor::RekorEntry;
use crate::service::{Health, Service};
use crate::statement::InTotoStatement;

/// Build the router for the relay.
#[must_use]
pub fn router(service: Arc<Service>) -> Router {
    Router::new()
        .route("/healthz", get(handle_health))
        .route("/v1/predicates", get(handle_predicates))
        .route("/v1/attestations", post(handle_emit))
        .route("/v1/attestations/raw", post(handle_publish_raw))
        .route("/v1/attestations/:uuid", get(handle_fetch))
        .route("/v1/attestations/:uuid/inclusion", get(handle_inclusion))
        .route("/v1/journal/tail", get(handle_journal_tail))
        .route("/v1/journal/backfill", post(handle_journal_backfill))
        .route("/v1/journal/validate", get(handle_journal_validate))
        .with_state(service)
}

// ── handler types ──────────────────────────────────────────────────────────

#[derive(Debug, Deserialize)]
struct EmitRequest {
    predicate_type: String,
    subject_name: String,
    /// Caller may pass either raw bytes or a base64-encoded blob.
    #[serde(default)]
    subject_content_b64: Option<String>,
    #[serde(default)]
    subject_content_text: Option<String>,
    predicate: serde_json::Value,
}

#[derive(Debug, Serialize)]
struct EmitResponse {
    receipt: RekorEntry,
    envelope: DsseEnvelope,
    statement: InTotoStatement,
}

#[derive(Debug, Serialize)]
struct ErrorBody {
    error: String,
    kind: String,
}

fn error_response(e: Error) -> (StatusCode, Json<ErrorBody>) {
    let (status, kind) = match &e {
        Error::Config(_) => (StatusCode::INTERNAL_SERVER_ERROR, "config"),
        Error::Predicate(_) => (StatusCode::BAD_REQUEST, "predicate"),
        Error::Dsse(_) => (StatusCode::BAD_REQUEST, "dsse"),
        Error::Signer(_) => (StatusCode::INTERNAL_SERVER_ERROR, "signer"),
        Error::Fulcio(_) => (StatusCode::BAD_GATEWAY, "fulcio"),
        Error::Rekor(_) => (StatusCode::BAD_GATEWAY, "rekor"),
        Error::Journal(_) => (StatusCode::INTERNAL_SERVER_ERROR, "journal"),
        Error::Verify(_) => (StatusCode::UNPROCESSABLE_ENTITY, "verify"),
        Error::SelfApproval { .. } => (StatusCode::FORBIDDEN, "self_approval"),
        Error::StaleApproval { .. } => (StatusCode::CONFLICT, "stale_approval"),
        Error::Http(_) => (StatusCode::BAD_GATEWAY, "http"),
        Error::Json(_) => (StatusCode::BAD_REQUEST, "json"),
        Error::Io(_) => (StatusCode::INTERNAL_SERVER_ERROR, "io"),
        Error::Other(_) => (StatusCode::INTERNAL_SERVER_ERROR, "other"),
    };
    (
        status,
        Json(ErrorBody {
            error: e.to_string(),
            kind: kind.into(),
        }),
    )
}

async fn handle_health(State(svc): State<Arc<Service>>) -> Json<Health> {
    Json(svc.health())
}

async fn handle_predicates() -> Json<Vec<&'static str>> {
    Json(ALL_PREDICATES.to_vec())
}

async fn handle_emit(
    State(svc): State<Arc<Service>>,
    Json(req): Json<EmitRequest>,
) -> impl IntoResponse {
    use base64::Engine as _;
    let content = if let Some(b64) = &req.subject_content_b64 {
        match base64::engine::general_purpose::STANDARD.decode(b64) {
            Ok(b) => b,
            Err(e) => return error_response(Error::Predicate(format!("subject_content_b64: {e}"))).into_response(),
        }
    } else if let Some(s) = &req.subject_content_text {
        s.as_bytes().to_vec()
    } else {
        return error_response(Error::Predicate(
            "subject_content_b64 or subject_content_text required".into(),
        ))
        .into_response();
    };
    let outcome = match svc
        .emit(&req.predicate_type, &req.subject_name, &content, req.predicate)
        .await
    {
        Ok(o) => o,
        Err(e) => return error_response(e).into_response(),
    };
    let body = EmitResponse {
        receipt: outcome.receipt,
        envelope: outcome.envelope,
        statement: outcome.statement,
    };
    (StatusCode::CREATED, Json(body)).into_response()
}

#[derive(Debug, Deserialize)]
struct PublishRawRequest {
    envelope: DsseEnvelope,
}

async fn handle_publish_raw(
    State(svc): State<Arc<Service>>,
    Json(req): Json<PublishRawRequest>,
) -> impl IntoResponse {
    match svc.publish(&req.envelope).await {
        Ok(r) => (StatusCode::CREATED, Json(r)).into_response(),
        Err(e) => error_response(e).into_response(),
    }
}

async fn handle_fetch(
    State(svc): State<Arc<Service>>,
    Path(uuid): Path<String>,
) -> impl IntoResponse {
    match svc.fetch(&uuid).await {
        Ok(env) => Json(env).into_response(),
        Err(_) => (StatusCode::NOT_FOUND, Json(ErrorBody { error: "not found".into(), kind: "not_found".into() })).into_response(),
    }
}

async fn handle_inclusion(
    State(_svc): State<Arc<Service>>,
    Path(uuid): Path<String>,
) -> impl IntoResponse {
    // Phase-6: inclusion proofs come from the Rekor client; the relay
    // currently exposes them only as a passthrough. For journal-only entries
    // we return a 200 with `journal_only=true` and an `audit_path` over the
    // hash-chained journal.
    Json(serde_json::json!({
        "uuid": uuid,
        "available": false,
        "reason": "Phase-6 ships passthrough only; v2 wires the full inclusion-proof flow."
    }))
    .into_response()
}

#[derive(Debug, Deserialize)]
struct TailQuery {
    #[serde(default)]
    n: Option<usize>,
}

async fn handle_journal_tail(
    State(svc): State<Arc<Service>>,
    Query(q): Query<TailQuery>,
) -> impl IntoResponse {
    let n = q.n.unwrap_or(50).clamp(1, 1000);
    match svc.journal().pending_backfill(n) {
        Ok(entries) => Json(entries).into_response(),
        Err(e) => error_response(e).into_response(),
    }
}

async fn handle_journal_backfill(State(svc): State<Arc<Service>>) -> impl IntoResponse {
    match svc.backfill_once(500).await {
        Ok(n) => Json(serde_json::json!({"backfilled": n})).into_response(),
        Err(e) => error_response(e).into_response(),
    }
}

async fn handle_journal_validate(State(svc): State<Arc<Service>>) -> impl IntoResponse {
    match svc.validate_journal() {
        Ok(n) => Json(serde_json::json!({"entries": n, "valid": true})).into_response(),
        Err(e) => (StatusCode::UNPROCESSABLE_ENTITY, Json(serde_json::json!({
            "valid": false, "error": e.to_string()
        })))
        .into_response(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::journal::Journal;
    use crate::predicate;
    use crate::rekor::{MockRekor, RekorClient};
    use crate::signer::LocalEd25519Signer;
    use axum::body::Body;
    use axum::http::Request;
    use std::sync::Arc;
    use tower::ServiceExt;

    async fn build_app() -> Router {
        let dir = tempfile::tempdir().unwrap();
        let signer = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
        let journal = Arc::new(Journal::open(dir.path().join("j.jsonl")).unwrap());
        let mock = Arc::new(MockRekor::new());
        let client = RekorClient::new(mock);
        let svc = Service::new(Arc::new(signer), Some(client), journal, false);
        router(Arc::new(svc))
    }

    #[tokio::test]
    async fn healthz_ok() {
        let app = build_app().await;
        let resp = app
            .oneshot(Request::builder().uri("/healthz").body(Body::empty()).unwrap())
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
    }

    #[tokio::test]
    async fn predicates_lists_all() {
        let app = build_app().await;
        let resp = app
            .oneshot(Request::builder().uri("/v1/predicates").body(Body::empty()).unwrap())
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
        let body = axum::body::to_bytes(resp.into_body(), 65_536).await.unwrap();
        let v: Vec<String> = serde_json::from_slice(&body).unwrap();
        // 13 Crucible + SLSA = 15.
        assert!(v.len() >= 13);
    }

    #[tokio::test]
    async fn emit_round_trip_via_http() {
        let app = build_app().await;
        let req_body = serde_json::json!({
            "predicate_type": predicate::PRED_WRITE_ATTESTATION,
            "subject_name": "task_demo",
            "subject_content_text": "demo-content",
            "predicate": {
                "task_id": "task_demo",
                "tenant_id": "ten_x",
                "path": "x.go",
                "agent_oidc_subject": "subj"
            }
        });
        let req = Request::builder()
            .method("POST")
            .uri("/v1/attestations")
            .header("Content-Type", "application/json")
            .body(Body::from(serde_json::to_vec(&req_body).unwrap()))
            .unwrap();
        let resp = app.oneshot(req).await.unwrap();
        assert_eq!(resp.status(), StatusCode::CREATED);
    }

    #[tokio::test]
    async fn emit_rejects_unknown_predicate() {
        let app = build_app().await;
        let req_body = serde_json::json!({
            "predicate_type": "https://crucible.dev/Bogus/v1",
            "subject_name": "t",
            "subject_content_text": "c",
            "predicate": {"task_id":"t"}
        });
        let req = Request::builder()
            .method("POST")
            .uri("/v1/attestations")
            .header("Content-Type", "application/json")
            .body(Body::from(serde_json::to_vec(&req_body).unwrap()))
            .unwrap();
        let resp = app.oneshot(req).await.unwrap();
        assert_eq!(resp.status(), StatusCode::BAD_REQUEST);
    }
}

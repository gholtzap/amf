use anyhow::Result;
use axum::{
    routing::{get, post, put, delete},
    Router,
};
use std::net::SocketAddr;
use tower_http::trace::TraceLayer;
use tracing::info;

use crate::config::SbiConfig;
use crate::context::{UeContextManager, RanContextManager};
use crate::db::Database;

pub async fn create_server(
    config: &SbiConfig,
    ue_context: UeContextManager,
    ran_context: RanContextManager,
    db: Database,
) -> Result<()> {
    let app = Router::new()
        .route("/namf-comm/v1/ue-contexts", post(create_ue_context))
        .route("/namf-comm/v1/ue-contexts/:ueContextId", get(get_ue_context))
        .route("/namf-comm/v1/ue-contexts/:ueContextId/release", post(release_ue_context))
        .route("/namf-comm/v1/ue-contexts/:ueContextId/n1-n2-messages", post(n1_n2_message_transfer))
        .route("/namf-evts/v1/subscriptions", post(create_event_subscription))
        .route("/namf-evts/v1/subscriptions/:subscriptionId", delete(delete_event_subscription))
        .route("/namf-loc/v1/provide-location-info", post(provide_location_info))
        .route("/namf-mt/v1/ue-contexts/:ueContextId/provide-domain-selection-info", post(provide_domain_selection_info))
        .route("/health", get(health_check))
        .layer(TraceLayer::new_for_http());

    let addr: SocketAddr = config.bind_addr.parse()?;
    let listener = tokio::net::TcpListener::bind(addr).await?;

    info!("SBI server listening on {}", addr);

    axum::serve(listener, app).await?;

    Ok(())
}

async fn create_ue_context() -> &'static str {
    "create_ue_context"
}

async fn get_ue_context() -> &'static str {
    "get_ue_context"
}

async fn release_ue_context() -> &'static str {
    "release_ue_context"
}

async fn n1_n2_message_transfer() -> &'static str {
    "n1_n2_message_transfer"
}

async fn create_event_subscription() -> &'static str {
    "create_event_subscription"
}

async fn delete_event_subscription() -> &'static str {
    "delete_event_subscription"
}

async fn provide_location_info() -> &'static str {
    "provide_location_info"
}

async fn provide_domain_selection_info() -> &'static str {
    "provide_domain_selection_info"
}

async fn health_check() -> &'static str {
    "OK"
}

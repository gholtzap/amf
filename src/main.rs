mod config;
mod context;
mod db;
mod nas;
mod ngap;
mod nf_client;
mod proto;
mod sbi;
mod security;
mod utils;

use anyhow::Result;
use clap::Parser;
use tracing::{info, error};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

#[derive(Parser, Debug)]
#[command(name = "amf-rust")]
#[command(about = "5G Access and Mobility Management Function", long_about = None)]
struct Args {
    #[arg(short, long, default_value = "config.json")]
    config: String,
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "amf_rust=debug,tower_http=debug".into()),
        )
        .with(tracing_subscriber::fmt::layer())
        .init();

    let args = Args::parse();

    info!("Starting AMF");
    info!("Loading configuration from: {}", args.config);

    let config = config::load_config(&args.config).await?;
    info!("Configuration loaded successfully");

    info!("Initializing database connection");
    let db = db::Database::new(&config.database).await?;
    info!("Database connected");

    info!("Initializing contexts");
    let ue_context = context::UeContextManager::new();
    let ran_context = context::RanContextManager::new();

    info!("Starting NRF client");
    let nrf_client = nf_client::nrf::NrfClient::new(&config.nrf).await?;
    nrf_client.register().await?;
    info!("Registered with NRF");

    info!("Starting SBI server on {}", config.sbi.bind_addr);
    let sbi_server = sbi::server::create_server(
        &config.sbi,
        ue_context.clone(),
        ran_context.clone(),
        db.clone(),
    ).await?;

    info!("Starting NGAP server on {}", config.ngap.bind_addr);
    let ngap_server = ngap::server::create_server(
        &config.ngap,
        ue_context.clone(),
        ran_context.clone(),
        db.clone(),
    );

    tokio::select! {
        result = sbi_server => {
            error!("SBI server exited: {:?}", result);
        }
        result = ngap_server => {
            error!("NGAP server exited: {:?}", result);
        }
        _ = tokio::signal::ctrl_c() => {
            info!("Received shutdown signal");
        }
    }

    info!("Deregistering from NRF");
    nrf_client.deregister().await?;

    info!("AMF shutdown complete");
    Ok(())
}

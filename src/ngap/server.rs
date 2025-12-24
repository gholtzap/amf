use anyhow::Result;
use tokio::net::UdpSocket;
use tracing::{info, error, debug};

use crate::config::NgapConfig;
use crate::context::{UeContextManager, RanContextManager};
use crate::db::Database;

pub async fn create_server(
    config: &NgapConfig,
    _ue_context: UeContextManager,
    _ran_context: RanContextManager,
    _db: Database,
) -> Result<()> {
    let socket = UdpSocket::bind(&config.bind_addr).await?;
    info!("NGAP server listening on {}", config.bind_addr);

    let mut buf = vec![0u8; 65536];

    loop {
        match socket.recv_from(&mut buf).await {
            Ok((len, addr)) => {
                debug!("Received {} bytes from {}", len, addr);
            }
            Err(e) => {
                error!("Error receiving NGAP message: {}", e);
            }
        }
    }
}

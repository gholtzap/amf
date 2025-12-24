use anyhow::Result;
use tokio::net::UdpSocket;
use std::sync::Arc;
use tracing::{info, error, debug};

use crate::config::{NgapConfig, Config};
use crate::context::{UeContextManager, RanContextManager};
use crate::db::Database;
use super::codec::{NgapPdu, NgapMessageValue};
use super::handlers;

pub async fn create_server(
    config: &Config,
    ue_context: UeContextManager,
    ran_context: RanContextManager,
    _db: Database,
) -> Result<()> {
    let socket = Arc::new(UdpSocket::bind(&config.ngap.bind_addr).await?);
    info!("NGAP server listening on {}", config.ngap.bind_addr);

    let mut buf = vec![0u8; 65536];

    loop {
        match socket.recv_from(&mut buf).await {
            Ok((len, addr)) => {
                debug!("Received {} bytes from {}", len, addr);

                match NgapPdu::decode(&buf[..len]) {
                    Ok(pdu) => {
                        let socket_clone = socket.clone();
                        let ran_context_clone = ran_context.clone();
                        let ue_context_clone = ue_context.clone();
                        let config_clone = config.clone();

                        tokio::spawn(async move {
                            if let Err(e) = handle_ngap_message(
                                pdu,
                                config_clone,
                                &ran_context_clone,
                                &ue_context_clone,
                                addr,
                                socket_clone,
                            ).await {
                                error!("Error handling NGAP message: {}", e);
                            }
                        });
                    }
                    Err(e) => {
                        error!("Failed to decode NGAP PDU: {}", e);
                    }
                }
            }
            Err(e) => {
                error!("Error receiving NGAP message: {}", e);
            }
        }
    }
}

async fn handle_ngap_message(
    pdu: NgapPdu,
    config: Config,
    ran_context: &RanContextManager,
    ue_context: &UeContextManager,
    addr: std::net::SocketAddr,
    socket: Arc<UdpSocket>,
) -> Result<()> {
    match pdu {
        NgapPdu::InitiatingMessage(msg) => {
            match msg.value {
                NgapMessageValue::NgSetupRequest(request) => {
                    let response_pdu = handlers::handle_ng_setup_request(
                        request,
                        &config,
                        ran_context,
                        addr,
                    ).await?;

                    let encoded = response_pdu.encode()?;
                    socket.send_to(&encoded, addr).await?;
                    info!("Sent NG Setup response to {}", addr);
                }
                NgapMessageValue::InitialUeMessage(message) => {
                    handlers::handle_initial_ue_message(
                        message,
                        ran_context,
                        ue_context,
                        addr,
                    ).await?;
                    info!("Processed Initial UE Message from {}", addr);
                }
                NgapMessageValue::UplinkNasTransport => {
                    debug!("Received Uplink NAS Transport");
                }
                _ => {
                    debug!("Received unknown initiating message");
                }
            }
        }
        NgapPdu::SuccessfulOutcome(_) => {
            debug!("Received successful outcome");
        }
        NgapPdu::UnsuccessfulOutcome(_) => {
            debug!("Received unsuccessful outcome");
        }
    }

    Ok(())
}

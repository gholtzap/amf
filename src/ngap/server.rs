use anyhow::{Result, Context};
use std::sync::Arc;
use std::os::unix::io::AsRawFd;
use tracing::{info, error, debug};
use socket2::{Socket, Domain, Type, Protocol};
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpStream;

use crate::config::{NgapConfig, Config};
use crate::context::{UeContextManager, RanContextManager};
use crate::db::Database;
use super::codec::{NgapPdu, NgapMessageValue};
use super::handlers;

const IPPROTO_SCTP: i32 = 132;

pub async fn create_server(
    config: &Config,
    ue_context: UeContextManager,
    ran_context: RanContextManager,
    _db: Database,
) -> Result<()> {
    let addr: std::net::SocketAddr = config.ngap.bind_addr.parse()
        .context("Failed to parse bind address")?;

    let socket = Socket::new(
        if addr.is_ipv4() { Domain::IPV4 } else { Domain::IPV6 },
        Type::STREAM,
        Some(Protocol::from(IPPROTO_SCTP)),
    )?;

    socket.set_reuse_address(true)?;
    socket.set_reuse_port(true)?;
    socket.bind(&addr.into())?;
    socket.listen(128)?;
    socket.set_nonblocking(true)?;

    let std_listener: std::net::TcpListener = socket.into();
    let listener = tokio::net::TcpListener::from_std(std_listener)?;

    info!("NGAP server listening on {} (SCTP)", config.ngap.bind_addr);

    loop {
        match listener.accept().await {
            Ok((stream, addr)) => {
                info!("Accepted SCTP association from {}", addr);

                let ran_context_clone = ran_context.clone();
                let ue_context_clone = ue_context.clone();
                let config_clone = config.clone();

                tokio::spawn(async move {
                    if let Err(e) = handle_sctp_association(
                        stream,
                        config_clone,
                        ran_context_clone,
                        ue_context_clone,
                        addr,
                    ).await {
                        error!("Error handling SCTP association from {}: {}", addr, e);
                    }
                });
            }
            Err(e) => {
                error!("Error accepting SCTP association: {}", e);
            }
        }
    }
}

async fn handle_sctp_association(
    mut stream: TcpStream,
    config: Config,
    ran_context: RanContextManager,
    ue_context: UeContextManager,
    addr: std::net::SocketAddr,
) -> Result<()> {
    let mut buf = vec![0u8; 65536];

    loop {
        match stream.read(&mut buf).await {
            Ok(0) => {
                info!("SCTP association closed by {}", addr);
                break;
            }
            Ok(len) => {
                debug!("Received {} bytes from {}", len, addr);
                info!("Raw NGAP PDU hex dump: {}", hex::encode(&buf[..len]));

                match NgapPdu::decode(&buf[..len]) {
                    Ok(pdu) => {
                        if let Err(e) = handle_ngap_message(
                            pdu,
                            &config,
                            &ran_context,
                            &ue_context,
                            addr,
                            &mut stream,
                        ).await {
                            error!("Error handling NGAP message: {}", e);
                        }
                    }
                    Err(e) => {
                        error!("Failed to decode NGAP PDU: {}", e);
                    }
                }
            }
            Err(e) => {
                error!("Error reading from SCTP stream: {}", e);
                break;
            }
        }
    }

    Ok(())
}

async fn handle_ngap_message(
    pdu: NgapPdu,
    config: &Config,
    ran_context: &RanContextManager,
    ue_context: &UeContextManager,
    addr: std::net::SocketAddr,
    stream: &mut TcpStream,
) -> Result<()> {
    match pdu {
        NgapPdu::InitiatingMessage(msg) => {
            match msg.value {
                NgapMessageValue::NgSetupRequest(request) => {
                    let response_pdu = handlers::handle_ng_setup_request(
                        request,
                        config,
                        ran_context,
                        addr,
                    ).await?;

                    let encoded = response_pdu.encode()?;
                    stream.write_all(&encoded).await?;
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

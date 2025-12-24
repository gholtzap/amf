use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PduSessionEstablishmentRequest {
    pub pdu_session_id: u8,
    pub pti: u8,
    pub integrity_protection_maximum_data_rate: IntegrityProtectionMaximumDataRate,
    pub pdu_session_type: Option<PduSessionType>,
    pub ssc_mode: Option<SscMode>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IntegrityProtectionMaximumDataRate {
    pub max_data_rate_uplink: u8,
    pub max_data_rate_downlink: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum PduSessionType {
    Ipv4,
    Ipv6,
    Ipv4v6,
    Unstructured,
    Ethernet,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SscMode {
    Mode1,
    Mode2,
    Mode3,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PduSessionEstablishmentAccept {
    pub pdu_session_id: u8,
    pub pti: u8,
    pub selected_pdu_session_type: PduSessionType,
    pub selected_ssc_mode: SscMode,
    pub authorized_qos_rules: Vec<QosRule>,
    pub session_ambr: SessionAmbr,
    pub pdu_address: Option<PduAddress>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QosRule {
    pub qos_rule_id: u8,
    pub rule_operation_code: u8,
    pub dqr: bool,
    pub qos_flow_id: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionAmbr {
    pub downlink: u64,
    pub uplink: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum PduAddress {
    Ipv4(String),
    Ipv6(String),
    Ipv4v6 { ipv4: String, ipv6: String },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PduSessionEstablishmentReject {
    pub pdu_session_id: u8,
    pub pti: u8,
    pub cause: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PduSessionReleaseRequest {
    pub pdu_session_id: u8,
    pub pti: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PduSessionReleaseCommand {
    pub pdu_session_id: u8,
    pub pti: u8,
    pub cause: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PduSessionReleaseComplete {
    pub pdu_session_id: u8,
    pub pti: u8,
}

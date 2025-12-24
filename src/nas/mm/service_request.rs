use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceRequest {
    pub ng_ksi: u8,
    pub service_type: ServiceType,
    pub s_tmsi: u32,
    pub uplink_data_status: Option<Vec<u8>>,
    pub pdu_session_status: Option<Vec<u8>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ServiceType {
    Signalling,
    Data,
    MobileTerminatedServices,
    Emergency,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceAccept {
    pub pdu_session_status: Option<Vec<u8>>,
    pub pdu_session_reactivation_result: Option<Vec<u8>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceReject {
    pub cause: u8,
    pub t3346_value: Option<u32>,
}

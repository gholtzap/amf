use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IdentityRequest {
    pub identity_type: IdentityType,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum IdentityType {
    Suci,
    Guti,
    Imei,
    STmsi,
    Imeisv,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IdentityResponse {
    pub mobile_identity: MobileIdentity,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MobileIdentity {
    Suci(String),
    Guti(String),
    Imei(String),
    STmsi(u32),
    Imeisv(String),
}

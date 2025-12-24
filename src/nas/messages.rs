use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NasMessageHeader {
    pub extended_protocol_discriminator: u8,
    pub security_header_type: u8,
    pub message_type: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum NasMessageType {
    RegistrationRequest = 0x41,
    RegistrationAccept = 0x42,
    RegistrationReject = 0x44,
    DeregistrationRequest = 0x45,
    DeregistrationAccept = 0x46,
    ServiceRequest = 0x4c,
    ServiceAccept = 0x4e,
    ServiceReject = 0x4d,
    AuthenticationRequest = 0x56,
    AuthenticationResponse = 0x57,
    AuthenticationReject = 0x58,
    AuthenticationFailure = 0x59,
    SecurityModeCommand = 0x5d,
    SecurityModeComplete = 0x5e,
    SecurityModeReject = 0x5f,
    IdentityRequest = 0x5b,
    IdentityResponse = 0x5c,
    PduSessionEstablishmentRequest = 0xc1,
    PduSessionEstablishmentAccept = 0xc2,
    PduSessionEstablishmentReject = 0xc3,
    PduSessionReleaseRequest = 0xd1,
    PduSessionReleaseCommand = 0xd2,
    PduSessionReleaseComplete = 0xd3,
}

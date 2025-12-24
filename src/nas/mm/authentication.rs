use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthenticationRequest {
    pub ng_ksi: u8,
    pub abba: Vec<u8>,
    pub authentication_parameter_rand: Vec<u8>,
    pub authentication_parameter_autn: Vec<u8>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthenticationResponse {
    pub authentication_response_parameter: Vec<u8>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthenticationFailure {
    pub cause: u8,
    pub authentication_failure_parameter: Option<Vec<u8>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthenticationReject {}

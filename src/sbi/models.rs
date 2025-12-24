use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UeContextCreateData {
    pub supi: String,
    pub pei: Option<String>,
    pub gpsi: Option<String>,
    pub ue_context_request: UeContextRequest,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum UeContextRequest {
    Initial,
    Existing,
    Emergency,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UeContextCreatedData {
    pub ue_context_id: String,
    pub supi: String,
    pub pei: Option<String>,
    pub gpsi: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct N1N2MessageTransferRequest {
    pub n1_message_container: Option<N1MessageContainer>,
    pub n2_info_container: Option<N2InfoContainer>,
    pub pdu_session_id: Option<u8>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct N1MessageContainer {
    pub n1_message_class: N1MessageClass,
    pub n1_message_content: N1MessageContent,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum N1MessageClass {
    Sm,
    Lpp,
    Sms,
    Updp,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct N1MessageContent {
    pub content_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct N2InfoContainer {
    pub n2_information_class: N2InformationClass,
    pub n2_info_content: N2InfoContent,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum N2InformationClass {
    Sm,
    Nrppa,
    Pws,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct N2InfoContent {
    pub ngap_message_type: u8,
    pub ngap_ie_type: u8,
    pub ngap_data: Vec<u8>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AmfEventSubscription {
    pub event_list: Vec<AmfEvent>,
    pub event_notif_uri: String,
    pub notif_id: String,
    pub supi: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AmfEvent {
    pub event_type: AmfEventType,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AmfEventType {
    LocationReport,
    PresenceInAoi,
    CommunicationFailure,
    Reachability,
    RegistrationStateReport,
    ConnectivityStateReport,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AmfCreatedEventSubscription {
    pub subscription: AmfEventSubscription,
    pub subscription_id: String,
}

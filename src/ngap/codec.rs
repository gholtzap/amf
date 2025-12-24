use anyhow::{Result, anyhow};
use bytes::{Bytes, BytesMut, Buf, BufMut};

use super::messages::*;

const NGAP_PROCEDURE_CODE_NG_SETUP: u8 = 21;
const NGAP_PROCEDURE_CODE_INITIAL_UE_MESSAGE: u8 = 15;
const NGAP_PROCEDURE_CODE_UPLINK_NAS_TRANSPORT: u8 = 46;

#[derive(Debug)]
pub enum NgapPdu {
    InitiatingMessage(InitiatingMessage),
    SuccessfulOutcome(SuccessfulOutcome),
    UnsuccessfulOutcome(UnsuccessfulOutcome),
}

#[derive(Debug)]
pub struct InitiatingMessage {
    pub procedure_code: u8,
    pub criticality: u8,
    pub value: NgapMessageValue,
}

#[derive(Debug)]
pub struct SuccessfulOutcome {
    pub procedure_code: u8,
    pub criticality: u8,
    pub value: NgapMessageValue,
}

#[derive(Debug)]
pub struct UnsuccessfulOutcome {
    pub procedure_code: u8,
    pub criticality: u8,
    pub value: NgapMessageValue,
}

#[derive(Debug)]
pub enum NgapMessageValue {
    NgSetupRequest(NgSetupRequest),
    NgSetupResponse(NgSetupResponse),
    NgSetupFailure(NgSetupFailure),
    InitialUeMessage(InitialUeMessage),
    UplinkNasTransport,
    Unknown,
}

impl NgapPdu {
    pub fn decode(data: &[u8]) -> Result<Self> {
        if data.len() < 3 {
            return Err(anyhow!("NGAP PDU too short"));
        }

        let pdu_type = data[0] & 0xE0;

        match pdu_type {
            0x00 => {
                let msg = decode_initiating_message(&data[1..])?;
                Ok(NgapPdu::InitiatingMessage(msg))
            }
            0x20 => {
                let msg = decode_successful_outcome(&data[1..])?;
                Ok(NgapPdu::SuccessfulOutcome(msg))
            }
            0x40 => {
                let msg = decode_unsuccessful_outcome(&data[1..])?;
                Ok(NgapPdu::UnsuccessfulOutcome(msg))
            }
            _ => Err(anyhow!("Unknown NGAP PDU type: {:02x}", pdu_type))
        }
    }

    pub fn encode(&self) -> Result<Bytes> {
        let mut buf = BytesMut::new();

        match self {
            NgapPdu::InitiatingMessage(msg) => {
                buf.put_u8(0x00);
                encode_initiating_message(msg, &mut buf)?;
            }
            NgapPdu::SuccessfulOutcome(msg) => {
                buf.put_u8(0x20);
                encode_successful_outcome(msg, &mut buf)?;
            }
            NgapPdu::UnsuccessfulOutcome(msg) => {
                buf.put_u8(0x40);
                encode_unsuccessful_outcome(msg, &mut buf)?;
            }
        }

        Ok(buf.freeze())
    }
}

fn decode_initiating_message(data: &[u8]) -> Result<InitiatingMessage> {
    if data.len() < 2 {
        return Err(anyhow!("Initiating message too short"));
    }

    let procedure_code = data[0];
    let criticality = (data[1] >> 6) & 0x03;

    let value = match procedure_code {
        NGAP_PROCEDURE_CODE_NG_SETUP => {
            let request = decode_ng_setup_request(&data[2..])?;
            NgapMessageValue::NgSetupRequest(request)
        }
        NGAP_PROCEDURE_CODE_INITIAL_UE_MESSAGE => {
            let message = decode_initial_ue_message(&data[2..])?;
            NgapMessageValue::InitialUeMessage(message)
        }
        NGAP_PROCEDURE_CODE_UPLINK_NAS_TRANSPORT => {
            NgapMessageValue::UplinkNasTransport
        }
        _ => NgapMessageValue::Unknown,
    };

    Ok(InitiatingMessage {
        procedure_code,
        criticality,
        value,
    })
}

fn encode_successful_outcome(msg: &SuccessfulOutcome, buf: &mut BytesMut) -> Result<()> {
    buf.put_u8(msg.procedure_code);
    buf.put_u8(msg.criticality << 6);

    match &msg.value {
        NgapMessageValue::NgSetupResponse(response) => {
            encode_ng_setup_response(response, buf)?;
        }
        _ => return Err(anyhow!("Unsupported successful outcome message")),
    }

    Ok(())
}

fn decode_successful_outcome(data: &[u8]) -> Result<SuccessfulOutcome> {
    if data.len() < 2 {
        return Err(anyhow!("Successful outcome too short"));
    }

    let procedure_code = data[0];
    let criticality = (data[1] >> 6) & 0x03;

    let value = match procedure_code {
        NGAP_PROCEDURE_CODE_NG_SETUP => {
            NgapMessageValue::NgSetupResponse(NgSetupResponse {
                amf_name: String::new(),
                served_guami_list: Vec::new(),
                relative_amf_capacity: 0,
                plmn_support_list: Vec::new(),
            })
        }
        _ => NgapMessageValue::Unknown,
    };

    Ok(SuccessfulOutcome {
        procedure_code,
        criticality,
        value,
    })
}

fn decode_unsuccessful_outcome(data: &[u8]) -> Result<UnsuccessfulOutcome> {
    if data.len() < 2 {
        return Err(anyhow!("Unsuccessful outcome too short"));
    }

    let procedure_code = data[0];
    let criticality = (data[1] >> 6) & 0x03;

    let value = match procedure_code {
        NGAP_PROCEDURE_CODE_NG_SETUP => {
            let failure = decode_ng_setup_failure(&data[2..])?;
            NgapMessageValue::NgSetupFailure(failure)
        }
        _ => NgapMessageValue::Unknown,
    };

    Ok(UnsuccessfulOutcome {
        procedure_code,
        criticality,
        value,
    })
}

fn encode_initiating_message(_msg: &InitiatingMessage, _buf: &mut BytesMut) -> Result<()> {
    Ok(())
}

fn encode_unsuccessful_outcome(msg: &UnsuccessfulOutcome, buf: &mut BytesMut) -> Result<()> {
    buf.put_u8(msg.procedure_code);
    buf.put_u8(msg.criticality << 6);

    match &msg.value {
        NgapMessageValue::NgSetupFailure(failure) => {
            encode_ng_setup_failure(failure, buf)?;
        }
        _ => return Err(anyhow!("Unsupported unsuccessful outcome message")),
    }

    Ok(())
}

fn decode_ng_setup_request(data: &[u8]) -> Result<NgSetupRequest> {
    let mut cursor = 0;
    let mut global_ran_node_id = None;
    let mut supported_ta_list = Vec::new();
    let mut default_paging_drx = 32;

    while cursor < data.len() {
        if cursor + 3 > data.len() {
            break;
        }

        let ie_id = data[cursor];
        cursor += 3;

        match ie_id {
            27 => {
                let (node_id, consumed) = decode_global_ran_node_id(&data[cursor..])?;
                global_ran_node_id = Some(node_id);
                cursor += consumed;
            }
            102 => {
                let (ta_list, consumed) = decode_supported_ta_list(&data[cursor..])?;
                supported_ta_list = ta_list;
                cursor += consumed;
            }
            96 => {
                if cursor + 1 <= data.len() {
                    default_paging_drx = data[cursor] as u32;
                    cursor += 1;
                }
            }
            _ => {
                if cursor < data.len() {
                    cursor += 1;
                }
            }
        }
    }

    Ok(NgSetupRequest {
        global_ran_node_id: global_ran_node_id.ok_or_else(|| anyhow!("Missing global RAN node ID"))?,
        supported_ta_list,
        default_paging_drx,
    })
}

fn decode_global_ran_node_id(data: &[u8]) -> Result<(GlobalRanNodeId, usize)> {
    if data.len() < 8 {
        return Err(anyhow!("Global RAN node ID too short"));
    }

    let plmn = decode_plmn_identity(&data[0..3]);
    let ran_node_id = format!("{:02x}{:02x}{:02x}{:02x}", data[4], data[5], data[6], data[7]);

    Ok((GlobalRanNodeId {
        plmn_identity: plmn,
        ran_node_id,
    }, 8))
}

fn decode_plmn_identity(data: &[u8]) -> PlmnIdentity {
    let mcc = format!("{}{}{}",
        data[0] & 0x0F,
        (data[0] >> 4) & 0x0F,
        data[1] & 0x0F
    );
    let mnc = format!("{}{}",
        (data[1] >> 4) & 0x0F,
        data[2] & 0x0F
    );

    PlmnIdentity { mcc, mnc }
}

fn encode_plmn_identity(plmn: &PlmnIdentity, buf: &mut BytesMut) {
    let mcc_bytes: Vec<u8> = plmn.mcc.chars().map(|c| c.to_digit(10).unwrap() as u8).collect();
    let mnc_bytes: Vec<u8> = plmn.mnc.chars().map(|c| c.to_digit(10).unwrap() as u8).collect();

    if mcc_bytes.len() >= 3 && mnc_bytes.len() >= 2 {
        buf.put_u8((mcc_bytes[1] << 4) | mcc_bytes[0]);
        buf.put_u8((mnc_bytes[0] << 4) | mcc_bytes[2]);
        buf.put_u8(mnc_bytes[1]);
    }
}

fn decode_supported_ta_list(data: &[u8]) -> Result<(Vec<SupportedTaItem>, usize)> {
    let mut list = Vec::new();
    let mut cursor = 0;

    if cursor + 1 > data.len() {
        return Ok((list, 0));
    }

    let count = data[cursor] as usize;
    cursor += 1;

    for _ in 0..count {
        if cursor + 6 > data.len() {
            break;
        }

        let tac = format!("{:02x}{:02x}{:02x}", data[cursor], data[cursor+1], data[cursor+2]);
        cursor += 3;

        let plmn_count = data[cursor] as usize;
        cursor += 1;

        let mut broadcast_plmn_list = Vec::new();
        for _ in 0..plmn_count {
            if cursor + 3 > data.len() {
                break;
            }

            let plmn = decode_plmn_identity(&data[cursor..cursor+3]);
            cursor += 3;

            let slice_count = if cursor < data.len() { data[cursor] as usize } else { 0 };
            cursor += 1;

            let mut tai_slice_support_list = Vec::new();
            for _ in 0..slice_count {
                if cursor + 1 > data.len() {
                    break;
                }

                let sst = data[cursor];
                cursor += 1;

                tai_slice_support_list.push(SliceSupportItem {
                    s_nssai: SNssai {
                        sst,
                        sd: None,
                    }
                });
            }

            broadcast_plmn_list.push(BroadcastPlmnItem {
                plmn_identity: plmn,
                tai_slice_support_list,
            });
        }

        list.push(SupportedTaItem {
            tac,
            broadcast_plmn_list,
        });
    }

    Ok((list, cursor))
}

fn encode_ng_setup_response(response: &NgSetupResponse, buf: &mut BytesMut) -> Result<()> {
    buf.put_u8(0x00);
    buf.put_u8(0x00);
    buf.put_u8(0x04);

    buf.put_u8(1);
    buf.put_u8(0x00);
    let name_len = response.amf_name.len().min(150);
    buf.put_u8(name_len as u8);
    buf.put_slice(response.amf_name.as_bytes());

    buf.put_u8(96);
    buf.put_u8(0x00);
    buf.put_u8(response.served_guami_list.len() as u8);
    for guami in &response.served_guami_list {
        encode_plmn_identity(&guami.plmn_identity, buf);
        buf.put_slice(guami.amf_region_id.as_bytes());
        buf.put_slice(guami.amf_set_id.as_bytes());
        buf.put_slice(guami.amf_pointer.as_bytes());
    }

    buf.put_u8(80);
    buf.put_u8(0x00);
    buf.put_u8(0x01);
    buf.put_u8(response.relative_amf_capacity);

    buf.put_u8(86);
    buf.put_u8(0x00);
    buf.put_u8(response.plmn_support_list.len() as u8);
    for plmn_support in &response.plmn_support_list {
        encode_plmn_identity(&plmn_support.plmn_identity, buf);
        buf.put_u8(plmn_support.slice_support_list.len() as u8);
        for slice in &plmn_support.slice_support_list {
            buf.put_u8(slice.s_nssai.sst);
        }
    }

    Ok(())
}

fn decode_ng_setup_failure(data: &[u8]) -> Result<NgSetupFailure> {
    let mut cursor = 0;
    let mut cause = None;

    while cursor + 3 < data.len() {
        let ie_id = data[cursor];
        cursor += 3;

        if ie_id == 15 {
            if cursor < data.len() {
                let cause_type = data[cursor];
                cursor += 1;
                if cursor < data.len() {
                    let cause_value = data[cursor];
                    cause = Some(Cause {
                        cause_type,
                        cause_value,
                    });
                    cursor += 1;
                }
            }
        } else {
            if cursor < data.len() {
                cursor += 1;
            }
        }
    }

    Ok(NgSetupFailure {
        cause: cause.ok_or_else(|| anyhow!("Missing cause in NG Setup Failure"))?,
        time_to_wait: None,
        critical_diagnostics: None,
    })
}

fn encode_ng_setup_failure(failure: &NgSetupFailure, buf: &mut BytesMut) -> Result<()> {
    buf.put_u8(0x00);
    buf.put_u8(0x00);
    buf.put_u8(0x01);

    buf.put_u8(15);
    buf.put_u8(0x00);
    buf.put_u8(0x02);
    buf.put_u8(failure.cause.cause_type);
    buf.put_u8(failure.cause.cause_value);

    if let Some(ttw) = failure.time_to_wait {
        buf.put_u8(12);
        buf.put_u8(0x00);
        buf.put_u8(0x01);
        buf.put_u8(ttw);
    }

    Ok(())
}

fn decode_initial_ue_message(data: &[u8]) -> Result<InitialUeMessage> {
    let mut cursor = 0;
    let mut ran_ue_ngap_id = None;
    let mut nas_pdu = Vec::new();
    let mut user_location_info = None;
    let mut rrc_establishment_cause = 0;

    while cursor < data.len() {
        if cursor + 3 > data.len() {
            break;
        }

        let ie_id = data[cursor];
        cursor += 1;
        let _criticality = data[cursor];
        cursor += 1;
        let length = data[cursor] as usize;
        cursor += 1;

        if cursor + length > data.len() {
            break;
        }

        match ie_id {
            85 => {
                if length >= 4 {
                    let id = u32::from_be_bytes([
                        data[cursor],
                        data[cursor + 1],
                        data[cursor + 2],
                        data[cursor + 3],
                    ]) as u64;
                    ran_ue_ngap_id = Some(id);
                }
                cursor += length;
            }
            38 => {
                nas_pdu = data[cursor..cursor + length].to_vec();
                cursor += length;
            }
            121 => {
                if length >= 9 {
                    let tai_plmn = decode_plmn_identity(&data[cursor..cursor + 3]);
                    let tac = format!("{:02x}{:02x}{:02x}",
                        data[cursor + 3], data[cursor + 4], data[cursor + 5]);

                    let nr_cgi = if length >= 15 {
                        let cgi_plmn = decode_plmn_identity(&data[cursor + 6..cursor + 9]);
                        let nr_cell_id = format!("{:02x}{:02x}{:02x}{:02x}{:02x}",
                            data[cursor + 9], data[cursor + 10], data[cursor + 11],
                            data[cursor + 12], data[cursor + 13]);
                        Some(NrCgi {
                            plmn_identity: cgi_plmn,
                            nr_cell_identity: nr_cell_id,
                        })
                    } else {
                        None
                    };

                    user_location_info = Some(UserLocationInfo {
                        nr_cgi,
                        tai: Tai {
                            plmn_identity: tai_plmn,
                            tac,
                        },
                    });
                }
                cursor += length;
            }
            90 => {
                if length >= 1 {
                    rrc_establishment_cause = data[cursor];
                }
                cursor += length;
            }
            _ => {
                cursor += length;
            }
        }
    }

    Ok(InitialUeMessage {
        ran_ue_ngap_id: ran_ue_ngap_id.ok_or_else(|| anyhow!("Missing RAN-UE-NGAP-ID"))?,
        nas_pdu,
        user_location_info: user_location_info.ok_or_else(|| anyhow!("Missing User Location Info"))?,
        rrc_establishment_cause,
    })
}

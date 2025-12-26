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
        use tracing::info;
        let mut buf = BytesMut::new();

        match self {
            NgapPdu::InitiatingMessage(msg) => {
                buf.put_u8(0x00);
                info!("Encoding InitiatingMessage, PDU type byte: 0x00");
                encode_initiating_message(msg, &mut buf)?;
            }
            NgapPdu::SuccessfulOutcome(msg) => {
                buf.put_u8(0x20);
                info!("Encoding SuccessfulOutcome, PDU type byte: 0x20");
                encode_successful_outcome(msg, &mut buf)?;
            }
            NgapPdu::UnsuccessfulOutcome(msg) => {
                buf.put_u8(0x40);
                info!("Encoding UnsuccessfulOutcome, PDU type byte: 0x40");
                encode_unsuccessful_outcome(msg, &mut buf)?;
            }
        }

        info!("Final NgapPdu encoded length: {} bytes", buf.len());
        Ok(buf.freeze())
    }
}

fn decode_aper_length(data: &[u8]) -> Result<(usize, usize)> {
    use tracing::debug;

    if data.is_empty() {
        return Err(anyhow!("No data for APER length field"));
    }

    if data[0] < 0x80 {
        debug!("APER length: single byte 0x{:02x} = {}", data[0], data[0]);
        Ok((data[0] as usize, 1))
    } else if data[0] < 0xC0 {
        if data.len() < 2 {
            return Err(anyhow!("Not enough data for 2-byte APER length"));
        }
        let length = (((data[0] & 0x3F) as usize) << 8) | (data[1] as usize);
        debug!("APER length: two bytes [{:02x}, {:02x}] = {}", data[0], data[1], length);
        Ok((length, 2))
    } else {
        Err(anyhow!("Fragmented APER length not supported"))
    }
}

fn decode_initiating_message(data: &[u8]) -> Result<InitiatingMessage> {
    use tracing::debug;

    if data.len() < 2 {
        return Err(anyhow!("Initiating message too short"));
    }

    let procedure_code = data[0];
    let criticality = (data[1] >> 6) & 0x03;

    debug!("decode_initiating_message: procedure_code={}, criticality={}, data_len={}", procedure_code, criticality, data.len());
    debug!("Initiating message data: {:02x?}", &data[..data.len().min(50)]);

    let (open_type_length, length_bytes) = decode_aper_length(&data[2..])?;
    let value_start = 2 + length_bytes;
    debug!("Open type length: {}, length_bytes: {}, value_start: {}", open_type_length, length_bytes, value_start);

    if value_start + open_type_length > data.len() {
        return Err(anyhow!("Open type length {} exceeds available data {}", open_type_length, data.len() - value_start));
    }

    let value_data = &data[value_start..value_start + open_type_length];

    let value = match procedure_code {
        NGAP_PROCEDURE_CODE_NG_SETUP => {
            debug!("Decoding NG Setup Request, passing {} bytes to decoder", value_data.len());
            let request = decode_ng_setup_request(value_data)?;
            NgapMessageValue::NgSetupRequest(request)
        }
        NGAP_PROCEDURE_CODE_INITIAL_UE_MESSAGE => {
            let message = decode_initial_ue_message(value_data)?;
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
    use tracing::info;

    info!("Encoding SuccessfulOutcome: procedure_code={}, criticality={}", msg.procedure_code, msg.criticality);
    buf.put_u8(msg.procedure_code);
    buf.put_u8(msg.criticality << 6);
    info!("Procedure code byte: 0x{:02x}, Criticality byte: 0x{:02x}", msg.procedure_code, msg.criticality << 6);

    let mut value_buf = BytesMut::new();
    match &msg.value {
        NgapMessageValue::NgSetupResponse(response) => {
            encode_ng_setup_response(response, &mut value_buf)?;
        }
        _ => return Err(anyhow!("Unsupported successful outcome message")),
    }

    info!("Encoded value length: {} bytes, adding APER length determinant", value_buf.len());
    encode_aper_length(value_buf.len(), buf);
    buf.put_slice(&value_buf);
    info!("SuccessfulOutcome complete with APER length determinant");

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
    use tracing::info;

    buf.put_u8(msg.procedure_code);
    buf.put_u8(msg.criticality << 6);
    info!("UnsuccessfulOutcome: procedure_code=0x{:02x}, criticality=0x{:02x}", msg.procedure_code, msg.criticality << 6);

    let mut value_buf = BytesMut::new();
    match &msg.value {
        NgapMessageValue::NgSetupFailure(failure) => {
            encode_ng_setup_failure(failure, &mut value_buf)?;
        }
        _ => return Err(anyhow!("Unsupported unsuccessful outcome message")),
    }

    info!("Encoded value length: {} bytes, adding APER length determinant", value_buf.len());
    encode_aper_length(value_buf.len(), buf);
    buf.put_slice(&value_buf);

    Ok(())
}

fn decode_ng_setup_request(data: &[u8]) -> Result<NgSetupRequest> {
    use tracing::{debug, warn};

    let mut cursor = 0;
    let mut global_ran_node_id = None;
    let mut supported_ta_list = Vec::new();
    let mut default_paging_drx = 32;

    debug!("Decoding NG Setup Request, data length: {}", data.len());
    debug!("Raw data: {:02x?}", &data[..data.len().min(100)]);

    if cursor + 3 > data.len() {
        return Err(anyhow!("NG Setup Request too short for header"));
    }

    let extension_bit = data[cursor];
    cursor += 1;
    debug!("Extension bit: {:02x} at position 0", extension_bit);

    debug!("IE count bytes at position {}: {:02x?}", cursor, &data[cursor..cursor+2.min(data.len()-cursor)]);
    if cursor + 2 > data.len() {
        return Err(anyhow!("Not enough data for IE count"));
    }
    let ie_count = u16::from_be_bytes([data[cursor], data[cursor + 1]]) as usize;
    cursor += 2;
    debug!("Number of IEs: {} (16-bit big-endian from bytes [{:02x}, {:02x}])", ie_count, data[cursor-2], data[cursor-1]);
    debug!("Cursor after IE count: {}, remaining bytes: {}", cursor, data.len() - cursor);

    for i in 0..ie_count {
        if cursor + 3 > data.len() {
            warn!("Not enough data for IE {} header at cursor {}", i, cursor);
            break;
        }

        debug!("IE {} header bytes at cursor {}: {:02x?}", i, cursor, &data[cursor..cursor+5.min(data.len()-cursor)]);
        let ie_id = u16::from_be_bytes([data[cursor], data[cursor + 1]]) as usize;
        let ie_criticality = (data[cursor + 2] >> 6) & 0x03;
        cursor += 3;
        debug!("IE {} ID bytes: [{:02x}, {:02x}] = {}, Criticality: {:02x} = {}",
            i, data[cursor-3], data[cursor-2], ie_id, data[cursor-1], ie_criticality);

        if cursor >= data.len() {
            warn!("No length field for IE {} (ID={}) at cursor {}", i, ie_id, cursor);
            break;
        }

        debug!("IE {} length bytes at cursor {}: {:02x?}", i, cursor, &data[cursor..cursor+2.min(data.len()-cursor)]);
        let (ie_length, length_bytes) = decode_aper_length(&data[cursor..])?;
        cursor += length_bytes;

        debug!("IE {}: ID={} (expected: 27=GlobalRANNodeID, 82=RANNodeName, 102=SupportedTAList, 21=PagingDRX), Criticality={}, Length={}, Cursor={}",
            i, ie_id, ie_criticality, ie_length, cursor);

        if cursor + ie_length > data.len() {
            warn!("IE {} length {} exceeds remaining data at cursor {}", i, ie_length, cursor);
            break;
        }

        match ie_id {
            27 => {
                debug!("Found Global RAN Node ID IE at cursor {}, decoding {} bytes...", cursor, ie_length);
                debug!("RAN Node ID data: {:02x?}", &data[cursor..cursor + ie_length.min(20)]);
                match decode_global_ran_node_id(&data[cursor..cursor + ie_length]) {
                    Ok((node_id, _consumed)) => {
                        debug!("Successfully decoded Global RAN Node ID: {:?}", node_id);
                        global_ran_node_id = Some(node_id);
                        cursor += ie_length;
                    }
                    Err(e) => {
                        warn!("Failed to decode Global RAN Node ID: {}", e);
                        cursor += ie_length;
                    }
                }
            }
            102 => {
                debug!("Found Supported TA List IE at cursor {}, decoding {} bytes...", cursor, ie_length);
                debug!("TA List data: {:02x?}", &data[cursor..cursor + ie_length.min(20)]);
                match decode_supported_ta_list(&data[cursor..]) {
                    Ok((ta_list, consumed)) => {
                        debug!("Decoded {} TAs, consumed {} bytes", ta_list.len(), consumed);
                        supported_ta_list = ta_list;
                        cursor += ie_length;
                    }
                    Err(e) => {
                        warn!("Failed to decode Supported TA List: {}", e);
                        cursor += ie_length;
                    }
                }
            }
            21 => {
                debug!("Found Default Paging DRX IE at cursor {}, {} bytes", cursor, ie_length);
                if ie_length >= 1 && cursor < data.len() {
                    default_paging_drx = data[cursor] as u32;
                    debug!("Paging DRX value: {}", default_paging_drx);
                }
                cursor += ie_length;
            }
            82 => {
                debug!("Found RAN Node Name IE at cursor {}, {} bytes", cursor, ie_length);
                if ie_length > 0 {
                    let name_bytes = &data[cursor..cursor + ie_length];
                    if let Ok(name) = std::str::from_utf8(name_bytes) {
                        debug!("RAN Node Name: {}", name);
                    } else {
                        debug!("RAN Node Name (hex): {:02x?}", &name_bytes[..name_bytes.len().min(20)]);
                    }
                }
                cursor += ie_length;
            }
            _ => {
                debug!("Unknown IE ID: {} at cursor {}, skipping {} bytes", ie_id, cursor, ie_length);
                debug!("Unknown IE data: {:02x?}", &data[cursor..cursor + ie_length.min(20)]);
                cursor += ie_length;
            }
        }
    }

    if global_ran_node_id.is_none() {
        warn!("No Global RAN Node ID found in NG Setup Request");
        warn!("IE parsing summary: parsed {} IEs, final cursor position: {}/{}", ie_count, cursor, data.len());
        warn!("HINT: If IE count seems wrong, check if bytes at position 1-2 should be interpreted as 16-bit IE count instead of APER length");
        warn!("HINT: Data bytes 0-10: {:02x?}", &data[..10.min(data.len())]);
    }

    debug!("NG Setup Request decode complete: GlobalRANNodeID={}, TAs={}, PagingDRX={}",
        global_ran_node_id.is_some(), supported_ta_list.len(), default_paging_drx);

    Ok(NgSetupRequest {
        global_ran_node_id: global_ran_node_id.ok_or_else(|| anyhow!("Missing global RAN node ID"))?,
        supported_ta_list,
        default_paging_drx,
    })
}

fn decode_length(data: &[u8]) -> Result<(usize, usize)> {
    if data.is_empty() {
        return Err(anyhow!("No data for length field"));
    }

    if data[0] < 128 {
        Ok((data[0] as usize, 1))
    } else {
        let num_bytes = (data[0] & 0x7F) as usize;
        if num_bytes == 0 || num_bytes > 4 {
            return Err(anyhow!("Invalid length encoding: {} bytes", num_bytes));
        }
        if data.len() < 1 + num_bytes {
            return Err(anyhow!("Not enough data for multi-byte length"));
        }

        let mut length = 0usize;
        for i in 0..num_bytes {
            length = (length << 8) | (data[1 + i] as usize);
        }
        Ok((length, 1 + num_bytes))
    }
}

fn decode_global_ran_node_id(data: &[u8]) -> Result<(GlobalRanNodeId, usize)> {
    use tracing::debug;

    debug!("decode_global_ran_node_id: input data length={}, data={:02x?}", data.len(), &data[..data.len().min(30)]);

    if data.len() < 5 {
        return Err(anyhow!("Global RAN node ID too short: {} bytes (need at least 5)", data.len()));
    }

    let mut cursor = 0;

    let choice_tag = data[cursor];
    cursor += 1;
    debug!("Choice tag at position 0: 0x{:02x} = {} (0=globalGNB-ID, 1=globalNgENB-ID, 2=globalN3IWF-ID, 3=globalTNGF-ID, 4=globalTWIF-ID, 5=globalW-AGF-ID)", choice_tag, choice_tag);

    if cursor + 3 > data.len() {
        return Err(anyhow!("Not enough data for PLMN at cursor {}", cursor));
    }

    debug!("PLMN bytes at position {}-{}: {:02x?}", cursor, cursor+2, &data[cursor..cursor+3]);
    let plmn = decode_plmn_identity(&data[cursor..cursor + 3]);
    debug!("Decoded PLMN: MCC={}, MNC={}", plmn.mcc, plmn.mnc);
    cursor += 3;

    if cursor >= data.len() {
        return Err(anyhow!("No RAN node ID present after PLMN at cursor {}", cursor));
    }

    let id_header = data[cursor];
    cursor += 1;
    debug!("RAN ID header at position {}: 0x{:02x}", cursor-1, id_header);

    let remaining = data.len() - cursor;
    if remaining == 0 {
        return Err(anyhow!("No RAN node ID value bytes at cursor {}", cursor));
    }

    let mut node_id_value = String::new();
    for i in 0..remaining {
        node_id_value.push_str(&format!("{:02x}", data[cursor + i]));
    }

    debug!("Decoded RAN Node ID value: {} ({} bytes, {} bits)", node_id_value, remaining, remaining * 8);

    let global_ran_node_id = match choice_tag {
        0 => GlobalRanNodeId::GNB(GlobalGnbId {
            plmn_identity: plmn,
            gnb_id: GnbId::GnbId {
                value: node_id_value,
                bit_length: (remaining * 8) as u8,
            },
        }),
        1 => GlobalRanNodeId::NgENB(GlobalNgEnbId {
            plmn_identity: plmn,
            ng_enb_id: NgEnbId::MacroNgEnbId(node_id_value),
        }),
        2 => GlobalRanNodeId::N3IWF(GlobalN3iwfId {
            plmn_identity: plmn,
            n3iwf_id: node_id_value,
        }),
        3 => GlobalRanNodeId::TNGF(GlobalTngfId {
            plmn_identity: plmn,
            tngf_id: node_id_value,
        }),
        4 => GlobalRanNodeId::TWIF(GlobalTwifId {
            plmn_identity: plmn,
            twif_id: node_id_value,
        }),
        5 => GlobalRanNodeId::WAGF(GlobalWagfId {
            plmn_identity: plmn,
            wagf_id: node_id_value,
        }),
        _ => return Err(anyhow!("Unknown RAN node type: {}", choice_tag)),
    };

    Ok((global_ran_node_id, data.len()))
}

fn decode_plmn_identity(data: &[u8]) -> PlmnIdentity {
    let mcc = format!("{}{}{}",
        data[0] & 0x0F,
        (data[0] >> 4) & 0x0F,
        data[1] & 0x0F
    );

    let mnc_digit3 = (data[1] >> 4) & 0x0F;
    let mnc = if mnc_digit3 == 0x0F {
        format!("{}{}",
            data[2] & 0x0F,
            (data[2] >> 4) & 0x0F
        )
    } else {
        format!("{}{}{}",
            data[2] & 0x0F,
            (data[2] >> 4) & 0x0F,
            mnc_digit3
        )
    };

    PlmnIdentity { mcc, mnc }
}

fn encode_plmn_identity(plmn: &PlmnIdentity, buf: &mut BytesMut) {
    let mcc_bytes: Vec<u8> = plmn.mcc.chars().map(|c| c.to_digit(10).unwrap() as u8).collect();
    let mnc_bytes: Vec<u8> = plmn.mnc.chars().map(|c| c.to_digit(10).unwrap() as u8).collect();

    if mcc_bytes.len() >= 3 && mnc_bytes.len() >= 2 {
        buf.put_u8((mcc_bytes[1] << 4) | mcc_bytes[0]);

        if mnc_bytes.len() == 2 {
            buf.put_u8(0xF0 | mcc_bytes[2]);
            buf.put_u8((mnc_bytes[1] << 4) | mnc_bytes[0]);
        } else {
            buf.put_u8((mnc_bytes[2] << 4) | mcc_bytes[2]);
            buf.put_u8((mnc_bytes[1] << 4) | mnc_bytes[0]);
        }
    }
}

fn decode_supported_ta_list(data: &[u8]) -> Result<(Vec<SupportedTaItem>, usize)> {
    use tracing::debug;

    let mut list = Vec::new();
    let mut cursor = 0;

    debug!("decode_supported_ta_list: input data length={}, data={:02x?}", data.len(), &data[..data.len().min(20)]);

    if data.len() < 5 {
        return Ok((list, 0));
    }

    let count = data[cursor] as usize + 1;
    cursor += 1;
    debug!("TA count: {}", count);

    for ta_idx in 0..count {
        if cursor >= data.len() {
            break;
        }

        let extension_bit = data[cursor];
        cursor += 1;
        debug!("TA item {} extension bit: {:02x}", ta_idx, extension_bit);

        if cursor + 3 > data.len() {
            break;
        }

        let tac = format!("{:02x}{:02x}{:02x}", data[cursor], data[cursor+1], data[cursor+2]);
        cursor += 3;
        debug!("TAC: {}", tac);

        if cursor >= data.len() {
            break;
        }

        let plmn_count = data[cursor] as usize + 1;
        cursor += 1;
        debug!("PLMN count: {}", plmn_count);

        let mut broadcast_plmn_list = Vec::new();
        for plmn_idx in 0..plmn_count {
            if cursor + 3 > data.len() {
                break;
            }

            let plmn = decode_plmn_identity(&data[cursor..cursor+3]);
            cursor += 3;
            debug!("Decoded PLMN {}: MCC={}, MNC={}", plmn_idx, plmn.mcc, plmn.mnc);

            if cursor >= data.len() {
                break;
            }

            let slice_count = data[cursor] as usize + 1;
            cursor += 1;
            debug!("Slice count: {}", slice_count);

            let mut tai_slice_support_list = Vec::new();
            for slice_idx in 0..slice_count {
                if cursor >= data.len() {
                    break;
                }

                let slice_extension_bit = data[cursor];
                cursor += 1;
                debug!("Slice {} extension bit: {:02x}", slice_idx, slice_extension_bit);

                if cursor >= data.len() {
                    break;
                }

                let sst = data[cursor];
                cursor += 1;
                debug!("SST: {}", sst);

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
    use tracing::{debug, info};

    let mut value_buf = BytesMut::new();

    value_buf.put_u8(0x00);
    debug!("Extension bit: 0x00");

    let ie_count = 4usize;
    value_buf.put_u16(ie_count as u16);
    debug!("IE count: {} encoded as 16-bit big-endian: 0x{:04x}", ie_count, ie_count);

    let ie_data = encode_ng_setup_response_ies(response)?;
    debug!("IE data length: {}, hex: {}", ie_data.len(), hex::encode(&ie_data));
    value_buf.put_slice(&ie_data);

    debug!("NG Setup Response value_buf length: {}", value_buf.len());
    info!("NG Setup Response value_buf hex: {}", hex::encode(&value_buf));

    encode_aper_length(value_buf.len(), buf);
    buf.put_slice(&value_buf);

    info!("Final encoded response buffer hex: {}", hex::encode(&buf));

    Ok(())
}

fn encode_ng_setup_response_ies(response: &NgSetupResponse) -> Result<Bytes> {
    use tracing::{debug, info};
    let mut buf = BytesMut::new();

    info!("Encoding IE 1 - AMFName: '{}'", response.amf_name);
    let ie1_start = buf.len();
    buf.put_u16(1);
    debug!("  IE ID: 0x0001 (2 bytes)");
    buf.put_u8(0x00);
    debug!("  Criticality byte: 0x00");
    let name_bytes = response.amf_name.as_bytes();
    let len_start = buf.len();
    encode_aper_length(name_bytes.len(), &mut buf);
    debug!("  Length: {} bytes, encoded as: {:02x?}", name_bytes.len(), &buf[len_start..]);
    buf.put_slice(name_bytes);
    debug!("  Value: {:02x?}", name_bytes);
    debug!("IE 1 complete: {:02x?}", &buf[ie1_start..]);

    info!("Encoding IE 96 - ServedGUAMIList ({} items)", response.served_guami_list.len());
    let ie96_start = buf.len();
    buf.put_u16(96);
    debug!("  IE ID: 0x0060 (2 bytes)");
    buf.put_u8(0x00);
    debug!("  Criticality byte: 0x00");
    let guami_data = encode_served_guami_list(&response.served_guami_list)?;
    debug!("  GUAMI data: {}", hex::encode(&guami_data));
    let len_start = buf.len();
    encode_aper_length(guami_data.len(), &mut buf);
    debug!("  Length: {} bytes, encoded as: {:02x?}", guami_data.len(), &buf[len_start..]);
    buf.put_slice(&guami_data);
    debug!("IE 96 complete: {:02x?}", &buf[ie96_start..]);

    info!("Encoding IE 80 - RelativeAMFCapacity: {}", response.relative_amf_capacity);
    let ie80_start = buf.len();
    buf.put_u16(80);
    debug!("  IE ID: 0x0050 (2 bytes)");
    buf.put_u8(0x00);
    debug!("  Criticality byte: 0x00");
    let len_start = buf.len();
    encode_aper_length(1, &mut buf);
    debug!("  Length: 1 byte, encoded as: {:02x?}", &buf[len_start..]);
    buf.put_u8(response.relative_amf_capacity);
    debug!("  Value: 0x{:02x}", response.relative_amf_capacity);
    debug!("IE 80 complete: {:02x?}", &buf[ie80_start..]);

    info!("Encoding IE 86 - PLMNSupportList ({} items)", response.plmn_support_list.len());
    let ie86_start = buf.len();
    buf.put_u16(86);
    debug!("  IE ID: 0x0056 (2 bytes)");
    buf.put_u8(0x00);
    debug!("  Criticality byte: 0x00");
    let plmn_data = encode_plmn_support_list(&response.plmn_support_list)?;
    debug!("  PLMN data: {}", hex::encode(&plmn_data));
    let len_start = buf.len();
    encode_aper_length(plmn_data.len(), &mut buf);
    debug!("  Length: {} bytes, encoded as: {:02x?}", plmn_data.len(), &buf[len_start..]);
    buf.put_slice(&plmn_data);
    debug!("IE 86 complete: {:02x?}", &buf[ie86_start..]);

    info!("All IEs encoded, final length: {}", buf.len());
    debug!("Complete IE buffer: {:02x?}", &buf[..]);
    Ok(buf.freeze())
}

fn encode_served_guami_list(guami_list: &[ServedGuami]) -> Result<Bytes> {
    let mut buf = BytesMut::new();

    buf.put_u8((guami_list.len() - 1) as u8);

    for guami in guami_list {
        buf.put_u8(0x00);
        encode_plmn_identity(&guami.plmn_identity, &mut buf);

        let region_id_bytes = hex::decode(&guami.amf_region_id)
            .unwrap_or_else(|_| vec![0]);
        buf.put_u8(region_id_bytes[0]);

        let set_id_val = u16::from_str_radix(&guami.amf_set_id, 16)
            .unwrap_or(0);
        buf.put_u16(set_id_val << 6);

        let pointer_val = u8::from_str_radix(&guami.amf_pointer, 16)
            .unwrap_or(0);
        buf.put_u8(pointer_val << 2);
    }

    Ok(buf.freeze())
}

fn encode_plmn_support_list(plmn_list: &[PlmnSupportItem]) -> Result<Bytes> {
    let mut buf = BytesMut::new();

    buf.put_u8((plmn_list.len() - 1) as u8);

    for plmn_support in plmn_list {
        buf.put_u8(0x00);
        encode_plmn_identity(&plmn_support.plmn_identity, &mut buf);

        buf.put_u8((plmn_support.slice_support_list.len() - 1) as u8);
        for slice in &plmn_support.slice_support_list {
            buf.put_u8(0x00);
            buf.put_u8(slice.s_nssai.sst);
        }
    }

    Ok(buf.freeze())
}

fn encode_aper_length(length: usize, buf: &mut BytesMut) {
    if length < 128 {
        buf.put_u8(length as u8);
    } else if length < 16384 {
        buf.put_u8(0x80 | (((length >> 8) & 0x3F) as u8));
        buf.put_u8((length & 0xFF) as u8);
    } else {
        buf.put_u8(0xC0);
        buf.put_u8((length >> 8) as u8);
        buf.put_u8((length & 0xFF) as u8);
    }
}

fn decode_ng_setup_failure(data: &[u8]) -> Result<NgSetupFailure> {
    use tracing::debug;

    let mut cursor = 0;
    let mut cause = None;

    if data.len() < 3 {
        return Err(anyhow!("NG Setup Failure data too short"));
    }

    let extension_bit = data[cursor];
    cursor += 1;
    debug!("NG Setup Failure extension bit: {:02x}", extension_bit);

    if cursor + 2 > data.len() {
        return Err(anyhow!("Not enough data for IE count"));
    }
    let ie_count = u16::from_be_bytes([data[cursor], data[cursor + 1]]) as usize;
    cursor += 2;
    debug!("NG Setup Failure IE count: {}", ie_count);

    for i in 0..ie_count {
        if cursor + 3 > data.len() {
            debug!("Not enough data for IE {} header", i);
            break;
        }

        let ie_id = u16::from_be_bytes([data[cursor], data[cursor + 1]]) as usize;
        let ie_criticality = (data[cursor + 2] >> 6) & 0x03;
        cursor += 3;

        if cursor >= data.len() {
            debug!("No length field for IE {}", i);
            break;
        }

        let (ie_length, length_bytes) = decode_aper_length(&data[cursor..])?;
        cursor += length_bytes;

        debug!("NG Setup Failure IE {}: ID={}, Length={}", i, ie_id, ie_length);

        if cursor + ie_length > data.len() {
            debug!("IE {} length exceeds data", i);
            break;
        }

        match ie_id {
            15 => {
                if ie_length >= 2 {
                    let cause_type = data[cursor];
                    let cause_value = data[cursor + 1];
                    cause = Some(Cause {
                        cause_type,
                        cause_value,
                    });
                    debug!("Found Cause IE: type={}, value={}", cause_type, cause_value);
                }
                cursor += ie_length;
            }
            107 => {
                debug!("Found TimeToWait IE (skipping)");
                cursor += ie_length;
            }
            _ => {
                debug!("Unknown IE ID: {}, skipping", ie_id);
                cursor += ie_length;
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
    use tracing::debug;

    let mut value_buf = BytesMut::new();

    value_buf.put_u8(0x00);

    let ie_count = if failure.time_to_wait.is_some() { 2usize } else { 1usize };
    value_buf.put_u16(ie_count as u16);
    debug!("NG Setup Failure IE count: {} encoded as 16-bit big-endian", ie_count);

    value_buf.put_u16(15);
    value_buf.put_u8(0x00);
    value_buf.put_u8(0x02);
    value_buf.put_u8(failure.cause.cause_type);
    value_buf.put_u8(failure.cause.cause_value);

    if let Some(ttw) = failure.time_to_wait {
        value_buf.put_u16(107);
        value_buf.put_u8(0x40);
        value_buf.put_u8(0x01);
        value_buf.put_u8(ttw);
    }

    encode_aper_length(value_buf.len(), buf);
    buf.put_slice(&value_buf);

    Ok(())
}

fn decode_initial_ue_message(data: &[u8]) -> Result<InitialUeMessage> {
    use tracing::debug;

    let mut cursor = 0;
    let mut ran_ue_ngap_id = None;
    let mut nas_pdu = Vec::new();
    let mut user_location_info = None;
    let mut rrc_establishment_cause = 0;

    if data.len() < 3 {
        return Err(anyhow!("Initial UE Message data too short"));
    }

    let extension_bit = data[cursor];
    cursor += 1;
    debug!("Initial UE Message extension bit: {:02x}", extension_bit);

    if cursor + 2 > data.len() {
        return Err(anyhow!("Not enough data for IE count"));
    }
    let ie_count = u16::from_be_bytes([data[cursor], data[cursor + 1]]) as usize;
    cursor += 2;
    debug!("Initial UE Message IE count: {}", ie_count);

    for i in 0..ie_count {
        if cursor + 3 > data.len() {
            debug!("Not enough data for IE {} header", i);
            break;
        }

        let ie_id = u16::from_be_bytes([data[cursor], data[cursor + 1]]) as usize;
        let ie_criticality = (data[cursor + 2] >> 6) & 0x03;
        cursor += 3;

        if cursor >= data.len() {
            debug!("No length field for IE {}", i);
            break;
        }

        let (ie_length, length_bytes) = decode_aper_length(&data[cursor..])?;
        cursor += length_bytes;

        debug!("Initial UE Message IE {}: ID={}, Length={}", i, ie_id, ie_length);

        if cursor + ie_length > data.len() {
            debug!("IE {} length exceeds data", i);
            break;
        }

        match ie_id {
            85 => {
                if ie_length >= 4 {
                    let id = u32::from_be_bytes([
                        data[cursor],
                        data[cursor + 1],
                        data[cursor + 2],
                        data[cursor + 3],
                    ]) as u64;
                    ran_ue_ngap_id = Some(id);
                    debug!("RAN-UE-NGAP-ID: {}", id);
                }
                cursor += ie_length;
            }
            38 => {
                nas_pdu = data[cursor..cursor + ie_length].to_vec();
                debug!("NAS-PDU length: {}", ie_length);
                cursor += ie_length;
            }
            121 => {
                if ie_length >= 9 {
                    let tai_plmn = decode_plmn_identity(&data[cursor..cursor + 3]);
                    let tac = format!("{:02x}{:02x}{:02x}",
                        data[cursor + 3], data[cursor + 4], data[cursor + 5]);

                    debug!("User Location Info: TAC={}, PLMN={}-{}", tac, tai_plmn.mcc, tai_plmn.mnc);

                    let nr_cgi = if ie_length >= 15 {
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
                cursor += ie_length;
            }
            90 => {
                if ie_length >= 1 {
                    rrc_establishment_cause = data[cursor];
                    debug!("RRC Establishment Cause: {}", rrc_establishment_cause);
                }
                cursor += ie_length;
            }
            _ => {
                debug!("Unknown IE ID: {}, skipping", ie_id);
                cursor += ie_length;
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

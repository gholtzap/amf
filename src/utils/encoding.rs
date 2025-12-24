use anyhow::Result;

pub fn encode_plmn(mcc: &str, mnc: &str) -> Result<Vec<u8>> {
    let mut bytes = Vec::new();

    if mcc.len() != 3 || (mnc.len() != 2 && mnc.len() != 3) {
        anyhow::bail!("Invalid MCC/MNC length");
    }

    let mcc_digits: Vec<u8> = mcc.chars()
        .map(|c| c.to_digit(10).unwrap() as u8)
        .collect();
    let mnc_digits: Vec<u8> = mnc.chars()
        .map(|c| c.to_digit(10).unwrap() as u8)
        .collect();

    bytes.push((mcc_digits[1] << 4) | mcc_digits[0]);

    if mnc.len() == 2 {
        bytes.push(0xF0 | mcc_digits[2]);
    } else {
        bytes.push((mnc_digits[2] << 4) | mcc_digits[2]);
    }

    bytes.push((mnc_digits[1] << 4) | mnc_digits[0]);

    Ok(bytes)
}

pub fn decode_plmn(bytes: &[u8]) -> Result<(String, String)> {
    if bytes.len() != 3 {
        anyhow::bail!("Invalid PLMN length");
    }

    let mcc = format!(
        "{}{}{}",
        bytes[0] & 0x0F,
        (bytes[0] >> 4) & 0x0F,
        bytes[1] & 0x0F
    );

    let mnc = if (bytes[1] >> 4) == 0x0F {
        format!(
            "{}{}",
            bytes[2] & 0x0F,
            (bytes[2] >> 4) & 0x0F
        )
    } else {
        format!(
            "{}{}{}",
            bytes[2] & 0x0F,
            (bytes[2] >> 4) & 0x0F,
            (bytes[1] >> 4) & 0x0F
        )
    };

    Ok((mcc, mnc))
}

pub fn encode_5g_s_tmsi(amf_set_id: u16, amf_pointer: u8, tmsi: u32) -> u64 {
    let mut result: u64 = 0;
    result |= (amf_set_id as u64) << 38;
    result |= (amf_pointer as u64) << 32;
    result |= tmsi as u64;
    result
}

pub fn decode_5g_s_tmsi(s_tmsi: u64) -> (u16, u8, u32) {
    let amf_set_id = ((s_tmsi >> 38) & 0x3FF) as u16;
    let amf_pointer = ((s_tmsi >> 32) & 0x3F) as u8;
    let tmsi = (s_tmsi & 0xFFFFFFFF) as u32;
    (amf_set_id, amf_pointer, tmsi)
}

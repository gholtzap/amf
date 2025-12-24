use anyhow::Result;
use sha2::{Sha256, Digest};
use hmac::{Hmac, Mac};

type HmacSha256 = Hmac<Sha256>;

pub fn kdf_hmac_sha256(key: &[u8], s: &[u8]) -> Result<Vec<u8>> {
    let mut mac = HmacSha256::new_from_slice(key)?;
    mac.update(s);
    Ok(mac.finalize().into_bytes().to_vec())
}

pub fn derive_key(key: &[u8], fc: u8, params: &[(&[u8], u16)]) -> Result<Vec<u8>> {
    let mut s = Vec::new();
    s.push(fc);

    for (param, length) in params {
        s.extend_from_slice(&length.to_be_bytes());
        s.extend_from_slice(param);
    }

    kdf_hmac_sha256(key, &s)
}

pub mod kdf;
pub mod crypto;

pub use kdf::*;
pub use crypto::*;

use anyhow::Result;

pub fn derive_kamf(kseaf: &[u8], supi: &str, abba: &[u8]) -> Result<Vec<u8>> {
    Ok(vec![0u8; 32])
}

pub fn derive_knas_enc(kamf: &[u8], algorithm_type: u8, algorithm_id: u8) -> Result<Vec<u8>> {
    Ok(vec![0u8; 32])
}

pub fn derive_knas_int(kamf: &[u8], algorithm_type: u8, algorithm_id: u8) -> Result<Vec<u8>> {
    Ok(vec![0u8; 32])
}

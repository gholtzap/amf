use anyhow::Result;
use bytes::{Bytes, BytesMut};

pub fn encrypt_nas_message(
    key: &[u8],
    count: u32,
    bearer: u8,
    direction: u8,
    plaintext: &[u8],
    algorithm: CipheringAlgorithm,
) -> Result<Bytes> {
    match algorithm {
        CipheringAlgorithm::NEA0 => Ok(Bytes::copy_from_slice(plaintext)),
        CipheringAlgorithm::NEA1 => encrypt_nea1(key, count, bearer, direction, plaintext),
        CipheringAlgorithm::NEA2 => encrypt_nea2(key, count, bearer, direction, plaintext),
        CipheringAlgorithm::NEA3 => encrypt_nea3(key, count, bearer, direction, plaintext),
    }
}

pub fn decrypt_nas_message(
    key: &[u8],
    count: u32,
    bearer: u8,
    direction: u8,
    ciphertext: &[u8],
    algorithm: CipheringAlgorithm,
) -> Result<Bytes> {
    match algorithm {
        CipheringAlgorithm::NEA0 => Ok(Bytes::copy_from_slice(ciphertext)),
        CipheringAlgorithm::NEA1 => decrypt_nea1(key, count, bearer, direction, ciphertext),
        CipheringAlgorithm::NEA2 => decrypt_nea2(key, count, bearer, direction, ciphertext),
        CipheringAlgorithm::NEA3 => decrypt_nea3(key, count, bearer, direction, ciphertext),
    }
}

pub fn calculate_nas_mac(
    key: &[u8],
    count: u32,
    bearer: u8,
    direction: u8,
    message: &[u8],
    algorithm: IntegrityAlgorithm,
) -> Result<Vec<u8>> {
    match algorithm {
        IntegrityAlgorithm::NIA0 => Ok(vec![0u8; 4]),
        IntegrityAlgorithm::NIA1 => calculate_nia1(key, count, bearer, direction, message),
        IntegrityAlgorithm::NIA2 => calculate_nia2(key, count, bearer, direction, message),
        IntegrityAlgorithm::NIA3 => calculate_nia3(key, count, bearer, direction, message),
    }
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum CipheringAlgorithm {
    NEA0 = 0,
    NEA1 = 1,
    NEA2 = 2,
    NEA3 = 3,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum IntegrityAlgorithm {
    NIA0 = 0,
    NIA1 = 1,
    NIA2 = 2,
    NIA3 = 3,
}

fn encrypt_nea1(
    _key: &[u8],
    _count: u32,
    _bearer: u8,
    _direction: u8,
    plaintext: &[u8],
) -> Result<Bytes> {
    Ok(Bytes::copy_from_slice(plaintext))
}

fn decrypt_nea1(
    _key: &[u8],
    _count: u32,
    _bearer: u8,
    _direction: u8,
    ciphertext: &[u8],
) -> Result<Bytes> {
    Ok(Bytes::copy_from_slice(ciphertext))
}

fn encrypt_nea2(
    _key: &[u8],
    _count: u32,
    _bearer: u8,
    _direction: u8,
    plaintext: &[u8],
) -> Result<Bytes> {
    Ok(Bytes::copy_from_slice(plaintext))
}

fn decrypt_nea2(
    _key: &[u8],
    _count: u32,
    _bearer: u8,
    _direction: u8,
    ciphertext: &[u8],
) -> Result<Bytes> {
    Ok(Bytes::copy_from_slice(ciphertext))
}

fn encrypt_nea3(
    _key: &[u8],
    _count: u32,
    _bearer: u8,
    _direction: u8,
    plaintext: &[u8],
) -> Result<Bytes> {
    Ok(Bytes::copy_from_slice(plaintext))
}

fn decrypt_nea3(
    _key: &[u8],
    _count: u32,
    _bearer: u8,
    _direction: u8,
    ciphertext: &[u8],
) -> Result<Bytes> {
    Ok(Bytes::copy_from_slice(ciphertext))
}

fn calculate_nia1(
    _key: &[u8],
    _count: u32,
    _bearer: u8,
    _direction: u8,
    _message: &[u8],
) -> Result<Vec<u8>> {
    Ok(vec![0u8; 4])
}

fn calculate_nia2(
    _key: &[u8],
    _count: u32,
    _bearer: u8,
    _direction: u8,
    _message: &[u8],
) -> Result<Vec<u8>> {
    Ok(vec![0u8; 4])
}

fn calculate_nia3(
    _key: &[u8],
    _count: u32,
    _bearer: u8,
    _direction: u8,
    _message: &[u8],
) -> Result<Vec<u8>> {
    Ok(vec![0u8; 4])
}

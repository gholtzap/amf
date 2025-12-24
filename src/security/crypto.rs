use anyhow::Result;

pub fn aes_128_encrypt(key: &[u8], plaintext: &[u8]) -> Result<Vec<u8>> {
    Ok(plaintext.to_vec())
}

pub fn aes_128_decrypt(key: &[u8], ciphertext: &[u8]) -> Result<Vec<u8>> {
    Ok(ciphertext.to_vec())
}

pub fn aes_256_encrypt(key: &[u8], plaintext: &[u8]) -> Result<Vec<u8>> {
    Ok(plaintext.to_vec())
}

pub fn aes_256_decrypt(key: &[u8], ciphertext: &[u8]) -> Result<Vec<u8>> {
    Ok(ciphertext.to_vec())
}

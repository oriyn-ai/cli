use anyhow::{Context, Result};
use std::env;

const SERVICE: &str = "bridge-cli";
const USER: &str = "default";

/// Resolve the API key: BRIDGE_API_KEY env var first, then OS keychain.
pub fn get_api_key() -> Result<String> {
    if let Ok(key) = env::var("BRIDGE_API_KEY") {
        if !key.is_empty() {
            return Ok(key);
        }
    }

    let entry =
        keyring::Entry::new(SERVICE, USER).context("failed to access OS keychain")?;

    entry
        .get_password()
        .context("not logged in — set BRIDGE_API_KEY or run `bridge login`")
}

/// Store an API key in the OS keychain.
pub fn store_api_key(key: &str) -> Result<()> {
    let entry =
        keyring::Entry::new(SERVICE, USER).context("failed to access OS keychain")?;

    entry
        .set_password(key)
        .context("failed to store credentials in OS keychain")
}

/// Delete the stored API key from the OS keychain.
pub fn delete_api_key() -> Result<()> {
    let entry =
        keyring::Entry::new(SERVICE, USER).context("failed to access OS keychain")?;

    // Ignore "no entry" errors — already logged out
    let _ = entry.delete_credential();
    Ok(())
}

use anyhow::Result;
use thiserror::Error;

const SERVICE: &str = "bridge-cli";
const USER: &str = "default";

#[derive(Debug, Error)]
pub enum LoginError {
    #[error("failed to store credentials in keychain: {0}")]
    KeychainStore(#[from] keyring::Error),
}

/// Run the login flow.
///
/// Currently a placeholder — will be replaced with a real OAuth flow.
pub async fn run() -> Result<()> {
    println!("Starting Bridge OAuth flow...");
    println!("(placeholder: in a future release this will open your browser)");

    // Simulate receiving a token from the OAuth callback
    let token = "placeholder-token";

    let entry = keyring::Entry::new(SERVICE, USER).map_err(LoginError::KeychainStore)?;
    entry
        .set_password(token)
        .map_err(LoginError::KeychainStore)?;

    println!("Credentials stored securely in your OS keychain.");
    Ok(())
}

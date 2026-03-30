use anyhow::Result;

use crate::auth;

/// Run the login flow.
///
/// Currently a placeholder — will be replaced with localhost callback + device flow.
pub async fn run() -> Result<()> {
    println!("Starting Bridge OAuth flow...");
    println!("(placeholder: in a future release this will open your browser)");

    // Simulate receiving a token from the OAuth callback
    let token = "placeholder-token";

    auth::store_api_key(token)?;

    println!("Credentials stored securely in your OS keychain.");
    Ok(())
}

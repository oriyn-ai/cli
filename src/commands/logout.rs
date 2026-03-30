use anyhow::Result;

use crate::auth;

/// Remove stored credentials from the OS keychain.
pub fn run() -> Result<()> {
    auth::delete_api_key()?;
    println!("Logged out.");
    Ok(())
}

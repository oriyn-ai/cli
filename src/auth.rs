use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::env;
use std::time::{SystemTime, UNIX_EPOCH};

const KEYRING_SERVICE: &str = "oriyn-cli";
const KEYRING_USER: &str = "credentials";

pub const SUPABASE_URL: &str = "https://ddykhzwjzbgpomlmkeji.supabase.co";
pub const SUPABASE_PUBLISHABLE_KEY: &str = "sb_publishable_FZtcboPlsEdA9tFS0bOWdQ_YBFpphBv";

#[derive(Debug, Serialize, Deserialize)]
pub struct Credentials {
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: i64,
}

/// Load credentials from the OS keychain.
pub fn load_credentials() -> Result<Credentials> {
    let entry =
        keyring::Entry::new(KEYRING_SERVICE, KEYRING_USER).context("failed to access OS keychain")?;

    let json = entry
        .get_password()
        .context("not logged in — run `oriyn login`")?;

    serde_json::from_str(&json).context("failed to parse stored credentials — run `oriyn login`")
}

/// Save credentials to the OS keychain.
pub fn save_credentials(creds: &Credentials) -> Result<()> {
    let entry =
        keyring::Entry::new(KEYRING_SERVICE, KEYRING_USER).context("failed to access OS keychain")?;

    let json = serde_json::to_string(creds).context("failed to serialize credentials")?;

    entry
        .set_password(&json)
        .context("failed to store credentials in OS keychain")
}

/// Delete credentials from the OS keychain.
pub fn delete_credentials() -> Result<()> {
    let entry =
        keyring::Entry::new(KEYRING_SERVICE, KEYRING_USER).context("failed to access OS keychain")?;

    // Ignore "no entry" errors — already logged out
    let _ = entry.delete_credential();
    Ok(())
}

fn unix_now() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("system clock before unix epoch")
        .as_secs() as i64
}

/// Get a valid access token, refreshing automatically if expired.
///
/// Checks `ORIYN_ACCESS_TOKEN` env var first (escape hatch for CI/Docker).
/// Otherwise loads from keychain and refreshes if within 60 seconds of expiry.
pub async fn get_valid_access_token() -> Result<String> {
    if let Ok(token) = env::var("ORIYN_ACCESS_TOKEN") {
        if !token.is_empty() {
            return Ok(token);
        }
    }

    let mut creds = load_credentials()?;

    if creds.expires_at - unix_now() > 60 {
        return Ok(creds.access_token);
    }

    // Token expired or expiring soon — refresh
    let new_creds = refresh_token(&creds.refresh_token).await?;
    creds.access_token = new_creds.access_token;
    creds.refresh_token = new_creds.refresh_token;
    creds.expires_at = new_creds.expires_at;
    save_credentials(&creds)?;

    Ok(creds.access_token)
}

#[derive(Deserialize)]
struct RefreshResponse {
    access_token: String,
    refresh_token: String,
    expires_in: i64,
}

async fn refresh_token(refresh_token: &str) -> Result<Credentials> {
    let url = format!("{SUPABASE_URL}/auth/v1/token?grant_type=refresh_token");

    let resp = reqwest::Client::new()
        .post(&url)
        .header("apikey", SUPABASE_PUBLISHABLE_KEY)
        .json(&serde_json::json!({ "refresh_token": refresh_token }))
        .send()
        .await
        .context("failed to reach Supabase for token refresh")?;

    if !resp.status().is_success() {
        let status = resp.status();
        let body = resp.text().await.unwrap_or_default();
        // Refresh token is invalid or revoked — clear credentials
        let _ = delete_credentials();
        anyhow::bail!("session expired ({status}) — run `oriyn login` again: {body}");
    }

    let data: RefreshResponse = resp
        .json()
        .await
        .context("failed to parse token refresh response")?;

    Ok(Credentials {
        access_token: data.access_token,
        refresh_token: data.refresh_token,
        expires_at: unix_now() + data.expires_in,
    })
}

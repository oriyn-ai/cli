use anyhow::{Context, Result};
use serde::Deserialize;

use crate::auth;

#[derive(Deserialize)]
struct MeResponse {
    user_id: String,
}

/// Show the currently authenticated user.
pub async fn run(api_base: &str) -> Result<()> {
    let key = match auth::get_api_key() {
        Ok(key) => key,
        Err(_) => {
            println!("Not logged in. Run `oriyn login` or set ORIYN_API_KEY.");
            return Ok(());
        }
    };

    let client = reqwest::Client::new();
    let resp = client
        .get(format!("{api_base}/v1/me"))
        .bearer_auth(&key)
        .send()
        .await
        .context("failed to reach the Oriyn API")?;

    if !resp.status().is_success() {
        let status = resp.status();
        let body = resp.text().await.unwrap_or_default();
        anyhow::bail!("API returned {status}: {body}");
    }

    let me: MeResponse = resp
        .json()
        .await
        .context("failed to parse response")?;

    println!("Logged in as user {}", me.user_id);
    Ok(())
}

use std::sync::Arc;
use std::time::Duration;

use anyhow::{Context, Result};
use axum::extract::{Query, State};
use axum::response::Html;
use axum::routing::get;
use axum::Router;
use serde::{Deserialize, Serialize};
use tokio::sync::{oneshot, Mutex};

use crate::auth;
use crate::telemetry;

// --- Localhost callback types ---

struct CallbackState {
    expected_state: String,
    tx: Mutex<Option<oneshot::Sender<CallbackResult>>>,
}

struct CallbackResult {
    key: String,
    email: String,
}

#[derive(Deserialize)]
struct CallbackParams {
    key: String,
    state: String,
    email: Option<String>,
}

// --- Device flow types ---

#[derive(Deserialize)]
struct DeviceCodeResponse {
    device_code: String,
    user_code: String,
    verification_uri: String,
    interval: u64,
    expires_in: u64,
}

#[derive(Serialize)]
struct DeviceTokenRequest {
    device_code: String,
}

#[derive(Deserialize)]
struct DeviceTokenResponse {
    key: String,
    email: String,
}

/// Run the login flow.
pub async fn run(web_base: &str, api_base: &str, device: bool) -> Result<()> {
    if device {
        device_flow(web_base, api_base).await
    } else {
        localhost_flow(web_base, api_base).await
    }
}

/// Fetch and store user_id for telemetry after successful login.
async fn store_user_id_for_telemetry(api_base: &str, api_key: &str) {
    #[derive(serde::Deserialize)]
    struct MeResponse {
        user_id: String,
    }
    let Ok(resp) = reqwest::Client::new()
        .get(format!("{api_base}/v1/me"))
        .bearer_auth(api_key)
        .send()
        .await
    else {
        return;
    };
    if let Ok(me) = resp.json::<MeResponse>().await {
        telemetry::store_user_id(&me.user_id);
    }
}

/// Localhost callback flow: start local server, open browser, wait for redirect back.
async fn localhost_flow(web_base: &str, api_base: &str) -> Result<()> {
    let (tx, rx) = oneshot::channel::<CallbackResult>();

    let state_param = uuid::Uuid::new_v4().to_string();

    let app_state = Arc::new(CallbackState {
        expected_state: state_param.clone(),
        tx: Mutex::new(Some(tx)),
    });

    let app = Router::new()
        .route("/callback", get(callback_handler))
        .with_state(app_state);

    let listener = tokio::net::TcpListener::bind("127.0.0.1:0")
        .await
        .context("failed to bind local server")?;

    let port = listener.local_addr()?.port();

    let server = tokio::spawn(async move {
        axum::serve(listener, app).await.ok();
    });

    let url = format!("{web_base}/auth/cli/authorize?port={port}&state={state_param}");

    if open::that(&url).is_err() {
        println!("Open this URL in your browser:\n\n  {url}\n");
    } else {
        println!("Opening browser to log in...");
    }
    println!("Waiting for authentication...");

    let result = tokio::time::timeout(Duration::from_secs(120), rx).await;

    server.abort();

    match result {
        Ok(Ok(callback)) => {
            auth::store_api_key(&callback.key)?;
            store_user_id_for_telemetry(api_base, &callback.key).await;
            println!("Logged in as {}", callback.email);
            Ok(())
        }
        Ok(Err(_)) => anyhow::bail!("login cancelled"),
        Err(_) => anyhow::bail!("login timed out after 120 seconds — try `oriyn login --device` for headless environments"),
    }
}

async fn callback_handler(
    Query(params): Query<CallbackParams>,
    State(state): State<Arc<CallbackState>>,
) -> Html<String> {
    if params.state != state.expected_state {
        return Html(
            "<html><body><h1>Error</h1><p>State mismatch — possible CSRF. Please try again.</p></body></html>"
                .to_string(),
        );
    }

    let mut tx = state.tx.lock().await;
    if let Some(tx) = tx.take() {
        let _ = tx.send(CallbackResult {
            key: params.key,
            email: params.email.unwrap_or_default(),
        });
    }

    Html(
        "<html><body style=\"font-family:system-ui;text-align:center;padding:4rem\">\
         <h1>Logged in!</h1>\
         <p>You can close this window and return to your terminal.</p>\
         </body></html>"
            .to_string(),
    )
}

/// Device code flow: generate code, user enters it on any browser, CLI polls for approval.
async fn device_flow(web_base: &str, api_base: &str) -> Result<()> {
    let client = reqwest::Client::new();

    // Step 1: Request device code
    let resp = client
        .post(format!("{web_base}/auth/device/code"))
        .send()
        .await
        .context("failed to request device code")?;

    if !resp.status().is_success() {
        let status = resp.status();
        let body = resp.text().await.unwrap_or_default();
        anyhow::bail!("failed to get device code ({status}): {body}");
    }

    let device: DeviceCodeResponse = resp
        .json()
        .await
        .context("failed to parse device code response")?;

    println!();
    println!("To authenticate, visit: {}", device.verification_uri);
    println!();
    println!("  Enter code: {}", device.user_code);
    println!();

    // Try to open browser (non-blocking, ignore failure)
    let _ = open::that(&device.verification_uri);

    // Step 2: Poll for approval
    let deadline =
        tokio::time::Instant::now() + Duration::from_secs(device.expires_in);
    let interval = Duration::from_secs(device.interval);

    loop {
        tokio::time::sleep(interval).await;

        if tokio::time::Instant::now() > deadline {
            anyhow::bail!("device code expired — run `oriyn login --device` again");
        }

        let resp = client
            .post(format!("{web_base}/auth/device/token"))
            .json(&DeviceTokenRequest {
                device_code: device.device_code.clone(),
            })
            .send()
            .await
            .context("failed to poll for token")?;

        match resp.status().as_u16() {
            200 => {
                let token: DeviceTokenResponse = resp
                    .json()
                    .await
                    .context("failed to parse token response")?;

                auth::store_api_key(&token.key)?;
                store_user_id_for_telemetry(api_base, &token.key).await;
                println!("Logged in as {}", token.email);
                return Ok(());
            }
            403 => continue,
            410 => anyhow::bail!("device code expired — run `oriyn login --device` again"),
            status => {
                let body = resp.text().await.unwrap_or_default();
                anyhow::bail!("unexpected response ({status}): {body}");
            }
        }
    }
}

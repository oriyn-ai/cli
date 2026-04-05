use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use anyhow::{Context, Result};
use axum::extract::{Query, State};
use axum::response::Html;
use axum::routing::get;
use axum::Router;
use serde::Deserialize;
use tokio::sync::{oneshot, Mutex};

use crate::auth::{self, Credentials};
use crate::telemetry;

// --- Localhost callback types ---

struct CallbackState {
    expected_state: String,
    tx: Mutex<Option<oneshot::Sender<CallbackResult>>>,
}

struct CallbackResult {
    access_token: String,
    refresh_token: String,
    expires_in: i64,
}

#[derive(Deserialize)]
struct CallbackParams {
    access_token: String,
    refresh_token: String,
    expires_in: i64,
    state: String,
}

/// Run the login flow.
pub async fn run(web_base: &str, api_base: &str) -> Result<()> {
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

    let url = format!("{web_base}/auth/cli/login?port={port}&state={state_param}");

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
            let now = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .expect("system clock before unix epoch")
                .as_secs() as i64;

            let creds = Credentials {
                access_token: callback.access_token,
                refresh_token: callback.refresh_token,
                expires_at: now + callback.expires_in,
            };

            auth::save_credentials(&creds)?;

            // Fetch user info for display and telemetry
            let email = fetch_user_email(api_base, &creds.access_token).await;
            if let Some(ref user_id) = fetch_user_id(api_base, &creds.access_token).await {
                telemetry::store_user_id(user_id);
            }

            match email {
                Some(email) => println!("Logged in as {email}"),
                None => println!("Logged in successfully."),
            }

            Ok(())
        }
        Ok(Err(_)) => anyhow::bail!("login cancelled"),
        Err(_) => anyhow::bail!(
            "login timed out after 120 seconds — please try again"
        ),
    }
}

#[derive(Deserialize)]
struct MeResponse {
    user_id: String,
    email: Option<String>,
}

async fn fetch_user_email(api_base: &str, token: &str) -> Option<String> {
    let resp = reqwest::Client::new()
        .get(format!("{api_base}/v1/me"))
        .bearer_auth(token)
        .send()
        .await
        .ok()?;
    let me: MeResponse = resp.json().await.ok()?;
    me.email
}

async fn fetch_user_id(api_base: &str, token: &str) -> Option<String> {
    let resp = reqwest::Client::new()
        .get(format!("{api_base}/v1/me"))
        .bearer_auth(token)
        .send()
        .await
        .ok()?;
    let me: MeResponse = resp.json().await.ok()?;
    Some(me.user_id)
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
            access_token: params.access_token,
            refresh_token: params.refresh_token,
            expires_in: params.expires_in,
        });
    }

    Html(
        "<html><body style=\"font-family:system-ui;text-align:center;padding:4rem\">\
         <h1>Authentication successful</h1>\
         <p>You may close this page and return to your terminal.</p>\
         </body></html>"
            .to_string(),
    )
}

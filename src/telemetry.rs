use std::env;
use std::fs;
use std::path::PathBuf;

use posthog_rs::{ClientOptionsBuilder, Event};

const POSTHOG_API_KEY: &str = "phc_RpuEAMMomACJxc7hG4mRMKURklt2BXtfzzwYQYlzr0W";

pub struct Telemetry {
    client: posthog_rs::Client,
    distinct_id: Option<String>,
}

impl Telemetry {
    pub async fn new() -> Self {
        let disabled = cfg!(debug_assertions)
            || matches!(
                env::var("ORIYN_TELEMETRY").as_deref(),
                Ok("0" | "false" | "off")
            )
            || config_dir().join("telemetry-disabled").exists();

        let options = ClientOptionsBuilder::default()
            .api_key(POSTHOG_API_KEY.to_string())
            .disabled(disabled)
            .build()
            .expect("valid posthog options");

        let client = posthog_rs::client(options).await;
        let distinct_id = load_user_id().or_else(load_or_create_anonymous_id);

        Self { client, distinct_id }
    }

    pub async fn capture(&self, event_name: &str, props: serde_json::Value) {
        let Some(ref did) = self.distinct_id else {
            return;
        };
        let mut event = Event::new(event_name, did);
        event.insert_prop("$lib", "oriyn-cli").ok();
        event
            .insert_prop("cli_version", env!("CARGO_PKG_VERSION"))
            .ok();
        event.insert_prop("$os", env::consts::OS).ok();
        if let Some(obj) = props.as_object() {
            for (k, v) in obj {
                event.insert_prop(k, v).ok();
            }
        }
        let _ = self.client.capture(event).await;
    }
}

fn config_dir() -> PathBuf {
    let home = env::var("HOME").unwrap_or_else(|_| ".".to_string());
    PathBuf::from(home).join(".config").join("oriyn")
}

pub fn get_user_id() -> Option<String> {
    load_user_id()
}

fn load_user_id() -> Option<String> {
    fs::read_to_string(config_dir().join("user-id"))
        .ok()
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty())
}

fn load_or_create_anonymous_id() -> Option<String> {
    let path = config_dir().join("anonymous-id");
    if let Ok(id) = fs::read_to_string(&path) {
        let id = id.trim().to_string();
        if !id.is_empty() {
            return Some(id);
        }
    }
    let id = uuid::Uuid::new_v4().to_string();
    fs::create_dir_all(config_dir()).ok()?;
    fs::write(&path, &id).ok()?;
    Some(id)
}

pub fn store_user_id(user_id: &str) {
    let dir = config_dir();
    let _ = fs::create_dir_all(&dir);
    let _ = fs::write(dir.join("user-id"), user_id);
}

pub fn clear_user_id() {
    let _ = fs::remove_file(config_dir().join("user-id"));
}

/// Manage telemetry opt-in/out.
pub fn manage(disable: bool, enable: bool, status: bool) {
    let flag_path = config_dir().join("telemetry-disabled");

    if disable {
        let _ = fs::create_dir_all(config_dir());
        let _ = fs::write(&flag_path, "");
        println!("Telemetry disabled.");
    } else if enable {
        let _ = fs::remove_file(&flag_path);
        println!("Telemetry enabled.");
    } else if status {
        let disabled = cfg!(debug_assertions)
            || matches!(
                env::var("ORIYN_TELEMETRY").as_deref(),
                Ok("0" | "false" | "off")
            )
            || flag_path.exists();
        if disabled {
            println!("Telemetry: disabled");
        } else {
            println!("Telemetry: enabled");
        }
    }
}

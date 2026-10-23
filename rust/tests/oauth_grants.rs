use platform_contracts::businessv1::{OAuthGrantRevoked,OAuthGrantsResponse};
use serde::Serialize;
use serde_json::{json,Value};

#[derive(Serialize)]
struct ProducerGrant<'a>{client_id:&'a str,client_name:&'a str,scopes:Vec<&'a str>,trusted_first_party:bool,granted_at:&'a str,last_used_at:&'a str}
#[derive(Serialize)]
struct ProducerPage<'a>{items:Vec<ProducerGrant<'a>>,next_cursor:Option<&'a str>}
#[test]
fn serialized_own_grant_page_has_only_safe_exact_fields(){
 let producer=ProducerPage{items:vec![ProducerGrant{client_id:"client-one",client_name:"Account client",scopes:vec!["openid","email","profile","offline_access"],trusted_first_party:false,granted_at:"2026-10-24T12:00:00Z",last_used_at:"2026-10-24T12:00:00Z"}],next_cursor:Some("Y2xpZW50LW9uZQ")};
 let raw=serde_json::to_vec(&producer).unwrap();let got:OAuthGrantsResponse=serde_json::from_slice(&raw).unwrap();assert_eq!(got.items.len(),1);assert_eq!(got.items[0].client_id,"client-one");assert_eq!(got.next_cursor.as_deref(),Some("Y2xpZW50LW9uZQ"));
 for field in ["access_token","refresh_token","client_secret"]{let mut bad:Value=serde_json::from_slice(&raw).unwrap();bad["items"][0][field]=json!("secret");assert!(serde_json::from_value::<OAuthGrantsResponse>(bad).is_err());}
 let mut bad:Value=serde_json::from_slice(&raw).unwrap();bad["items"][0]["last_used_at"]=Value::Null;assert!(serde_json::from_value::<OAuthGrantsResponse>(bad).is_err());
 let mut bad:Value=serde_json::from_slice(&raw).unwrap();bad["items"][0]["scopes"]=json!(["admin"]);assert!(serde_json::from_value::<OAuthGrantsResponse>(bad).is_err());
 let mut bad:Value=serde_json::from_slice(&raw).unwrap();bad["items"]=Value::Array(vec![bad["items"][0].clone();101]);assert!(serde_json::from_value::<OAuthGrantsResponse>(bad).is_err());
}
#[test]
fn own_grant_revocation_proof_is_strict_and_idempotent_shape(){
 let bytes=serde_json::to_vec(&json!({"client_id":"client-one","revoked":true})).unwrap();let result:OAuthGrantRevoked=serde_json::from_slice(&bytes).unwrap();assert!(result.revoked);assert_eq!(result.client_id,"client-one");
 assert!(serde_json::from_value::<OAuthGrantRevoked>(json!({"client_id":"client-one","revoked":false})).is_err());
 assert!(serde_json::from_value::<OAuthGrantRevoked>(json!({"client_id":"client-one","revoked":true,"client_secret":"secret"})).is_err());
}

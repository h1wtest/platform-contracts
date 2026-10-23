use platform_contracts::businessv1::{AuditResponse,CommandStatus};
use serde::Serialize;
use serde_json::{json,Value};

#[derive(Serialize)]
struct ProducerAuditRecord<'a> {
 id:&'a str,event_type:&'a str,actor_id:Option<&'a str>,user_id:Option<&'a str>,
 #[serde(skip_serializing_if="Option::is_none")] aggregate_type:Option<&'a str>,
 #[serde(skip_serializing_if="Option::is_none")] aggregate_id:Option<&'a str>,
 #[serde(skip_serializing_if="Option::is_none")] aggregate_version:Option<i64>,
 #[serde(skip_serializing_if="Option::is_none")] producer:Option<&'a str>,
 created_at:&'a str,
 #[serde(skip_serializing_if="Option::is_none")] reason:Option<&'a str>,
 #[serde(skip_serializing_if="Option::is_none")] operation:Option<&'a str>,
 #[serde(skip_serializing_if="Option::is_none")] target_type:Option<&'a str>,
 #[serde(skip_serializing_if="Option::is_none")] target_id:Option<&'a str>,
}
#[derive(Serialize)]
struct ProducerAuditPage<'a>{items:Vec<ProducerAuditRecord<'a>>,next_cursor:Option<&'a str>}
#[test]
fn serialized_owner_audit_page_decodes_without_invented_envelope_fields(){
 let id="019a13cf-0000-7000-8000-000000000002";
 let producer=ProducerAuditPage{items:vec![
  ProducerAuditRecord{id,event_type:"identity.user_registered",actor_id:None,user_id:Some(id),aggregate_type:None,aggregate_id:None,aggregate_version:None,producer:None,created_at:"2026-10-24T12:00:00Z",reason:None,operation:None,target_type:None,target_id:None},
  ProducerAuditRecord{id,event_type:"identity.admin_action_recorded",actor_id:Some(id),user_id:None,aggregate_type:Some("identity_admin_action"),aggregate_id:Some(id),aggregate_version:Some(1),producer:Some("identity"),created_at:"2026-10-24T12:00:00Z",reason:Some("narrow access"),operation:Some("role.permissions"),target_type:Some("role"),target_id:Some("support")},
 ],next_cursor:Some("opaque-cursor")};
 let bytes=serde_json::to_vec(&producer).unwrap();let page:AuditResponse=serde_json::from_slice(&bytes).unwrap();
 assert_eq!(page.items.len(),2);assert_eq!(page.items[0].id.to_string(),id);assert!(page.items[0].actor_id.is_none());assert_eq!(page.items[1].target_id.as_deref(),Some("support"));assert_eq!(page.next_cursor.as_deref(),Some("opaque-cursor"));
 assert!(serde_json::from_str::<AuditResponse>("{\"items\":[],\"next_cursor\":null}").is_ok());
 for invalid in [json!([]),json!({"items":[]}),json!({"items":[],"next_cursor":null,"raw_payload":"secret"})]{assert!(serde_json::from_value::<AuditResponse>(invalid).is_err());}
 let mut unsafe_page:Value=serde_json::from_slice(&bytes).unwrap();unsafe_page["items"][0]["raw_payload"]=json!("secret");assert!(serde_json::from_value::<AuditResponse>(unsafe_page).is_err());
}
#[test]
fn serialized_public_command_projection_needs_no_private_journal_fields(){
 for status in ["requested","unknown","denied","succeeded"]{
  let mut producer=json!({"command_id":"019a13cf-0000-7000-8000-000000000002","status":status});
  if status=="succeeded"{producer["result"]=json!({"revoked":2});}
  let bytes=serde_json::to_vec(&producer).unwrap();let response:CommandStatus=serde_json::from_slice(&bytes).unwrap();assert_eq!(response.status,status);
 }
 let invalid=json!({"command_id":"019a13cf-0000-7000-8000-000000000002","status":"succeeded","result":{"provider_secret":"secret"}});assert!(serde_json::from_value::<CommandStatus>(invalid).is_err());
}

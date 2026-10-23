package businessv1

import (
 "encoding/json"
 "testing"
 "time"
)

// This projection models the existing audit owner's public JSON (AuditEvent and
// paginated HTTP response), independently of the generated consumer DTO.
type auditProducerRecord struct {
 ID string `json:"id"`
 EventType string `json:"event_type"`
 ActorID *string `json:"actor_id"`
 UserID *string `json:"user_id"`
 AggregateType *string `json:"aggregate_type,omitempty"`
 AggregateID *string `json:"aggregate_id,omitempty"`
 AggregateVersion *int64 `json:"aggregate_version,omitempty"`
 Producer *string `json:"producer,omitempty"`
 CreatedAt time.Time `json:"created_at"`
 Reason *string `json:"reason,omitempty"`
 Operation *string `json:"operation,omitempty"`
 TargetType *string `json:"target_type,omitempty"`
 TargetID *string `json:"target_id,omitempty"`
}
func TestAuditOwnerSerializedPageDecodesStrictly(t *testing.T) {
 id:="019a13cf-0000-7000-8000-000000000002";actor:="019a13cf-0000-7000-8000-000000000003"
 aggregateType:="identity_admin_action";producer:="identity";operation:="role.permissions";targetType:="role";targetID:="support";reason:="narrow support access";version:=int64(1)
 created:=time.Date(2026,10,24,12,0,0,0,time.UTC);cursor:="opaque-next-cursor"
 produced:=struct{Items []auditProducerRecord `json:"items"`;NextCursor *string `json:"next_cursor"`}{Items:[]auditProducerRecord{
  {ID:id,EventType:"identity.user_registered",UserID:&id,CreatedAt:created},
  {ID:actor,EventType:"identity.admin_action_recorded",ActorID:&actor,AggregateType:&aggregateType,AggregateID:&actor,AggregateVersion:&version,Producer:&producer,CreatedAt:created,Reason:&reason,Operation:&operation,TargetType:&targetType,TargetID:&targetID},
 },NextCursor:&cursor}
 raw,err:=json.Marshal(produced);if err!=nil {t.Fatal(err)}
 var page AuditResponse;if err=json.Unmarshal(raw,&page);err!=nil {t.Fatalf("owner page rejected: %s: %v",raw,err)}
 if len(page.Items)!=2||page.Items[0].ID!=id||page.Items[0].ActorID!=nil||page.Items[1].TargetID==nil||*page.Items[1].TargetID!="support"||page.NextCursor==nil||*page.NextCursor!=cursor {t.Fatalf("lost audit projection: %+v",page)}
 if err=json.Unmarshal([]byte(`{"items":[],"next_cursor":null}`),&page);err!=nil {t.Fatal(err)}
 for _,bad:=range []string{`[]`,`{"items":[]}`,`{"items":[],"next_cursor":null,"payload":{"secret":"no"}}`,`{"items":[{"id":"019a13cf-0000-7000-8000-000000000002","event_type":"identity.user_registered","actor_id":null,"user_id":null,"created_at":"2026-10-24T12:00:00Z","raw_payload":"secret"}],"next_cursor":null}`} {if json.Unmarshal([]byte(bad),&page)==nil {t.Fatalf("accepted wrong audit wire: %s",bad)}}
}
func TestActualCommandStatusProjectionNeedsNoJournalFields(t *testing.T) {
 id:="019a13cf-0000-7000-8000-000000000002";user:="019a13cf-0000-7000-8000-000000000003";banned:=true
 for _,status:=range []string{"requested","unknown","denied","succeeded"} {
  produced:=struct{CommandID string `json:"command_id"`;Status string `json:"status"`;Result *struct{UserID string `json:"user_id"`;IsBanned bool `json:"is_banned"`} `json:"result,omitempty"`}{CommandID:id,Status:status}
  if status=="succeeded" {produced.Result=&struct{UserID string `json:"user_id"`;IsBanned bool `json:"is_banned"`}{user,banned}}
  raw,err:=json.Marshal(produced);if err!=nil {t.Fatal(err)};var got CommandStatus;if err=json.Unmarshal(raw,&got);err!=nil {t.Fatalf("actual command projection rejected: %s: %v",raw,err)};if got.CommandID!=id||got.Status!=status {t.Fatalf("lost command result: %+v",got)}
 }
 var got CommandStatus
 for _,bad:=range []string{`{"command_id":"019a13cf-0000-7000-8000-000000000002","status":"succeeded","result":{"provider_secret":"no"}}`,`{"command_id":"019a13cf-0000-7000-8000-000000000002","status":"succeeded","action":"ban","reason":"journal only"}`} {if json.Unmarshal([]byte(bad),&got)==nil {t.Fatal("accepted private journal fields",bad)}}
}

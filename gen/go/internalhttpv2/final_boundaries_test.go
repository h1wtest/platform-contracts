package internalhttpv2

import (
 "encoding/json"
 "strings"
 "testing"
)

func TestImmutableSubscriptionOutcomeProducerBytes(t *testing.T) {
 raw := `{"action_id":"019a13cf-0000-7000-8000-000000000001","order_id":"019a13cf-0000-7000-8000-000000000002","user_id":"019a13cf-0000-7000-8000-000000000003","state":"APPLIED","subscription_id":"019a13cf-0000-7000-8000-000000000004","subscription_version":9223372036854775807,"period_start":"2026-10-24T00:00:00Z","period_end":"2026-11-24T00:00:00Z","reason":null}`
 var value SubscriptionActionOutcome
 if err := json.Unmarshal([]byte(raw), &value); err != nil { t.Fatal(err) }
 encoded,err := json.Marshal(value); if err != nil {t.Fatal(err)}
 if !strings.Contains(string(encoded), `"subscription_version":9223372036854775807`) {t.Fatal("lost exact outcome version")}
 for _,bad := range []string{strings.Replace(raw,`"reason":null`, `"reason":null,"extra":true`,1),strings.Replace(raw,`"reason":null`, `"reason":null,"state":"PENDING"`,1),strings.Replace(raw,`,"reason":null`, ``,1)} {
  if json.Unmarshal([]byte(bad),&value)==nil {t.Fatal("accepted invalid outcome",bad)}
 }
 rejected:=`{"action_id":"019a13cf-0000-7000-8000-000000000001","order_id":"019a13cf-0000-7000-8000-000000000002","user_id":"019a13cf-0000-7000-8000-000000000003","state":"REJECTED","subscription_id":null,"subscription_version":null,"period_start":null,"period_end":null,"reason":"conflict"}`
 if err:=json.Unmarshal([]byte(rejected),&value);err!=nil {t.Fatal(err)}
}

func TestTemplatePreviewProducerBytes(t *testing.T) {
 raw:=`{"locales":{"en":{"subject":"Preview","body":"{{reset_url}}"},"ru":{"subject":"Просмотр","body":"{{reset_url}}"}}}`
 var request TemplatePreviewRequest
 if err:=json.Unmarshal([]byte(raw),&request);err!=nil {t.Fatal(err)}
 var response TemplatePreviewResponse
 if err:=json.Unmarshal([]byte(raw),&response);err!=nil {t.Fatal(err)}
 for _,bad:=range []string{`{"locales":null}`,strings.Replace(raw,`"locales":`, `"extra":true,"locales":`,1),`{"locales":{"en":{"subject":"Preview","body":"x"}}}`} {
  if json.Unmarshal([]byte(bad),&request)==nil {t.Fatal("accepted invalid preview",bad)}
 }
}

func TestPrepareProducerBytesOmitAbsentNonnullableFields(t *testing.T) {
 id:="019a13cf-0000-7000-8000-000000000001"
 var offer OfferSnapshot
 raw:=`{"product_id":"`+id+`","plan_id":"`+id+`","plan_key":"pro","upgrade_rank":1,"revision_id":"`+id+`","offer_id":"`+id+`","duration_days":30,"price":"1.000000","currency":"USDT","name":{"en":"Pro","ru":"Про"},"features":[],"snapshot_hash":"`+strings.Repeat("a",64)+`"}`
 if err:=json.Unmarshal([]byte(raw),&offer);err!=nil {t.Fatal(err)}
 version:=int64(1)
 for _,kind:=range []string{"INITIAL","RENEW","UPGRADE"} {
  producer:=PrepareActionRequest{OperationID:id,UserID:id,Kind:kind,OfferSnapshot:offer}
  if kind!="INITIAL" {producer.SubscriptionID=&id;producer.ExpectedVersion=&version}
  bytes,err:=json.Marshal(producer);if err!=nil {t.Fatal(err)}
  var decoded PrepareActionRequest
  if err=json.Unmarshal(bytes,&decoded);err!=nil {t.Fatal(err)}
  if kind=="INITIAL"&&(strings.Contains(string(bytes),"subscription_id")||strings.Contains(string(bytes),"expected_version")) {t.Fatal("optional fields not omitted",string(bytes))}
  for _,field:=range []string{"subscription_id","expected_version","unknown"} {
   bad:=strings.TrimSuffix(string(bytes),"}")+`,"`+field+`":null}`
   if json.Unmarshal([]byte(bad),&decoded)==nil {t.Fatal("accepted null/unknown",bad)}
  }
 }
}

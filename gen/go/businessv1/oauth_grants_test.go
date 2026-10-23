package businessv1

import (
 "encoding/json"
 "testing"
 "time"
)

func TestSerializedOwnOAuthGrantPageStrictBoundary(t *testing.T) {
 type producerGrant struct {ClientID string `json:"client_id"`;ClientName string `json:"client_name"`;Scopes []string `json:"scopes"`;TrustedFirstParty bool `json:"trusted_first_party"`;GrantedAt time.Time `json:"granted_at"`;LastUsedAt time.Time `json:"last_used_at"`}
 cursor:="Y2xpZW50LW9uZQ";at:=time.Date(2026,10,24,12,0,0,0,time.UTC)
 produced:=struct {Items []producerGrant `json:"items"`;NextCursor *string `json:"next_cursor"`}{[]producerGrant{{"client-one","Account client",[]string{"openid","email","profile","offline_access"},false,at,at}},&cursor}
 raw,err:=json.Marshal(produced);if err!=nil {t.Fatal(err)}
 var got OAuthGrantsResponse;if err=json.Unmarshal(raw,&got);err!=nil {t.Fatal(err)}
 if len(got.Items)!=1||got.Items[0].ClientID!="client-one"||got.Items[0].TrustedFirstParty||got.NextCursor==nil||*got.NextCursor!=cursor||!got.Items[0].LastUsedAt.Equal(at) {t.Fatalf("lost grant projection %+v",got)}
 for _,invalid:=range []string{`{"items":[],"next_cursor":null,"access_token":"secret"}`,`{"items":[{"client_id":"one","client_name":"Client","scopes":["admin"],"trusted_first_party":false,"granted_at":"2026-10-24T12:00:00Z","last_used_at":"2026-10-24T12:00:00Z"}],"next_cursor":null}`,`{"items":[{"client_id":"one","client_name":"Client","scopes":["openid"],"trusted_first_party":false,"granted_at":"2026-10-24T12:00:00Z","last_used_at":null}],"next_cursor":null}`} {if json.Unmarshal([]byte(invalid),&got)==nil {t.Fatal("accepted invalid grant wire",invalid)}}
 produced.Items=make([]producerGrant,101);for i:=range produced.Items {produced.Items[i]=producerGrant{"client-one","Client",[]string{"openid"},true,at,at}}
 raw,err=json.Marshal(produced);if err!=nil {t.Fatal(err)};if json.Unmarshal(raw,&got)==nil {t.Fatal("accepted more than100 grants")}
}
func TestSerializedOwnOAuthGrantRevocationStrictBoundary(t *testing.T) {
 produced:=struct {ClientID string `json:"client_id"`;Revoked bool `json:"revoked"`}{"client-one",true}
 raw,err:=json.Marshal(produced);if err!=nil {t.Fatal(err)};var got OAuthGrantRevoked;if err=json.Unmarshal(raw,&got);err!=nil {t.Fatal(err)}
 if got.ClientID!="client-one"||!got.Revoked {t.Fatalf("lost revoke proof %+v",got)}
 for _,invalid:=range []string{`{"client_id":"client-one","revoked":false}`,`{"client_id":"client-one","revoked":true,"client_secret":"secret"}`} {if json.Unmarshal([]byte(invalid),&got)==nil {t.Fatal("accepted invalid revoke proof",invalid)}}
}

import test from 'node:test';
import assert from 'node:assert/strict';
import {pathToFileURL} from 'node:url';
import {resolve} from 'node:path';

// Parent compiles gen/typescript with pinned frontend tsc to this directory first.
if (!process.env.CONTRACTS_TS_DIR) throw new Error('CONTRACTS_TS_DIR required: compiled generated TypeScript output');
const event = await import(pathToFileURL(resolve(process.env.CONTRACTS_TS_DIR, 'eventsv2.js')).href);
const business = await import(pathToFileURL(resolve(process.env.CONTRACTS_TS_DIR, 'businessv1.js')).href);
const wire = await import(pathToFileURL(resolve(process.env.CONTRACTS_TS_DIR, 'validation.js')).href);
const envelope = {schema_version:2,event_id:'019a13cf-0000-7000-8000-000000000001',event_type:'identity.user_registered',producer:'identity',aggregate_type:'user',aggregate_id:'019a13cf-0000-7000-8000-000000000002',aggregate_version:1,occurred_at:'2026-10-24T00:00:00Z',payload:{user_id:'019a13cf-0000-7000-8000-000000000002',email:'u@example.test',locale:'en',registration_id:'019a13cf-0000-7000-8000-000000000003',referral_claim_id:null}};

test('event int64 wire preserves 2^53 and int64 maximum without quoted integers',()=>{
 for(const version of [9007199254740991,9007199254740992n,9223372036854775807n]){
  const text=event.stringifyEnvelope({...envelope,aggregate_version:version});
  assert.ok(text.includes('"aggregate_version":'+version.toString()));
  assert.equal(event.parseEnvelope(text).aggregate_version,version);
 }
 const unsafeNumber={...envelope,aggregate_version:Number(9223372036854775807n)};
 assert.throws(()=>event.stringifyEnvelope(unsafeNumber));
 assert.throws(()=>event.stringifyEnvelope({...envelope,aggregate_version:9223372036854775808n}));
 assert.throws(()=>event.parseEnvelope(JSON.stringify(envelope).replace('"aggregate_version":1','"aggregate_version":9223372036854775808')));
});
test('public versions remain safe Numbers and nullable locale enum rejects other strings',()=>{
 assert.deepEqual(business.decodeProfileChanges({website_locale:null}),{website_locale:null});
 assert.deepEqual(business.decodeProfileChanges({notification_locale:'ru'}),{notification_locale:'ru'});
 assert.throws(()=>business.decodeProfileChanges({website_locale:'xx'}));
 assert.throws(()=>business.decodeVersionCommand({expected_version:9007199254740992n,idempotency_key:'key'}));
});
test('strict integer parser rejects duplicates and JSON syntax errors',()=>{
 assert.equal(wire.parseStrictJSON('9223372036854775807'),9223372036854775807n);
 assert.throws(()=>wire.parseStrictJSON('{"version":1,"version":2}'));
 assert.throws(()=>wire.parseStrictJSON('01'));
 assert.throws(()=>wire.parseStrictJSON('['.repeat(130)+'0'+']'.repeat(130)));
});
test('serialized integer subset rejects fractional and exponent tokens before rounding',()=>{
 for(const token of ['1.0000000000000001','9007199254740991.1','1e0','1E+0','0.0','-0.0','1e-999']){
  assert.throws(()=>wire.parseStrictJSON('{"expected_version":'+token+'}'),token);
 }
 assert.deepEqual(wire.parseStrictJSON('[true,false,null]'),[true,false,null]);
 for(const token of ['-9223372036854775808','9223372036854775807','9007199254740991','-9007199254740991'])assert.equal(wire.stringifyStrictJSON(wire.parseStrictJSON(token)),token);
});

test('serialized audit owner pages decode with nullable identities and optional projection metadata',()=>{
 const id='019a13cf-0000-7000-8000-000000000002';
 const producer={items:[
  {id,event_type:'identity.user_registered',actor_id:null,user_id:id,created_at:'2026-10-24T12:00:00Z'},
  {id,event_type:'identity.admin_action_recorded',actor_id:id,user_id:null,aggregate_type:'identity_admin_action',aggregate_id:id,aggregate_version:1,producer:'identity',created_at:'2026-10-24T12:00:00Z',reason:'narrow access',operation:'role.permissions',target_type:'role',target_id:'support'},
 ],next_cursor:'opaque-cursor'};
 const bytes=JSON.stringify(producer);const response=business.decodeAuditResponse(wire.parseStrictJSON(bytes));
 assert.equal(response.items.length,2);assert.equal(response.items[0].id,id);assert.equal(response.items[0].actor_id,null);assert.equal(response.items[1].target_id,'support');assert.equal(response.next_cursor,'opaque-cursor');
 assert.doesNotThrow(()=>business.decodeAuditResponse(wire.parseStrictJSON('{"items":[],"next_cursor":null}')));
 for(const invalid of [[],{items:[]},{...producer,raw_payload:'secret'},{items:[{...producer.items[0],raw_payload:'secret'}],next_cursor:null}])assert.throws(()=>business.decodeAuditResponse(wire.parseStrictJSON(JSON.stringify(invalid))));
});
test('serialized command status keeps exact public projection and rejects private result fields',()=>{
 for(const status of ['requested','unknown','denied','succeeded']){
  const producer={command_id:'019a13cf-0000-7000-8000-000000000002',status,...(status==='succeeded'?{result:{revoked:2}}:{})};
  assert.equal(business.decodeCommandStatus(wire.parseStrictJSON(JSON.stringify(producer))).status,status);
 }
 const invalid={command_id:'019a13cf-0000-7000-8000-000000000002',status:'succeeded',result:{provider_secret:'secret'}};
 assert.throws(()=>business.decodeCommandStatus(wire.parseStrictJSON(JSON.stringify(invalid))));
});

test('serialized own OAuth grants expose safe fields with strict scopes and timestamps',()=>{
 const grant={client_id:'client-one',client_name:'Account client',scopes:['openid','email','profile','offline_access'],trusted_first_party:false,granted_at:'2026-10-24T12:00:00Z',last_used_at:'2026-10-24T12:00:00Z'};
 const producer={items:[grant],next_cursor:'Y2xpZW50LW9uZQ'};
 const response=business.decodeOAuthGrantsResponse(wire.parseStrictJSON(JSON.stringify(producer)));
 assert.equal(response.items[0].client_id,'client-one');assert.equal(response.next_cursor,'Y2xpZW50LW9uZQ');
 for(const field of ['access_token','refresh_token','client_secret'])assert.throws(()=>business.decodeOAuthGrantsResponse(wire.parseStrictJSON(JSON.stringify({...producer,items:[{...grant,[field]:'secret'}]}))));
 for(const invalid of [{...producer,items:[{...grant,last_used_at:null}]},{...producer,items:[{...grant,scopes:['admin']}]},{...producer,items:Array.from({length:101},()=>grant)}])assert.throws(()=>business.decodeOAuthGrantsResponse(wire.parseStrictJSON(JSON.stringify(invalid))));
 const revoked=business.decodeOAuthGrantRevoked(wire.parseStrictJSON(JSON.stringify({client_id:'client-one',revoked:true})));assert.equal(revoked.revoked,true);
 assert.throws(()=>business.decodeOAuthGrantRevoked({client_id:'client-one',revoked:false}));
});
test('template authority variables and nonpublishing preview DTOs consume strict producer bytes',()=>{
 const locales={en:{subject:'Preview',body:'{{reset_url}}'},ru:{subject:'Просмотр',body:'{{reset_url}}'}};
 for(const decode of [business.decodeTemplatePreviewRequest,business.decodeTemplatePreviewResponse]){
  const decoded=decode(wire.parseStrictJSON(JSON.stringify({locales})));
  for(const locale of ['en','ru']){
   assert.equal(decoded.locales[locale].subject,locales[locale].subject);
   assert.equal(decoded.locales[locale].body,locales[locale].body);
  }
  for(const invalid of [{locales:null},{locales,extra:true},{locales:{en:locales.en}}])assert.throws(()=>decode(wire.parseStrictJSON(JSON.stringify(invalid))));
 }
 const revision={event_type:'notification.password_reset',version:1,locales,variables:['expires_at','reset_url']};
 assert.equal(business.decodeTemplateRevision(wire.parseStrictJSON(JSON.stringify(revision))).variables[1],'reset_url');
 for(const invalid of [{...revision,variables:['secret']},{...revision,variables:Array(33).fill('reset_url')},{event_type:revision.event_type,version:1,locales}])assert.throws(()=>business.decodeTemplateRevision(wire.parseStrictJSON(JSON.stringify(invalid))));
});

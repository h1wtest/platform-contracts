import {schema, type ErrorResponse, decodeErrorResponse} from './businessv1.js';
import {parseStrictJSON,validate} from './validation.js';

export class APIError extends Error {
 constructor(public readonly status:number,public readonly body:ErrorResponse){super(body.code);}
}
// Responses are decoded from bytes through duplicate-key-aware parsing and schema
// validation. Browser credentials and CSRF remain owned by the caller's live session.
export async function request<T>(operation:{method:string;path:string;response:string;request?:string},path:string,options:{body?:unknown;csrfToken?:string;signal?:AbortSignal;fetch?:typeof fetch}={}):Promise<T>{
 if(!path.startsWith('/api/v1/')||path.startsWith('//'))throw new TypeError('Relative platform API path required');
 const url=new URL(path,globalThis.location?.origin??'https://platform.local');
 if(url.pathname!==operation.path.replace(/\{[^}]+\}/g,match=>{const at=operation.path.indexOf(match);return url.pathname.slice(at).split('/')[0];})){
  const escaped=operation.path.split('/').map(part=>part.startsWith('{')?'[^/]+':part.replace(/[.*+?^${}()|[\]\\]/g,'\\$&')).join('/');
  if(!new RegExp('^'+escaped+'$').test(url.pathname))throw new TypeError('Operation path mismatch');
 }
 const headers=new Headers({'Accept':'application/json'});
 if(options.csrfToken)headers.set('X-CSRF-Token',options.csrfToken);
 let body:string|undefined;
 if(operation.request){if(options.body===undefined)throw new TypeError('Request body required');validate(schema,operation.request,options.body);body=JSON.stringify(options.body);headers.set('Content-Type','application/json');}
 else if(options.body!==undefined)throw new TypeError('Unexpected request body');
 const response=await(options.fetch??fetch)(path,{method:operation.method,credentials:'same-origin',redirect:'error',headers,body,signal:options.signal});
 const text=await response.text();const decoded=parseStrictJSON(text);
 if(!response.ok)throw new APIError(response.status,decodeErrorResponse(decoded));
 validate(schema,operation.response,decoded);return decoded as T;
}
export const operations=schema['x-endpoints'];

'use strict';
const {test}=require('node:test');const assert=require('node:assert/strict');const fs=require('node:fs');const path=require('node:path');const os=require('node:os');const crypto=require('node:crypto');
const {platformAsset,verifiedBinary}=require('../lib/runtime.cjs');
test('six supported targets and fail-closed unsupported systems',()=>{
 const expected={darwin:['screen-cli_darwin_amd64','screen-cli_darwin_arm64'],linux:['screen-cli_linux_amd64','screen-cli_linux_arm64'],win32:['screen-cli_windows_amd64.exe','screen-cli_windows_arm64.exe']};
 for(const [platform,names] of Object.entries(expected)){assert.equal(platformAsset(platform,'x64'),names[0]);assert.equal(platformAsset(platform,'arm64'),names[1]);}
 assert.throws(()=>platformAsset('linux','ia32'));assert.throws(()=>platformAsset('freebsd','x64'));
});
test('corrupt, linked, missing and mismatched-version binaries cannot execute',()=>{
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'screen-cli-npm-runtime-'));try{
  fs.mkdirSync(path.join(root,'dist'));const name=platformAsset('linux','x64');const raw=Buffer.from('original correct binary');
  fs.writeFileSync(path.join(root,'package.json'),JSON.stringify({version:'0.4.0'}));
  const manifest={version:'0.4.0',files:[{kind:'binary',name,size:raw.length,sha256:crypto.createHash('sha256').update(raw).digest('hex')}]};
  const manifestPath=path.join(root,'dist/manifest.json');fs.writeFileSync(manifestPath,JSON.stringify(manifest));
  const binary=path.join(root,'dist',name);fs.writeFileSync(binary,raw);assert.equal(verifiedBinary(root,'linux','x64'),binary);
  fs.writeFileSync(binary,Buffer.alloc(raw.length,65));assert.throws(()=>verifiedBinary(root,'linux','x64'),/校验失败/);
  fs.unlinkSync(binary);assert.throws(()=>verifiedBinary(root,'linux','x64'));
  if(process.platform!=='win32'){fs.writeFileSync(path.join(root,'outside'),raw);fs.symlinkSync(path.join(root,'outside'),binary);assert.throws(()=>verifiedBinary(root,'linux','x64'));fs.unlinkSync(binary);}
  fs.writeFileSync(binary,raw);manifest.version='0.3.0';fs.writeFileSync(manifestPath,JSON.stringify(manifest));assert.throws(()=>verifiedBinary(root,'linux','x64'),/版本不一致/);
 }finally{fs.rmSync(root,{recursive:true,force:true});}
});

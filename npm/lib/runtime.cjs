'use strict';
const fs=require('node:fs');
const path=require('node:path');
const crypto=require('node:crypto');
function platformAsset(platform=process.platform,arch=process.arch){
 const architecture={x64:'amd64',arm64:'arm64'}[arch];
 if(!['darwin','linux','win32'].includes(platform)||!architecture)throw new Error(`不支持当前系统或处理器：${platform}/${arch}`);
 return `screen-cli_${platform==='win32'?'windows':platform}_${architecture}${platform==='win32'?'.exe':''}`;
}
function verifiedBinary(root,platform=process.platform,arch=process.arch){
 const name=platformAsset(platform,arch);
 const manifest=JSON.parse(fs.readFileSync(path.join(root,'dist/manifest.json'),'utf8'));
 const pkg=JSON.parse(fs.readFileSync(path.join(root,'package.json'),'utf8'));
 if(manifest.version!==pkg.version)throw new Error('CLI 安装版本不一致，请重新安装');
 const asset=manifest.files.find(file=>file.name===name&&file.kind==='binary');
 const binary=path.join(root,'dist',name);
 if(!asset||!fs.lstatSync(binary).isFile())throw new Error('CLI 安装包缺失，请重新安装');
 const raw=fs.readFileSync(binary);
 if(raw.length!==asset.size||crypto.createHash('sha256').update(raw).digest('hex')!==asset.sha256)throw new Error('CLI 二进制校验失败，请重新安装');
 return binary;
}
module.exports={platformAsset,verifiedBinary};

#!/usr/bin/env node
import fs from 'node:fs';import path from 'node:path';import crypto from 'node:crypto';import {fileURLToPath} from 'node:url';
const source=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const check=process.argv[2]==='--check';const destination=process.argv[check?3:2];
if(!destination)throw new Error('Usage: export-repository.mjs [--check] destination');
const target=path.resolve(destination);
if(target===source||source.startsWith(target+path.sep)||target.startsWith(source+path.sep))throw new Error('Export must use a separate directory');
const rootFiles=['go.mod','README.md','AGENTS.md','VERSION','package.json','package-lock.json','.gitignore','RELEASE_NOTES.md'];
const directories=['npm','skills','scripts','.github'];
const files=[...rootFiles,...fs.readdirSync(source).filter(name=>name.endsWith('.go'))];
function visit(relative){for(const entry of fs.readdirSync(path.join(source,relative),{withFileTypes:true})){const name=`${relative}/${entry.name}`;if(entry.isSymbolicLink())throw new Error('Source links are not allowed');if(entry.isDirectory())visit(name);else if(entry.isFile())files.push(name);else throw new Error('Unsupported source entry');}}
for(const directory of directories)visit(directory);
files.sort();const digest=file=>crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
const expected=Object.fromEntries(files.map(name=>[name,digest(path.join(source,name))]));
const marker=path.join(target,'.screen-cli-export.json');let previous={};
if(fs.existsSync(marker))previous=JSON.parse(fs.readFileSync(marker,'utf8')).files;
if(fs.existsSync(target)){
 const allowed=new Set([...Object.keys(previous),...files,'.screen-cli-export.json']);
 function inspect(relative=''){for(const entry of fs.readdirSync(path.join(target,relative),{withFileTypes:true})){
  if(relative===''&&entry.name==='.git')continue;
  if(relative===''&&['dist','node_modules','packages'].includes(entry.name)&&entry.isDirectory())continue;
  const name=relative?`${relative}/${entry.name}`:entry.name;
  if(entry.isSymbolicLink())throw new Error('Destination links are not allowed');if(entry.isDirectory())inspect(name);else if(!allowed.has(name))throw new Error(`Unexpected destination file: ${name}`);
  else if(!check&&previous[name]&&digest(path.join(target,name))!==previous[name])throw new Error(`Destination has local edits: ${name}`);
 }}inspect();
}
if(check){for(const [name,hash] of Object.entries(expected))if(!fs.existsSync(path.join(target,name))||digest(path.join(target,name))!==hash)throw new Error(`Mirror differs: ${name}`);if(Object.keys(previous).some(name=>!expected[name]))throw new Error('Mirror has removed source files');console.log(`Mirror verified: ${files.length} allowlisted files`);}
else{
 for(const name of files){const output=path.join(target,name);fs.mkdirSync(path.dirname(output),{recursive:true});fs.copyFileSync(path.join(source,name),output);fs.chmodSync(output,fs.statSync(path.join(source,name)).mode&0o777);}
 for(const name of Object.keys(previous))if(!expected[name])fs.unlinkSync(path.join(target,name));
 fs.writeFileSync(marker,JSON.stringify({version:fs.readFileSync(path.join(source,'VERSION'),'utf8').trim(),files:expected},null,2)+'\n');console.log(`Exported ${files.length} allowlisted CLI files`);
}

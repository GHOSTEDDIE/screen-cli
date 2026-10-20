#!/usr/bin/env node
'use strict';
const path=require('node:path');
const {spawn,spawnSync}=require('node:child_process');
const {verifiedBinary}=require('../lib/runtime.cjs');
const {installSkill}=require('../lib/skill.cjs');
const root=path.resolve(__dirname,'../..');const args=process.argv.slice(2);
function output(data){process.stdout.write(JSON.stringify({ok:true,data})+'\n');}
function fail(error){process.stdout.write(JSON.stringify({ok:false,error:{error:'INSTALLATION_FAILED',message:error.message}})+'\n');process.exitCode=1;}
try{
 if(args[0]==='skill'){
  if(args.length===4&&args[1]==='install'&&args[2]==='--agent')output(installSkill(root,args[3]));
  else if(args.length===1||args.includes('--help'))output({usage:'screen-cli skill install --agent codex|claude'});
  else throw new Error('用法：screen-cli skill install --agent codex|claude');
 }else{
  const binary=verifiedBinary(root);
  if(args.length===0||['--help','help'].includes(args[0])){
   const help=spawnSync(binary,args,{encoding:'utf8',windowsHide:true});
   if(help.error)throw help.error;
   if(help.status!==0){process.stdout.write(help.stdout||'');process.stderr.write(help.stderr||'');process.exitCode=help.status||1;}
   else{const result=JSON.parse(help.stdout);result.data.usage.push('screen-cli skill install --agent codex|claude');process.stdout.write(JSON.stringify(result)+'\n');}
  }else{
   const child=spawn(binary,args,{stdio:'inherit',windowsHide:true});
   const handlers={};
   for(const signal of ['SIGINT','SIGTERM']){handlers[signal]=()=>{if(!child.killed)child.kill(signal);};process.on(signal,handlers[signal]);}
   const cleanup=()=>{for(const [signal,handler] of Object.entries(handlers))process.removeListener(signal,handler);};
   child.once('error',error=>{cleanup();fail(error);});
   child.once('exit',(code,signal)=>{cleanup();process.exitCode=code===null?(signal==='SIGINT'?130:143):code;});
  }
 }
}catch(error){fail(error);}

'use strict';
const fs=require('node:fs');
const path=require('node:path');
const os=require('node:os');
const crypto=require('node:crypto');
const metadata='.screen-cli-install.json';
function contentHashes(root){
 const hashes={};
 function visit(relative=''){
  for(const entry of fs.readdirSync(path.join(root,relative),{withFileTypes:true})){
   const name=relative?`${relative}/${entry.name}`:entry.name;
   if(name===metadata)continue;
   if(entry.isSymbolicLink())throw new Error('Skill 目录不能包含链接');
   if(entry.isDirectory())visit(name);
   else if(entry.isFile())hashes[name]=crypto.createHash('sha256').update(fs.readFileSync(path.join(root,name))).digest('hex');
   else throw new Error('Skill 包含不支持的文件');
  }
 }
 visit();
 return Object.fromEntries(Object.entries(hashes).sort(([a],[b])=>a.localeCompare(b)));
}
function validateSkill(root){
 const hashes=contentHashes(root);
 const skill=fs.readFileSync(path.join(root,'SKILL.md'),'utf8');
 const frontmatter=skill.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n/);
 if(!frontmatter||!/(?:^|\n)name: screen-cli(?:\r?\n|$)/.test(frontmatter[1])||!hashes['references/commands.md'])throw new Error('Skill 安装包无效');
 return hashes;
}
function same(a,b){return JSON.stringify(a)===JSON.stringify(b);}
function installSkill(root,agent,options={}){
 if(!['codex','claude'].includes(agent))throw new Error('请使用 skill install --agent codex 或 --agent claude');
 const home=options.home||os.homedir();const env=options.env||process.env;
 const base=agent==='codex'?(env.CODEX_HOME||path.join(home,'.codex')):(env.CLAUDE_CONFIG_DIR||path.join(home,'.claude'));
 const parent=path.join(base,'skills');const target=path.join(parent,'screen-cli');
 const source=path.join(root,'skills/screen-cli');const expected=validateSkill(source);
 const version=JSON.parse(fs.readFileSync(path.join(root,'package.json'),'utf8')).version;
 let exists=false;
 try{const stat=fs.lstatSync(target);if(!stat.isDirectory()||stat.isSymbolicLink())throw new Error('现有 Skill 路径不是普通目录');exists=true;}catch(error){if(error.code!=='ENOENT')throw error;}
 if(exists){
  const actual=validateSkill(target);
  if(same(actual,expected))return {agent,path:target,version,changed:false};
  let previous;
  try{previous=JSON.parse(fs.readFileSync(path.join(target,metadata),'utf8'));}catch{throw new Error('现有同名 Skill 不是本工具安装的版本，请先核对并移走该目录');}
  if(!same(actual,previous.hashes))throw new Error('现有 Skill 含有本地修改，已保留，请先核对后再安装');
 }
 fs.mkdirSync(parent,{recursive:true});
 const stage=fs.mkdtempSync(path.join(parent,'.screen-cli-stage-'));const candidate=path.join(stage,'screen-cli');const previous=path.join(stage,'previous');let committed=false;
 try{
  fs.cpSync(source,candidate,{recursive:true,errorOnExist:true,force:false});
  if(!same(validateSkill(candidate),expected))throw new Error('Skill 暂存校验失败');
  fs.writeFileSync(path.join(candidate,metadata),JSON.stringify({version,hashes:expected})+'\n',{mode:0o600});
  if(exists)fs.renameSync(target,previous);
  try{fs.renameSync(candidate,target);committed=true;}catch(error){if(exists){try{fs.renameSync(previous,target);}catch{throw new Error(`安装失败，原 Skill 已保留在 ${previous}，请恢复后重试`);}}throw error;}
 }finally{if(committed||!fs.existsSync(previous))fs.rmSync(stage,{recursive:true,force:true});}
 return {agent,path:target,version,changed:true};
}
module.exports={installSkill,validateSkill,contentHashes};

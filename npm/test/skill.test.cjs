'use strict';
const {test}=require('node:test');const assert=require('node:assert/strict');const fs=require('node:fs');const path=require('node:path');const os=require('node:os');
const {installSkill}=require('../lib/skill.cjs');const source=path.resolve(__dirname,'../..');
test('skill installation, repeat, safe upgrade and preservation of customized or unrelated skills',()=>{
 const work=fs.mkdtempSync(path.join(os.tmpdir(),'screen-cli-skills-'));try{
  const root=path.join(work,'package');fs.mkdirSync(root);fs.copyFileSync(path.join(source,'package.json'),path.join(root,'package.json'));fs.cpSync(path.join(source,'skills'),path.join(root,'skills'),{recursive:true});
  const home=path.join(work,'user home');const options={home,env:{}};
  for(const agent of ['codex','claude']){const result=installSkill(root,agent,options);assert.equal(result.changed,true);assert(fs.readFileSync(path.join(result.path,'SKILL.md'),'utf8').includes('screen-cli'));assert.equal(installSkill(root,agent,options).changed,false);}
  const codex=path.join(home,'.codex/skills');fs.mkdirSync(path.join(codex,'unrelated'));fs.writeFileSync(path.join(codex,'unrelated/keep'),'original');
  fs.appendFileSync(path.join(root,'skills/screen-cli/references/commands.md'),'\nNew upstream instructions\n');
  const upgraded=installSkill(root,'codex',options);assert(upgraded.changed);assert(fs.readFileSync(path.join(upgraded.path,'references/commands.md'),'utf8').includes('New upstream instructions'));assert.equal(fs.readFileSync(path.join(codex,'unrelated/keep'),'utf8'),'original');
  fs.appendFileSync(path.join(upgraded.path,'SKILL.md'),'\nUser customization\n');assert.throws(()=>installSkill(root,'codex',options),/本地修改/);assert(fs.readFileSync(path.join(upgraded.path,'SKILL.md'),'utf8').includes('User customization'));
  assert(!fs.readdirSync(codex).some(name=>name.startsWith('.screen-cli-stage-')));assert.throws(()=>installSkill(root,'unknown',options));
 }finally{fs.rmSync(work,{recursive:true,force:true});}
});
test('custom skill and linked target preserved; custom agent home respected',()=>{
 const work=fs.mkdtempSync(path.join(os.tmpdir(),'screen-cli-skill-custom-'));try{
  const options={home:work,env:{CODEX_HOME:path.join(work,'custom codex')}};const result=installSkill(source,'codex',options);assert(result.path.startsWith(options.env.CODEX_HOME));
  fs.unlinkSync(path.join(result.path,'.screen-cli-install.json'));fs.appendFileSync(path.join(result.path,'SKILL.md'),'\nCustomized\n');assert.throws(()=>installSkill(source,'codex',options),/不是本工具/);
  if(process.platform!=='win32'){fs.renameSync(result.path,result.path+'-original');fs.symlinkSync(result.path+'-original',result.path);assert.throws(()=>installSkill(source,'codex',options),/普通目录/);assert(fs.lstatSync(result.path).isSymbolicLink());}
 }finally{fs.rmSync(work,{recursive:true,force:true});}
});

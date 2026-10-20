'use strict';
const fs=require('node:fs');const path=require('node:path');
const {verifiedBinary}=require('../npm/lib/runtime.cjs');const {validateSkill}=require('../npm/lib/skill.cjs');
const root=path.resolve(__dirname,'..');const pkg=require('../package.json');
if(fs.readFileSync(path.join(root,'VERSION'),'utf8').trim()!==pkg.version)throw new Error('VERSION 与 package.json 不一致');
for(const platform of ['darwin','linux','win32'])for(const arch of ['x64','arm64'])verifiedBinary(root,platform,arch);
validateSkill(path.join(root,'skills/screen-cli'));
if(pkg.scripts.install||pkg.scripts.postinstall||pkg.dependencies)throw new Error('npm 分发不应依赖安装脚本或额外运行依赖');
console.log('Release verified: six binaries, matching version, hashes, bundled Skill, and no install scripts.');

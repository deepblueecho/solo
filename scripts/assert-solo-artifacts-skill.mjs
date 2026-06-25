import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const root = new URL('../', import.meta.url).pathname;
const skillDir = join(root, 'skills/solo-artifacts');
const oldCodexSkillDir = join(root, '.codex/skills/solo-artifacts');
const read = (path) => readFileSync(join(skillDir, path), 'utf8');
const assert = (condition, message) => {
  if (!condition) throw new Error(message);
};

assert(existsSync(skillDir), 'solo-artifacts skill directory should exist');
assert(!existsSync(oldCodexSkillDir), 'solo-artifacts should live in Solo skills/, not .codex/skills/');
assert(!existsSync(join(skillDir, '.git')), 'solo-artifacts skill should not contain nested git metadata');

const skill = read('SKILL.md');
const starter = read('assets/starter.html');
const css = read('assets/base.css');
const review = read('references/review-decision.md');

assert(skill.includes('name: solo-artifacts'), 'SKILL.md should use solo-artifacts as the skill name');
assert(!skill.includes('name: work-canvas'), 'SKILL.md should not keep the old work-canvas name');
assert(skill.includes('Solo') && skill.includes('artifact'), 'SKILL.md should describe Solo artifact usage');
assert(starter.includes('solo-artifacts STARTER'), 'starter should identify itself as solo-artifacts');
assert(css.includes('--solo-yellow') && css.includes('--solo-cream') && css.includes('shadow-brutal'), 'base.css should use Solo brutal yellow/cream tokens');
assert(!css.includes('[data-theme="dark"]'), 'base.css should not include a dark theme');
assert(!starter.includes('data-theme') && !starter.includes('data-theme-toggle'), 'starter should not include theme switching');
assert(css.includes('border: 2px solid var(--text)') || css.includes('border: 2px solid var(--ink)'), 'base.css should use hard brutal borders');
assert(review.includes('Solo-brutal') && review.includes('review-decision'), 'review-decision reference should describe the Solo-brutal variant');

console.log('solo-artifacts skill source checks passed');

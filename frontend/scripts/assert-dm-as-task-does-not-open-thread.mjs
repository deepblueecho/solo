import { readFileSync } from 'node:fs';

const source = readFileSync(new URL('../components/dashboard/dm-view.tsx', import.meta.url), 'utf8');
const match = source.match(/const handleAsTask = useCallback\([\s\S]*?\n  \);/);

if (!match) throw new Error('DMView handleAsTask should exist');
if (match[0].includes("setActiveRightPanel('thread')")) {
  throw new Error('DM as-task conversion should not open the thread panel');
}
if (!match[0].includes('await onAsTask(message)')) {
  throw new Error('DM as-task conversion should use the parent convert handler');
}

console.log('DM as-task source check passed');

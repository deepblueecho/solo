import { readFileSync } from 'node:fs';

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8');
const assert = (condition, message) => {
  if (!condition) {
    throw new Error(message);
  }
};

const teamsPage = read('app/teams/page.tsx');
const relationshipsPage = read('app/relationships/page.tsx');
const detailPanel = read('components/relationships/relationship-detail-panel.tsx');
const teamsAgentProfile = read('components/teams/teams-agent-profile.tsx');

assert(
  relationshipsPage.includes('export function RelationshipWorkspace'),
  'relationships page should expose a reusable RelationshipWorkspace component',
);
assert(
  teamsPage.includes('<RelationshipWorkspace') && !teamsPage.includes('TeamsLeftColumn'),
  'teams page should render the relationship workspace directly instead of the old TeamsLeftColumn layout',
);
assert(
  detailPanel.includes('TeamsAgentProfile') && detailPanel.includes('TeamsAgentWorkspace'),
  'agent node detail should reuse the existing Teams profile/workspace components',
);
assert(
  relationshipsPage.includes('AgentForm') && relationshipsPage.includes('Create from Template'),
  'relationship workspace should preserve single-agent and template creation',
);
assert(
  !relationshipsPage.includes('+ Agent'),
  'toolbar should not show a duplicate plus in the Agent button label',
);
assert(
  detailPanel.includes('agentPanelWidth') && detailPanel.includes('cursor-col-resize'),
  'agent detail panel should be resizable like the channel thread panel',
);
assert(
  detailPanel.includes('showProfileHeader={false}'),
  'embedded agent profile should hide its duplicate avatar header',
);
assert(
  !detailPanel.includes('<div className="tab">Runtime</div>'),
  'agent node detail should not add a standalone Runtime tab',
);
assert(
  detailPanel.includes('redirectAfterDelete={false}') && teamsAgentProfile.includes('redirectAfterDelete = true'),
  'embedded agent profile should delete in-place without redirecting away from the relationship graph',
);
assert(
  relationshipsPage.includes('onAgentDeleted={handleAgentDeleted}'),
  'relationship graph should refresh agents after deleting one from the embedded profile',
);

console.log('team relationship-first source checks passed');

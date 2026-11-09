# Claude Code skills inventory

Inventory verified on 2026-11-09. All Skills CLI installations are project-local
copies under `.claude/skills/`; no global skill installation was performed.

## Installed and reusable

| Skill(s) | Source | License / status | Use in RYZE |
| --- | --- | --- | --- |
| `frontend-design` | Anthropic official plugin `frontend-design@claude-plugins-official`, project scope | Official Anthropic plugin, enabled in `.claude/settings.json`, v1.1.1 | Intentional visual direction and production UI design |
| `ui-ux-pro-max` | `nextlevelbuilder/ui-ux-pro-max-skill` | MIT; established community skill | Searchable UI/UX, typography, color, accessibility and stack guidance |
| `vercel-react-best-practices` | `vercel-labs/agent-skills` | MIT; Vercel-maintained | React performance compatible with Vite; ignore Next-only advice |
| `vercel-composition-patterns` | `vercel-labs/agent-skills` | MIT; Vercel-maintained | Component composition and state boundaries |
| `web-design-guidelines` | `vercel-labs/agent-skills` | MIT; Vercel-maintained | Responsive/accessibility/UI review |
| `react-state-management` | `wshobson/agents` | MIT; community collection | Apply state-boundary guidance only; the verified root package currently has no Zustand dependency |
| `responsive-design` | `wshobson/agents` | MIT; community collection | Mobile-first layout and breakpoint review |
| `interaction-design` | `wshobson/agents` | MIT; community collection | Purposeful feedback, transitions and reduced motion |
| `design-system-patterns` | `wshobson/agents` | MIT; community collection | Tokens and reusable visual foundations without over-abstraction |
| `typescript-advanced-types` | `wshobson/agents` | MIT; community collection | Type-safe component/data APIs; avoid needless complexity |
| `threejs-*` (13 skills) | `cloudai-x/threejs-skills` | MIT; community collection | Current Three.js core, assets, materials, animation, interaction, performance and R3F references |
| `webapp-testing` | `anthropics/skills` | Apache-2.0; official Anthropic skill | Playwright browser inspection, screenshots and interaction verification |

The 13 Three.js skills are: `threejs-fundamentals`, `threejs-geometry`,
`threejs-materials`, `threejs-lighting`, `threejs-textures`,
`threejs-animation`, `threejs-loaders`, `threejs-shaders`,
`threejs-postprocessing`, `threejs-interaction`, `threejs-performance`,
`threejs-react-three-fiber`, and `threejs-asset-pipeline`.

CloudAI-X targets Three.js r186, matching RYZE's verified `three@0.186.1`.
RYZE currently has direct Three.js exercise demonstrations but no verified R3F
or Drei dependency; the R3F skill does not authorize adding either dependency.

`webapp-testing` includes the Apache license and a helper that manages a caller-
provided server command. Its security scan reported one Socket alert and a
medium Snyk result; use the helper only with explicit, trusted local commands
and the existing Playwright configuration. Do not pass secrets or arbitrary
unreviewed shell text.

## Installation and discovery evidence

The current CLI is `skills@1.7.6`, with Node `v26.13.0` and npm/npx `12.2.1`.
`skills-lock.json` pins the copied sources and hashes. `npx skills list` and
inspection of every installed `SKILL.md` confirmed the names above. The Claude
Code plugin list confirms the official frontend-design plugin is enabled at
project scope.

Before this setup there were no usable project or user-level Claude skills in
the inspected locations. The Impertio Three.js alternative was not installed:
the CloudAI-X MIT collection already provides the needed coverage without a
second overlapping collection. No additional discovery result was installed.

## Invocation guidance

Select the smallest useful set; do not load this entire inventory for ordinary
work:

- Visual redesign: `frontend-design`, `ui-ux-pro-max`, `responsive-design`,
  `design-system-patterns`, optionally `web-design-guidelines`.
- React implementation: the three Vercel skills, `react-state-management`,
  `typescript-advanced-types`.
- Existing Three.js work: only the relevant `threejs-*` skills; use R3F guidance
  only if RYZE intentionally adopts R3F later.
- Browser verification: `webapp-testing`, `web-design-guidelines`, and the
  relevant responsive guidance.
- Backend/database/API work: the RYZE-specific skills, not frontend skills.

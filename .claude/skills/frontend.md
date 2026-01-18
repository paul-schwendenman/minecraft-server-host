# Frontend Expert

Specialized guidance for working in `minecraft-ui/` - Svelte 5 monorepo.

## Project Structure

```
minecraft-ui/
├── apps/
│   ├── manager/    # Vite + Svelte 5 SPA (legacy control panel)
│   └── worlds/     # SvelteKit + static adapter (map viewer)
└── libs/
    ├── ui/         # @minecraft/ui - shared components
    └── data/       # @minecraft/data - API calls, stores, types
```

## Conventions

### Components
- **PascalCase** filenames: `AsyncButton.svelte`, `ServerStatus.svelte`
- **Location**: `libs/ui/src/lib/components/` for shared, `src/lib/` for app-specific
- **Exports**: Barrel files in `src/lib/index.ts`

### State Management
```typescript
// Custom store pattern from @minecraft/data
function createStatus() {
  const { subscribe, set } = writable({} as ServerStatusResponse);
  return {
    subscribe,
    refresh: async () => { set(await reducer(null)); },
    dispatch: async (action: Action) => { set(await reducer(action)); }
  };
}
export const status = createStatus();
```

### Styling
- **Tailwind CSS v4** + **DaisyUI** for design tokens
- Use semantic classes: `bg-base-200`, `btn-primary`, `text-base-content/70`
- Responsive: `sm:` prefix for breakpoints
- Custom utilities in `app.css` with `@apply`

### API Layer
- Separate file per endpoint in `@minecraft/data/src/lib/api/`
- Use native `fetch`, no HTTP library
- Types in `@minecraft/data/src/lib/types/`

### SvelteKit Patterns (worlds app)
```typescript
// +page.ts
export const load: PageLoad = async ({ fetch }) => {
  const worlds = await listWorlds(fetch);
  return { worlds };
};

// +page.svelte
<script lang="ts">
  export let data;  // Receives load() return
</script>
```

## Guidelines

1. **Prefer shared libs** - Components in `@minecraft/ui`, data in `@minecraft/data`
2. **TypeScript** - Use `<script lang="ts">` for typed components
3. **Async patterns** - Use `AsyncButton` for loading states, `{#await}` blocks
4. **No CSS-in-JS** - Use Tailwind classes or scoped `<style>` blocks
5. **Workspace deps** - `"@minecraft/data": "workspace:^"`

## Commands

```bash
cd minecraft-ui
pnpm install
pnpm dev:manager     # Run manager app
pnpm dev:worlds      # Run worlds app
pnpm -r test         # Run all tests
pnpm -r lint         # Lint all
pnpm -r check        # Type check (svelte-check)
```

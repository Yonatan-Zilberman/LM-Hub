# LM Hub TUI Manual QA Checklist

## Startup & Configuration
- [ ] First run wizard opens and prompts for LM Studio URL
- [ ] Wizard saves correctly to `~/.lmhub/config.yaml`
- [ ] TUI starts successfully after wizard
- [ ] Context budget and metrics bar render without overflow
- [ ] No emojis present in the UI

## Model Selection & Loading
- [ ] Press `Ctrl+M` to open Model Select view
- [ ] Verify LM Studio models are listed
- [ ] Select a model and press `Enter` to load
- [ ] Status bar updates from `[OFF]` to `[ON]` with green indicator
- [ ] Context bar accurately displays model context length

## Ask Mode
- [ ] Input a query in Ask Mode
- [ ] Verify token streaming updates progressively
- [ ] History correctly persists when switching modes

## Plan Mode
- [ ] Press `Ctrl+P` to switch to Plan Mode
- [ ] Verify context bar is visible
- [ ] Input a planning query and verify progress spinner
- [ ] View generated plan, accept with `Ctrl+B` (switches to Build Mode)

## Build Mode
- [ ] Verify plan step 1 is queued
- [ ] Verify Agent executes steps
- [ ] If required, `ShowAskUser` modal triggers and correctly handles input
- [ ] Verify `Ctrl+S` extracts memory

## Cross-Mode Support
- [ ] `Ctrl+T` to open Prompt Templates modal
- [ ] `Ctrl+E` to open Memory Center modal
- [ ] Verify Window resize correctly adjusts `ContentHeight`
- [ ] Verify modalless background operations (like network check) still happen while a modal is open

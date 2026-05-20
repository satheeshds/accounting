# Recipe management feature plan

## Problem
Add a new CRUD feature to track each menu item together with the ingredients it uses, the quantity/unit for each ingredient line, and an instruction field for preparation notes.

## Feature name
- **User-facing name:** `Recipe Management`
- The backend can still use stable resource names such as `menu_items`, `ingredients`, and `menu_item_ingredients`.

## Confirmed decisions
- Ingredients should be a reusable catalog referenced by menu items.
- The plan should treat quantity and unit as properties of the menu-item-to-ingredient relationship, not of the ingredient master record.
- The feature must include UI, not just backend/API work.
- Recipe records need an `instruction` column/field.
- The UI should include both a dedicated ingredient catalog screen and a recipe editor.

## Current state
- The service is a Go + Chi JSON API with a consistent resource pattern across `models`, `store`, `handlers`, `db/migrations`, `main.go`, and handler tests.
- Existing child-collection CRUD already exists for bill and invoice line items:
  - routes are registered in `main.go`
  - request/response models and validation live in `models`
  - SQL access lives in `store`
  - handler tests exercise create/list/update/delete flows against DuckDB migrations
- Swagger output is generated from handler annotations, so a new API surface should follow the same annotation pattern and then refresh generated docs.
- The frontend is a vanilla JS single-page UI under `static/`:
  - navigation is declared in `static/index.html`
  - section rendering, API calls, and modal forms live in `static/app.js`
  - styling lives in `static/style.css`

## Proposed approach
1. Model this as a reusable-catalog domain rather than a single flat table:
   - `ingredients` master records for reusable ingredients
   - `menu_items` parent records
   - `menu_item_ingredients` join rows containing `menu_item_id`, `ingredient_id`, `quantity`, and `unit`
2. Add an `instruction` field on `menu_items` so each recipe stores preparation steps/notes at the parent level.
3. Keep quantity and unit on the join row so the same ingredient can be used differently across menu items.
4. Mirror existing CRUD conventions:
   - top-level CRUD for `menu_items`
   - nested CRUD for `/menu-items/{id}/ingredients`
   - include ingredient lines on `GET /menu-items/{id}`
5. Add UI support in the existing SPA pattern:
   - a new sidebar entry for `Recipe Management`
   - a list view with search/table actions
   - modal forms for recipe CRUD and ingredient-line editing
   - a dedicated ingredient catalog screen for ingredient CRUD
6. Reuse the current implementation pattern used by bill/invoice items:
   - migration-first
   - model validation
   - transactional store methods for create/update/delete
   - handler tests covering inline nested data plus dedicated nested endpoints
   - frontend additions implemented directly in `static/app.js`, `static/index.html`, and `static/style.css`

## Todos
1. Confirm feature scope and naming
   - Ingredients are confirmed as reusable catalog records referenced by menu items.
   - Use `Recipe Management` as the product name.
2. Add schema and data model
   - Create migrations for `ingredients`, `menu_items`, and `menu_item_ingredients`.
   - Add an `instruction` column on `menu_items`.
   - Define IDs, timestamps, and required fields using the repository's existing migration style.
3. Add models and validation
   - Add input/output structs for menu items, ingredient catalog records, and ingredient lines.
   - Validate required recipe names, instruction handling, positive quantities, and non-empty units where applicable.
4. Add store layer CRUD
   - Implement list/get/create/update/delete for ingredients.
   - Implement list/get/create/update/delete for menu items.
   - Implement nested ingredient-line CRUD and transactional replacement for inline updates.
5. Add handlers, routes, and API docs
   - Add ingredient endpoints for standalone catalog management.
   - Register routes in `main.go`.
   - Add handler annotations so Swagger can include the new endpoints.
6. Add UI
   - Add a `Recipe Management` navigation entry and renderer in the SPA.
   - Add a dedicated ingredient catalog screen with search, create, edit, and delete actions.
   - Add recipe forms with instruction input and ingredient-line editing UX.
7. Add tests
   - Follow `handlers/items_test.go` style for full CRUD coverage and deleted-parent behavior.
   - Cover validation failures and nested create/update behavior.
   - Add frontend smoke coverage only if the repository already contains a pattern for it; otherwise keep UI verification manual.

## Notes and assumptions
- The safest fit with current code is to follow the bill/invoice item pattern, but expanded into a reusable-catalog design because ingredient reuse is now confirmed.
- The frontend already follows a hand-written SPA pattern, so UI work should extend that structure rather than introducing a new framework.
- If the final product language changes later, the backend resource names can still stay stable while the UI label changes independently.
- Ingredient catalog CRUD is confirmed as a standalone UI surface in addition to recipe editing.

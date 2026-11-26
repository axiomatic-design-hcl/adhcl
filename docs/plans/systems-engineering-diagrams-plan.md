# Systems engineering diagrams plan

This plan adds fourteen systems engineering diagrams to the page that `adhcl diagram` writes. Each diagram uses only facts that the current `.ad.hcl` language already holds. The plan changes no language rule.

Status: proposed. No step is started. The plan has three parts: a [review of the current page](#review-of-the-current-page) with fixes, the [new diagrams](#diagrams), and an [integration guide](#integration-guide) for adding them to the page.

## Terms

- A functional requirement (FR) states what the system must do.
- A design parameter (DP) is the design choice that controls one FR.
- A process variable (PV) says how a DP is built or delivered.
- A sibling set is the group of FRs and DPs that have the same parent.
- A view model is the data that the page shows, without HTML.
- A gap is data that the model does not contain. A gap is not a failure.
- A diagram is one rendered view on the page: a matrix, a table, or a static SVG drawing.

## Why

Systems engineers, testers, and reviewers know a standard set of diagrams. These include the requirement diagram, the design structure matrix (DSM), the N² chart, and the verification cross-reference matrix (VCRM). The current page uses Axiomatic Design terms only. A reader with a systems engineering background must translate those terms in their head.

The `.ad.hcl` facts already contain the links that these diagrams need. Each diagram in this plan reads typed references only. No diagram guesses a link from free text.

## Scope

The plan covers the fourteen diagrams that the language fully supports:

| No. | Diagram | Main reader |
|---|---|---|
| D1 | Domain mapping matrix (FR × DP) | Architect |
| D2 | Design structure matrix (DP × DP) | Architect, engineering lead |
| D3 | Multi-domain matrix | Systems engineer |
| D4 | Requirement diagram | Systems engineer, product manager |
| D5 | Functional decomposition tree | Everyone |
| D6 | Requirements traceability matrix | Tester, reviewer |
| D7 | Verification cross-reference matrix | Tester, quality lead |
| D8 | Allocation matrix | Architect, engineering lead |
| D9 | Module structure diagram | Software engineer |
| D10 | Module dependency N² chart | Software engineer, architect |
| D11 | Change impact and propagation matrix | Architect, tester |
| D12 | Trade study table | Program manager, architect |
| D13 | Measures tree | Product manager, quality lead |
| D14 | Module junction diagram | Architect, engineering lead |

These diagrams are outside this plan because the language does not hold the data that they need: the functional flow block diagram (FFBD), the internal block diagram, IDEF0, the context diagram, FMEA, the parametric diagram, the sequence diagram, the state machine, and the use case diagram. Some of these can show partial data. They need a separate plan that says how to show the missing parts.

## Baseline

These facts come from the current code:

| Fact | Location |
|---|---|
| `adhcl diagram` writes one HTML page with eight tabs. | `cmd/adhcl/diagram.go`, `internal/diagram/presentation.go` |
| `diagram.Build` makes a `Page` view model from `validate.Result`. It does no I/O. | `internal/diagram/build.go` |
| `diagram.Render` runs embedded templates. Each tab has its own template file. | `internal/diagram/render.go`, `internal/diagram/assets/tab_*.tmpl` |
| The architecture tab shows one design matrix for each scope and for the master scope. | `internal/diagram/matrix.go`, `tab_architecture.tmpl` |
| The trace tab has a trace map in static SVG, a change impact table, and a trace table with one row for each leaf FR. | `internal/diagram/layout.go`, `lists.go`, `tab_trace.tmpl` |
| The engineering tab lists PVs, modules, flows, and module junctions. | `internal/diagram/views.go`, `tab_engineering.tmpl` |
| `assessment.Impacts` gives the known and possible effects of each master DP. | `internal/assessment/impact.go` |
| `assessment.ScopeResult` gives the matrix, pairing, classification, design order, and cycles of each scope. | `internal/assessment/types.go` |
| The model holds `Module.Contains`, `Within`, `Dependencies`, `Criterion.AllocatedFrom`, `Scenario.Vary`, `ExpectAffected`, and `ExpectUnaffected`. | `internal/model/core.go` |
| Each record has one home row. Other mentions link to it through `Ref`. | `internal/diagram/build.go` |
| The page labels each claim as Source, Inferred, adhcl, Derived, or Not in source. | `internal/diagram/presentation.go` |

The [diagram implementation plan](diagram-implementation-plan.md) set the rules for the page. This plan keeps all of them.

## Review of the current page

Fix the current page before you add diagrams. The new diagrams reuse its matrix layout, its tables, and its map. Each defect below would spread to them.

### Method

The review used adhcl at commit `2709d45`, built with `make build`. These commands generated the pages:

```sh
adhcl diagram examples/minimal.ad.hcl
adhcl diagram examples/architecture.ad.hcl
adhcl diagram examples/project/model.ad.project.hcl
adhcl diagram ../spec/examples/requirements_hcl.ad.hcl
```

The last model is the largest available input: 12 DPs, 9 leaf FRs, and a 9 × 9 master matrix. Each page opened in Chromium through Playwright at 1280 × 900 and at 390 × 844. Each tab was captured as a full-page screenshot. A script compared the document width with the viewport and read the computed styles. No page had a script error or a console error. `examples/architecture-diagram.html` equals the generated output.

The review applied Nielsen's ten heuristics, Krug's trunk test, WCAG 2.2 AA, and the consistency checks of the `refactoring-ui` skill. Severity uses Nielsen's scale: 4 blocks a task or breaks a stated constraint, 3 is a major problem, 2 is a minor problem, and 1 is cosmetic.

### What works

Keep these properties. The new diagrams must have them too.

- The page is one file with no network load. The same input gives the same bytes.
- The five claim labels (Source, Inferred, adhcl, Derived, Not in source) differ by shape as well as by color.
- The start tab passes the trunk test. It names the design, the tabs, and the reader of each tab.
- The matrices never show an unknown as `0`. A hatched `?` cell is easy to tell from a `0` cell.
- Each tab works without JavaScript, and each record has one home row.

### Findings

| ID | Issue | Evidence | Heuristic or rule | Severity | Fix | Status |
|---|---|---|---|---|---|---|
| R1 | A wide matrix pushes the architecture tab wider than the screen. | Self-design page: document width 900 px at a 390 px viewport, and 1300 px at a 1280 px viewport. The text column beside the master matrix is 550 px wide and outside any scroll region. On desktop it shrinks to one word per line. | WCAG 1.4.10 Reflow (AA), plan constraint C9 | 4 | Stack `.mx` into one column. Put the classification, the meaning, and the design order above the matrix, and let only the matrix table scroll. Add a test that the document width is at most the viewport width at 390 px for every tab. | Open |
| R2 | When a DP is selected on the trace map, the other boxes have too little contrast to read. | `.active .node:not(.hit){opacity:.3}` leaves the text at about 2:1 against `#141413`. | WCAG 1.4.3 Contrast (AA) | 4 | Keep the text at 4.5:1 or more. Mark boxes outside the impact with a muted, dashed border instead of opacity, and keep opacity only for edges. | Open |
| R3 | The influence lines on the trace map cannot be read with more than about 8 leaf FRs. | Self-design map: dozens of dashed curves and `?` labels in the 35-unit gap between the FR column and the DP column. | Aesthetic and minimalist design, recognition rather than recall | 3 | Draw influence edges only for the selected DP, both with and without the script, using `DefaultImpact` when the script is off. Show the full cross-effect picture in the new DSM (D2) instead of on the map. | Open |
| R4 | Wide tables cut off columns with no sign that more exists. | At 1280 px: "Scenarios to run again" in change impact and "Realized by" in the DP table are cut. At 390 px: the DP table shows three of nine columns. | Visibility of system status | 3 | Add a scroll shadow on the right edge of `.scroll` with pure CSS (`background-attachment: local`). Make the ID column sticky. Let the mono header labels wrap. At less than 720 px, show list rows as stacked cards. | Open |
| R5 | Tabs on a phone cut off the active tab with no scroll cue. | At 390 px the bar ends in "Arc" for the architecture tab. | Visibility of system status, trunk test ("Where am I?") | 2 | Scroll the active tab into view on route. Add an edge fade. Below 480 px, wrap the tabs into two rows. | Open |
| R6 | Some cells show two gap labels for one fact. | PV and "Realized by" cells show a "Not in source" tag and the word "None". | Consistency and standards | 2 | Show one gap tag with the text inside it. Use this one gap form on every tab and in every new diagram. | Open |
| R7 | Map boxes for modules show only a cut ID. | `module.candidate_selec…` with an empty body. | Recognition rather than recall | 2 | Wrap the ID onto two lines before cutting text. Fill the body with the DPs that the module realizes. | Open |
| R8 | The impact detail panel lists IDs as plain text. | The text of `#detail` after a DP is selected. | Consistency with the rule that every mention links to its home row | 2 | Build the links in the script from the `Ref` anchors, or render one hidden detail block for each DP in the template and show the selected one. The second option needs no markup in the script. | Open |
| R9 | "Reasons for each cell" looks like a heading, not a control. | 18 px summary text with no marker beside a 14 px explanation. | Affordance, consistency | 2 | Give the summary a marker and a count, for example "Show 20 cell reasons". Use the body text size. | Open |
| R10 | The start tab does not say what the reader should do first. | Self-design: 50 warnings and an unresolved top level, but the first thing on the page is the view cards. | Steps to value; the "One Thing" check of the `steve-jobs-design-review` skill | 1 | Above the view cards, show one sentence that gives the top open item and links to it, for example "The top-level matrix is unresolved: 2 cells are unknown." The tool already derives this in `openItems`. | Open |
| R11 | The guide tab is 4,000 to 6,300 px tall with no index. | Screenshot heights | Help and documentation | 1 | Add an "On this tab" bar, as the other tabs have. | Open |

R1 and R2 break constraint C9 of the first diagram plan. Fix them before any release that claims WCAG 2.2 AA.

### Cut list

These cuts make the page smaller. None of them removes a fact.

- Show influence edges on the trace map for the selected DP only (R3). The DSM shows them all.
- Do not repeat a matrix in full on a second tab. Link to its home instead. See the [integration guide](#integration-guide).

### Tests to add

- A Playwright check in `cmd/adhcl/diagram_acceptance_test.go`, or a Go test that runs a headless browser when one is available. It checks that no tab is wider than the viewport at 390 px and at 1280 px, on the architecture example and on the self-design example. If no browser is available, the test is skipped and says so. It never passes without running.
- A contrast check on the computed colors of map text in the selected state.
- A unit test that each gap cell renders one tag (R6).

## Command contract

The command line does not change:

```
adhcl diagram [--profile source|architecture] [--candidate REF] [--context REF]
              [-o FILE | --output FILE] [--] PATH
```

The exit codes, the output rules, and the blocked-assessment behavior do not change. The page gets one new tab. See decision Q1 for other options.

## Constraints

The constraints of the first diagram plan still apply. These matter most for this work:

| ID | Constraint | Origin |
|---|---|---|
| C1 | Go code derives every value. The templates and the script compute nothing about the design. | Diagram plan C1 |
| C2 | The page is one self-contained file. It loads nothing from the network. | Diagram plan C2 |
| C3 | The same input and the same tool version give the same bytes. | Diagram plan C3 |
| C4 | Add no new Go module dependency. Draw SVG by hand in Go. Do not use Mermaid or Graphviz. | Diagram plan C4 |
| C5 | A diagram never shows a link that the model does not declare or that the assessment does not derive. | Diagram plan C5, spec principle 1 |
| C6 | Every string from the model is escaped. | Diagram plan C6 |
| C7 | An unknown cell shows as `?`. It never shows as `0` or as an empty cell. | Spec principle 2 |
| C8 | Each diagram has an "About this diagram" part with three headings: what it is, how to use it, and who reads it. | User request |
| C9 | The fixed text follows the `simple-english` skill. | Diagram plan C8 |
| C10 | The page meets WCAG 2.2 AA. Each SVG drawing has a table or a list with the same data. | Diagram plan C9 |
| C11 | The existing tabs, anchors, and tests do not change, except where this plan says so. | Compatibility |
| C12 | The SE names describe the form of a diagram. They do not claim SysML conformance or an exchange format such as XMI. | The language is not SysML |

## Design

### Placement

Add one tab, "Systems engineering", with the ID `systems`. Put it after "Risks and evidence" and before "Guide". The tab opens with an index of all fourteen diagrams. D1, D6, D11, and D14 extend existing views on other tabs, and their index cards link there. The other ten diagrams have a section on the new tab. The [integration guide](#integration-guide) gives the details.

The new tab reuses records. It does not create new home rows. Each mention of a record links to its home row on the existing tabs, as on every other tab.

### Shared parts

Add these parts once and use them in every diagram:

1. `About` struct with three fields: `What`, `How`, and `Who`. Each field is a list of short paragraphs. The text is fixed. It lives in one Go file, `internal/diagram/systems_text.go`, so that a test can check that every diagram has all three parts.
2. A template `about` that renders the struct as a `<details open>` block with three `<h4>` headings: "What it is", "How to use it", and "Who reads it". With the script off, the block stays open.
3. `GridMatrix`, a general matrix view model: row headers, column headers, and cells. Each cell has a symbol, a label kind, an explanation, and a source location. D1, D2, D3, D7, D8, D10, and D11 use it. `matrix.go` keeps the existing `Matrix` type for the architecture tab.
4. A template `grid` that renders `GridMatrix` as an accessible table with `scope` attributes, a caption, and a scroll region. It uses the same CSS classes as the existing matrices: `x`, `off`, `zero`, and `unk`.
5. A tree layout function in `internal/diagram/layout_tree.go`. It places the nodes of a tree in columns by depth and in rows by authored order. It returns integer coordinates. D5, D9, and D13 use it.
6. A box-and-edge graph layout function in `internal/diagram/layout_graph.go`. It places nodes in fixed lanes and routes each edge as a cubic curve from the right side of the source to the left side of the target. D4 and D14 use it. It reuses `wrap` and the box size from `layout.go`.
7. A size guard. If a diagram has more than 40 rows, 40 columns, or 120 nodes, the page splits it by top-level branch. Each part goes in its own `<details>` block. The guard never removes a row, a column, or a node.

### Data flow

`Build` calls one new function, `b.systems(page)`, after it fills `Lists`, `Matrices`, `Impacts`, and `Trace`. The function returns a `Systems` struct with one field for each diagram. Each diagram builder is a pure function that reads the builder indexes and the `validate.Result`. No diagram builder reads another diagram's output. This keeps the builders independent, so that you can build them in any order.

### Rules for every diagram

1. Each cell, node, and edge has one label kind: Source, Inferred, adhcl, Derived, or Not in source.
2. A diagram with no data shows one "Not in source" card that names the missing block kind. It does not disappear.
3. Inactive DPs (proposed, rejected, superseded, or outside the snapshot) do not appear in D1, D2, D3, D8, D11, or D14. D4, D5, and D12 show them with a dashed border and the word "Inactive".
4. Every diagram reads the selected candidate and context through the same snapshot as `validate`.
5. Each diagram cites the spec field that each kind of link comes from, in its "About" part or in a legend.

## Diagrams

Each section below gives the fixed "About" text, the source facts, the rendering, the gaps and limits, and the tests. The "About" text is the text that goes on the page.

### D1. Domain mapping matrix (FR × DP)

#### About this diagram

**What it is.** A domain mapping matrix (DMM) links the items of two domains. Here the rows are leaf functional requirements (FRs) and the columns are leaf design parameters (DPs). In Axiomatic Design this matrix is the master design matrix. `X` means that the DP changes whether the FR meets its criteria. `0` means that the model states a reason why it does not. `?` means that no effect is declared.

**How to use it.** Read across a row to see every design choice that can move one outcome. Read down a column to see every outcome that one design choice can move. An `X` off the diagonal is a cross effect. Each cross effect is a reason to set the DPs in a fixed order or to change the design. Each `?` is a question for the team. Hover a cell to read its mechanism or its reason.

**Who reads it.** Software architects and systems engineers who check independence. Reviewers who want one picture of the whole design.

#### Source facts

| Element | Source |
|---|---|
| Rows and columns | `assessment.MasterScope(res.Scopes).Matrix.FRs` and `.DPs` |
| Cell value | `Matrix.Cell(fr, dp)`, through the same accessor as the architecture tab |
| Diagonal | `ScopeResult.Pairing` |
| Explanation | `influence.mechanism` or `influence.because`, with `mechanism_source` |

#### Render

A `grid` table. Order the rows and columns by the design order when the scope has one, and by authored order otherwise. Shade the diagonal. Mark a conflict with "(conflict)". Show the classification and a count of unknown cells above the table.

#### Gaps and limits

If there is no master scope, show "The tool has no master matrix." D1 is the master matrix on the architecture tab, which gets the SE name and this "About" block. The SE tab links to it and does not draw it again.

#### Tests

- Every cell in D1 equals the master cell on the architecture tab for `examples/architecture.ad.hcl`.
- A fixture with an unlisted cell and no `cells` default shows `?`.
- A fixture with a decoupled scope orders rows and columns by the design order.

### D2. Design structure matrix (DP × DP)

#### About this diagram

**What it is.** A design structure matrix (DSM) is a square matrix with the same items on both axes. Here the items are the leaf design parameters (DPs). A mark in row A, column B means that DP B also changes the FR that DP A controls. So you must settle DP B before you tune DP A. The tool derives this matrix from the FR × DP matrix and the control pairing. It is not authored.

**How to use it.** Marks below the diagonal follow the design order. Marks above the diagonal are feedback. A block of marks around the diagonal is a cycle: those DPs must be tuned together. Use the matrix to plan the build order, to find modules that teams can build in parallel, and to find the cycles that need a coupling record.

**Who reads it.** Software architects and engineering leads who plan the order of work. Systems engineers who use DSM tools and methods.

#### Source facts

| Element | Source |
|---|---|
| Axes | Master DPs that have a pairing, in `ScopeResult.Order` |
| Cell (A, B) | `Matrix.Cell(Pairing⁻¹(A), B)` for A ≠ B |
| Cycles | `ScopeResult.Cycles` |

#### Render

A `grid` table with the same order on both axes. Mark the diagonal with the DP number. Show `X` for a material effect, `?` for an unknown, and leave a known `0` as `0`. Draw a border around each cycle block. Under the table, list each cycle and link it to its coupling record, if one exists.

#### Gaps and limits

The DSM needs a unique pairing. If the master scope has no pairing, show "The FRs and DPs do not pair, so the tool cannot build a DSM." Then link to the finding that explains why. Do not invent a pairing.

#### Tests

- A decoupled fixture has no mark above the diagonal.
- A coupled fixture shows its cycle as one block, and the block members equal `ScopeResult.Cycles`.
- An ambiguous pairing gives the gap card and no table.

### D3. Multi-domain matrix

#### About this diagram

**What it is.** A multi-domain matrix (MDM) puts several domains on one square grid. Each block of the grid links two domains. Here the domains are needs, FRs, DPs, PVs, modules, states, and scenarios. Each block shows one kind of declared link, for example "DP realized by module".

**How to use it.** Use the MDM to see the whole model in one place. Find empty blocks, which mean that a kind of link is missing everywhere. Find rows or columns with no mark, which are items with no link of that kind. Open one block to see its cells.

**Who reads it.** Systems engineers and reviewers who check that the model is complete. Tool builders who check the model graph.

#### Source facts

| Block | Source |
|---|---|
| Need × FR | Effective needs: `assessment.EffectiveNeeds`. Mark inherited needs as Derived. |
| FR × DP | D1 cells |
| DP × DP | D2 cells |
| DP × PV | `pv.realizes` |
| DP × module | `module.realizes` |
| DP × state | `dp.owns_state` (`O`) and `dp.reads_state` (`R`) |
| Module × module | `module.dependencies` and `module.contains` (`C`) |
| Module × state | `module.owns_state` |
| Scenario × FR | `scenario.verifies` through the criterion's FR, `expect_affected`, `expect_unaffected` |

#### Render

A summary grid first: one row and one column for each domain, and each block cell gives the count of marks. Each count links to a detail `grid` for that block. Each detail block uses its own symbol set, which a legend explains. Blocks with no source field, for example Need × PV, are not drawn.

#### Gaps and limits

A full MDM of a large model is too big to read. The summary grid always fits. The detail blocks follow the size guard. An empty block shows "Not in source" with the field that would fill it.

#### Tests

- The count in each summary cell equals the number of marks in its detail block.
- The FR × DP block equals D1.
- A fixture with no PV shows the DP × PV block as a gap.

### D4. Requirement diagram

#### About this diagram

**What it is.** A requirement diagram draws requirements as boxes and their relations as labeled arrows. It uses the relation names of SysML: trace, derive, refine, satisfy, and verify. It is not a SysML model file. Here a need traces to a root FR. A child FR derives from its parent FR through the parent DP. A criterion refines its FR. A DP satisfies the FR that it controls. A scenario verifies a criterion. A constraint limits an FR or a DP.

**How to use it.** Start at a need on the left and follow the arrows to the right. Check that each FR has a criterion, a DP that satisfies it, and a scenario that verifies it. A box with no arrow to its right is a gap. Use one diagram for each top-level FR when you review one capability with its owner.

**Who reads it.** Systems engineers and requirement engineers. Product managers who own the FRs. Reviewers who know SysML.

#### Source facts

| Arrow | Source |
|---|---|
| «trace» FR → need | `fr.need`, or the inherited need (Derived) |
| «deriveReqt» child FR → parent FR | `fr.parent = dp.X` and `dp.X.controls` |
| «refine» criterion → FR | `fr.criterion` |
| «deriveReqt» child criterion → parent criterion | `criterion.allocated_from` |
| «satisfy» DP → FR | `dp.controls` |
| «verify» scenario → criterion | `scenario.verifies` |
| «bounded_by» constraint → FR or DP | `fr.bounded_by`, `dp.bounded_by` |
| Citation | `cites`, shown in the box title, not as an arrow |

#### Render

One SVG for each top-level FR, made with the graph layout. Lanes from left to right: needs and constraints, FRs by depth, criteria, DPs, scenarios. Each box shows the stereotype in angle quotes, the ID, and two lines of the statement. Each arrow has its relation label. Inactive DPs have a dashed border. Under each SVG, a table lists the same arrows: source, relation, target, label kind, and location.

#### Gaps and limits

An FR with no criterion, no satisfying DP, or no verifying scenario gets a dashed "Not in source" stub box on that side. The diagram does not show `influence` edges. Those belong to D1 and D2.

#### Tests

- Each arrow in the table has a declared reference or a Derived label.
- Each FR without a criterion has exactly one stub box.
- The SVG has no overlapping boxes, and each arrow ends on its two boxes.

### D5. Functional decomposition tree

#### About this diagram

**What it is.** A functional decomposition tree breaks one top function into smaller functions, level by level. In Axiomatic Design the FR tree and the DP tree grow together. Each DP is decomposed into the child FRs that it creates. This back-and-forth is called zigzagging. The tree shows both hierarchies side by side, with the hierarchy numbers such as `1.2.1`.

**How to use it.** Read the FR tree to see what the system must do at each level. Read the DP tree to see the design choice at each level. Follow a zigzag arrow from a DP to the child FRs that it creates. A leaf with no children is where the design is detailed enough to build. A non-leaf DP with no child FRs is a gap in the decomposition.

**Who reads it.** Everyone. Business leads and product managers read the top levels. Architects and engineers read the lower levels.

#### Source facts

| Element | Source |
|---|---|
| FR nodes and levels | `fr.parent`, `Graph.SiblingSets`, the existing hierarchy numbers |
| DP nodes | `dp.controls` |
| Zigzag arrow DP → child FRs | `fr.parent` |
| Classification of each child set | `ScopeResult.Classification` |

#### Render

One SVG with two trees, made with the tree layout. The FR tree is on the left and the DP tree is on the right, mirrored, so that each FR faces its DP. A thin line joins each FR to its DP. A curved arrow joins each DP to its child FRs. Each child set has a small badge with its classification. A nested `<ul>` list gives the same tree without SVG.

#### Gaps and limits

The tree shows inactive DPs with a dashed border and the word "Inactive". Their child FRs are not in the snapshot, so the tree does not show them. A DP that controls two FRs draws one box with two lines.

#### Tests

- The tree has one node for each FR and DP in the hierarchy, and the numbers equal the product tab.
- A rejected DP is dashed and has no child.
- The nested list and the SVG contain the same IDs in the same order.

### D6. Requirements traceability matrix

#### About this diagram

**What it is.** A requirements traceability matrix (RTM) is a table with one row for each acceptance criterion. Each row follows the links from the stakeholder need to the requirement, the design choice, the implementation, the test, and the evidence. The trace table on the trace tab has one row for each leaf FR. This matrix has one row for each criterion, which is the unit that a test checks.

**How to use it.** Read a row from left to right to see how one acceptance condition is met and checked. Sort or scan a column for gaps. A row with no scenario is not tested. A row with no evidence has no recorded result. Use the matrix in audits and in release reviews.

**Who reads it.** Testers and quality leads. Reviewers and auditors who must show that each requirement is designed, built, and checked. Program managers who track open items.

#### Source facts

| Column | Source |
|---|---|
| Stakeholder, need | `need.stakeholder`, effective needs |
| FR and its path | `fr`, `fr.parent` |
| Criterion | `fr.criterion`, with `metric`, `op`, `value`, and `unit` |
| Constraint | `fr.bounded_by` |
| DP | `dp.controls` |
| Module and PV | `module.realizes`, `pv.realizes` of that DP or its leaf descendants |
| Scenario | `scenario.verifies` |
| Evidence | `evidence.scenario` for those scenarios, with the estimate from `Result.Candidates` |

#### Render

A table with the columns above. Each cell lists links to the home rows. An empty cell shows a dashed "Missing" label. The caption gives the number of rows and the number of rows with a full chain.

#### Gaps and limits

An FR with no criterion gets one row with "No criterion" in the criterion column, so that the FR does not disappear. Evidence shows "No recorded result" when no `evidence` block names the scenario. The matrix does not say that a test passed. The model holds no test runs.

#### Tests

- The matrix has one row for each criterion, plus one row for each FR without a criterion.
- Each module in a row realizes the row's DP or one of its leaf descendants.
- A scenario with an `evidence` block shows that block, and a scenario without one shows "No recorded result".

### D7. Verification cross-reference matrix

#### About this diagram

**What it is.** A verification cross-reference matrix (VCRM) shows which test checks which requirement. Here the rows are acceptance criteria and the columns are scenarios. A mark means that the scenario lists the criterion in `verifies`. The columns also show the kind of each scenario: change, operational, or failure.

**How to use it.** Find each row with no mark. That criterion has no planned check. Find each column with no mark. That scenario checks no stated criterion. Use the scenario kind to see whether you test only normal operation, or also changes and failures. The matrix shows planned checks. It does not show results.

**Who reads it.** Testers and quality leads who plan verification. Reviewers who check test coverage before a release.

#### Source facts

| Element | Source |
|---|---|
| Rows | Every criterion, grouped by FR |
| Columns | Every scenario, with `scenario.kind` |
| Mark | `scenario.verifies` |
| Evidence row at the bottom | `evidence.scenario` |

#### Render

A `grid` table. The column header shows the scenario ID and its kind. Each mark is `V`. A last row shows, for each scenario, the linked evidence or "No recorded result". A last column shows, for each criterion, the count of scenarios.

#### Gaps and limits

The language has no verification method field, such as test, analysis, inspection, or demonstration. The "About" part says so. The matrix does not show a method column and does not guess one from the scenario kind.

#### Tests

- Each `V` equals one `verifies` reference.
- A criterion with no scenario has a count of 0 and the "Missing" label.
- A scenario with an empty `verifies` list has an empty column and a "No criterion" label.

### D8. Allocation matrix

#### About this diagram

**What it is.** An allocation matrix shows which part of the implementation carries each design choice. Here the rows are leaf design parameters (DPs). The columns are modules and process variables (PVs). A mark means that the module or the PV realizes the DP. A second table shows which DP owns each piece of state and which DPs read it.

**How to use it.** Each leaf DP needs at least one mark. A row with no mark is a design choice that nothing implements yet. A column with marks in many rows is a module that carries many choices. That module is a place where a change can spread. In the state table, a state read by many DPs needs an influence review, because shared state can create cross effects.

**Who reads it.** Software architects who assign design choices to modules. Engineering leads who assign modules to teams.

#### Source facts

| Element | Source |
|---|---|
| Rows | Active leaf DPs |
| Module columns | `module.realizes` |
| PV columns | `pv.realizes` |
| State table | `dp.owns_state`, `dp.reads_state`, `module.owns_state` |

#### Render

One `grid` table with a column group for modules and a column group for PVs. A mark is `R`. A second `grid` table with states as rows and DPs and modules as columns. The marks are `O` for owns and `R` for reads.

#### Gaps and limits

The matrix shows declarations, not a working implementation. A PV with a binding attachment shows the owner result, which is `unknown` today. The "About" part says that a mark is a claim, not proof.

#### Tests

- Each `R` equals one `realizes` reference.
- A leaf DP with no module and no PV has a row with "Not realized".
- The state table marks a state that two DPs read in both rows.

### D9. Module structure diagram

#### About this diagram

**What it is.** A module structure diagram shows how the logical modules of the system nest. It is similar to a SysML block definition diagram. A composite module contains other modules. A realization module realizes one or more DPs. Each box lists the contract of the module: what it realizes, the state it owns, its inputs, its outputs, and its invariants.

**How to use it.** Read the boxes from the top to see the large parts of the system, then open each part. Check that each realization module names its DPs and that each module has a contract. Use the diagram when you split work between teams or when you plan a package layout.

**Who reads it.** Software engineers and engineering leads. Architects who review the module boundaries.

#### Source facts

| Element | Source |
|---|---|
| Composition | `module.contains` and `module.within` |
| Compartments | `realizes`, `owns_state`, `inputs`, `outputs`, `invariants`, `failure_behavior` |
| Acceptance checks | `module.acceptance_checks` |

#### Render

One SVG made with the tree layout. Each module is a box with a name row and compartments. A composition line with a filled diamond joins a composite to each child. A nested list gives the same tree with the same compartments.

#### Gaps and limits

The `inputs` and `outputs` fields are free text. The diagram lists them inside the box. It does not draw ports or connectors between modules, because the language has no typed interface. A missing compartment shows "Not in source". A module that no composite contains is a root.

#### Tests

- Each `contains` reference has one composition line.
- A module with no `invariants` shows the gap label in that compartment.
- The diagram draws no line between two modules from matching input and output strings.

### D10. Module dependency N² chart

#### About this diagram

**What it is.** An N² chart places the modules on the diagonal of a square grid. A mark in row A, column B means that module A depends on module B. The chart is read like a DSM. It shows the declared `dependencies` of each module.

**How to use it.** Scan each row to see what a module needs. Scan each column to see what needs a module. A pair of marks that mirror each other across the diagonal is a cycle. A dependency cycle is an architecture fact. It does not alone mean that the FRs are coupled. Use the chart to plan the build order and to find modules that are hard to change.

**Who reads it.** Software engineers and architects. Engineering leads who plan the order of integration.

#### Source facts

| Element | Source |
|---|---|
| Diagonal | Every realization and composite module |
| Mark | `module.dependencies` |
| Cycles | Strongly connected parts of the dependency graph |

#### Render

A `grid` table with the module IDs on the diagonal cells. A mark is `D`. Order the modules by a stable topological order when one exists, and by authored order inside each cycle. Draw a border around each cycle. List the cycles under the table.

#### Gaps and limits

A module with no `dependencies` field shows a dashed row header with "Not declared". This is different from `dependencies = []`, which is a known empty list. The chart does not label the interface on each mark, because inputs and outputs are free text.

#### Tests

- Each `D` equals one `dependencies` reference.
- An omitted field and an empty list render differently.
- A fixture with a two-module cycle shows one cycle block.

### D11. Change impact and propagation matrix

#### About this diagram

**What it is.** A change propagation matrix shows what can change when one design choice changes. Here the rows are leaf DPs and the columns are leaf FRs. "A" means that the FR is affected through a material effect. "?" means that the effect is unknown, so the tool cannot rule it out. Extra columns list the scenarios to run again and the modules and PVs to review. A second table compares the change scenarios in the model with this result.

**How to use it.** Before a change, find the row of the DP that you plan to change. Run again every scenario in that row. Ask the owners of each "?" FR to confirm or rule out the effect. In the second table, a scenario that expects an FR to stay unaffected while the matrix says "A" is a point to discuss. Either the scenario or an influence is wrong.

**Who reads it.** Software architects who judge the risk of a change. Testers who choose which tests to run again. Program managers who plan change work.

#### Source facts

| Element | Source |
|---|---|
| Matrix rows and cells | `assessment.Impacts`: `Affected` and `Possible` |
| Scenarios, modules, PVs, needs | `Impact.Scenarios`, `Modules`, `PVs`, `Needs` |
| Comparison table | `scenario.vary`, `expect_affected`, `expect_unaffected` |

#### Render

A `grid` table with the rows in design order. Then a table with one row for each change scenario: the DPs it varies, the FRs it expects to change, the FRs it expects to stay the same, and, for each FR, the result from the matrix. Label each comparison result as Derived. The result is "agrees", "differs", or "unknown".

#### Gaps and limits

The comparison is a view of two declared facts. It is not a `validate` finding. Decision Q3 asks whether the assessment should report it as a finding. The matrix uses the same impact sets as the trace map, so the two cannot disagree.

#### Tests

- Each row equals the `Impact` of its DP.
- A scenario that expects an FR to stay the same while the matrix has a material effect shows "differs".
- An FR with an unknown cell shows "unknown", not "agrees".

### D12. Trade study table

#### About this diagram

**What it is.** A trade study compares alternative designs against the same requirements. Here each decision is one trade study. The columns are the candidates. The rows are the FRs that the decision addresses, with their criteria. Each cell names the DPs of the candidate that control the FR. The last rows give the evidence and the information content. Information content measures the chance of success: a lower value is better.

**How to use it.** Compare the candidates row by row. A candidate with no DP for an FR does not address that FR. Compare information content only where the tool shows an estimate from qualified evidence. Without evidence, the table says "Insufficient evidence", and the selection stays provisional. The table does not give weights or scores, because the language does not hold them.

**Who reads it.** Program managers and product owners who approve a decision. Architects who propose the candidates.

#### Source facts

| Element | Source |
|---|---|
| Columns | `decision.candidate`, with `version`, `context`, `workload`, `horizon` |
| Rows | `decision.addresses` and their criteria |
| Cells | `candidate.dps` intersected with the DPs whose `controls` names the row's FR |
| Evidence rows | `candidate.information`, `evidence`, `Result.Candidates` estimates and feasibility |
| Selected | `decision.selected`, `decision.provisional`, `decision.why` |

#### Render

One table for each decision. Mark the selected candidate column. Show inactive DPs with "Inactive". The last row shows feasibility and the information bounds from the assessment, or "Insufficient evidence".

#### Gaps and limits

The table never computes a probability. It shows only the estimates in `Result.Candidates`. It does not rank candidates without them.

#### Tests

- A decision with two candidates gives two columns, and the selected one is marked.
- A candidate with no evidence shows "Insufficient evidence".
- Each cell lists only DPs that the candidate names and that control the row's FR.

### D13. Measures tree

#### About this diagram

**What it is.** A measures tree shows how a high-level acceptance condition breaks down into lower-level conditions. Systems engineering calls the top level measures of effectiveness (MoE) and the lower levels measures of performance (MoP). Here a node is a criterion. A child criterion names its parent through `allocated_from`. A leaf with a metric, an operator, a value, and a unit is a measurable target.

**How to use it.** Start at a criterion of a top FR. Follow it down to the measurable targets that support it. A criterion with a statement but no metric is a target that no one can measure yet. An allocation link records the split. It does not prove that meeting the children meets the parent. That needs a separate argument.

**Who reads it.** Product managers who own the acceptance conditions. Quality leads who define metrics. Systems engineers who track technical performance.

#### Source facts

| Element | Source |
|---|---|
| Nodes | `fr.criterion`, with `statement`, `metric`, `op`, `value`, `unit`, `under`, `origin` |
| Edges | `criterion.allocated_from` |
| Group | The FR that holds each criterion |

#### Render

One SVG made with the tree layout, with roots on the left. Each box shows the criterion ID, its FR, and its predicate, for example `p95_latency <= 200 ms`. A qualitative criterion shows its statement with a dashed "Not measurable" tag. A nested list gives the same tree.

#### Gaps and limits

A criterion with no `allocated_from` is a root, also on a child FR. The tree does not add a link from a child FR's criterion to a parent criterion. The spec says that children inherit context, not criteria.

#### Tests

- Each edge equals one `allocated_from` reference.
- A criterion without a metric shows the "Not measurable" tag.
- The predicate text matches `criterionValue` for each value kind.

### D14. Module junction diagram

#### About this diagram

**What it is.** A module junction diagram shows how the modules combine at each level of the design. It comes from Axiomatic Design, and it is close to a functional flow diagram. Each junction is a circle. S (summation) means that the modules are independent and you can combine them in any order. C (control) means that you must settle them in the design order. F (feedback) means that they adjust each other and the work can repeat. The tool derives the junction from the matrix classification.

**How to use it.** Read the diagram from left to right in the design order. At an S junction, teams can work in parallel. At a C junction, follow the arrows. At an F junction, plan for iteration and check that a coupling record states the stopping rule. A junction is a design sequence. It is a runtime order only when the module contracts require it.

**Who reads it.** Software architects and engineering leads who plan the build and the integration. Readers of the Axiomatic Design literature.

#### Source facts

| Element | Source |
|---|---|
| Junction kind | `JunctionRow`, from `ScopeResult.Classification` |
| Order | `ScopeResult.Order` |
| Modules | `module.realizes` of each DP in the scope |
| Nesting | The parent DP of each scope |
| Coupling record | `coupling.scope` |

#### Render

One SVG made with the graph layout. Each scope is a group: a junction circle with S, C, or F, and the module boxes of its DPs in design order. A nested scope replaces the box of its parent DP. A C junction draws arrows between boxes in order. An F junction draws a return arrow. A DP with no module shows its DP ID in a dashed box. A list gives the same structure.

#### Gaps and limits

An unresolved scope shows a junction with "?" and links to its cause. It does not guess S, C, or F. The diagram uses the same `JunctionRow` data as the engineering tab, so the two cannot disagree.

#### Tests

- Each junction kind equals the engineering tab row for the same scope.
- An unresolved scope shows "?".
- A DP with no module shows a dashed box with its DP ID.

## Integration guide

This guide says how each new diagram joins the page. Follow it for every diagram, so that the page stays one system and not fourteen added parts.

### Rule 1: extend before you add

Four of the new diagrams show data that the page already shows. Extend the existing view in place. Do not draw a second copy.

| New diagram | Existing view | Integration |
|---|---|---|
| D1 Domain mapping matrix | Master matrix on the architecture tab | Add the SE name "Domain mapping matrix (DMM)" and the "About" block to the master matrix. The SE tab shows an index card that links to it. |
| D6 Requirements traceability matrix | Trace table on the trace tab | Keep the trace table, one row for each leaf FR. Add D6 as a second view of the same section, with one row for each criterion. Two buttons switch between the views. Without the script, both views show, one after the other. |
| D11 Change propagation matrix | Change impact table on the trace tab | Add the matrix form above the table, and the scenario comparison below it. Both read the same `assessment.Impact` values as the table and the map. |
| D14 Module junction diagram | Module junctions table on the engineering tab | Add the SVG above the existing table. The table becomes its text form, as C10 requires. |

The other ten diagrams are new views. They live on the SE tab. This changes the default of decision Q6 to "link".

### Rule 2: one tab, one index, many homes

- Add `tabSystems = "systems"` to `presentation.go`. Put it before `tabGuide` in `tabDefinitions`. Set `Reader` to "Systems engineer, tester, reviewer" and `Question` to "Which standard SE diagrams describe this design?". Fill `Has` with the number of diagrams that have data, for example "11 of 14 diagrams have data".
- The tab opens with an index of fourteen cards. Use the `.views` grid and `.view` card from the start tab. Each card shows the diagram number, its SE name, its main reader, and one line from "What it is". The four cards from rule 1 link to their homes on the other tabs. The other cards link to sections on this tab.
- Add the template `assets/tab_systems.tmpl` and include it in `page.html.tmpl` before `tab_guide.tmpl`. In each template, find tabs by ID, not by position. The current `index .Tabs 3` lookups break when a tab is inserted. Replace them with a `tab "architecture"` template function, and change that in the same step.
- Give each section the anchor `se-dN`, for example `se-d2`. Keep the existing anchors. `TestPageLinksResolve` must also cover the new anchors.

### Rule 3: one "About" block, everywhere

- Render "About this diagram" with the shared `about` template. Put it between the section heading and the diagram, collapsed after the first visit of the tab. Use `<details>`, open by default, so that the page stays readable without the script.
- Use the same three `<h4>` headings in every block: "What it is", "How to use it", and "Who reads it". Keep the text in `systems_text.go`. Do not put text in the templates.
- The four extended views from rule 1 get the same block. So each of the fourteen diagrams has its explanation wherever it lives.

### Rule 4: reuse the design tokens

- Use only the existing CSS custom properties: `--paper`, `--panel`, `--ink`, `--muted`, `--line`, `--clay`, `--clay-light`, `--clay-wash`, and `--dark-line`. Add no new color. Clay marks only the selection, cross effects, and focus, as it does today.
- Matrix cells use the existing classes: `x`, `x off`, `zero`, and `unk`. Symbols that the new diagrams add (`V` verify, `R` realize, `O` own, `D` depend, `C` contain, `A` affected) use the `x` style. Their meaning goes in a legend line under each diagram, not in a new color.
- SVG drawings use the classes of the trace map: `.node`, `.edge`, `.edge.influence`, and `.map-label`. Draw SVG drawings on the ivory background, not the dark band, unless the drawing is interactive. Only the trace map uses the dark band today. Keep it as the one feature band of the page.
- Use the spacing steps 4, 8, 12, 16, 24, and 32 px, and the type sizes 12, 14, 16, 20, 24, and 32 px.

### Rule 5: shared components first

Build these before the first diagram. Fix the matching review findings in them, so that the new diagrams get the fixes for free:

| Component | File | Also fixes |
|---|---|---|
| `GridMatrix` and the `grid` template, explanation above the table and a scroll shadow | `matrix.go`, `page.html.tmpl` | R1, R4, R9 |
| One gap form, `{{gap "None"}}`, with no second word | `render.go` | R6 |
| Box text wrapping that keeps the full ID | `layout.go` (`wrap`) | R7 |
| Tree and graph layout | `layout_tree.go`, `layout_graph.go` | Reuses the R7 wrapping |
| Size guard with `<details>` parts | `systems.go` | R3, for large models |

Then move the existing architecture matrices to `GridMatrix`. The architecture tab then proves the component before any new diagram uses it.

### Rule 6: interaction stays optional

- No new diagram needs JavaScript to show its data.
- If a diagram has a selection, it reuses the `impact-data` JSON block and the `select` function. Do not add a second data block. D2 and D11 can highlight the row of the DP selected on the trace map. The selection is kept in the URL hash, for example `#se-d2?dp=dp.x`.
- The script only adds and removes classes and copies prepared text, as today. It computes no design value (C1).

### Rule 7: the legend and the guide grow together

- Add each new symbol to the page legend in the footer only when the page uses it.
- Add one guide entry for each SE term: DMM, DSM, MDM, N² chart, RTM, VCRM, MoE, MoP, and junction. Link each "About" block to its guide entry.
- Add one line to the start tab's "Choose a view" grid for the SE tab, as for every other tab.

### Rule 8: protect the bytes and the build

- Keep each slice in authored or derived order. Do not iterate over a map (C3).
- Regenerate `examples/architecture-diagram.html` in the same commit as each template change. `make diagram-example-check` must pass on every commit.
- Measure the page size of the self-design example before and after. It is 266 KB today. Record the new size in the handoff. If the page grows by more than 50%, collapse the SE sections by default.

### Integration checklist for one diagram

Copy this list into the pull request of each diagram:

- [ ] The builder is a pure function in `internal/diagram`, with a test written first.
- [ ] The view model field is on `Page.Systems`, or on the existing view that rule 1 names.
- [ ] The section has the anchor `se-dN` and an index card.
- [ ] The "About" block has all three parts, with text from `systems_text.go`.
- [ ] Every mention of a record is a `Ref` that links to its home row.
- [ ] Empty data shows one gap card. Unknown cells show `?`.
- [ ] Each SVG has a table or a list with the same data.
- [ ] No tab is wider than the viewport at 390 px and at 1280 px.
- [ ] Text contrast is 4.5:1 or more in every state.
- [ ] The legend and the guide have the new symbols and terms.
- [ ] The committed example page is regenerated, and `make verify` passes.

## Fixtures

The current examples do not use every field that these diagrams read. Add one fixture, `internal/diagram/testdata/systems.ad.hcl`. Keep it small, with about 12 FRs. It must contain:

- two levels of FRs and DPs, one decoupled scope, and one coupled scope with a coupling record
- one unknown cell and one `cells` default
- a rejected DP in a decision with two candidates, one with evidence
- criteria with `allocated_from`, one qualitative criterion, and one criterion with no scenario
- scenarios of all three kinds, one change scenario whose expectation differs from the matrix
- composite and realization modules, a dependency cycle, one module with an omitted `dependencies` field, and one with `dependencies = []`
- PVs, states that two DPs read, and a DP with no realization

Run `adhcl validate` on the fixture under the source profile before you use it. Record the findings in the test as the expected findings.

## Work order and completion gates

| Step | Work | Diagrams | Completion gate |
|---|---|---|---|
| 0 | Fix the review findings R1 to R11, with R1 and R2 first | Existing page | The layout test passes at 390 px and 1280 px on both examples. Map text has a contrast of 4.5:1 or more in every state. |
| 1 | Add the fixture | All | `validate` runs on it, and the test records its findings. |
| 2 | Add the tab, the index, `About`, `GridMatrix`, the `about` and `grid` templates, and the size guard. Look up tabs by ID. Move the architecture matrices to `GridMatrix`. | All | The new tab renders with fourteen index cards and ten empty sections, each with its full "About" part. A test checks that each `About` has three non-empty parts. The architecture tab looks the same as before, apart from the R1 fix. |
| 3 | Add the matrix diagrams | D1, D2, D3, D7, D8, D10, D11 | Each test in the diagram section passes. Each mark traces to one source fact. |
| 4 | Add the table diagrams | D6, D12 | Each test in the diagram section passes. |
| 5 | Add the tree layout and the tree diagrams | D5, D9, D13 | No boxes overlap. Each SVG has a list with the same IDs. |
| 6 | Add the graph layout and the graph diagrams | D4, D14 | No boxes overlap. Each edge ends on its two boxes. Each SVG has a table with the same edges. |
| 7 | Check the page | All | The page for each example and for the fixture opens with no browser error. An automated scan finds no WCAG AA failure. Two runs give equal bytes. |
| 8 | Update documents and examples | All | The documents describe observed output. |

Step 0 comes first, because steps 2 to 6 reuse the fixed components. Step 2 comes next. Steps 3 to 6 are independent of each other, because no diagram builder reads another one. Step 7 needs steps 3 to 6.

Write each test before the code, as `CONTRIBUTING.md` requires. Use `// Given`, `// When`, and `// Then` comments.

## Acceptance scenarios

Add these cases to `cmd/adhcl/diagram_acceptance_test.go`. Each case runs the built binary.

| Scenario | Input | Required outcome |
|---|---|---|
| S1 | `diagram examples/architecture.ad.hcl` | Exit code 0. The page has the `systems` tab with fourteen index cards in order. Each card links to an anchor that exists. |
| S2 | Each of the fourteen diagrams on the S1 page, on any tab | Each diagram has the three headings "What it is", "How to use it", and "Who reads it". |
| S3 | The fixture | No matrix is drawn twice. D11 and the change impact table read the same impact sets. D14 and the junction table give the same junction for each scope. |
| S4 | The fixture with `--candidate` set to the second candidate | D1, D2, D11, and D12 follow the selected candidate. |
| S5 | `examples/minimal.ad.hcl` | Each diagram without data shows its "Not in source" card. No section is missing. |
| S6 | A statement with `<script>` and a quote, used in D4, D5, D9, and D13 | The text is escaped in each SVG and table. |
| S7 | Run S1 two times | The outputs are equal byte for byte. |
| S8 | A model with 60 leaf FRs | The size guard splits D1, D3, and D4. No row or node is lost. |

## Documents

Do this step after the smoke runs, so that the documents describe observed output.

1. Regenerate `examples/architecture-diagram.html`. Make sure that `make diagram-example-check` passes.
2. `README.md`: add "systems engineering" to the list of tabs in the `adhcl diagram` text. Name the fourteen diagrams in one short list.
3. `docs/axiomatic-design.md`: add the FRs of this plan and their matrix.
4. Guide tab: add one entry for each SE term: DMM, DSM, MDM, N² chart, RTM, VCRM, MoE, and MoP.
5. `CHANGELOG.md`: add the new tab under "Added".

## Decisions to confirm

Each decision has a default. The plan uses the default until the owner selects another option.

| ID | Decision | Default | Other options |
|---|---|---|---|
| Q1 | Where the diagrams go | One new "Systems engineering" tab | Put each diagram on the tab of its reader. Or add a `--view se` option that writes a separate page. |
| Q2 | Whether the page always includes the tab | Always | Only with a new `--systems` flag, which keeps the default page smaller |
| Q3 | The D11 comparison of change scenarios with the matrix | A Derived view on the page only | A new assessment finding, which needs a diagnostic code in the spec repository |
| Q4 | Size guard limits | 40 rows, 40 columns, 120 nodes | Other limits after the S8 run |
| Q5 | Relation names in D4 | SysML names in angle quotes | Plain English names such as "verifies" |
| Q6 | Whether D1, D6, D11, and D14 repeat existing views on the SE tab | Link to them, and extend them in place. See the integration guide, rule 1. | Repeat them in full on the SE tab |
| Q7 | Whether the review fixes ship before the diagrams | Yes. R1 and R2 break the WCAG 2.2 AA constraint. | Ship them in the same release as the diagrams |

## Outside this plan

- A change to the `.ad.hcl` language. A diagram that needs a new field needs a spec change first.
- The partial diagrams: FFBD, internal block diagram, IDEF0, context diagram, and FMEA.
- Export to SysML, XMI, ReqIF, CSV, or image files.
- An extension owner for SE views. The `html_view/1` contract admits only `layout` and `show_source_locations`.
- Test runs and live observations. The page shows `evidence` blocks only.
- Interactive editing, filtering, and sorting.

## Definition of complete

The work is complete only when all eight gates pass and `make verify` passes. The command must run from a built binary. The page for each example and for the fixture must open in a browser with no error. Each of the fourteen sections must show its "About" part. The handoff must record the commands that were run, the observed outcomes, and each limit that was not exercised.

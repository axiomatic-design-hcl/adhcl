# adhcl diagram implementation plan

This plan records the work to add the `adhcl diagram` command. The command reads a design model and writes one HTML page that shows the design as a diagram. Validation exposes the selected snapshot.

The sample page is [architecture-diagram.html](architecture-diagram.html). A person wrote it by hand for [examples/architecture.ad.hcl](../../examples/architecture.ad.hcl). The command must generate a page of the same kind from any valid model.

Status: v0.2.0 ships the command. The released page has one tab for each reader instead of the sections in this plan. The plan stays as the record of the first design. See [CHANGELOG.md](../../CHANGELOG.md) for the released behavior.

## Terms

- A functional requirement (FR) states what the system must do.
- A design parameter (DP) is the mechanism that controls one FR.
- A process variable (PV) says how a DP is built or delivered.
- A sibling set is the group of FRs and DPs that have the same parent.
- A view model is the data that the page shows, without HTML.
- A gap is data that the model does not contain. A gap is not a failure.

## Baseline

The tool has two commands today: `version` and `validate`. These facts come from the current code:

| Fact | Location |
|---|---|
| The command router accepts only `version` and `validate`. | `internal/cli/arguments.go`, `internal/cli/cli.go` |
| `validate.Run` returns a `validate.Result` with findings, scopes, candidates, owners, and the selected snapshot. | `internal/validate/validate.go`, `internal/validate/result.go` |
| The selected snapshot gives access to the resolved graph. | `internal/selection/selection.go` |
| The text report is the only presentation. | `internal/report` |
| The model has 21 block kinds, with hierarchy, constraints, decisions, evidence, and PVs. | `internal/model/core.go` |
| The `html_view` owner accepts a `layout` value but renders nothing. | `internal/owners/registry.go` |
| The README and the design document say that the tool does not render HTML. | `README.md:3`, `docs/axiomatic-design.md:195` |
| The module has two direct dependencies: `hcl/v2` and `go-cty`. | `go.mod` |

The [v0.1.0 plan](implementation-plan.md) excludes HTML output. This feature is new scope. It does not change the behavior of `version` or `validate`.

## Command contract

```
adhcl diagram [--profile source|architecture] [--candidate REF] [--context REF]
              [-o FILE | --output FILE] [--] PATH
```

1. `PATH` accepts the same three forms as `validate`: one `.ad.hcl` file, a directory, or a manifest.
2. `--profile`, `--candidate`, and `--context` have the same meaning and the same defaults as in `validate`.
3. If `-o` or `--output` is absent, the command writes the HTML to standard output.
4. If `-o FILE` or `--output FILE` is present, the command writes the HTML to `FILE` and writes nothing to standard output. `--output=FILE` is also valid.
5. The command replaces an existing `FILE`. It does not create missing parent directories.
6. `diagram --help` and `diagram -h` print the help text when given as the only argument.

Exit codes:

| Code | Meaning |
|---|---|
| 0 | The command delivered the complete HTML page. |
| 1 | The command delivered no page. The core assessment was blocked, or the output failed. |
| 2 | The command line was invalid. |

If the core assessment is blocked, the command writes the findings to standard error in the text format of `validate`. It writes no HTML and it does not create or change `FILE`.

Exit code 0 does not mean that the design is valid. A model with error findings and a complete core assessment still produces a page. The page shows the errors. Use `validate` as the gate in CI. See decision Q2.

## Needs and constraints

| ID | Need | Why |
|---|---|---|
| N1 | Generate the diagram from any `.ad.hcl` model with one command. | The sample page is handwritten and goes out of date. |
| N2 | Send the page to a file or to standard output. | A person saves a file. A script pipes the output. |
| N3 | Show the same findings as `validate`. | Two different answers for one model destroy trust. |
| N4 | Keep declared facts, tool findings, and gaps visibly apart. | A reader must not take a gap for a pass or for a failure. |
| N5 | See what a change to one DP touches. | This is the main question that the diagram answers. |

| ID | Constraint | Origin |
|---|---|---|
| C1 | The page does not derive design findings. The Go code derives them. | Audit recommendation, three layers |
| C2 | The page is one self-contained file. It loads nothing from the network. | Sample page |
| C3 | The same input and the same tool version produce the same bytes. | Proposed, for review diffs and tests |
| C4 | Add no new Go module dependency. Use `html/template` and `embed`. | Proposed, same reasoning as the v0.1.0 plan |
| C5 | The page never shows data that the model does not contain. | N4 |
| C6 | Every string from the model is escaped. A model cannot inject markup or script. | Proposed, security |
| C7 | The visual style follows the `claude` design skill. | User request |
| C8 | The fixed text on the page follows the `simple-english` skill. | User request |
| C9 | The page meets WCAG 2.2 AA and works with a keyboard. | `claude` design skill |
| C10 | `validate` output and exit codes do not change. | Proposed, compatibility |

## Design

FR0 is the parent outcome: deliver an HTML diagram of one design model. DP0 selects a separate `diagram` command path that reuses the validation pipeline. The FRs below arise within DP0.

| FR | Need | Proposed acceptance criterion | Paired DP and responsibility |
|---|---|---|---|
| FR1 | N1, N2 | Each valid command line gives one request. Each invalid command line gives exit code 2. | DP1: the diagram argument grammar in `internal/cli`. |
| FR2 | N3 | The findings, scopes, and candidates equal those of `validate` for the same request. | DP2: `validate.Run`, with the snapshot exposed in `validate.Result`. |
| FR3 | N4, N5 | Each declared record appears once in the view model. Each absent block kind appears as a gap. Each impact set equals the assessed cells. | DP3: a pure view-model builder, `diagram.Build`. |
| FR4 | N4 | The page shows the complete view model, passes the checks for C2, C6, C7, and C9, and has no browser error. | DP4: an embedded template with CSS and a small script, `diagram.Render`. |
| FR5 | N2 | The complete page reaches the selected target, or no page reaches it. | DP5: an output sink that buffers the page and replaces the file in one step. |

### Influence matrix

Rows are FRs. Columns are DPs. `X` is a material influence. `0` is an absence of influence. This matrix is a proposal. It has no reviewed support.

| | DP1 arguments | DP2 pipeline | DP3 builder | DP4 template | DP5 sink |
|---|---|---|---|---|---|
| FR1 accept the invocation | X | 0 | 0 | 0 | 0 |
| FR2 provide the assessed design | X | X | 0 | 0 | 0 |
| FR3 select what the page shows | 0 | X | X | 0 | 0 |
| FR4 present the page | 0 | 0 | X | X | 0 |
| FR5 deliver the bytes | X | 0 | 0 | 0 | X |

Reasons for the entries off the diagonal:

- DP1 to FR2: the arguments select the path, the profile, the candidate, and the context.
- DP2 to FR3: the builder reads the result types. A change to those types changes the view model.
- DP3 to FR4: the template reads the view model. A new field needs new markup.
- DP1 to FR5: the arguments select the output target.

Reasons for the important zeros:

- DP4 to FR5: the sink handles the page as bytes. It does not read the content.
- DP4 to FR3: the template has no logic that selects or computes design data. Constraint C1 enforces this.
- DP3 to FR2: the builder does not change the result.

The matrix is a triangle, so the structure is decoupled. Establish the DPs in the order DP1, DP2, DP3, DP4. DP5 depends only on DP1. You can build DP5 at the same time as DP3 and DP4.

### Children of FR3

DP3 creates these sibling outcomes. They are decoupled. The trace map uses the impact sets to place scenarios. The health summary uses the gap list from the sections. Build the impact sets before the trace map and the sections before the health summary.

| FR | Outcome | DP |
|---|---|---|
| FR3.1 | Place each need, FR, DP, module, PV, and scenario in the trace map, with its links. | DP3.1: a column and row layout function. |
| FR3.2 | Give the set of items to test again for each DP. | DP3.2: an impact function over the master matrix. |
| FR3.3 | Give the content of each section from the block kinds that belong to it. | DP3.3: one mapper for each section. |
| FR3.4 | Give the health summary: what holds, what needs a review, and the gaps. | DP3.4: a summary function over the findings and the block counts. |

| | DP3.2 impact | DP3.1 layout | DP3.3 sections | DP3.4 summary |
|---|---|---|---|---|
| FR3.2 impact sets | X | 0 | 0 | 0 |
| FR3.1 trace map | X | X | 0 | 0 |
| FR3.3 section content | 0 | 0 | X | 0 |
| FR3.4 health summary | 0 | 0 | X | X |

### Module contracts

| Module / FR | Input and output | Failure behavior |
|---|---|---|
| `internal/cli` / FR1 | Arguments to a `Request` with `Output`. | Return a usage error. Never open a file. |
| `internal/validate` / FR2 | `Request` to `Result`. `Result.Snapshot` is nil when the core is blocked. | Unchanged. Return findings, never panic. |
| `internal/diagram` build / FR3 | `validate.Result` to `diagram.Page`. No I/O. | Return an error if `Snapshot` is nil. |
| `internal/diagram` render / FR4 | `diagram.Page` and an `io.Writer` to HTML bytes. | Return the template or write error. |
| `cmd/adhcl` sink / FR5 | Bytes and a target to a file or to standard output. | Remove the temporary file. Leave the old `FILE` unchanged. |

### Alternatives and the Information Axiom

No success probability exists for these choices. No information score is given. The comparison is qualitative and the selection is provisional.

| Choice | Alternative | Decision and risk |
|---|---|---|
| Layout in Go, static SVG | Layout in browser script | Use Go. The page then reads correctly without script, and the bytes are stable (C3). The risk is text that does not fit a box. Step 4 handles it. |
| `html/template` with `embed` | A string builder | Use `html/template`. It escapes by context (C6). A string builder risks one missed escape. |
| Hand-drawn SVG | Mermaid or Graphviz | Use hand-drawn SVG. A library adds a dependency or a network load (C2, C4), and it does not support the impact highlight. |
| Reuse `validate.Run` | A second assessment in the diagram package | Reuse. A second assessment can give a different answer (N3). |
| Buffer, then write | Stream the template to the target | Buffer. A template error after a partial write leaves a broken file (FR5). |

## Page contract

The page has one main diagram and seven sections. Each section shows declared data if the model has it. If the model has none, the section shows one dashed "Not in source" card. The sample page shows the form of each part.

| Part | Content | Model source |
|---|---|---|
| Top bar | Design ID and links to each part. | `design` |
| Introduction | Design ID, level, and boundary. | `design` |
| Health | Error count, warning count by code, and the list of gaps. | `Result.Findings`, block counts |
| Map | Trace columns, links, cross influences, and the impact of each DP. | needs, leaf FRs, leaf DPs, modules, PVs, scenarios, master matrix |
| 01 Customer needs | Statement, reason, origin, and stakeholder. | `need`, `stakeholder`, `source` |
| 02 Requirements and constraints | FR, criteria, linked needs, linked scenarios, constraints, assumptions. | `fr`, `criterion`, `constraint`, `assumption` |
| 03 Design decomposition | The FR and DP tree, level by level. | `fr.parent`, `Graph.SiblingSets` |
| 04 Architecture and independence | One matrix and one verdict for each sibling set and for the master scope. State ownership. Coupling records. | `Result.Scopes`, `expect`, `coupling`, `state`, `cells` |
| 05 Design evaluation | Decisions, candidates, feasibility, and evidence estimates. | `decision`, `Result.Candidates`, `evidence`, `context`, `workload` |
| 06 Implementation and process | Module contracts, flows, PVs, and owner results. | `module`, `flow`, `pv`, `Result.Owners` |
| 07 Verification | Criterion to scenario coverage, scenario steps, and the findings list. | `scenario`, `evidence`, `Result.Findings` |

Rules for every part:

1. Each item has one of three labels. "Source" marks text from the model. "adhcl" marks a finding from the tool. "Not in source" marks a gap.
2. The form of the label carries the meaning: a solid border, a filled box, or a dashed border. Color does not carry it.
3. Each finding shows its code, its severity, and its source location as `file:line`.
4. A matrix cell shows `X`, `0`, or `?`. A `?` is an unknown effect. The page never shows an unknown as `0`.
5. A classification of `unresolved` shows as "Unresolved", with the cause from the findings. It does not show as a failure or as a pass.
6. A scenario without a linked `evidence` block shows "No recorded result".
7. The Information Axiom part shows an estimate only when `Result.Candidates` has one. If it has none, the part shows "Insufficient evidence".

### What the sample has that the generator cannot produce

The sample contains text that a person wrote. The generator has no source for it. Use these rules:

| Sample content | Generated content |
|---|---|
| The headline "Count the right scans. See what is not proven." | The design ID. The boundary is the lead paragraph. |
| Short box titles such as "Accept 12-digit scans". | The record ID in the mono font, then the statement, wrapped to two lines and cut with "…". The full statement is in an SVG `<title>` and in the detail panel. |
| Buttons "Change the digit rule" and "Change the counting rule". | No buttons with invented names. The reader selects a DP box. The page selects the first DP that changes more than one FR on load, if one exists. |
| Hand-placed curves and labels. | Step 4 computes them. |
| The pasted `adhcl validate` output. | A findings table from `Result.Findings`. |

### Style contract

Take the tokens and the components from the sample page. They follow the `claude` design skill.

- Paper `#FAF9F6`, ink `#141413`, and one clay accent. Clay marks only the selection, the cross influences, and the focus ring.
- No shadows, no gradients, and square corners. The primary button has a flat top and a round bottom.
- Ivory sections and near-black feature bands alternate. The serif font is used only on the dark bands.
- Type sizes are 12, 14, 16, 20, 24, and 32 pixels. Spacing steps are 4, 8, 12, 16, 24, and 32 pixels.
- The fonts are system fallbacks. The page embeds no font file (C2).
- Body text has a contrast ratio of 4.5 to 1 or more. Each target is 44 pixels or more in one dimension. Focus is always visible.
- The script only adds and removes classes and fills the detail panel. It reads the impact sets from one embedded JSON block. It computes nothing about the design.
- With the script off, all the content stays readable. Only the highlight is absent.

## Work order and completion gates

| Step | Responsibility | Design coverage | Completion gate |
|---|---|---|---|
| 1 | Route the command and parse its arguments | FR1 | Each case in the command contract gives the stated request or usage error. |
| 2 | Expose the snapshot from validation | FR2 | Complete. The result exposes the snapshot when the core assessment completes. |
| 3 | Build the view model | FR3.2, FR3.3, FR3.4 | Each block kind maps to a part. Impact sets and gaps are correct on the fixtures. |
| 4 | Lay out the trace map | FR3.1 | No box overlaps another box. Each link ends on its two boxes. |
| 5 | Render the page | FR4 | The page for each example opens with no browser error and passes the style rules. |
| 6 | Deliver the output | FR5 | Standard output, file output, and each failure case behave as stated. |
| 7 | Prove the complete command | FR0 | A built binary passes scenarios D1 to D12. |
| 8 | Update documents and examples | N1 | The documents describe observed output. The sample page is generated. |

Steps 3 and 4 need step 2. Step 5 needs steps 3 and 4. Step 6 needs only step 1. Do not close a gate by removing a part of the page.

## 1. Route the command and parse its arguments

1. Add `diagram` to the accepted commands in `parseArguments`.
2. Add an `Output` field to `cli.Request`. `Output` is empty for standard output.
3. Use separate flag tables for `validate` and `diagram`. Both commands share the parser for common argument rules. `validate -o x` stays a usage error.
4. Keep the existing rules: no repeated flag, no empty value, `--` ends the options, and a path that starts with a dash works after `--`.
5. Accept `-o FILE`, `--output FILE`, and `--output=FILE`. Reject `-oFILE`.
6. Add a `DiagramFunc` parameter to `cli.Run`, or replace the two function parameters with one small interface. Use the form that keeps `cli` free of imports from other internal packages.
7. Update `helpText` with the `diagram` usage line, the command, and the `--output` option.
8. Update the "expected version or validate command" message to name all three commands.

Tests: extend the table tests in `internal/cli/arguments_test.go` with the new valid and invalid forms.

## 2. Expose the snapshot from validation

1. Run a reference search on `validate.Result` and `coreResult` before the change.
2. Add `Snapshot *selection.Snapshot` to `validate.Result`. Set it from `core.snapshot` in `validate.Run`. It is nil when the core is blocked.
3. Do not change `report.Render`. It ignores the new field.
4. Make sure that the full `validate` test suite and the acceptance tests pass with no changed expectation.

The snapshot gives the graph, the active FRs and DPs, the active modules, and the applied influences. The builder needs no other access to the pipeline.

## 3. Build the view model

Create the package `internal/diagram`. `Build(result validate.Result) (Page, error)` is a pure function. It performs no I/O and holds no global state.

1. Define `Page` and its parts as plain structs with exported fields. Use strings, numbers, and slices only, so the template needs no logic. Keep authored order in every slice (C3).
2. Give each item its label kind: source, tool, or gap.
3. Write one mapper for each part in the page contract. Each mapper returns its items, or one gap record when the model has no data for the part.
4. Mark inactive records. A DP that is rejected, superseded, or outside the snapshot stays visible in section 05 and is absent from the map.
5. Build the health summary:
   - Count the errors and the warnings. Group the warnings by code.
   - Derive "what holds" only from facts in the result: each scope with a classification that is not `unresolved`, and each `expect` block with no failed-expectation finding.
   - List one gap for each absent block kind that a section needs: child levels, constraints, decisions, evidence, PVs.
6. Build the matrices. For each `ScopeResult`, read each coordinate through the matrix cell accessor, so that scoped defaults and implicit unknowns resolve the same way as in the assessment. Keep the mechanism or reason text and the source location for each cell. Mark conflicts.
7. Build the impact sets from the master scope. For each active DP:
   - Take each FR whose cell is `material`. Put each FR whose cell is `unknown` in a separate "possibly affected" list.
   - Add the needs of those FRs, the scenarios that verify their criteria, and the modules that list those scenarios or criteria in `acceptance_checks`.
   - Add the modules and PVs that realize the DP.
8. Build the trace sets for the other item kinds. Follow the declared links in both directions: need, FR, DP, module or PV, scenario.
9. Convert each source range to `file:line` text. Use the path as given in the input, not an absolute path (C3).

Tests in `internal/diagram`: use small models as fixtures and assert on the view model.

- The impact set of each DP in `examples/architecture.ad.hcl`. `dp.batch_counter` touches one row. `dp.scan_admission` touches both rows.
- An unknown cell appears in "possibly affected" and shows as `?`.
- A model with constraints, a decision with two candidates, and a PV produces no gap for those parts.
- A model with a parent FR and child FRs produces a tree of two levels and one matrix for each sibling set.
- A rejected DP is absent from the map and present in section 05.
- `Build` returns an error when `Snapshot` is nil.

## 4. Lay out the trace map

The layout is a pure function from the map items to coordinates. It runs in Go, so the SVG is static.

1. Use five fixed columns: needs, FRs, DPs, modules and PVs, scenarios. The column width and the box size come from the sample: boxes of 190 by 96 units, with a gap of 40 units.
2. The map shows leaf FRs and leaf DPs only. Section 03 shows the parents. Group the leaves by parent, in authored order, and draw one thin rule between groups.
3. Give each leaf FR and its paired DP the same row. If a scope has no unique pairing, give each FR and each DP its own row in authored order.
4. Place each need, module, PV, and scenario on the row of the first item that it links to. If the row position is taken, use the next free position in that column. The SVG height grows with the longest column.
5. Draw a link on one row as a straight line. Draw a link between rows as a cubic curve that leaves the right side of the source box and enters the left side of the target box.
6. Draw each material cell off the pairing as a dashed clay curve from the DP box to the FR box. Draw each unknown cell the same way with a `?` label.
7. Draw a module dependency as a vertical line in the module column.
8. Wrap each statement to two lines of 26 characters at word limits. Cut longer text with "…". Wrap by character count, because Go cannot measure the rendered font.
9. Use integer coordinates. Sort nothing by map iteration order (C3).

Tests: assert that no two boxes in a column overlap, that each link starts and ends on the edges of its boxes, and that two runs give equal output. Do not assert exact coordinates.

Known limit: the map is readable up to about 30 leaf FRs. Record the observed limit in the handoff. Collapsible groups are outside this plan.

## 5. Render the page

1. Put `page.html.tmpl`, `page.css`, and `page.js` in `internal/diagram/assets`. Load them with `embed`. Parse the template once.
2. Start from the sample page. Move its CSS and script into the asset files with no style change. Replace each handwritten value with a template action.
3. Insert the CSS and the script inline in the page (C2). Write the impact sets and the detail texts as one JSON block in a `<script type="application/json">` element. Let `html/template` escape every value. Do not use `template.HTML`, `template.JS`, or `template.CSS` on model text (C6).
4. Remove the model-specific data from the sample script. The script reads the JSON block and uses `data-id` attributes. It contains no record ID.
5. Write the fixed text once, in the template. Follow the `simple-english` rules: short sentences, active voice, one meaning for each word, and each term defined at its first use.
6. Use one `<h1>`, one `<h2>` for each part, a skip link, `aria-live` on the detail panel, `aria-pressed` on each selectable box, and a visible focus state.
7. Add `<meta name="generator" content="adhcl VERSION">`. Add no date and no time (C3).
8. `Render(w io.Writer, page Page) error` executes the template into `w`.

Checks for this step:

- Run the command on the three examples and open each page in a browser. Make sure that the console shows no error.
- Select each DP on the architecture page and compare the highlight with the impact set from step 3.
- Compare the generated architecture page with the sample by eye, at 1280 pixels and at 390 pixels of width.
- Run an automated accessibility scan and fix each AA failure.

## 6. Deliver the output

1. In `cmd/adhcl`, add `runDiagram`. It calls `validate.Run`, then `diagram.Build`, then `diagram.Render` into a `bytes.Buffer`.
2. If `Result.Snapshot` is nil, render the result with `report.Render` to standard error and return exit code 1. Write nothing to standard output and do not touch `FILE`.
3. If `Output` is empty, write the buffer to standard output. Report a write error through the existing delivery-failure path.
4. If `Output` is set, create a temporary file in the directory of `FILE`. Write the buffer, close the file, and rename it to `FILE`. If a step fails, remove the temporary file and return exit code 1.
5. Print nothing to standard output when `Output` is set. Print warnings to no stream. The page contains them.

## 7. Prove the complete command

Add cases to `cmd/adhcl/acceptance_test.go`. Each case runs the built binary. Follow the existing rule: test behavior that a user can see, not wording or forwarding.

| Scenario | Change or input | Required outcome |
|---|---|---|
| D1 | `diagram examples/architecture.ad.hcl` | Exit code 0. Standard output is one HTML document. Standard error is empty. |
| D2 | The same input with `-o`, with `--output`, and with `--output=` | Exit code 0. The file has the same bytes as D1. Standard output is empty. |
| D3 | Run D1 two times | The two outputs are equal byte for byte. |
| D4 | `-o` to an existing file | The file has the new content. |
| D5 | `-o` to a path in a missing directory | Exit code 1. No file exists. Standard error names the path. |
| D6 | Standard output is `/dev/full` | Exit code 1. Standard error reports the delivery failure. |
| D7 | A model with a syntax error, with `-o` to an existing file | Exit code 1. The findings are on standard error. The file is unchanged. |
| D8 | `diagram` with no path, `-o` with no value, `-o` two times, and `validate -o x` | Exit code 2 for each. |
| D9 | A statement that contains `<script>alert(1)</script>` and a quote | The page contains the escaped text. The page contains no new `script` element. |
| D10 | A model with a failed expectation | Exit code 0. The page shows the error finding. |
| D11 | A manifest and a directory as `PATH` | Exit code 0. The findings in the page equal the findings of `validate` for the same path. |
| D12 | `--candidate` and `--context` on the self-design example | The page shows the selected candidate and context. The matrices equal the `validate` scopes. |

Add `diagram` runs for the three examples to the `validate-examples` target, with the output sent to a temporary directory.

## 8. Update documents and examples

Do this step after the smoke runs, so that the documents describe observed output.

1. Regenerate `examples/architecture-diagram.html` with the command. From then on, a person does not edit the file.
2. Add a Makefile target that generates the example again and fails if the committed file differs.
3. `README.md`: remove "It does not render HTML". Add the `diagram` command, its options, and its exit codes. State that exit code 0 does not mean a valid design.
4. `examples/README.md`: add the `diagram` command line for each example.
5. `docs/axiomatic-design.md`: add the diagram FRs and the matrix from this plan. Replace the sentence that puts HTML delivery outside the release.
6. `CHANGELOG.md`: add the command under "Added". Record the new `validate.Result.Snapshot` field under "Changed".
7. Ask the owner what to do with `examples/architecture.html` and `examples/architecture-workspace.html`. They are earlier handwritten samples and are not tracked in git.

## Decisions to confirm

Each decision has a default. The plan uses the default until the owner selects another option.

| ID | Decision | Default | Other option |
|---|---|---|---|
| Q1 | Input forms for `PATH` | The same three forms as `validate` | Only one `.ad.hcl` file, as the request text says |
| Q2 | Exit code when the page shows error findings | 0, because the page was delivered | 1, the same as `validate` |
| Q3 | The `html_view` layout value in a manifest | Ignored in this plan. The part order is fixed. | `matrices_first` moves section 04 above section 01 |
| Q4 | Default profile | `source`, the same as `validate` | `architecture` |
| Q5 | The release that gets the command | The first release after v0.1.0 | v0.1.0, which has no tag yet |

## Outside this plan

- A change to the `.ad.hcl` language. The page shows only what the current schema holds.
- Test execution results from an external test runner. The page shows `evidence` blocks only.
- Change history and a comparison of two revisions.
- A watch mode, a server, and a live reload.
- Output formats other than HTML.
- Themes and a dark mode setting.

## Definition of complete

The feature is complete only when all eight gates pass. The command must run from a built binary, not only compile. `make verify` must pass. The generated page for each of the three examples must open in a browser with no error. The committed sample must equal the generated output. The handoff must record the commands that were run, the observed outcomes, and each limit that was not exercised.

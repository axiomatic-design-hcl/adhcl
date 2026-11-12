# Examples

These examples show the input format for `adhcl`. Each one describes a small, made-up design. None of them describes a real system or a measured result.

Build the tool to `bin/adhcl` first. Run every command from the repository root.

## minimal.ad.hcl

A thermostat with two root requirements, two design parameters, and explicit influences. It has no module contracts or scenarios, so it does not pass the architecture profile.

Command: `./bin/adhcl validate examples/minimal.ad.hcl`

Expected exit code: 0.

Expected warnings: `support_gap` for the influence claims, because the example has no reviewed support. The source profile also reports `incomplete_module_contract` and `missing_leaf_scenario` as warnings, because the example has no modules or scenarios.

## architecture.ad.hcl

A pallet batch counter with two root requirements, two design parameters, one module for each design parameter, and one triangular mapping.

Command: `./bin/adhcl validate --profile architecture examples/architecture.ad.hcl`

Expected exit code: 0.

Expected warnings: `support_gap` for the authored claims.

## project/model.ad.project.hcl

A badge gate in a project manifest. The manifest lists one core file, `design.ad.hcl`, and the three standard extensions. Each extension has one attachment.

Command: `./bin/adhcl validate examples/project/model.ad.project.hcl`

Command for the architecture profile: `./bin/adhcl validate --profile architecture examples/project/model.ad.project.hcl`

Expected exit code: 0 for both commands.

Expected warnings: `support_gap` for the authored claims, and `unavailable_external_input` for the binding attachment and the behavior attachment. The tool receives no observations and no behavior model, so the results for those two attachments stay unknown.

A zero exit code means that validation completed under the chosen profile without error findings. It does not prove that the design is independent or that the system is built.

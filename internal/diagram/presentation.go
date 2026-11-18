package diagram

import "github.com/axiomatic-design-hcl/adhcl/internal/assessment"

// Tab IDs. The order of tabDefinitions is the order of the tab bar.
const (
	tabStart        = "start"
	tabBusiness     = "business"
	tabProduct      = "product"
	tabArchitecture = "architecture"
	tabEngineering  = "engineering"
	tabTrace        = "trace"
	tabRisks        = "risks"
	tabGuide        = "guide"
)

// Labels say where a claim on the page comes from.
const (
	labelSource   = "Source"
	labelInferred = "Inferred"
	labelTool     = "adhcl"
	labelDerived  = "Derived"
	labelGap      = "Not in source"
)

// Tab is one reader view of the page.
type Tab struct {
	ID, Title, Reader, Question, Kicker, Heading, Lede string
	Has                                                string // counts for the start card
}

var tabDefinitions = []Tab{
	{ID: tabStart, Title: "Start"},
	{ID: tabBusiness, Title: "Business view", Reader: "Business lead", Question: "Whose problem does the product solve, and what limits the solution?", Kicker: "Business view · Customer domain", Heading: "Purpose and scope", Lede: "Who has the problem, what they need, and what limits the solution."},
	{ID: tabProduct, Title: "Product view", Reader: "Product manager", Question: "What must the product do, and how do we accept it?", Kicker: "Product view · Functional domain", Heading: "Capabilities and acceptance", Lede: "What the product must do, and the conditions that make the result acceptable."},
	{ID: tabArchitecture, Title: "Architecture view", Reader: "Software architect", Question: "Which design choices control which outcomes, and where do they interfere?", Kicker: "Architecture view · Physical domain", Heading: "Design choices and independence", Lede: "Which design choice controls which outcome, and where the choices interfere."},
	{ID: tabEngineering, Title: "Engineering view", Reader: "Software engineer", Question: "What do we build, and in which order does it run?", Kicker: "Engineering view · Process domain", Heading: "Components and operating flow", Lede: "What the team builds for each design choice, and in which order it runs."},
	{ID: tabTrace, Title: "Trace and impact", Reader: "Architect, tester", Question: "If one design choice changes, what else can change?", Kicker: "Trace and impact · All domains", Heading: "Trace map and change impact", Lede: "Follow the links from needs to requirements, design choices, and modules. See what a change can affect."},
	{ID: tabRisks, Title: "Risks and evidence", Reader: "Program manager", Question: "What is still open, and which proof is missing?", Kicker: "Risks and evidence", Heading: "Open questions and remaining risk", Lede: "What the model does not yet prove, and what a change can break."},
	{ID: tabGuide, Title: "Guide", Reader: "Every reader", Question: "What do FR, DP, X, and \"decoupled\" mean?", Kicker: "Appendix", Heading: "Guide to Axiomatic Design and this page", Lede: "A short cheat sheet. It explains the terms, the matrix symbols, and the labels on every tab."},
}

// classificationMeaning tells the team what a matrix classification means for its work.
var classificationMeaning = map[assessment.Classification]string{
	assessment.Uncoupled:  "Each DP affects only its own FR. You can set the DPs in any order.",
	assessment.Decoupled:  "The matrix is triangular. Set the DPs in the order below, and each FR stays under control.",
	assessment.Coupled:    "The DPs affect the FRs of other DPs in a cycle. A change to one DP needs a check of the other FRs, and tuning can repeat.",
	assessment.Unresolved: "Some cells have no declared effect (?), or the FRs and DPs do not pair. The tool cannot classify this matrix until the model declares them.",
}

// junctionMeaning explains the module junction of each classification.
var junctionMeaning = map[string]string{
	"S": "S (summation): run the modules in any order and combine the results.",
	"C": "C (control): settle and run the modules in the design order.",
	"F": "F (feedback): the modules adjust each other, so the work can repeat.",
}

// CodeGuide explains one finding code.
type CodeGuide struct{ Code, Meaning, Action string }

// codeGuides holds a plain meaning and an action for every diagnostic code.
var codeGuides = map[string]CodeGuide{
	"input_failure":                {Meaning: "The tool cannot read an input file, or the file is outside the allowed set.", Action: "Make sure that the path exists and that each file ends in .ad.hcl."},
	"invalid_syntax":               {Meaning: "A file is not valid HCL, or an expression is not allowed.", Action: "Correct the syntax at the location."},
	"invalid_structure":            {Meaning: "A block or field does not match the schema, for example an unknown field.", Action: "Correct the field name, type, or value at the location."},
	"duplicate_identity":           {Meaning: "Two records or files use the same name.", Action: "Give each record a unique label."},
	"invalid_reference":            {Meaning: "A reference names a record that does not exist or has the wrong kind.", Action: "Correct the reference."},
	"invalid_hierarchy":            {Meaning: "The FR and DP tree has a cycle, an unreachable record, or a wrong parent.", Action: "Correct the parent and controls links."},
	"invalid_selection":            {Meaning: "The selected candidate or conditions contradict each other.", Action: "Select one valid candidate and context."},
	"conflicting_influence":        {Meaning: "Two records state both an effect and no effect for the same cell.", Action: "Keep one effect for the cell, or add conditions that separate them."},
	"ambiguous_pairing":            {Meaning: "The DPs can pair with the FRs in more than one way.", Action: "Change controls so that each FR has exactly one DP."},
	"uncontrolled_requirement":     {Meaning: "An FR has no DP that controls it.", Action: "Add a DP that controls the FR."},
	"redundant_control":            {Meaning: "A DP has no FR of its own in the pairing.", Action: "Remove the extra DP or pair it with an FR."},
	"ineffective_control":          {Meaning: "A DP has no effect on the FR that it controls.", Action: "Choose a DP that changes its FR."},
	"conflicting_ownership":        {Meaning: "Two leaf DPs own the same state, or two modules realize the same DP.", Action: "Give each state and each DP one owner."},
	"incomplete_criterion":         {Meaning: "A requirement or criterion has no metric, operator, and value. Nobody can test it with a number.", Action: "Add a metric, an operator, a value, and a unit."},
	"incomplete_module_contract":   {Meaning: "A leaf DP has no module, or its module contract is incomplete, for example no inputs.", Action: "Add the module or the missing contract fields."},
	"missing_leaf_scenario":        {Meaning: "A leaf requirement has no scenario with steps that tests its criteria.", Action: "Add a scenario that verifies the criteria of the leaf."},
	"incomplete_feedback_contract": {Meaning: "A coupled scope repeats work without a complete coupling record.", Action: "Add the iteration bound, convergence, stopping rule, and failure behavior."},
	"unresolved_refinement":        {Meaning: "A parent-level effect has no matching effects between the child FRs and child DPs.", Action: "Declare the child-level effects that explain the parent effect."},
	"unresolved_applicability":     {Meaning: "A condition or a rejected assumption leaves a required link unknown.", Action: "Resolve the condition or review the assumption."},
	"unresolved_influence":         {Meaning: "A matrix has cells with no declared effect, so the tool cannot classify it.", Action: "Declare the effect (material or none) for each ? cell."},
	"shared_resource_review_gap":   {Meaning: "DPs share state, a resource, or a PV, and the model does not assess the effect.", Action: "Declare the effect between the DPs that share the item."},
	"support_gap":                  {Meaning: "A claimed effect or assumption has no reviewed support.", Action: "Add a review, a source, or evidence for the claim."},
	"unresolved_feasibility":       {Meaning: "A candidate has no conclusive evidence that it meets all criteria and constraints.", Action: "Record trial evidence for the candidate."},
	"infeasible_candidate":         {Meaning: "Evidence shows that a candidate fails a required criterion or constraint.", Action: "Change or reject the candidate."},
	"invalid_evidence":             {Meaning: "An evidence record breaks the evidence rules, for example more failures than trials.", Action: "Correct the evidence record."},
	"incomparable_evidence":        {Meaning: "The candidates have no common outcomes or conditions, so the tool cannot compare them.", Action: "Record evidence for the same outcomes and conditions."},
	"failed_expectation":           {Meaning: "An expected classification or order does not match the assessed matrix.", Action: "Correct the design or the expectation."},
	"invalid_attachment":           {Meaning: "An extension file breaks its contract.", Action: "Correct the extension file."},
	"unavailable_owner":            {Meaning: "No tool is available for a required extension.", Action: "Install the owner tool or remove the extension."},
	"owner_failure":                {Meaning: "The tool for an extension cannot finish its work.", Action: "Read the message and correct the extension input."},
	"duplicate_binding":            {Meaning: "A binding repeats the address of a PV.", Action: "Remove the repeated binding."},
	"binding_mismatch":             {Meaning: "Observations contradict a required binding.", Action: "Correct the binding or the implementation."},
	"behavior_target_missing":      {Meaning: "A behavior revision does not have the required target.", Action: "Add the target to the behavior model."},
	"unavailable_external_input":   {Meaning: "Required observation data or a behavior revision is not available.", Action: "Supply the external input."},
	"render_failure":               {Meaning: "The tool cannot render or write the report.", Action: "Make sure that the output path is writable."},
}

// openItemCodes are the finding codes that the open-items table counts, with the row text.
var openItemCodes = []struct{ Code, Text, Target string }{
	{"incomplete_criterion", "Requirements and criteria that nobody can measure", "p-criteria"},
	{"missing_leaf_scenario", "Leaf requirements without a scenario", "p-scenarios"},
	{"support_gap", "Claims without reviewed support", "findings"},
	{"incomplete_module_contract", "Leaf DPs without a complete module contract", "e-modules"},
}

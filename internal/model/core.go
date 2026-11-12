package model

import (
	"math/big"

	"github.com/hashicorp/hcl/v2"
)

// Core holds every decoded core record. Each slice keeps authored order and
// keeps duplicates and repeated endpoints, so that resolution can report them.
// Nested records (criteria, candidates, information, loops) live inside their
// owning record.
type Core struct {
	Designs      []Design
	Sources      []SourceRecord
	Stakeholders []Stakeholder
	Needs        []Need
	Constraints  []Constraint
	Assumptions  []Assumption
	Contexts     []Context
	Workloads    []Workload
	States       []State
	FRs          []FR
	DPs          []DP
	Influences   []Influence
	Cells        []Cells
	Expects      []Expect
	Couplings    []Coupling
	Decisions    []Decision
	Evidence     []Evidence
	PVs          []PV
	Modules      []Module
	Flows        []Flow
	Scenarios    []Scenario
}

// Optional is a list that records whether it was authored. Present with no
// items is an explicit empty list, which differs from an omitted field.
// Plain slices are used where an omitted list means the same as an empty list.
type Optional[T any] struct {
	Present bool
	Items   []T
}

// AssumptionStatus is the authored status of an assumption.
type AssumptionStatus string

const (
	AssumptionAssumed     AssumptionStatus = "assumed"
	AssumptionToValidate  AssumptionStatus = "to_validate"
	AssumptionValidated   AssumptionStatus = "validated"
	AssumptionInvalidated AssumptionStatus = "invalidated"
)

// CriterionOp is the comparison that a criterion applies to its value.
type CriterionOp string

const (
	CriterionOpLessEqual    CriterionOp = "<="
	CriterionOpLess         CriterionOp = "<"
	CriterionOpGreaterEqual CriterionOp = ">="
	CriterionOpGreater      CriterionOp = ">"
	CriterionOpEqual        CriterionOp = "=="
	CriterionOpIn           CriterionOp = "in"
)

// ValueKind tags a criterion value or one scalar. Scalar kinds are number,
// string, and bool. List and interval are composite kinds.
type ValueKind string

const (
	ValueNumber   ValueKind = "number"
	ValueString   ValueKind = "string"
	ValueBool     ValueKind = "bool"
	ValueList     ValueKind = "list"
	ValueInterval ValueKind = "interval"
)

// Scalar is one string, number, or Boolean literal. Kind selects the payload field.
type Scalar struct {
	Kind   ValueKind
	Number *big.Float
	Text   string
	Flag   bool
}

// CriterionValue is the typed operand of a criterion. Kind selects one payload:
// Scalar, List, or Interval.
type CriterionValue struct {
	Kind     ValueKind
	Scalar   Scalar
	List     []Scalar
	Interval Interval
}

// Interval is an inclusive numeric range.
type Interval struct {
	Min *big.Float
	Max *big.Float
}

// Effect is the claimed effect of one control on one outcome.
type Effect string

const (
	EffectMaterial Effect = "material"
	EffectNone     Effect = "none"
	EffectUnknown  Effect = "unknown"
)

// MechanismOrigin says whether an influence mechanism is source-stated or inferred.
type MechanismOrigin string

const (
	MechanismFromSource MechanismOrigin = "source"
	MechanismInferred   MechanismOrigin = "inferred"
)

// UnlistedEffect is the default effect for cells that no exact influence lists.
type UnlistedEffect string

const (
	UnlistedNone    UnlistedEffect = "none"
	UnlistedUnknown UnlistedEffect = "unknown"
)

// Classification is the expected coupling classification of one sibling set.
type Classification string

const (
	ClassificationUncoupled  Classification = "uncoupled"
	ClassificationDecoupled  Classification = "decoupled"
	ClassificationCoupled    Classification = "coupled"
	ClassificationUnresolved Classification = "unresolved"
)

// ExpectBasis selects the kind of expectation that an expect block asserts.
type ExpectBasis string

const (
	ExpectBasisSource      ExpectBasis = "source"
	ExpectBasisConditional ExpectBasis = "conditional"
	ExpectBasisSupported   ExpectBasis = "supported"
)

// LifecycleState is the authored lifecycle state of a design parameter.
type LifecycleState string

const (
	LifecycleProposed   LifecycleState = "proposed"
	LifecycleSelected   LifecycleState = "selected"
	LifecycleRejected   LifecycleState = "rejected"
	LifecycleSuperseded LifecycleState = "superseded"
)

// InformationBasis says whether a candidate's information is unknown or measured.
type InformationBasis string

const (
	InformationUnknown  InformationBasis = "unknown"
	InformationMeasured InformationBasis = "measured"
)

// EvidenceBasis is the basis of an evidence block. Core version 1 admits only measured evidence.
type EvidenceBasis string

const EvidenceMeasured EvidenceBasis = "measured"

// Dependence says how the trials of an evidence block relate. Core version 1 admits only joint trials.
type Dependence string

const DependenceJoint Dependence = "joint"

// ConfidenceMethod names the interval convention of an evidence block.
type ConfidenceMethod string

const ConfidenceWilson ConfidenceMethod = "wilson"

// ScenarioKind is the category of a scenario.
type ScenarioKind string

const (
	ScenarioChange      ScenarioKind = "change"
	ScenarioOperational ScenarioKind = "operational"
	ScenarioFailure     ScenarioKind = "failure"
)

// Design is the merged design root. A merged core has exactly one.
type Design struct {
	Header
	Boundary string
	Level    *string
	Workload *Ref
}

// SourceRecord is a source block. The name keeps the file-level Source type distinct.
type SourceRecord struct {
	Header
	Title string
	DOI   *string
	URL   *string
	Pages *string
}

// Stakeholder is an affected person or group.
type Stakeholder struct {
	Header
	Role string
}

// Need is a desired stakeholder outcome.
type Need struct {
	Header
	Statement   string
	Why         *string
	Stakeholder *Ref
}

// Constraint is a limit on acceptable solutions.
type Constraint struct {
	Header
	Statement string
	Kind      *string
}

// Assumption is a premise that the design asserts.
type Assumption struct {
	Header
	Statement string
	Status    *AssumptionStatus
}

// Context is a named operating condition.
type Context struct {
	Header
	Description string
	Assumes     []Ref
	Excludes    []Ref
	Workload    *Ref
}

// Workload describes an input population and exposure.
type Workload struct {
	Header
	Description string
	Ranges      Optional[RangeDimension]
}

// RangeDimension is one measured dimension of a workload range map, in authored order.
type RangeDimension struct {
	Name  string
	Min   *big.Float
	Max   *big.Float
	Unit  *string
	Range hcl.Range
}

// State is information owned or read by leaf controls.
type State struct {
	Header
	Description string
}

// FR is a functional requirement. Need distinguishes an omitted list from an explicit empty list.
type FR struct {
	Header
	Statement string
	Need      Optional[Ref]
	Parent    *Ref
	BoundedBy []Ref
	Criteria  []Criterion
}

// Criterion is an acceptance condition of one FR. Its ID has the form fr.F.criterion.C.
type Criterion struct {
	Header
	Statement     *string
	Metric        *string
	Op            *CriterionOp
	Value         *CriterionValue
	Unit          *string
	Under         *Ref
	AllocatedFrom *Ref
}

// DP is a design parameter. Its controls stay within one sibling set.
type DP struct {
	Header
	Controls       []Ref
	Responsibility string
	Decision       LifecycleState
	Why            *string
	Role           *string
	OwnsState      Optional[Ref]
	ReadsState     []Ref
	BoundedBy      []Ref
	HoldsIf        []Ref
}

// Influence is one independently asserted claim. Records with the same endpoints stay separate.
type Influence struct {
	Unlabeled
	From            []Ref
	To              Ref
	Effect          Effect
	Mechanism       *string
	Because         *string
	MechanismSource *MechanismOrigin
	SourceLocator   *string
	Under           *Ref
	HoldsIf         []Ref
}

// Cells sets the default effect for unlisted cells of one sibling set.
type Cells struct {
	Header
	Scope    Ref
	Unlisted UnlistedEffect
	Because  *string
}

// Expect asserts the expected classification of one sibling set. Order distinguishes an omitted list from an explicit empty list.
type Expect struct {
	Header
	Scope          Ref
	Classification Classification
	Order          Optional[Ref]
	Basis          *ExpectBasis
	Under          *Ref
}

// Coupling records residual coupling in one sibling set. Its numeric bound is a pointer so
// that the decoder can check that it is a positive integer.
type Coupling struct {
	Header
	Scope           Ref
	Affects         []Ref
	Reason          string
	Condition       *string
	Decision        *string
	NormalPractice  *bool
	ResidualRisk    *string
	IterationBound  *big.Float
	Convergence     *string
	Stopping        *string
	FailureBehavior *string
}

// Decision is a comparison of one or more candidates for the same outcomes.
type Decision struct {
	Header
	Addresses   []Ref
	Context     *Ref
	Workload    *Ref
	Horizon     *string
	Selected    *string
	Provisional *bool
	Why         *string
	Candidates  []Candidate
}

// Candidate is one alternative inside a decision. Its ID has the form decision.D.candidate.C.
type Candidate struct {
	Header
	DPs         []Ref
	Version     *string
	Context     *Ref
	Workload    *Ref
	Horizon     *string
	Information *Information
}

// Information is the optional information block of a candidate.
type Information struct {
	Unlabeled
	Basis    InformationBasis
	Evidence []Ref
}

// Evidence is one block of joint trial observations. Every field is required. The
// numeric fields are pointers so that the decoder can tell absence from zero.
type Evidence struct {
	Header
	Basis            EvidenceBasis
	Event            string
	Scenario         Ref
	Candidate        Ref
	Version          string
	Context          Ref
	Workload         Ref
	Horizon          string
	Dependence       Dependence
	Sampling         string
	Trials           *big.Float
	Failures         *big.Float
	ConfidenceMethod ConfidenceMethod
	Confidence       *big.Float
}

// PV is a process variable that realizes one or more design parameters.
type PV struct {
	Header
	Realizes []Ref
	Kind     *string
	Address  *string
	Note     *string
}

// Module is a realization module or a composite module. Realizes and Contains distinguish an omitted list
// from an explicit empty list, and they are mutually exclusive.
type Module struct {
	Header
	Realizes         Optional[Ref]
	Contains         Optional[Ref]
	Within           *Ref
	Inputs           Optional[string]
	Outputs          Optional[string]
	OwnsState        Optional[Ref]
	Dependencies     Optional[Ref]
	Invariants       Optional[string]
	FailureBehavior  *string
	AcceptanceChecks Optional[Ref]
}

// Flow is an ordered runtime sequence. Each step is one ref or a group of refs that
// share one outer position.
type Flow struct {
	Header
	Steps  [][]Ref
	Owner  *Ref
	Input  *string
	Output *string
	After  []Ref
	Loops  []Loop
}

// Loop is an unlabeled feedback record inside its enclosing flow.
type Loop struct {
	Unlabeled
	After  Ref
	BackTo Ref
	When   string
	Bound  *big.Float
}

// Scenario is a change, operational, or failure case that serves as an acceptance check.
type Scenario struct {
	Header
	Kind             ScenarioKind
	Verifies         Optional[Ref]
	Vary             []Ref
	ExpectAffected   []Ref
	ExpectUnaffected []Ref
	Steps            Optional[string]
	Description      *string
	Under            *Ref
}

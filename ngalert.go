package sdk

import (
	"encoding/json"
	"errors"
	"time"
)

// original source code:
// - https://github.com/grafana/grafana/blob/main/pkg/services/ngalert/models/alert_rule.go
// - https://github.com/grafana/grafana/blob/main/pkg/services/ngalert/models/alert_query.go

const (
	DefaultMaxDataPoints = 43200 // 12 hours at 1sec interval
	DefaultIntervalMs    = 1000
)

type NoDataState string

const (
	NoDataStateAlerting NoDataState = "Alerting"
	NoDataStateNoData   NoDataState = "NoData"
	NoDataStateOk       NoDataState = "OK"
)

type ExecErrorState string

const (
	ExecErrorStateAlerting ExecErrorState = "Alerting"
	ExecErrorStateError    ExecErrorState = "Error"
	ExecErrorStateOk       ExecErrorState = "OK"
)

type NgAlert struct {
	Id    int64  `json:"id,omitempty"`
	Uid   string `json:"uid,omitempty"`
	OrgID int64  `json:"orgID,omitempty"`

	FolderUID string `json:"folderUID"`
	RuleGroup string `json:"ruleGroup"`
	Title     string `json:"title"`

	Condition string         `json:"condition"`
	Data      []NgAlertQuery `json:"data"`

	Updated time.Time `json:"updated,omitempty"`

	NoDataState  NoDataState    `json:"noDataState"`
	ExecErrState ExecErrorState `json:"execErrState"`

	For string `json:"for"` // format: 1m

	Annotations map[string]string `json:"annotations"`
	Labels      map[string]string `json:"labels"`

	Provenance string `json:"provenance,omitempty"`
	IsPaused   bool   `json:"isPaused"`
}

type NgAlertQuery struct {
	RefId             string            `json:"refId"`
	QueryType         string            `json:"queryType"`
	RelativeTimeRange RelativeTimeRange `json:"relativeTimeRange"`
	DatasourceUid     string            `json:"datasourceUid"` // datasource or "__expr__"
	Model             NgAlertQueryModel `json:"model"`
}

type RelativeTimeRange struct {
	FromSeconds int `json:"from"`
	ToSeconds   int `json:"to"`
}

type NgAlertQueryModel struct {
	*NgAlertQueryModelQuery
	*NgAlertQueryModelExpression
}

func (m NgAlertQueryModel) MarshalJSON() ([]byte, error) {
	if m.NgAlertQueryModelQuery != nil {
		return json.Marshal(m.NgAlertQueryModelQuery)
	}
	if m.NgAlertQueryModelExpression != nil {
		return json.Marshal(m.NgAlertQueryModelExpression)
	}
	return nil, errors.New("model is empty")
}

type NgAlertQueryModelQuery struct {
	EditorMode    string `json:"editorMode"` // default: code
	Expr          string `json:"expr"`
	Instant       bool   `json:"instant"`       // default: true (opposite of Range)
	IntervalMs    int    `json:"intervalMs"`    // default: DefaultIntervalMs
	LegendFormat  string `json:"legendFormat"`  // default: __auto
	MaxDataPoints int    `json:"maxDataPoints"` // default: DefaultMaxDataPoints
	Range         bool   `json:"range"`         // default: false (opposite of Instant)
	RefId         string `json:"refId"`         // same as NgAlertQuery.RefId
}

type NgAlertQueryModelExpression struct {
	NgAlertQueryModelExpressionParams
	Cmd NgAlertQueryModelCommand `json:"-"`
}

type NgAlertQueryModelExpressionParams struct {
	Datasource    NgAlertQueryModelDatasource `json:"datasource"`
	Hide          bool                        `json:"hide"`          // default: false
	IntervalMs    int                         `json:"intervalMs"`    // default: DefaultIntervalMs
	MaxDataPoints int                         `json:"maxDataPoints"` // default: DefaultMaxDataPoints
	RefId         string                      `json:"refId"`         // same as NgAlertQuery.RefId
}

func (c NgAlertQueryModelExpression) MarshalJSON() ([]byte, error) {
	data, err := structToMap(c.NgAlertQueryModelExpressionParams)
	if err != nil {
		return nil, err
	}

	cmd, err := c.Cmd.marshal()
	if err != nil {
		return nil, err
	}
	for k, v := range cmd {
		data[k] = v
	}

	return json.Marshal(data)
}

type NgAlertQueryModelDatasource struct {
	Type string `json:"type"` // default: __expr__
	Uid  string `json:"uid"`  // default: __expr__
}

type NgAlertQueryModelCommand struct {
	Type CommandType `json:"type"`

	*MathCommand
	*ReduceCommand
	*ResampleCommand
	*ClassicConditionsCommand
	*ThresholdCommand
}

func structToMap(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	data := map[string]any{}
	err = json.Unmarshal(raw, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c NgAlertQueryModelCommand) marshal() (map[string]any, error) {
	data, err := c.marshalCommand()
	if err != nil {
		return nil, err
	}
	data["type"] = c.Type
	return data, nil
}

func (c NgAlertQueryModelCommand) marshalCommand() (map[string]any, error) {
	switch c.Type {
	case CommandTypeMath:
		return structToMap(c.MathCommand)
	case CommandTypeReduce:
		return structToMap(c.ReduceCommand)
	case CommandTypeResample:
		return structToMap(c.ResampleCommand)
	case CommandTypeClassicConditions:
		return structToMap(c.ClassicConditionsCommand)
	case CommandTypeThreshold:
		return structToMap(c.ThresholdCommand)
	default:
		return nil, errors.New("unknown command type: " + string(c.Type))
	}
}

// CommandType indicates which command struct is used for the model expression.
// source: https://github.com/grafana/grafana/blob/main/pkg/expr/commands.go#L335
type CommandType string

const (
	CommandTypeMath              CommandType = "math"
	CommandTypeReduce            CommandType = "reduce"
	CommandTypeResample          CommandType = "resample"
	CommandTypeClassicConditions CommandType = "classic_conditions"
	CommandTypeThreshold         CommandType = "threshold"
)

// ---------------------- math command ----------------------
// source: https://github.com/grafana/grafana/blob/main/pkg/expr/commands.go#L46

type MathCommand struct {
	Expression string `json:"expression"`
}

// ---------------------- reduce command ----------------------
// source: https://github.com/grafana/grafana/blob/main/pkg/expr/commands.go#L102

type ReduceCommand struct {
	Expression string                 `json:"expression"`
	Reducer    ReducerFunc            `json:"reducer"`
	Settings   *ReduceCommandSettings `json:"settings,omitempty"`
}

type ReducerFunc string

const (
	ReducerFuncSum   ReducerFunc = "sum"
	ReducerFuncMean  ReducerFunc = "mean"
	ReducerFuncMin   ReducerFunc = "min"
	ReducerFuncMax   ReducerFunc = "max"
	ReducerFuncCount ReducerFunc = "count"
	ReducerFuncLast  ReducerFunc = "last"
)

type ReduceCommandSettings struct {
	Mode             ReduceMode `json:"mode"`
	ReplaceWithValue *float64   `json:"replaceWithValue,omitempty"` // required if mode == ReduceModeReplaceNonNumeric
}

type ReduceMode string

const (
	ReduceModeStrict            ReduceMode = "" // empty on purpose
	ReduceModeDropNonNumeric    ReduceMode = "dropNN"
	ReduceModeReplaceNonNumeric ReduceMode = "replaceNN"
)

// ---------------------- resample command ----------------------
// source: https://github.com/grafana/grafana/blob/main/pkg/expr/commands.go#L102

type ResampleCommand struct {
	Expression  string              `json:"expression"`
	Window      string              `json:"window"` // format: 1m
	DownSampler ResampleDownSampler `json:"downsampler"`
	UpSampler   ResampleUpSampler   `json:"upsampler"`
}

type ResampleDownSampler string

const (
	ResampleDownSamplerSum  ResampleDownSampler = "sum"
	ResampleDownSamplerMean ResampleDownSampler = "mean"
	ResampleDownSamplerMin  ResampleDownSampler = "min"
	ResampleDownSamplerMax  ResampleDownSampler = "max"
	ResampleDownSamplerLast ResampleDownSampler = "last"
)

type ResampleUpSampler string

const (
	ResampleUpSamplerPad         ResampleUpSampler = "pad"
	ResampleUpSamplerBackFilling ResampleUpSampler = "backfilling"
	ResampleUpSamplerFillNa      ResampleUpSampler = "fillna"
)

// ---------------------- classic conditions command ----------------------
// source: https://github.com/grafana/grafana/blob/main/pkg/expr/classic/classic.go#L279

type ClassicConditionsCommand struct {
	Conditions []ClassicCondition `json:"conditions"`
}

type ClassicCondition struct {
	Evaluator ClassicConditionEval     `json:"evaluator"`
	Operator  ClassicConditionOperator `json:"operator"`
	Query     ClassicConditionQuery    `json:"query"`
	Reducer   ClassicConditionReducer  `json:"reducer"`
}

type ClassicConditionEval struct {
	Params []float64                `json:"params"`
	Type   ClassicConditionEvalType `json:"type"`
}

type ClassicConditionEvalType string

const (
	ClassicConditionEvalTypeGt           ClassicConditionEvalType = "gt"
	ClassicConditionEvalTypeLt           ClassicConditionEvalType = "lt"
	ClassicConditionEvalTypeWithinRange  ClassicConditionEvalType = "within_range"
	ClassicConditionEvalTypeOutsideRange ClassicConditionEvalType = "outside_range"
	ClassicConditionEvalTypeNoValue      ClassicConditionEvalType = "no_value"
)

type ClassicConditionOperator struct {
	Type ClassicConditionOperatorType `json:"type"`
}

type ClassicConditionOperatorType string

const (
	ClassicConditionOperatorTypeAnd ClassicConditionOperatorType = "and"
	ClassicConditionOperatorTypeOr  ClassicConditionOperatorType = "or"
)

type ClassicConditionQuery struct {
	Params []string `json:"params"`
}

type ClassicConditionReducer struct {
	Type ClassicConditionReducerType `json:"type"`
}

type ClassicConditionReducerType string

const (
	ClassicConditionReducerTypeAvg            ClassicConditionReducerType = "avg"
	ClassicConditionReducerTypeSum            ClassicConditionReducerType = "sum"
	ClassicConditionReducerTypeMin            ClassicConditionReducerType = "min"
	ClassicConditionReducerTypeMax            ClassicConditionReducerType = "max"
	ClassicConditionReducerTypeCount          ClassicConditionReducerType = "count"
	ClassicConditionReducerTypeLast           ClassicConditionReducerType = "last"
	ClassicConditionReducerTypeMedian         ClassicConditionReducerType = "median"
	ClassicConditionReducerTypeDiff           ClassicConditionReducerType = "diff"
	ClassicConditionReducerTypeDiffAbs        ClassicConditionReducerType = "diff_abs"
	ClassicConditionReducerTypePercentDiff    ClassicConditionReducerType = "percent_diff"
	ClassicConditionReducerTypePercentDiffAbs ClassicConditionReducerType = "percent_diff_abs"
	ClassicConditionReducerTypeCountNonNull   ClassicConditionReducerType = "count_non_null"
)

// ---------------------- threshold command ----------------------
// source: https://github.com/grafana/grafana/blob/main/pkg/expr/threshold.go#L62

type ThresholdCommand struct {
	Expression string               `json:"expression"`
	Conditions []ThresholdCondition `json:"conditions"`
}

type ThresholdCondition struct {
	Evaluator ThresholdConditionEval `json:"evaluator"`
}

type ThresholdConditionEval struct {
	Params []float64                  `json:"params"`
	Type   ThresholdConditionEvalType `json:"type"`
}

type ThresholdConditionEvalType string

const (
	ThresholdConditionEvalTypeTypeGt           ThresholdConditionEvalType = "gt"
	ThresholdConditionEvalTypeTypeLt           ThresholdConditionEvalType = "lt"
	ThresholdConditionEvalTypeTypeWithinRange  ThresholdConditionEvalType = "within_range"
	ThresholdConditionEvalTypeTypeOutsideRange ThresholdConditionEvalType = "outside_range"
)

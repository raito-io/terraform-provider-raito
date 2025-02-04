// Code generated by "enumer -type=UnaryOperator -values -gqlgen -yaml -json -trimprefix=UnaryOperator"; DO NOT EDIT.

package abac_expression

import (
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"io"
	"strconv"
	"strings"
)

const _UnaryOperatorName = "Not"

var _UnaryOperatorIndex = [...]uint8{0, 3}

const _UnaryOperatorLowerName = "not"

func (i UnaryOperator) String() string {
	if i < 0 || i >= UnaryOperator(len(_UnaryOperatorIndex)-1) {
		return fmt.Sprintf("UnaryOperator(%d)", i)
	}
	return _UnaryOperatorName[_UnaryOperatorIndex[i]:_UnaryOperatorIndex[i+1]]
}

func (UnaryOperator) Values() []string {
	return UnaryOperatorStrings()
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _UnaryOperatorNoOp() {
	var x [1]struct{}
	_ = x[UnaryOperatorNot-(0)]
}

var _UnaryOperatorValues = []UnaryOperator{UnaryOperatorNot}

var _UnaryOperatorNameToValueMap = map[string]UnaryOperator{
	_UnaryOperatorName[0:3]:      UnaryOperatorNot,
	_UnaryOperatorLowerName[0:3]: UnaryOperatorNot,
}

var _UnaryOperatorNames = []string{
	_UnaryOperatorName[0:3],
}

// UnaryOperatorString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func UnaryOperatorString(s string) (UnaryOperator, error) {
	if val, ok := _UnaryOperatorNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _UnaryOperatorNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, errors.Errorf("%s does not belong to UnaryOperator values", s)
}

// UnaryOperatorValues returns all values of the enum
func UnaryOperatorValues() []UnaryOperator {
	return _UnaryOperatorValues
}

// UnaryOperatorStrings returns a slice of all String values of the enum
func UnaryOperatorStrings() []string {
	strs := make([]string, len(_UnaryOperatorNames))
	copy(strs, _UnaryOperatorNames)
	return strs
}

// IsAUnaryOperator returns "true" if the value is listed in the enum definition. "false" otherwise
func (i UnaryOperator) IsAUnaryOperator() bool {
	for _, v := range _UnaryOperatorValues {
		if i == v {
			return true
		}
	}
	return false
}

// MarshalJSON implements the json.Marshaler interface for UnaryOperator
func (i UnaryOperator) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface for UnaryOperator
func (i *UnaryOperator) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return errors.Errorf("UnaryOperator should be a string, got %s", data)
	}

	var err error
	*i, err = UnaryOperatorString(s)
	return err
}

// MarshalYAML implements a YAML Marshaler for UnaryOperator
func (i UnaryOperator) MarshalYAML() (interface{}, error) {
	return i.String(), nil
}

// UnmarshalYAML implements a YAML Unmarshaler for UnaryOperator
func (i *UnaryOperator) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	var err error
	*i, err = UnaryOperatorString(s)
	return err
}

// MarshalGQL implements the graphql.Marshaler interface for UnaryOperator
func (i UnaryOperator) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(i.String()))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface for UnaryOperator
func (i *UnaryOperator) UnmarshalGQL(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return errors.Errorf("UnaryOperator should be a string, got %T", value)
	}

	var err error
	*i, err = UnaryOperatorString(str)
	return err
}

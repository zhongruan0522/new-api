package shared

import (
	"strconv"

	"github.com/NookMux/NookMux/pkg/jsonx"
)

type IntValue int

func (i *IntValue) UnmarshalJSON(b []byte) error {
	var n int
	if err := jsonx.Unmarshal(b, &n); err == nil {
		*i = IntValue(n)
		return nil
	}
	var s string
	if err := jsonx.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*i = IntValue(v)
	return nil
}

func (i IntValue) MarshalJSON() ([]byte, error) {
	return jsonx.Marshal(int(i))
}

type BoolValue bool

func (b *BoolValue) UnmarshalJSON(data []byte) error {
	var boolean bool
	if err := jsonx.Unmarshal(data, &boolean); err == nil {
		*b = BoolValue(boolean)
		return nil
	}
	var str string
	if err := jsonx.Unmarshal(data, &str); err != nil {
		return err
	}
	if str == "true" {
		*b = BoolValue(true)
	} else if str == "false" {
		*b = BoolValue(false)
	} else {
		return jsonx.Unmarshal(data, &boolean)
	}
	return nil
}
func (b BoolValue) MarshalJSON() ([]byte, error) {
	return jsonx.Marshal(bool(b))
}

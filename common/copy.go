package common

import (
	"fmt"
	"github.com/antlabs/pcopy"
)

func DeepCopy[T any](src *T) (*T, error) {
	if src == nil {
		return nil, fmt.Errorf("copy source cannot be nil")
	}
	var dst T
	err := pcopy.Copy(&dst, src)
	if err != nil {
		return nil, err
	}
	if &dst == nil {
		return nil, fmt.Errorf("copy result cannot be nil")
	}
	return &dst, nil
}

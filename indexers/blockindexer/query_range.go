package indexer

import (
	"time"

	"github.com/tendermint/tendermint/libs/pubsub/query"
)

type QueryRanges map[string]QueryRange

type QueryRange struct {
	LowerBound        interface{}
	UpperBound        interface{}
	Key               string
	IncludeLowerBound bool
	IncludeUpperBound bool
}

func (qr QueryRange) AnyBound() interface{} {
	if qr.LowerBound != nil {
		return qr.LowerBound
	}

	return qr.UpperBound
}

func (qr QueryRange) LowerBoundValue() interface{} {
	if qr.LowerBound == nil {
		return nil
	}

	if qr.IncludeLowerBound {
		return qr.LowerBound
	}

	switch t := qr.LowerBound.(type) {
	case int64:
		return t + 1

	case time.Time:
		return t.Unix() + 1

	default:
		panic("not implemented")
	}
}

func (qr QueryRange) UpperBoundValue() interface{} {
	if qr.UpperBound == nil {
		return nil
	}

	if qr.IncludeUpperBound {
		return qr.UpperBound
	}

	switch t := qr.UpperBound.(type) {
	case int64:
		return t - 1

	case time.Time:
		return t.Unix() - 1

	default:
		panic("not implemented")
	}
}

func LookForRanges(conditions []query.Condition) (ranges QueryRanges, indexes []int) {
	ranges = make(QueryRanges)
	for i, c := range conditions {
		if IsRangeOperation(c.Op) {
			r, ok := ranges[c.CompositeKey]
			if !ok {
				r = QueryRange{Key: c.CompositeKey}
			}

			switch c.Op {
			case query.OpGreater:
				r.LowerBound = c.Operand

			case query.OpGreaterEqual:
				r.IncludeLowerBound = true
				r.LowerBound = c.Operand

			case query.OpLess:
				r.UpperBound = c.Operand

			case query.OpLessEqual:
				r.IncludeUpperBound = true
				r.UpperBound = c.Operand
			}

			ranges[c.CompositeKey] = r
			indexes = append(indexes, i)
		}
	}

	return ranges, indexes
}

func IsRangeOperation(op query.Operator) bool {
	switch op {
	case query.OpGreater, query.OpGreaterEqual, query.OpLess, query.OpLessEqual:
		return true

	default:
		return false
	}
}

// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueInterface(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		val  Value
	}{
		{name: "null", val: Null{}},
		{name: "bool", val: Bool(true)},
		{name: "int", val: Int(1)},
		{name: "long", val: Long(1)},
		{name: "float", val: Float(1)},
		{name: "double", val: Double(1)},
		{name: "bytes", val: Bytes{0x01}},
		{name: "string", val: String("a")},
		{name: "record", val: Record{Fields: []Field{{Name: "a", Value: Int(1)}}}},
		{name: "enum", val: Enum{Symbol: "X"}},
		{name: "array", val: Array{Int(1), Int(2)}},
		{name: "map", val: Map{"k": Long(2)}},
		{name: "union", val: Union{Index: 1, Value: String("v")}},
		{name: "fixed", val: Fixed{0x01, 0x02, 0x03, 0x04}},
		{name: "decimal", val: Decimal{Unscaled: big.NewInt(123)}},
		{name: "uuid", val: UUID("00000000-0000-0000-0000-000000000000")},
		{name: "date", val: Date(0)},
		{name: "time millis", val: TimeMillis(0)},
		{name: "time micros", val: TimeMicros(0)},
		{name: "timestamp millis", val: TimestampMillis(0)},
		{name: "timestamp micros", val: TimestampMicros(0)},
		{name: "timestamp nanos", val: TimestampNanos(0)},
		{name: "local timestamp millis", val: LocalTimestampMillis(0)},
		{name: "local timestamp micros", val: LocalTimestampMicros(0)},
		{name: "local timestamp nanos", val: LocalTimestampNanos(0)},
		{name: "duration", val: Duration{Months: 1, Days: 2, Millis: 3}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NotNil(t, tc.val)
		})
	}
}

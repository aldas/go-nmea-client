package test_test

import (
	"github.com/aldas/go-nmea-client"
	"github.com/stretchr/testify/assert"
	"testing"
)

func AssertRawMessage(t *testing.T, expect nmea.Message, actual nmea.Message, delta float64) {
	assert.Equal(t, expect.Header, actual.Header)
	AssertFieldValues(t, expect.Fields, actual.Fields, delta)
}

func AssertFieldValues(t *testing.T, expect nmea.FieldValues, actual nmea.FieldValues, delta float64) {
	assert.Len(t, actual, len(expect))

	for _, actualFieldValue := range actual {
		expectedFieldValue, ok := expect.FindByID(actualFieldValue.ID)
		if !ok {
			t.Errorf("actual fields contains field with ID `%v` that is not in expected fields", actualFieldValue.ID)
			continue
		}
		AssertFieldValue(t, expectedFieldValue, actualFieldValue, delta)
	}
}

func AssertFieldValue(t *testing.T, expect nmea.FieldValue, actual nmea.FieldValue, delta float64) {
	switch actual.Value.(type) {
	case float64:
		assert.InDelta(
			t,
			expect.Value,
			actual.Value,
			delta,
			"Field ID: `%v` value %v is different from expected %v",
			expect.ID,
			actual.Value,
			expect.Value,
		)
		expect.Value = nil
		actual.Value = nil
	}
	assert.Equal(t, expect, actual)
}

package test_test

import "time"

// UTCTime creates instance of time in UTC timezone this helps avoid problems running tests with different timezone computers
func UTCTime(sec int64) time.Time {
	return time.Unix(sec, 0).In(time.UTC)
}

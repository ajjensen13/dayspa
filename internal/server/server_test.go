package server

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

func Test_connData_MarshalText(t *testing.T) {
	tests := []struct {
		name     string
		c        *connData
		wantText string
		wantErr  bool
	}{
		{
			`happy path`,
			&connData{
				log: []tx{
					connStateTx{state: http.StateNew, ts: time.Date(2020, 05, 15, 0, 0, 0, 0, time.UTC)},
					connStateTx{state: http.StateActive, ts: time.Date(2020, 05, 15, 0, 0, 1, 0, time.UTC)},
					connStateTx{state: http.StateClosed, ts: time.Date(2020, 05, 15, 0, 0, 2, 0, time.UTC)},
				},
				begin: time.Date(2020, 05, 15, 0, 0, 0, 0, time.UTC),
				end:   time.Date(2020, 05, 15, 0, 0, 2, 0, time.UTC),
			},
			`` +
				"BEGIN CONN DATA: 2020-05-15 00:00:00 +0000 UTC\n" +
				" +0.000s: CONN STATE = new\n" +
				" +1.000s: CONN STATE = active\n" +
				" +2.000s: CONN STATE = closed\n" +
				"END CONN DATA: 2020-05-15 00:00:02 +0000 UTC\n",
			false,
		},
		{
			`never started`,
			&connData{
				log: []tx{
					connStateTx{state: http.StateNew, ts: time.Date(2020, 05, 15, 0, 0, 0, 0, time.UTC)},
					connStateTx{state: http.StateActive, ts: time.Date(2020, 05, 15, 0, 0, 1, 0, time.UTC)},
					connStateTx{state: http.StateClosed, ts: time.Date(2020, 05, 15, 0, 0, 2, 0, time.UTC)},
				},
				end: time.Date(2020, 05, 15, 0, 0, 2, 0, time.UTC),
			},
			``,
			false,
		},
		{
			`never ended`,
			&connData{
				log: []tx{
					connStateTx{state: http.StateNew, ts: time.Date(2020, 05, 15, 0, 0, 0, 0, time.UTC)},
					connStateTx{state: http.StateActive, ts: time.Date(2020, 05, 15, 0, 0, 1, 0, time.UTC)},
					connStateTx{state: http.StateClosed, ts: time.Date(2020, 05, 15, 0, 0, 2, 0, time.UTC)},
				},
				begin: time.Date(2020, 05, 15, 0, 0, 0, 0, time.UTC),
			},
			`` +
				"BEGIN CONN DATA: 2020-05-15 00:00:00 +0000 UTC\n" +
				" +0.000s: CONN STATE = new\n" +
				" +1.000s: CONN STATE = active\n" +
				" +2.000s: CONN STATE = closed\n" +
				"...\n",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, err := tt.c.MarshalText()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(gotText) != tt.wantText {
				t.Errorf("MarshalText() gotText = \n%q\nwant = \n%q\n", string(gotText), tt.wantText)
			}
		})
	}
}

func compareConnData(t *testing.T, want, got []*connData) {
	if len(want) != len(got) {
		t.Errorf("len(want) = %d, len(got) = %d", len(want), len(got))
	}

	for i := 0; i < len(want) || i < len(got); i++ {
		if i < len(want) && i < len(got) {
			compareTxs(t, want[i].log, got[i].log)
			continue
		}

		if i < len(want) {
			t.Errorf("extra log: %v", want[i])
		}

		if i < len(got) {
			t.Errorf("missing log: %v", got[i])
		}
	}
}

func compareTxs(t *testing.T, want, got []tx) {
	if len(want) != len(got) {
		t.Errorf("len(want) = %d, len(got) = %d", len(want), len(got))
	}

	for i := 0; i < len(want) || i < len(got); i++ {
		if i < len(want) && i < len(got) {
			compareTx(t, want[i], got[i])
			continue
		}

		if i < len(want) {
			t.Errorf("extra log entry: %v", want[i])
		}

		if i < len(got) {
			t.Errorf("missing log entry: %v", got[i])
		}
	}
}

func compareTx(t *testing.T, want, got tx) {
	t.Helper()

	switch wantC := want.(type) {
	case connStateTx:
		wantC.ts = got.(connStateTx).ts
		if !reflect.DeepEqual(want, got) {
			t.Errorf("want = %v, got = %v", want, got)
		}
	}
}

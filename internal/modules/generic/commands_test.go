// Copyright 2024 Kelvin Clement Mwinuka
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generic_test

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/echovault/echovault/echovault"
	"github.com/echovault/echovault/internal"
	"github.com/echovault/echovault/internal/clock"
	"github.com/echovault/echovault/internal/config"
	"github.com/echovault/echovault/internal/constants"
	"github.com/tidwall/resp"
)

type KeyData struct {
	Value    interface{}
	ExpireAt time.Time
}

func Test_Generic(t *testing.T) {
	mockClock := clock.NewClock()
	port, err := internal.GetFreePort()
	if err != nil {
		t.Error(err)
		return
	}

	mockServer, err := echovault.NewEchoVault(
		echovault.WithConfig(config.Config{
			BindAddr:       "localhost",
			Port:           uint16(port),
			DataDir:        "",
			EvictionPolicy: constants.NoEviction,
		}),
	)
	if err != nil {
		t.Error(err)
		return
	}

	go func() {
		mockServer.Start()
	}()

	t.Cleanup(func() {
		mockServer.ShutDown()
	})

	t.Run("Test_HandleSET", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name             string
			command          []string
			presetValues     map[string]KeyData
			expectedResponse interface{}
			expectedValue    interface{}
			expectedExpiry   time.Time
			expectedErr      error
		}{
			{
				name:             "1. Set normal string value",
				command:          []string{"SET", "SetKey1", "value1"},
				presetValues:     nil,
				expectedResponse: "OK",
				expectedValue:    "value1",
				expectedExpiry:   time.Time{},
				expectedErr:      nil,
			},
			{
				name:             "2. Set normal integer value",
				command:          []string{"SET", "SetKey2", "1245678910"},
				presetValues:     nil,
				expectedResponse: "OK",
				expectedValue:    "1245678910",
				expectedExpiry:   time.Time{},
				expectedErr:      nil,
			},
			{
				name:             "3. Set normal float value",
				command:          []string{"SET", "SetKey3", "45782.11341"},
				presetValues:     nil,
				expectedResponse: "OK",
				expectedValue:    "45782.11341",
				expectedExpiry:   time.Time{},
				expectedErr:      nil,
			},
			{
				name:             "4. Only set the value if the key does not exist",
				command:          []string{"SET", "SetKey4", "value4", "NX"},
				presetValues:     nil,
				expectedResponse: "OK",
				expectedValue:    "value4",
				expectedExpiry:   time.Time{},
				expectedErr:      nil,
			},
			{
				name:    "5. Throw error when value already exists with NX flag passed",
				command: []string{"SET", "SetKey5", "value5", "NX"},
				presetValues: map[string]KeyData{
					"SetKey5": {
						Value:    "preset-value5",
						ExpireAt: time.Time{},
					},
				},
				expectedResponse: nil,
				expectedValue:    "preset-value5",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("key SetKey5 already exists"),
			},
			{
				name:    "6. Set new key value when key exists with XX flag passed",
				command: []string{"SET", "SetKey6", "value6", "XX"},
				presetValues: map[string]KeyData{
					"SetKey6": {
						Value:    "preset-value6",
						ExpireAt: time.Time{},
					},
				},
				expectedResponse: "OK",
				expectedValue:    "value6",
				expectedExpiry:   time.Time{},
				expectedErr:      nil,
			},
			{
				name:             "7. Return error when setting non-existent key with XX flag",
				command:          []string{"SET", "SetKey7", "value7", "XX"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    nil,
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("key SetKey7 does not exist"),
			},
			{
				name:             "8. Return error when NX flag is provided after XX flag",
				command:          []string{"SET", "SetKey8", "value8", "XX", "NX"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    nil,
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("cannot specify NX when XX is already specified"),
			},
			{
				name:             "9. Return error when XX flag is provided after NX flag",
				command:          []string{"SET", "SetKey9", "value9", "NX", "XX"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    nil,
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("cannot specify XX when NX is already specified"),
			},
			{
				name:             "10. Set expiry time on the key to 100 seconds from now",
				command:          []string{"SET", "SetKey10", "value10", "EX", "100"},
				presetValues:     nil,
				expectedResponse: "OK",
				expectedValue:    "value10",
				expectedExpiry:   mockClock.Now().Add(100 * time.Second),
				expectedErr:      nil,
			},
			{
				name:             "11. Return error when EX flag is passed without seconds value",
				command:          []string{"SET", "SetKey11", "value11", "EX"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("seconds value required after EX"),
			},
			{
				name:             "12. Return error when EX flag is passed with invalid (non-integer) value",
				command:          []string{"SET", "SetKey12", "value12", "EX", "seconds"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("seconds value should be an integer"),
			},
			{
				name:             "13. Return error when trying to set expiry seconds when expiry is already set",
				command:          []string{"SET", "SetKey13", "value13", "PX", "100000", "EX", "100"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    nil,
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("cannot specify EX when expiry time is already set"),
			},
			{
				name:             "14. Set expiry time on the key in unix milliseconds",
				command:          []string{"SET", "SetKey14", "value14", "PX", "4096"},
				presetValues:     nil,
				expectedResponse: "OK",
				expectedValue:    "value14",
				expectedExpiry:   mockClock.Now().Add(4096 * time.Millisecond),
				expectedErr:      nil,
			},
			{
				name:             "15. Return error when PX flag is passed without milliseconds value",
				command:          []string{"SET", "SetKey15", "value15", "PX"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("milliseconds value required after PX"),
			},
			{
				name:             "16. Return error when PX flag is passed with invalid (non-integer) value",
				command:          []string{"SET", "SetKey16", "value16", "PX", "milliseconds"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("milliseconds value should be an integer"),
			},
			{
				name:             "17. Return error when trying to set expiry milliseconds when expiry is already provided",
				command:          []string{"SET", "SetKey17", "value17", "EX", "10", "PX", "1000000"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    nil,
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("cannot specify PX when expiry time is already set"),
			},
			{
				name: "18. Set exact expiry time in seconds from unix epoch",
				command: []string{
					"SET", "SetKey18", "value18",
					"EXAT", fmt.Sprintf("%d", mockClock.Now().Add(200*time.Second).Unix()),
				},
				presetValues:     nil,
				expectedResponse: "OK",
				expectedValue:    "value18",
				expectedExpiry:   mockClock.Now().Add(200 * time.Second),
				expectedErr:      nil,
			},
			{
				name: "19. Return error when trying to set exact seconds expiry time when expiry time is already provided",
				command: []string{
					"SET", "SetKey19", "value19",
					"EX", "10",
					"EXAT", fmt.Sprintf("%d", mockClock.Now().Add(200*time.Second).Unix()),
				},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("cannot specify EXAT when expiry time is already set"),
			},
			{
				name:             "20. Return error when no seconds value is provided after EXAT flag",
				command:          []string{"SET", "SetKey20", "value20", "EXAT"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("seconds value required after EXAT"),
			},
			{
				name:             "21. Return error when invalid (non-integer) value is passed after EXAT flag",
				command:          []string{"SET", "SekKey21", "value21", "EXAT", "seconds"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("seconds value should be an integer"),
			},
			{
				name: "22. Set exact expiry time in milliseconds from unix epoch",
				command: []string{
					"SET", "SetKey22", "value22",
					"PXAT", fmt.Sprintf("%d", mockClock.Now().Add(4096*time.Millisecond).UnixMilli()),
				},
				presetValues:     nil,
				expectedResponse: "OK",
				expectedValue:    "value22",
				expectedExpiry:   mockClock.Now().Add(4096 * time.Millisecond),
				expectedErr:      nil,
			},
			{
				name: "23. Return error when trying to set exact milliseconds expiry time when expiry time is already provided",
				command: []string{
					"SET", "SetKey23", "value23",
					"PX", "1000",
					"PXAT", fmt.Sprintf("%d", mockClock.Now().Add(4096*time.Millisecond).UnixMilli()),
				},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("cannot specify PXAT when expiry time is already set"),
			},
			{
				name:             "24. Return error when no milliseconds value is provided after PXAT flag",
				command:          []string{"SET", "SetKey24", "value24", "PXAT"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("milliseconds value required after PXAT"),
			},
			{
				name:             "25. Return error when invalid (non-integer) value is passed after EXAT flag",
				command:          []string{"SET", "SetKey25", "value25", "PXAT", "unix-milliseconds"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "",
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("milliseconds value should be an integer"),
			},
			{
				name:    "26. Get the previous value when GET flag is passed",
				command: []string{"SET", "SetKey26", "value26", "GET", "EX", "1000"},
				presetValues: map[string]KeyData{
					"SetKey26": {
						Value:    "previous-value",
						ExpireAt: time.Time{},
					},
				},
				expectedResponse: "previous-value",
				expectedValue:    "value26",
				expectedExpiry:   mockClock.Now().Add(1000 * time.Second),
				expectedErr:      nil,
			},
			{
				name:             "27. Return nil when GET value is passed and no previous value exists",
				command:          []string{"SET", "SetKey27", "value27", "GET", "EX", "1000"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    "value27",
				expectedExpiry:   mockClock.Now().Add(1000 * time.Second),
				expectedErr:      nil,
			},
			{
				name:             "28. Throw error when unknown optional flag is passed to SET command.",
				command:          []string{"SET", "SetKey28", "value28", "UNKNOWN-OPTION"},
				presetValues:     nil,
				expectedResponse: nil,
				expectedValue:    nil,
				expectedExpiry:   time.Time{},
				expectedErr:      errors.New("unknown option UNKNOWN-OPTION for set command"),
			},
			{
				name:             "29. Command too short",
				command:          []string{"SET"},
				expectedResponse: nil,
				expectedValue:    nil,
				expectedErr:      errors.New(constants.WrongArgsResponse),
			},
			{
				name:             "30. Command too long",
				command:          []string{"SET", "SetKey30", "value1", "value2", "value3", "value4", "value5", "value6"},
				expectedResponse: nil,
				expectedValue:    nil,
				expectedErr:      errors.New(constants.WrongArgsResponse),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.presetValues != nil {
					for k, v := range test.presetValues {
						cmd := []resp.Value{
							resp.StringValue("SET"),
							resp.StringValue(k),
							resp.StringValue(v.Value.(string))}
						err := client.WriteArray(cmd)
						if err != nil {
							t.Error(err)
						}
						rd, _, err := client.ReadValue()
						if err != nil {
							t.Error(err)
						}
						if !strings.EqualFold(rd.String(), "ok") {
							t.Errorf("expected preset response to be \"OK\", got %s", rd.String())
						}
					}
				}

				command := make([]resp.Value, len(test.command))
				for j, c := range test.command {
					command[j] = resp.StringValue(c)
				}

				if err = client.WriteArray(command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()

				if test.expectedErr != nil {
					if !strings.Contains(res.Error().Error(), test.expectedErr.Error()) {
						t.Errorf("expected error \"%s\", got \"%s\"", test.expectedErr.Error(), err.Error())
					}
					return
				}
				if err != nil {
					t.Error(err)
				}

				switch test.expectedResponse.(type) {
				case string:
					if test.expectedResponse != res.String() {
						t.Errorf("expected response \"%s\", got \"%s\"", test.expectedResponse, res.String())
					}
				case nil:
					if !res.IsNull() {
						t.Errorf("expcted nil response, got %+v", res)
					}
				default:
					t.Error("test expected result should be nil or string")
				}

				key := test.command[1]

				// Compare expected value to response value
				if err = client.WriteArray([]resp.Value{resp.StringValue("GET"), resp.StringValue(key)}); err != nil {
					t.Error(err)
				}
				res, _, err = client.ReadValue()
				if err != nil {
					t.Error(err)
				}
				if res.String() != test.expectedValue.(string) {
					t.Errorf("expected value %s, got %s", test.expectedValue.(string), res.String())
				}

				// Compare expected expiry to response expiry
				if !test.expectedExpiry.Equal(time.Time{}) {
					if err = client.WriteArray([]resp.Value{resp.StringValue("EXPIRETIME"), resp.StringValue(key)}); err != nil {
						t.Error(err)
					}
					res, _, err = client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					if res.Integer() != int(test.expectedExpiry.Unix()) {
						t.Errorf("expected expiry time %d, got %d", test.expectedExpiry.Unix(), res.Integer())
					}
				}
			})
		}
	})

	t.Run("Test_HandleMSET", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name             string
			command          []string
			expectedResponse string
			expectedValues   map[string]interface{}
			expectedErr      error
		}{
			{
				name:             "1. Set multiple key value pairs",
				command:          []string{"MSET", "MsetKey1", "value1", "MsetKey2", "10", "MsetKey3", "3.142"},
				expectedResponse: "OK",
				expectedValues:   map[string]interface{}{"MsetKey1": "value1", "MsetKey2": 10, "MsetKey3": 3.142},
				expectedErr:      nil,
			},
			{
				name:             "2. Return error when keys and values are not even",
				command:          []string{"MSET", "MsetKey1", "value1", "MsetKey2", "10", "MsetKey3"},
				expectedResponse: "",
				expectedValues:   make(map[string]interface{}),
				expectedErr:      errors.New("each key must be paired with a value"),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				command := make([]resp.Value, len(test.command))
				for j, c := range test.command {
					command[j] = resp.StringValue(c)
				}

				if err = client.WriteArray(command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if test.expectedErr != nil {
					if !strings.Contains(res.Error().Error(), test.expectedErr.Error()) {
						t.Errorf("expected error %s, got %s", test.expectedErr.Error(), err.Error())
					}
					return
				}

				if res.String() != test.expectedResponse {
					t.Errorf("expected response %s, got %s", test.expectedResponse, res.String())
				}

				for key, expectedValue := range test.expectedValues {
					// Get value from server
					if err = client.WriteArray([]resp.Value{resp.StringValue("GET"), resp.StringValue(key)}); err != nil {
						t.Error(err)
					}
					res, _, err = client.ReadValue()
					switch expectedValue.(type) {
					default:
						t.Error("unexpected type for expectedValue")
					case int:
						ev, _ := expectedValue.(int)
						if res.Integer() != ev {
							t.Errorf("expected value %d for key %s, got %d", ev, key, res.Integer())
						}
					case float64:
						ev, _ := expectedValue.(float64)
						if res.Float() != ev {
							t.Errorf("expected value %f for key %s, got %f", ev, key, res.Float())
						}
					case string:
						ev, _ := expectedValue.(string)
						if res.String() != ev {
							t.Errorf("expected value %s for key %s, got %s", ev, key, res.String())
						}
					}
				}
			})
		}
	})

	t.Run("Test_HandleGET", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name  string
			key   string
			value string
		}{
			{
				name:  "1. String",
				key:   "GetKey1",
				value: "value1",
			},
			{
				name:  "2. Integer",
				key:   "GetKey2",
				value: "10",
			},
			{
				name:  "3. Float",
				key:   "GetKey3",
				value: "3.142",
			},
		}
		// Test successful Get command
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				func(key, value string) {
					// Preset the values
					err = client.WriteArray([]resp.Value{resp.StringValue("SET"), resp.StringValue(key), resp.StringValue(value)})
					if err != nil {
						t.Error(err)
					}

					res, _, err := client.ReadValue()
					if err != nil {
						t.Error(err)
					}

					if !strings.EqualFold(res.String(), "ok") {
						t.Errorf("expected preset response to be \"OK\", got %s", res.String())
					}

					if err = client.WriteArray([]resp.Value{resp.StringValue("GET"), resp.StringValue(key)}); err != nil {
						t.Error(err)
					}

					res, _, err = client.ReadValue()
					if err != nil {
						t.Error(err)
					}

					if res.String() != test.value {
						t.Errorf("expected value %s, got %s", test.value, res.String())
					}
				}(test.key, test.value)
			})
		}

		// Test get non-existent key
		if err = client.WriteArray([]resp.Value{resp.StringValue("GET"), resp.StringValue("test4")}); err != nil {
			t.Error(err)
		}
		res, _, err := client.ReadValue()
		if err != nil {
			t.Error(err)
		}
		if !res.IsNull() {
			t.Errorf("expected nil, got: %+v", res)
		}

		errorTests := []struct {
			name     string
			command  []string
			expected string
		}{
			{
				name:     "1. Return error when no GET key is passed",
				command:  []string{"GET"},
				expected: constants.WrongArgsResponse,
			},
			{
				name:     "2. Return error when too many GET keys are passed",
				command:  []string{"GET", "GetKey1", "test"},
				expected: constants.WrongArgsResponse,
			},
		}
		for _, test := range errorTests {
			t.Run(test.name, func(t *testing.T) {
				command := make([]resp.Value, len(test.command))
				for i, c := range test.command {
					command[i] = resp.StringValue(c)
				}

				if err = client.WriteArray(command); err != nil {
					t.Error(err)
				}

				res, _, err = client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if !strings.Contains(res.Error().Error(), test.expected) {
					t.Errorf("expected error '%s', got: %s", test.expected, err.Error())
				}
			})
		}
	})

	t.Run("Test_HandleMGET", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name          string
			presetKeys    []string
			presetValues  []string
			command       []string
			expected      []interface{}
			expectedError error
		}{
			{
				name:          "1. MGET multiple existing values",
				presetKeys:    []string{"MgetKey1", "MgetKey2", "MgetKey3", "MgetKey4"},
				presetValues:  []string{"value1", "value2", "value3", "value4"},
				command:       []string{"MGET", "MgetKey1", "MgetKey4", "MgetKey2", "MgetKey3", "MgetKey1"},
				expected:      []interface{}{"value1", "value4", "value2", "value3", "value1"},
				expectedError: nil,
			},
			{
				name:          "2. MGET multiple values with nil values spliced in",
				presetKeys:    []string{"MgetKey5", "MgetKey6", "MgetKey7"},
				presetValues:  []string{"value5", "value6", "value7"},
				command:       []string{"MGET", "MgetKey5", "MgetKey6", "non-existent", "non-existent", "MgetKey7", "non-existent"},
				expected:      []interface{}{"value5", "value6", nil, nil, "value7", nil},
				expectedError: nil,
			},
			{
				name:          "3. Return error when MGET is invoked with no keys",
				presetKeys:    []string{"MgetKey5"},
				presetValues:  []string{"value5"},
				command:       []string{"MGET"},
				expected:      nil,
				expectedError: errors.New(constants.WrongArgsResponse),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				// Set up the values
				for i, key := range test.presetKeys {
					if err = client.WriteArray([]resp.Value{
						resp.StringValue("SET"),
						resp.StringValue(key),
						resp.StringValue(test.presetValues[i]),
					}); err != nil {
						t.Error(err)
					}
					res, _, err := client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					if !strings.EqualFold(res.String(), "ok") {
						t.Errorf("expected preset response to be \"OK\", got \"%s\"", res.String())
					}
				}

				// Test the command and its results
				command := make([]resp.Value, len(test.command))
				for i, c := range test.command {
					command[i] = resp.StringValue(c)
				}

				if err = client.WriteArray(command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if test.expectedError != nil {
					// If we expect and error, branch out and check error
					if !strings.Contains(res.Error().Error(), test.expectedError.Error()) {
						t.Errorf("expected error %+v, got: %+v", test.expectedError, err)
					}
					return
				}

				if res.Type().String() != "Array" {
					t.Errorf("expected type Array, got: %s", res.Type().String())
				}
				for i, value := range res.Array() {
					if test.expected[i] == nil {
						if !value.IsNull() {
							t.Errorf("expected nil value, got %+v", value)
						}
						continue
					}
					if value.String() != test.expected[i] {
						t.Errorf("expected value %s, got: %s", test.expected[i], value.String())
					}
				}
			})
		}
	})

	t.Run("Test_HandleDEL", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name             string
			command          []string
			presetValues     map[string]string
			expectedResponse int
			expectToExist    map[string]bool
			expectedErr      error
		}{
			{
				name:    "1. Delete multiple keys",
				command: []string{"DEL", "DelKey1", "DelKey2", "DelKey3", "DelKey4", "DelKey5"},
				presetValues: map[string]string{
					"DelKey1": "value1",
					"DelKey2": "value2",
					"DelKey3": "value3",
					"DelKey4": "value4",
				},
				expectedResponse: 4,
				expectToExist: map[string]bool{
					"DelKey1": false,
					"DelKey2": false,
					"DelKey3": false,
					"DelKey4": false,
					"DelKey5": false,
				},
				expectedErr: nil,
			},
			{
				name:             "2. Return error when DEL is called with no keys",
				command:          []string{"DEL"},
				presetValues:     nil,
				expectedResponse: 0,
				expectToExist:    nil,
				expectedErr:      errors.New(constants.WrongArgsResponse),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.presetValues != nil {
					for k, v := range test.presetValues {
						if err = client.WriteArray([]resp.Value{
							resp.StringValue("SET"),
							resp.StringValue(k),
							resp.StringValue(v),
						}); err != nil {
							t.Error(err)
						}
						res, _, err := client.ReadValue()
						if err != nil {
							t.Error(err)
						}
						if !strings.EqualFold(res.String(), "ok") {
							t.Errorf("expected preset response to be \"OK\", got %s", res.String())
						}
					}
				}

				command := make([]resp.Value, len(test.command))
				for i, c := range test.command {
					command[i] = resp.StringValue(c)
				}

				if err = client.WriteArray(command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if test.expectedErr != nil {
					if !strings.Contains(res.Error().Error(), test.expectedErr.Error()) {
						t.Errorf("expected error \"%s\", got \"%s\"", test.expectedErr.Error(), res.Error().Error())
					}
					return
				}

				if res.Integer() != test.expectedResponse {
					t.Errorf("expected response %d, got %d", test.expectedResponse, res.Integer())
				}

				for key, expected := range test.expectToExist {
					if err = client.WriteArray([]resp.Value{resp.StringValue("GET"), resp.StringValue(key)}); err != nil {
						t.Error(err)
					}
					res, _, err = client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					exists := !res.IsNull()
					if exists != expected {
						t.Errorf("expected existence of key %s to be %v, got %v", key, expected, exists)
					}
				}
			})
		}
	})

	t.Run("Test_HandlePERSIST", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name             string
			command          []string
			presetValues     map[string]KeyData
			expectedResponse int
			expectedValues   map[string]KeyData
			expectedError    error
		}{
			{
				name:    "1. Successfully persist a volatile key",
				command: []string{"PERSIST", "PersistKey1"},
				presetValues: map[string]KeyData{
					"PersistKey1": {Value: "value1", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"PersistKey1": {Value: "value1", ExpireAt: time.Time{}},
				},
				expectedError: nil,
			},
			{
				name:             "2. Return 0 when trying to persist a non-existent key",
				command:          []string{"PERSIST", "PersistKey2"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    nil,
			},
			{
				name:    "3. Return 0 when trying to persist a non-volatile key",
				command: []string{"PERSIST", "PersistKey3"},
				presetValues: map[string]KeyData{
					"PersistKey3": {Value: "value3", ExpireAt: time.Time{}},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"PersistKey3": {Value: "value3", ExpireAt: time.Time{}},
				},
				expectedError: nil,
			},
			{
				name:             "4. Command too short",
				command:          []string{"PERSIST"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
			{
				name:             "5. Command too long",
				command:          []string{"PERSIST", "PersistKey5", "key6"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.presetValues != nil {
					for k, v := range test.presetValues {
						command := []resp.Value{resp.StringValue("SET"), resp.StringValue(k), resp.StringValue(v.Value.(string))}
						if !v.ExpireAt.Equal(time.Time{}) {
							command = append(command, []resp.Value{
								resp.StringValue("PX"),
								resp.StringValue(fmt.Sprintf("%d", v.ExpireAt.Sub(mockClock.Now()).Milliseconds())),
							}...)
						}
						if err = client.WriteArray(command); err != nil {
							t.Error(err)
						}
						res, _, err := client.ReadValue()
						if err != nil {
							t.Error(err)
						}
						if !strings.EqualFold(res.String(), "ok") {
							t.Errorf("expected preset response to be OK, got %s", res.String())
						}
					}
				}

				command := make([]resp.Value, len(test.command))
				for i, c := range test.command {
					command[i] = resp.StringValue(c)
				}

				if err = client.WriteArray(command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if test.expectedError != nil {
					if !strings.Contains(res.Error().Error(), test.expectedError.Error()) {
						t.Errorf("expected error \"%s\", got \"%s\"", test.expectedError.Error(), err.Error())
					}
					return
				}

				if res.Integer() != test.expectedResponse {
					t.Errorf("expected response %d, got %d", test.expectedResponse, res.Integer())
				}

				if test.expectedValues == nil {
					return
				}

				for key, expected := range test.expectedValues {
					// Compare the value of the key with what's expected
					if err = client.WriteArray([]resp.Value{resp.StringValue("GET"), resp.StringValue(key)}); err != nil {
						t.Error(err)
					}
					res, _, err = client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					if res.String() != expected.Value.(string) {
						t.Errorf("expected value %s, got %s", expected.Value.(string), res.String())
					}
					// Compare the expiry of the key with what's expected
					if err = client.WriteArray([]resp.Value{resp.StringValue("PTTL"), resp.StringValue(key)}); err != nil {
						t.Error(err)
					}
					res, _, err = client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					if expected.ExpireAt.Equal(time.Time{}) {
						if res.Integer() != -1 {
							t.Error("expected key to be persisted, it was not.")
						}
						continue
					}
					if res.Integer() != int(expected.ExpireAt.UnixMilli()) {
						t.Errorf("expected expiry %d, got %d", expected.ExpireAt.UnixMilli(), res.Integer())
					}
				}
			})
		}
	})

	t.Run("Test_HandleEXPIRETIME", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name             string
			command          []string
			presetValues     map[string]KeyData
			expectedResponse int
			expectedError    error
		}{
			{
				name:    "1. Return expire time in seconds",
				command: []string{"EXPIRETIME", "ExpireTimeKey1"},
				presetValues: map[string]KeyData{
					"ExpireTimeKey1": {Value: "value1", ExpireAt: mockClock.Now().Add(100 * time.Second)},
				},
				expectedResponse: int(mockClock.Now().Add(100 * time.Second).Unix()),
				expectedError:    nil,
			},
			{
				name:    "2. Return expire time in milliseconds",
				command: []string{"PEXPIRETIME", "ExpireTimeKey2"},
				presetValues: map[string]KeyData{
					"ExpireTimeKey2": {Value: "value2", ExpireAt: mockClock.Now().Add(4096 * time.Millisecond)},
				},
				expectedResponse: int(mockClock.Now().Add(4096 * time.Millisecond).UnixMilli()),
				expectedError:    nil,
			},
			{
				name:    "3. If the key is non-volatile, return -1",
				command: []string{"PEXPIRETIME", "ExpireTimeKey3"},
				presetValues: map[string]KeyData{
					"ExpireTimeKey3": {Value: "value3", ExpireAt: time.Time{}},
				},
				expectedResponse: -1,
				expectedError:    nil,
			},
			{
				name:             "4. If the key is non-existent return -2",
				command:          []string{"PEXPIRETIME", "ExpireTimeKey4"},
				presetValues:     nil,
				expectedResponse: -2,
				expectedError:    nil,
			},
			{
				name:             "5. Command too short",
				command:          []string{"PEXPIRETIME"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
			{
				name:             "6. Command too long",
				command:          []string{"PEXPIRETIME", "ExpireTimeKey5", "ExpireTimeKey6"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.presetValues != nil {
					for k, v := range test.presetValues {
						command := []resp.Value{resp.StringValue("SET"), resp.StringValue(k), resp.StringValue(v.Value.(string))}
						if !v.ExpireAt.Equal(time.Time{}) {
							command = append(command, []resp.Value{
								resp.StringValue("PX"),
								resp.StringValue(fmt.Sprintf("%d", v.ExpireAt.Sub(mockClock.Now()).Milliseconds())),
							}...)
						}
						if err = client.WriteArray(command); err != nil {
							t.Error(err)
						}
						res, _, err := client.ReadValue()
						if err != nil {
							t.Error(err)
						}
						if !strings.EqualFold(res.String(), "ok") {
							t.Errorf("expected preset response to be OK, got %s", res.String())
						}
					}
				}

				command := make([]resp.Value, len(test.command))
				for i, c := range test.command {
					command[i] = resp.StringValue(c)
				}

				if err = client.WriteArray(command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if test.expectedError != nil {
					if !strings.Contains(res.Error().Error(), test.expectedError.Error()) {
						t.Errorf("expected error \"%s\", got \"%s\"", test.expectedError.Error(), err.Error())
					}
					return
				}

				if res.Integer() != test.expectedResponse {
					t.Errorf("expected response %d, got %d", test.expectedResponse, res.Integer())
				}
			})
		}
	})

	t.Run("Test_HandleTTL", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name             string
			command          []string
			presetValues     map[string]KeyData
			expectedResponse int
			expectedError    error
		}{
			{
				name:    "1. Return TTL time in seconds",
				command: []string{"TTL", "TTLKey1"},
				presetValues: map[string]KeyData{
					"TTLKey1": {Value: "value1", ExpireAt: mockClock.Now().Add(100 * time.Second)},
				},
				expectedResponse: 100,
				expectedError:    nil,
			},
			{
				name:    "2. Return TTL time in milliseconds",
				command: []string{"PTTL", "TTLKey2"},
				presetValues: map[string]KeyData{
					"TTLKey2": {Value: "value2", ExpireAt: mockClock.Now().Add(4096 * time.Millisecond)},
				},
				expectedResponse: 4096,
				expectedError:    nil,
			},
			{
				name:    "3. If the key is non-volatile, return -1",
				command: []string{"TTL", "TTLKey3"},
				presetValues: map[string]KeyData{
					"TTLKey3": {Value: "value3", ExpireAt: time.Time{}},
				},
				expectedResponse: -1,
				expectedError:    nil,
			},
			{
				name:             "4. If the key is non-existent return -2",
				command:          []string{"TTL", "TTLKey4"},
				presetValues:     nil,
				expectedResponse: -2,
				expectedError:    nil,
			},
			{
				name:             "5. Command too short",
				command:          []string{"TTL"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
			{
				name:             "6. Command too long",
				command:          []string{"TTL", "TTLKey5", "TTLKey6"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.presetValues != nil {
					for k, v := range test.presetValues {
						command := []resp.Value{resp.StringValue("SET"), resp.StringValue(k), resp.StringValue(v.Value.(string))}
						if !v.ExpireAt.Equal(time.Time{}) {
							command = append(command, []resp.Value{
								resp.StringValue("PX"),
								resp.StringValue(fmt.Sprintf("%d", v.ExpireAt.Sub(mockClock.Now()).Milliseconds())),
							}...)
						}
						if err = client.WriteArray(command); err != nil {
							t.Error(err)
						}
						res, _, err := client.ReadValue()
						if err != nil {
							t.Error(err)
						}
						if !strings.EqualFold(res.String(), "ok") {
							t.Errorf("expected preset response to be OK, got %s", res.String())
						}
					}
				}

				command := make([]resp.Value, len(test.command))
				for i, c := range test.command {
					command[i] = resp.StringValue(c)
				}

				if err = client.WriteArray(command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if test.expectedError != nil {
					if !strings.Contains(res.Error().Error(), test.expectedError.Error()) {
						t.Errorf("expected error \"%s\", got \"%s\"", test.expectedError.Error(), err.Error())
					}
					return
				}

				if res.Integer() != test.expectedResponse {
					t.Errorf("expected response %d, got %d", test.expectedResponse, res.Integer())
				}
			})
		}
	})

	t.Run("Test_HandleEXPIRE", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name             string
			command          []string
			presetValues     map[string]KeyData
			expectedResponse int
			expectedValues   map[string]KeyData
			expectedError    error
		}{
			{
				name:    "1. Set new expire by seconds",
				command: []string{"EXPIRE", "ExpireKey1", "100"},
				presetValues: map[string]KeyData{
					"ExpireKey1": {Value: "value1", ExpireAt: time.Time{}},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireKey1": {Value: "value1", ExpireAt: mockClock.Now().Add(100 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name:    "2. Set new expire by milliseconds",
				command: []string{"PEXPIRE", "ExpireKey2", "1000"},
				presetValues: map[string]KeyData{
					"ExpireKey2": {Value: "value2", ExpireAt: time.Time{}},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireKey2": {Value: "value2", ExpireAt: mockClock.Now().Add(1000 * time.Millisecond)},
				},
				expectedError: nil,
			},
			{
				name:    "3. Set new expire only when key does not have an expiry time with NX flag",
				command: []string{"EXPIRE", "ExpireKey3", "1000", "NX"},
				presetValues: map[string]KeyData{
					"ExpireKey3": {Value: "value3", ExpireAt: time.Time{}},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireKey3": {Value: "value3", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name:    "4. Return 0, when NX flag is provided and key already has an expiry time",
				command: []string{"EXPIRE", "ExpireKey4", "1000", "NX"},
				presetValues: map[string]KeyData{
					"ExpireKey4": {Value: "value4", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"ExpireKey4": {Value: "value4", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name:    "5. Set new expire time from now key only when the key already has an expiry time with XX flag",
				command: []string{"EXPIRE", "ExpireKey5", "1000", "XX"},
				presetValues: map[string]KeyData{
					"ExpireKey5": {Value: "value5", ExpireAt: mockClock.Now().Add(30 * time.Second)},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireKey5": {Value: "value5", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name:    "6. Return 0 when key does not have an expiry and the XX flag is provided",
				command: []string{"EXPIRE", "ExpireKey6", "1000", "XX"},
				presetValues: map[string]KeyData{
					"ExpireKey6": {Value: "value6", ExpireAt: time.Time{}},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"ExpireKey6": {Value: "value6", ExpireAt: time.Time{}},
				},
				expectedError: nil,
			},
			{
				name:    "7. Set expiry time when the provided time is after the current expiry time when GT flag is provided",
				command: []string{"EXPIRE", "ExpireKey7", "1000", "GT"},
				presetValues: map[string]KeyData{
					"ExpireKey7": {Value: "value7", ExpireAt: mockClock.Now().Add(30 * time.Second)},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireKey7": {Value: "value7", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name:    "8. Return 0 when GT flag is passed and current expiry time is greater than provided time",
				command: []string{"EXPIRE", "ExpireKey8", "1000", "GT"},
				presetValues: map[string]KeyData{
					"ExpireKey8": {Value: "value8", ExpireAt: mockClock.Now().Add(3000 * time.Second)},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"ExpireKey8": {Value: "value8", ExpireAt: mockClock.Now().Add(3000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name:    "9. Return 0 when GT flag is passed and key does not have an expiry time",
				command: []string{"EXPIRE", "ExpireKey9", "1000", "GT"},
				presetValues: map[string]KeyData{
					"ExpireKey9": {Value: "value9", ExpireAt: time.Time{}},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"ExpireKey9": {Value: "value9", ExpireAt: time.Time{}},
				},
				expectedError: nil,
			},
			{
				name:    "10. Set expiry time when the provided time is before the current expiry time when LT flag is provided",
				command: []string{"EXPIRE", "ExpireKey10", "1000", "LT"},
				presetValues: map[string]KeyData{
					"ExpireKey10": {Value: "value10", ExpireAt: mockClock.Now().Add(3000 * time.Second)},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireKey10": {Value: "value10", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name:    "11. Return 0 when LT flag is passed and current expiry time is less than provided time",
				command: []string{"EXPIRE", "ExpireKey11", "5000", "LT"},
				presetValues: map[string]KeyData{
					"ExpireKey11": {Value: "value11", ExpireAt: mockClock.Now().Add(3000 * time.Second)},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"ExpireKey11": {Value: "value11", ExpireAt: mockClock.Now().Add(3000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name:    "12. Return 0 when LT flag is passed and key does not have an expiry time",
				command: []string{"EXPIRE", "ExpireKey12", "1000", "LT"},
				presetValues: map[string]KeyData{
					"ExpireKey12": {Value: "value12", ExpireAt: time.Time{}},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireKey12": {Value: "value12", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name:    "13. Return error when unknown flag is passed",
				command: []string{"EXPIRE", "ExpireKey13", "1000", "UNKNOWN"},
				presetValues: map[string]KeyData{
					"ExpireKey13": {Value: "value13", ExpireAt: time.Time{}},
				},
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    errors.New("unknown option UNKNOWN"),
			},
			{
				name:             "14. Return error when expire time is not a valid integer",
				command:          []string{"EXPIRE", "ExpireKey14", "expire"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    errors.New("expire time must be integer"),
			},
			{
				name:             "15. Command too short",
				command:          []string{"EXPIRE"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
			{
				name:             "16. Command too long",
				command:          []string{"EXPIRE", "ExpireKey16", "10", "NX", "GT"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.presetValues != nil {
					for k, v := range test.presetValues {
						command := []resp.Value{resp.StringValue("SET"), resp.StringValue(k), resp.StringValue(v.Value.(string))}
						if !v.ExpireAt.Equal(time.Time{}) {
							command = append(command, []resp.Value{
								resp.StringValue("PX"),
								resp.StringValue(fmt.Sprintf("%d", v.ExpireAt.Sub(mockClock.Now()).Milliseconds())),
							}...)
						}
						if err = client.WriteArray(command); err != nil {
							t.Error(err)
						}
						res, _, err := client.ReadValue()
						if err != nil {
							t.Error(err)
						}
						if !strings.EqualFold(res.String(), "ok") {
							t.Errorf("expected preset response to be OK, got %s", res.String())
						}
					}
				}

				command := make([]resp.Value, len(test.command))
				for i, c := range test.command {
					command[i] = resp.StringValue(c)
				}

				if err = client.WriteArray(command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if test.expectedError != nil {
					if !strings.Contains(res.Error().Error(), test.expectedError.Error()) {
						t.Errorf("expected error \"%s\", got \"%s\"", test.expectedError.Error(), err.Error())
					}
					return
				}

				if res.Integer() != test.expectedResponse {
					t.Errorf("expected response %d, got %d", test.expectedResponse, res.Integer())
				}

				if test.expectedValues == nil {
					return
				}

				for key, expected := range test.expectedValues {
					// Compare the value of the key with what's expected
					if err = client.WriteArray([]resp.Value{resp.StringValue("GET"), resp.StringValue(key)}); err != nil {
						t.Error(err)
					}
					res, _, err = client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					if res.String() != expected.Value.(string) {
						t.Errorf("expected value %s, got %s", expected.Value.(string), res.String())
					}
					// Compare the expiry of the key with what's expected
					if err = client.WriteArray([]resp.Value{resp.StringValue("PTTL"), resp.StringValue(key)}); err != nil {
						t.Error(err)
					}
					res, _, err = client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					if expected.ExpireAt.Equal(time.Time{}) {
						if res.Integer() != -1 {
							t.Error("expected key to be persisted, it was not.")
						}
						continue
					}
					if res.Integer() != int(expected.ExpireAt.Sub(mockClock.Now()).Milliseconds()) {
						t.Errorf("expected expiry %d, got %d", expected.ExpireAt.Sub(mockClock.Now()).Milliseconds(), res.Integer())
					}
				}
			})
		}
	})

	t.Run("Test_HandleEXPIREAT", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name             string
			command          []string
			presetValues     map[string]KeyData
			expectedResponse int
			expectedValues   map[string]KeyData
			expectedError    error
		}{
			{
				name:    "1. Set new expire by unix seconds",
				command: []string{"EXPIREAT", "ExpireAtKey1", fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).Unix())},
				presetValues: map[string]KeyData{
					"ExpireAtKey1": {Value: "value1", ExpireAt: time.Time{}},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireAtKey1": {Value: "value1", ExpireAt: time.Unix(mockClock.Now().Add(1000*time.Second).Unix(), 0)},
				},
				expectedError: nil,
			},
			{
				name:    "2. Set new expire by milliseconds",
				command: []string{"PEXPIREAT", "ExpireAtKey2", fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).UnixMilli())},
				presetValues: map[string]KeyData{
					"ExpireAtKey2": {Value: "value2", ExpireAt: time.Time{}},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireAtKey2": {Value: "value2", ExpireAt: time.UnixMilli(mockClock.Now().Add(1000 * time.Second).UnixMilli())},
				},
				expectedError: nil,
			},
			{
				name:    "3. Set new expire only when key does not have an expiry time with NX flag",
				command: []string{"EXPIREAT", "ExpireAtKey3", fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).Unix()), "NX"},
				presetValues: map[string]KeyData{
					"ExpireAtKey3": {Value: "value3", ExpireAt: time.Time{}},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireAtKey3": {Value: "value3", ExpireAt: time.Unix(mockClock.Now().Add(1000*time.Second).Unix(), 0)},
				},
				expectedError: nil,
			},
			{
				name:    "4. Return 0, when NX flag is provided and key already has an expiry time",
				command: []string{"EXPIREAT", "ExpireAtKey4", fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).Unix()), "NX"},
				presetValues: map[string]KeyData{
					"ExpireAtKey4": {Value: "value4", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"ExpireAtKey4": {Value: "value4", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name: "5. Set new expire time from now key only when the key already has an expiry time with XX flag",
				command: []string{
					"EXPIREAT", "ExpireAtKey5",
					fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).Unix()), "XX",
				},
				presetValues: map[string]KeyData{
					"ExpireAtKey5": {Value: "value5", ExpireAt: mockClock.Now().Add(30 * time.Second)},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireAtKey5": {Value: "value5", ExpireAt: time.Unix(mockClock.Now().Add(1000*time.Second).Unix(), 0)},
				},
				expectedError: nil,
			},
			{
				name: "6. Return 0 when key does not have an expiry and the XX flag is provided",
				command: []string{
					"EXPIREAT", "ExpireAtKey6",
					fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).Unix()), "XX",
				},
				presetValues: map[string]KeyData{
					"ExpireAtKey6": {Value: "value6", ExpireAt: time.Time{}},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"ExpireAtKey6": {Value: "value6", ExpireAt: time.Time{}},
				},
				expectedError: nil,
			},
			{
				name: "7. Set expiry time when the provided time is after the current expiry time when GT flag is provided",
				command: []string{
					"EXPIREAT", "ExpireAtKey7",
					fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).Unix()), "GT",
				},
				presetValues: map[string]KeyData{
					"ExpireAtKey7": {Value: "value7", ExpireAt: mockClock.Now().Add(30 * time.Second)},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireAtKey7": {Value: "value7", ExpireAt: time.Unix(mockClock.Now().Add(1000*time.Second).Unix(), 0)},
				},
				expectedError: nil,
			},
			{
				name: "8. Return 0 when GT flag is passed and current expiry time is greater than provided time",
				command: []string{
					"EXPIREAT", "ExpireAtKey8",
					fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).Unix()), "GT",
				},
				presetValues: map[string]KeyData{
					"ExpireAtKey8": {Value: "value8", ExpireAt: mockClock.Now().Add(3000 * time.Second)},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"ExpireAtKey8": {Value: "value8", ExpireAt: mockClock.Now().Add(3000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name: "9. Return 0 when GT flag is passed and key does not have an expiry time",
				command: []string{
					"EXPIREAT", "ExpireAtKey9",
					fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).Unix()), "GT",
				},
				presetValues: map[string]KeyData{
					"ExpireAtKey9": {Value: "value9", ExpireAt: time.Time{}},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"ExpireAtKey9": {Value: "value9", ExpireAt: time.Time{}},
				},
				expectedError: nil,
			},
			{
				name: "10. Set expiry time when the provided time is before the current expiry time when LT flag is provided",
				command: []string{
					"EXPIREAT", "ExpireAtKey10",
					fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).Unix()), "LT",
				},
				presetValues: map[string]KeyData{
					"ExpireAtKey10": {Value: "value10", ExpireAt: mockClock.Now().Add(3000 * time.Second)},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireAtKey10": {Value: "value10", ExpireAt: time.Unix(mockClock.Now().Add(1000*time.Second).Unix(), 0)},
				},
				expectedError: nil,
			},
			{
				name: "11. Return 0 when LT flag is passed and current expiry time is less than provided time",
				command: []string{
					"EXPIREAT", "ExpireAtKey11",
					fmt.Sprintf("%d", mockClock.Now().Add(3000*time.Second).Unix()), "LT",
				},
				presetValues: map[string]KeyData{
					"ExpireAtKey11": {Value: "value11", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedResponse: 0,
				expectedValues: map[string]KeyData{
					"ExpireAtKey11": {Value: "value11", ExpireAt: mockClock.Now().Add(1000 * time.Second)},
				},
				expectedError: nil,
			},
			{
				name: "12. Return 0 when LT flag is passed and key does not have an expiry time",
				command: []string{
					"EXPIREAT", "ExpireAtKey12",
					fmt.Sprintf("%d", mockClock.Now().Add(1000*time.Second).Unix()), "LT",
				},
				presetValues: map[string]KeyData{
					"ExpireAtKey12": {Value: "value12", ExpireAt: time.Time{}},
				},
				expectedResponse: 1,
				expectedValues: map[string]KeyData{
					"ExpireAtKey12": {Value: "value12", ExpireAt: time.Unix(mockClock.Now().Add(1000*time.Second).Unix(), 0)},
				},
				expectedError: nil,
			},
			{
				name:    "13. Return error when unknown flag is passed",
				command: []string{"EXPIREAT", "ExpireAtKey13", "1000", "UNKNOWN"},
				presetValues: map[string]KeyData{
					"ExpireAtKey13": {Value: "value13", ExpireAt: time.Time{}},
				},
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    errors.New("unknown option UNKNOWN"),
			},
			{
				name:             "14. Return error when expire time is not a valid integer",
				command:          []string{"EXPIREAT", "ExpireAtKey14", "expire"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    errors.New("expire time must be integer"),
			},
			{
				name:             "15. Command too short",
				command:          []string{"EXPIREAT"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
			{
				name:             "16. Command too long",
				command:          []string{"EXPIREAT", "ExpireAtKey16", "10", "NX", "GT"},
				presetValues:     nil,
				expectedResponse: 0,
				expectedValues:   nil,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.presetValues != nil {
					for k, v := range test.presetValues {
						command := []resp.Value{resp.StringValue("SET"), resp.StringValue(k), resp.StringValue(v.Value.(string))}
						if !v.ExpireAt.Equal(time.Time{}) {
							command = append(command, []resp.Value{
								resp.StringValue("PX"),
								resp.StringValue(fmt.Sprintf("%d", v.ExpireAt.Sub(mockClock.Now()).Milliseconds())),
							}...)
						}
						if err = client.WriteArray(command); err != nil {
							t.Error(err)
						}
						res, _, err := client.ReadValue()
						if err != nil {
							t.Error(err)
						}
						if !strings.EqualFold(res.String(), "ok") {
							t.Errorf("expected preset response to be OK, got %s", res.String())
						}
					}
				}

				command := make([]resp.Value, len(test.command))
				for i, c := range test.command {
					command[i] = resp.StringValue(c)
				}

				if err = client.WriteArray(command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if test.expectedError != nil {
					if !strings.Contains(res.Error().Error(), test.expectedError.Error()) {
						t.Errorf("expected error \"%s\", got \"%s\"", test.expectedError.Error(), err.Error())
					}
					return
				}

				if res.Integer() != test.expectedResponse {
					t.Errorf("expected response %d, got %d", test.expectedResponse, res.Integer())
				}

				if test.expectedValues == nil {
					return
				}

				for key, expected := range test.expectedValues {
					// Compare the value of the key with what's expected
					if err = client.WriteArray([]resp.Value{resp.StringValue("GET"), resp.StringValue(key)}); err != nil {
						t.Error(err)
					}
					res, _, err = client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					if res.String() != expected.Value.(string) {
						t.Errorf("expected value %s, got %s", expected.Value.(string), res.String())
					}
					// Compare the expiry of the key with what's expected
					if err = client.WriteArray([]resp.Value{resp.StringValue("PTTL"), resp.StringValue(key)}); err != nil {
						t.Error(err)
					}
					res, _, err = client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					if expected.ExpireAt.Equal(time.Time{}) {
						if res.Integer() != -1 {
							t.Error("expected key to be persisted, it was not.")
						}
						continue
					}
					if res.Integer() != int(expected.ExpireAt.Sub(mockClock.Now()).Milliseconds()) {
						t.Errorf("expected expiry %d, got %d", expected.ExpireAt.Sub(mockClock.Now()).Milliseconds(), res.Integer())
					}
				}
			})
		}
	})

	t.Run("Test_HandlerINCR", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name             string
			key              string
			presetValue      interface{}
			command          []resp.Value
			expectedResponse int64
			expectedError    error
		}{
			{
				name:             "1. Increment non-existent key",
				key:              "IncrKey1",
				presetValue:      nil,
				command:          []resp.Value{resp.StringValue("INCR"), resp.StringValue("IncrKey1")},
				expectedResponse: 1,
				expectedError:    nil,
			},
			{
				name:             "2. Increment existing key with integer value",
				key:              "IncrKey2",
				presetValue:      "5",
				command:          []resp.Value{resp.StringValue("INCR"), resp.StringValue("IncrKey2")},
				expectedResponse: 6,
				expectedError:    nil,
			},
			{
				name:             "3. Increment existing key with non-integer value",
				key:              "IncrKey3",
				presetValue:      "not_an_int",
				command:          []resp.Value{resp.StringValue("INCR"), resp.StringValue("IncrKey3")},
				expectedResponse: 0,
				expectedError:    errors.New("value is not an integer or out of range"),
			},
			{
				name:             "4. Increment existing key with int64 value",
				key:              "IncrKey4",
				presetValue:      int64(10),
				command:          []resp.Value{resp.StringValue("INCR"), resp.StringValue("IncrKey4")},
				expectedResponse: 11,
				expectedError:    nil,
			},
			{
				name:             "5. Command too short",
				key:              "IncrKey5",
				presetValue:      nil,
				command:          []resp.Value{resp.StringValue("INCR")},
				expectedResponse: 0,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
			{
				name:        "6. Command too long",
				key:         "IncrKey6",
				presetValue: nil,
				command: []resp.Value{
					resp.StringValue("INCR"),
					resp.StringValue("IncrKey6"),
					resp.StringValue("IncrKey6"),
				},
				expectedResponse: 0,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.presetValue != nil {
					command := []resp.Value{resp.StringValue("SET"), resp.StringValue(test.key), resp.StringValue(fmt.Sprintf("%v", test.presetValue))}
					if err = client.WriteArray(command); err != nil {
						t.Error(err)
					}
					res, _, err := client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					if !strings.EqualFold(res.String(), "ok") {
						t.Errorf("expected preset response to be OK, got %s", res.String())
					}
				}

				if err = client.WriteArray(test.command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if test.expectedError != nil {
					if !strings.Contains(res.Error().Error(), test.expectedError.Error()) {
						t.Errorf("expected error \"%s\", got \"%s\"", test.expectedError.Error(), err.Error())
					}
					return
				}

				if err != nil {
					t.Error(err)
				} else {
					responseInt, err := strconv.ParseInt(res.String(), 10, 64)
					if err != nil {
						t.Errorf("error parsing response to int64: %s", err)
					}
					if responseInt != test.expectedResponse {
						t.Errorf("expected response %d, got %d", test.expectedResponse, responseInt)
					}
				}
			})
		}
	})

	t.Run("Test_HandlerDECR", func(t *testing.T) {
		t.Parallel()
		conn, err := internal.GetConnection("localhost", port)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		client := resp.NewConn(conn)

		tests := []struct {
			name             string
			key              string
			presetValue      interface{}
			command          []resp.Value
			expectedResponse int64
			expectedError    error
		}{
			{
				name:             "1. Increment non-existent key",
				key:              "DecrKey1",
				presetValue:      nil,
				command:          []resp.Value{resp.StringValue("DECR"), resp.StringValue("DecrKey1")},
				expectedResponse: -1,
				expectedError:    nil,
			},
			{
				name:             "2. Decrement existing key with integer value",
				key:              "DecrKey2",
				presetValue:      "5",
				command:          []resp.Value{resp.StringValue("DECR"), resp.StringValue("DecrKey2")},
				expectedResponse: 4,
				expectedError:    nil,
			},
			{
				name:             "3. Decrement existing key with non-integer value",
				key:              "DecrKey3",
				presetValue:      "not_an_int",
				command:          []resp.Value{resp.StringValue("DECR"), resp.StringValue("DecrKey3")},
				expectedResponse: 0,
				expectedError:    errors.New("value is not an integer or out of range"),
			},
			{
				name:             "4. Decrement existing key with int64 value",
				key:              "DecrKey4",
				presetValue:      int64(10),
				command:          []resp.Value{resp.StringValue("DECR"), resp.StringValue("DecrKey4")},
				expectedResponse: 9,
				expectedError:    nil,
			},
			{
				name:             "5. Command too short",
				key:              "DencrKey5",
				presetValue:      nil,
				command:          []resp.Value{resp.StringValue("DECR")},
				expectedResponse: 0,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
			{
				name:        "6. Command too long",
				key:         "DecrKey6",
				presetValue: nil,
				command: []resp.Value{
					resp.StringValue("DECR"),
					resp.StringValue("DecrKey6"),
					resp.StringValue("DecrKey6"),
				},
				expectedResponse: 0,
				expectedError:    errors.New(constants.WrongArgsResponse),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.presetValue != nil {
					command := []resp.Value{resp.StringValue("SET"), resp.StringValue(test.key), resp.StringValue(fmt.Sprintf("%v", test.presetValue))}
					if err = client.WriteArray(command); err != nil {
						t.Error(err)
					}
					res, _, err := client.ReadValue()
					if err != nil {
						t.Error(err)
					}
					if !strings.EqualFold(res.String(), "ok") {
						t.Errorf("expected preset response to be OK, got %s", res.String())
					}
				}

				if err = client.WriteArray(test.command); err != nil {
					t.Error(err)
				}

				res, _, err := client.ReadValue()
				if err != nil {
					t.Error(err)
				}

				if test.expectedError != nil {
					if !strings.Contains(res.Error().Error(), test.expectedError.Error()) {
						t.Errorf("expected error \"%s\", got \"%s\"", test.expectedError.Error(), err.Error())
					}
					return
				}

				if err != nil {
					t.Error(err)
				} else {
					responseInt, err := strconv.ParseInt(res.String(), 10, 64)
					if err != nil {
						t.Errorf("error parsing response to int64: %s", err)
					}
					if responseInt != test.expectedResponse {
						t.Errorf("expected response %d, got %d", test.expectedResponse, responseInt)
					}
				}
			})
		}
	})
}

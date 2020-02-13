package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/wrp-go/v2"
	"github.com/xmidt-org/wrp-go/v2/wrphttp"
)

func TestNewFanoutHandler(t *testing.T) {
	assert := assert.New(t)

	assert.Panics(func() {
		NewWRPFanoutHandler(nil)
	})

	assert.NotPanics(func() {
		assert.NotNil(NewWRPFanoutHandler(http.NotFoundHandler()))
	})
}

func TestNewWRPFanoutHandlerWithPIDCheck(t *testing.T) {
	assert := assert.New(t)

	assert.Panics(func() {
		NewWRPFanoutHandlerWithPIDCheck(http.NotFoundHandler(), nil)
	})

	assert.Panics(func() {
		NewWRPFanoutHandlerWithPIDCheck(nil, &wrpPartnersAccess{})
	})
}

func TestFanoutRequest(t *testing.T) {
	testCases := []struct {
		Name         string
		Recorder     *httptest.ResponseRecorder
		Entity       *wrphttp.Entity
		Err          error
		ExpectedCode int
		Modify       bool
	}{
		{
			Name:         "wrp check errors",
			Recorder:     httptest.NewRecorder(),
			Entity:       new(wrphttp.Entity),
			Err:          ErrTokenMissing,
			ExpectedCode: 500,
		},
		{
			Name:     "wrp gets modified - happy path",
			Modify:   true,
			Recorder: httptest.NewRecorder(),
			Entity: &wrphttp.Entity{
				Format: wrp.Msgpack,
				Message: wrp.Message{
					Destination: "mac:1122334455/service",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			mockWRPAccessAuthority := new(mockWRPAccessAuthority)
			wrpFanoutHandler := NewWRPFanoutHandlerWithPIDCheck(
				http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}), mockWRPAccessAuthority)

			wrpResponseWriter := newTestWRPResponseWriter(testCase.Recorder)

			r := httptest.NewRequest(http.MethodGet, "http://localhost", nil)

			wrpRequest := &wrphttp.Request{
				Original: r,
				Entity:   testCase.Entity,
			}

			mockWRPAccessAuthority.On("authorizeWRP", r.Context(), &testCase.Entity.Message).Return(testCase.Modify, testCase.Err)

			wrpFanoutHandler.ServeWRP(wrpResponseWriter, wrpRequest)

			if testCase.Err != nil {
				assert.Equal(testCase.ExpectedCode, testCase.Recorder.Code)
			} else {
				outgoingBody, err := ioutil.ReadAll(r.Body)
				assert.Nil(err)
				assert.Equal(int64(len(outgoingBody)), r.ContentLength)
				assert.Equal(testCase.Entity.Format.ContentType(), r.Header.Get("Content-Type"))
				assert.Equal(testCase.Entity.Message.Destination, r.Header.Get("X-Webpa-Device-Name"))
			}
		})
	}
}

type testWRPResponseWriter struct {
	http.ResponseWriter
}

func (t *testWRPResponseWriter) WriteWRP(i interface{}) (int, error) {
	return 0, nil
}

func newTestWRPResponseWriter(w *httptest.ResponseRecorder) *testWRPResponseWriter {
	return &testWRPResponseWriter{
		ResponseWriter: w,
	}
}

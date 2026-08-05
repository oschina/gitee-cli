package cmdtest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

type TestFactory struct {
	*cmdutil.Factory
	OutBuf    *bytes.Buffer
	ErrOutBuf *bytes.Buffer
	Server    *httptest.Server
}

func NewTestFactory(handler http.Handler) *TestFactory {
	srv := httptest.NewServer(handler)

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    outBuf,
		ErrOut: errBuf,
	}

	f := &cmdutil.Factory{
		IOStreams: ios,
		Context:   context.Background(),
		GiteeClient: func() (*gitee.Client, error) {
			return gitee.NewClient("test-token", gitee.WithBaseURL(srv.URL)), nil
		},
	}

	return &TestFactory{
		Factory:   f,
		OutBuf:    outBuf,
		ErrOutBuf: errBuf,
		Server:    srv,
	}
}

func (tf *TestFactory) Close() {
	tf.Server.Close()
}

func (tf *TestFactory) Output() string {
	return tf.OutBuf.String()
}

func JSONHandler(t interface{}, statusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(t)
	}
}

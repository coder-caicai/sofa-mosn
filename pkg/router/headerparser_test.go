package router

import (
	"reflect"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func Test_headerParser_evaluateHeaders(t *testing.T) {
	parser := &headerParser{
		headersToAdd: []*headerPair{&headerPair{
			headerName: &lowerCaseString{"level"},
			headerFormatter: &plainHeaderFormatter{
				isAppend:    false,
				staticValue: "1",
			},
		},
		},
		headersToRemove: []*lowerCaseString{&lowerCaseString{"status"}},
	}
	type args struct {
		headers     types.HeaderMap
		requestInfo types.RequestInfo
	}

	tests := []struct {
		name string
		args args
		want types.HeaderMap
	}{
		{
			name: "case1",
			args: args{
				headers:     protocol.CommonHeader{"status": "normal"},
				requestInfo: nil,
			},
			want: protocol.CommonHeader{"level": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser.evaluateHeaders(tt.args.headers, tt.args.requestInfo)
			if !reflect.DeepEqual(tt.args.headers, tt.want) {
				t.Errorf("(h *headerParser) evaluateHeaders(headers map[string]string, requestInfo types.RequestInfo) = %v, want %v", tt.args.headers, tt.want)
			}
		})
	}
}
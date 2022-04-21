package http

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewParser(t *testing.T) {
	type args struct {
		config *ParserConfig
	}
	tests := []struct {
		name string
		args args
		want *parser
	}{
		{
			name: "nil",
			args: args{
				config: nil,
			},
			want: nil,
		},
		{
			name: "success",
			args: args{
				config: &ParserConfig{
					RealIPHeader:           "",
					SendHeaders:            true,
					SendAllHeaders:         true,
					HeadersWhitelist:       nil,
					IncludeRequestBodyFor:  nil,
					IncludeResponseBodyFor: nil,
				},
			},
			want: &parser{
				config: &parserConfig{
					realIPHeader:           "",
					sendHeaders:            true,
					sendAllHeaders:         true,
					headersWhitelist:       nil,
					includeRequestBodyFor:  nil,
					includeResponseBodyFor: nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewParser(tt.args.config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewParser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parser_Parse(t *testing.T) {
	raw := `POST /api/v1/alert/common/2988466/574c8739021047c35049b2ac76472c2be8d46e15 HTTP/1.1
Content-Type: application/json
User-Agent: PostmanRuntime/7.29.0
Accept: */*
Cache-Control: no-cache
Postman-Token: bb12d58b-d4eb-4d4e-85e6-126545640c3f
Host: 192.168.100.163:8149
Accept-Encoding: gzip, deflate, br
Connection: keep-alive
Content-Length: 336

{
    "alertId": "test",
    "org": 8888,
    "objectId": "HOST",
    "source": "zabbix",
    "instanceId": "5b1339263cc7a",
    "notifyContent": "<kestrel>",
    "metricName": "system_load_15",
    "content": "第三方事件信息",
    "accessId": "574c8739021047c35049b2ac76472c2be8d46e15",
    "isRecover": false,
    "value": 32
}`
	request := []byte(strings.Replace(raw, "\n", "\r\n", -1))

	raw = `HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Date: Wed, 13 Apr 2022 10:41:16 GMT
Content-Length: 66

{"code":0,"codeExplain":"","error":"","data":{"status":"success"}}`
	response := []byte(strings.Replace(raw, "\n", "\r\n", -1))

	type fields struct {
		config *ParserConfig
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Message
		want1  bool
		want2  bool
	}{
		{
			name: "request",
			fields: fields{
				config: &ParserConfig{
					RealIPHeader:           "",
					SendHeaders:            true,
					SendAllHeaders:         true,
					HeadersWhitelist:       nil,
					IncludeRequestBodyFor:  nil,
					IncludeResponseBodyFor: nil,
				},
			},
			args: args{
				data: request,
			},
			want: &Message{
				Version: Version{
					Major: 1,
					Minor: 1,
				},
				IsRequest:  true,
				StatusCode: 0,
			},
			want1: true,
			want2: true,
		},
		{
			name: "response",
			fields: fields{
				config: &ParserConfig{
					RealIPHeader:           "",
					SendHeaders:            true,
					SendAllHeaders:         true,
					HeadersWhitelist:       nil,
					IncludeRequestBodyFor:  nil,
					IncludeResponseBodyFor: nil,
				},
			},
			args: args{
				data: response,
			},
			want: &Message{
				Version: Version{
					Major: 1,
					Minor: 1,
				},
				IsRequest:  false,
				StatusCode: 200,
			},
			want1: true,
			want2: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.fields.config)
			got, got1, got2 := parser.Parse(tt.args.data)
			if got.StatusCode != tt.want.StatusCode || !reflect.DeepEqual(got.Version, tt.want.Version) ||
				got.IsRequest != tt.want.IsRequest {
				t.Errorf("Parse() code = %d, want %d", got.StatusCode, tt.want.StatusCode)
				t.Errorf("Parse() version = %s, want %s", got.Version.String(), tt.want.Version.String())
				t.Errorf("Parse() isRequest = %v, want %v", got.IsRequest, tt.want.IsRequest)
			}
			if got1 != tt.want1 {
				t.Errorf("Parse() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("Parse() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func Test_convertConfig(t *testing.T) {
	type args struct {
		config *ParserConfig
	}
	tests := []struct {
		name string
		args args
		want *parserConfig
	}{
		{
			name: "success",
			args: args{
				config: &ParserConfig{
					RealIPHeader:           "",
					SendHeaders:            true,
					SendAllHeaders:         true,
					HeadersWhitelist:       nil,
					IncludeRequestBodyFor:  nil,
					IncludeResponseBodyFor: nil,
				},
			},
			want: &parserConfig{
				realIPHeader:           "",
				sendHeaders:            true,
				sendAllHeaders:         true,
				headersWhitelist:       nil,
				includeRequestBodyFor:  nil,
				includeResponseBodyFor: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertConfig(tt.args.config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertMessage(t *testing.T) {
	type args struct {
		message *message
	}
	tests := []struct {
		name string
		args args
		want *Message
	}{
		{
			name: "success",
			args: args{
				message: &message{
					version: version{
						major: 1,
						minor: 1,
					},
					statusCode: 200,
				},
			},
			want: &Message{
				Version: Version{
					Major: 1,
					Minor: 1,
				},
				StatusCode: 200,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertMessage(tt.args.message); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	type fields struct {
		Major uint8
		Minor uint8
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "success",
			fields: fields{
				Major: 1,
				Minor: 1,
			},
			want: "1.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Version{
				Major: tt.fields.Major,
				Minor: tt.fields.Minor,
			}
			if got := v.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertVersion(t *testing.T) {
	type args struct {
		ver version
	}
	tests := []struct {
		name string
		args args
		want Version
	}{
		{
			name: "success",
			args: args{
				ver: version{
					major: 1,
					minor: 1,
				},
			},
			want: Version{
				Major: 1,
				Minor: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertVersion(tt.args.ver); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

package module

import "testing"

func TestCheckModuleFormatAcceptsBlackbox028Fields(t *testing.T) {
	data := []byte(`
modules:
  http_http3:
    prober: http
    timeout: 5s
    http:
      method: GET
      enable_http3: true
      enable_http2: false
      follow_redirects: true
  grpc_metadata:
    prober: grpc
    grpc:
      tls: true
      metadata:
        authorization:
          - Bearer token
        x-tenant-id:
          - tenant-a
  tcp_expect_bytes:
    prober: tcp
    tcp:
      query_response:
        - expect_bytes: "\\x00\\x01"
          send: "ping"
`)

	ok, err := CheckModuleFormat(data)
	if !ok || err != nil {
		t.Fatalf("expected new 0.28 fields to pass validation, got ok=%v err=%v", ok, err)
	}
}

func TestCheckModuleFormatRejectsInvalidBlackbox028FieldTypes(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "http_enable_http3_type",
			data: `
modules:
  http_http3:
    prober: http
    http:
      enable_http3: "true"
`,
		},
		{
			name: "grpc_metadata_type",
			data: `
modules:
  grpc_metadata:
    prober: grpc
    grpc:
      metadata: invalid
`,
		},
		{
			name: "tcp_expect_bytes_type",
			data: `
modules:
  tcp_expect_bytes:
    prober: tcp
    tcp:
      query_response:
        - expect_bytes: 12
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := CheckModuleFormat([]byte(tt.data))
			if ok || err == nil {
				t.Fatalf("expected validation failure, got ok=%v err=%v", ok, err)
			}
		})
	}
}

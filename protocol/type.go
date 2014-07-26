package protocol

import (
	"net/http"
)

type HttpRequestEvent struct {
	SessionID uint32
	RawReq    *http.Request
}
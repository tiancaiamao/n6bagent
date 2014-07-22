package proxy

import "github.com/tiancaiamao/n6bagent/event"

var c4_cfg *C4Config = &C4Config{
	Compressor:             event.COMPRESSOR_SNAPPY,
	Encrypter:              event.ENCRYPTER_RC4,
	UA:                     "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:15.0) Gecko/20100101 Firefox/15.0.1",
	ReadTimeout:            25,
	MaxConn:                5,
	WSConnKeepAlive:        180,
	InjectRange:            initHostMatchRegex("*.c.youtube.com|av.vimeo.com|av.voanews.com"),
	FetchLimitSize:         256000,
	ConcurrentRangeFetcher: 5,
}

var workerNode []string = []string{
	"n6bagent.herokuapp.com",
}

var RC4Key = "8976501f8451f03c5c4067b47882f2e5"

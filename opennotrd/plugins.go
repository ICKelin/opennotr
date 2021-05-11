package opennotrd

import (
	// plugin import
	_ "github.com/ICKelin/opennotr/opennotrd/plugin/dummy"
	_ "github.com/ICKelin/opennotr/opennotrd/plugin/restyproxy"
	_ "github.com/ICKelin/opennotr/opennotrd/plugin/tcpproxy"
	_ "github.com/ICKelin/opennotr/opennotrd/plugin/udpproxy"
)

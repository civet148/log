package log

import (
	"fmt"
	"github.com/civet148/gotools/gopprof"
)

func StartProf(strListenAddr string) error {
	strMessage := fmt.Sprintf(`
	http://%s/debug/pprof/threadcreate?debug=1
	http://%s/debug/pprof/goroutine?debug=1
	http://%s/debug/pprof/heap?debug=1`,
	strListenAddr, strListenAddr, strListenAddr)

	Infof(strMessage)
	if err := gopprof.Start(strListenAddr, false); err != nil {
		return fmt.Errorf("listen %s error %s", strListenAddr, err)
	}
	return nil
}

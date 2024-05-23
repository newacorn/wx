package helpers

import "log"

func init() {
	log.SetFlags(log.Llongfile | log.Ltime)
}
func Log(message ...interface{}) {
	log.Println(message...)
}

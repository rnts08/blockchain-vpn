package tunnel

import (
	"io"
	"log"
)

func copyStreamWithControl(dst io.Writer, src io.Reader, onRead func(int), limiter *rateEnforcer) {
	buf := make([]byte, 32*1024)
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if limiter != nil {
				limiter.accountAndThrottle(n)
			}
			if onRead != nil {
				onRead(n)
			}
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				log.Printf("Info: stream copy write ended: %v", writeErr)
				return
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				log.Printf("Info: stream copy read ended: %v", readErr)
			}
			return
		}
	}
}

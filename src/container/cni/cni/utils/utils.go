package utils

import (
	"log"
	"time"
)

func DoWithRetries(retryCount int, timeWait time.Duration, f func() error) error {
	var err error
	for retryCount > 0 {
		err = f()
		if err != nil {
			retryCount--
			log.Printf("Got error %v. Retries left %d.\n", err, retryCount)
			time.Sleep(timeWait)
		} else {
			break
		}
	}
	return err
}

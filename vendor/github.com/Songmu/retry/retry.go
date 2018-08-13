package retry

import "time"

// Retry retry N times
func Retry(trial uint, interval time.Duration, f func() error) (err error) {
	for trial > 0 {
		trial--
		err = f()
		if err == nil || trial <= 0 {
			break
		}
		time.Sleep(interval)
	}
	return err
}

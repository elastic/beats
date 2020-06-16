package tlsconfig

import (
	"crypto/x509"
	"fmt"
	"time"
)

const timeFormat = "2006-01-02 15:04:05 MST"

func checkExpiration(cert *x509.Certificate) error {
	now := time.Now()

	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid: validity starts at %s but current time is %s", cert.NotBefore.Format(timeFormat), now.Format(timeFormat))
	}

	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate has expired: validity ended at %s but current time is %s", cert.NotAfter.Format(timeFormat), now.Format(timeFormat))
	}

	return nil
}

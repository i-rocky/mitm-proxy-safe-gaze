package main

import "crypto/tls"

// CertsStorage is an example of a custom cert storage.
type CertsStorage struct {
	// certsCache is a cache with the generated certificates.
	certsCache map[string]*tls.Certificate
}

// Get gets the certificate from the storage.
func (c *CertsStorage) Get(key string) (cert *tls.Certificate, ok bool) {
	cert, ok = c.certsCache[key]

	return cert, ok
}

// Set saves the certificate to the storage.
func (c *CertsStorage) Set(key string, cert *tls.Certificate) {
	c.certsCache[key] = cert
}

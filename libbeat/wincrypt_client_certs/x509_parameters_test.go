package wincrypt_client_certs

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestX509Parameters_Get(t *testing.T) {
	InitMocks(false)

	p := (*X509Parameters)(NytimesX509Cert)

	get := func(name string) interface{} {
		result, err := p.Get(name)
		assert.NoError(t, err)
		return result
	}

	assert.Equal(t, 3, get("X509_Version"))
	assert.Equal(t, "bfd6231bea59aad6cb566ed0878d60ed", get("X509_SerialNumber"))
	assert.Equal(t, "SHA256-RSA", get("X509_SignatureAlgorithm"))
	assert.Equal(t, "CN=COMODO RSA Organization Validation Secure Server CA,O=COMODO CA Limited,L=Salford,ST=Greater Manchester,C=GB", get("X509_Issuer"))
	assert.Equal(t, "COMODO RSA Organization Validation Secure Server CA", get("X509_Issuer_CommonName"))
	assert.Equal(t, "", get("X509_Issuer_SerialNumber"))
	assert.Equal(t, []interface{}{"GB"}, get("X509_Issuer_Country"))
	assert.Equal(t, []interface{}{"Salford"}, get("X509_Issuer_Locality"))
	assert.Equal(t, []interface{}{"COMODO CA Limited"}, get("X509_Issuer_Organization"))
	assert.Equal(t, []interface{}([]interface{}{}), get("X509_Issuer_OrganizationalUnit"))
	assert.Equal(t, []interface{}{"Greater Manchester"}, get("X509_Issuer_Province"))
	assert.Equal(t, []interface{}{}, get("X509_Issuer_StreetAddress"))
	assert.Equal(t, []interface{}{}, get("X509_Issuer_PostalCode"))
	assert.Equal(t, "CN=nytimes.com,OU=The New York Times+OU=Multi-Domain SSL,O=The New York Times,POSTALCODE=10018,STREET=620 8th Ave,L=New York,ST=New York,C=US", get("X509_Subject"))
	assert.Equal(t, "nytimes.com", get("X509_Subject_CommonName"))
	assert.Equal(t, "", get("X509_Subject_SerialNumber"))
	assert.Equal(t, []interface{}{"US"}, get("X509_Subject_Country"))
	assert.Equal(t, []interface{}{"New York"}, get("X509_Subject_Locality"))
	assert.Equal(t, []interface{}{"The New York Times"}, get("X509_Subject_Organization"))
	assert.Equal(t, []interface{}{"The New York Times", "Multi-Domain SSL"}, get("X509_Subject_OrganizationalUnit"))
	assert.Equal(t, []interface{}{"New York"}, get("X509_Subject_Province"))
	assert.Equal(t, []interface{}{"620 8th Ave"}, get("X509_Subject_StreetAddress"))
	assert.Equal(t, []interface{}{"10018"}, get("X509_Subject_PostalCode"))
	assert.Equal(t, "RSA", get("X509_PublicKeyAlgorithm"))
	assert.Equal(t, "9af32bdacfad4fb62fbb2a48482a12b71b42c124", get("X509_AuthorityKeyId"))
	assert.Equal(t, "8622ac0657dd85d0603d0300b7ddee16dc990534", get("X509_SubjectKeyId"))
	assert.Equal(t, "309686edf9bff832c6e2ab9f0954190e10241655", get("X509_Fingerprint"))
	assert.Equal(t, []interface{}{"ServerAuth", "ClientAuth"}, get("X509_ExtKeyUsage"))

	from, _ := time.Parse(time.RFC3339, "2018-11-29T00:00:00+00:00")
	assert.Equal(t, from.Unix(), get("X509_ValidFrom").(time.Time).Unix())

	to, _ := time.Parse(time.RFC3339, "2020-01-18T23:59:59+00:00")
	assert.Equal(t, to.Unix(), get("X509_ValidTo").(time.Time).Unix())

	_, err := p.Get("BOGUS")
	assert.Error(t, err)
}

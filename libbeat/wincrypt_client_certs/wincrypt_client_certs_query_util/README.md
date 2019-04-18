# Wincrypt Client Certs Query Util

This util dumps all variables for all certificates with an accessible private key for the given search criteria (query and stores).

Run `go run github.com/elastic/beats/libbeat/wincrypt_client_certs/wincrypt_client_certs_query_util --help` for more information.

**Example**:
```
$ go run github.com/elastic/beats/libbeat/wincrypt_client_certs/wincrypt_client_certs_query_util \
    --stores "LocalMachine/My" \
    --stores "CurrentUser/My" \
    --query "X509_Issuer_CommonName=='Puppet CA: mngt-emperor-rz1-01.lxprod.ka.de.dm-drogeriemarkt.com'"
time="2019-04-22T11:47:30+02:00" level=info msg=certificate
X509_Version: 3
X509_SerialNumber: 7b0d
X509_SignatureAlgorithm: SHA256-RSA
X509_Issuer: CN=Puppet CA: mngt-emperor-rz1-01.lxprod.ka.de.dm-drogeriemarkt.com
X509_Issuer_CommonName: Puppet CA: mngt-emperor-rz1-01.lxprod.ka.de.dm-drogeriemarkt.com
X509_Issuer_SerialNumber:
X509_Issuer_Country: []
X509_Issuer_Locality: []
X509_Issuer_Organization: []
X509_Issuer_OrganizationalUnit: []
X509_Issuer_Province: []
X509_Issuer_StreetAddress: []
X509_Issuer_PostalCode: []
X509_Subject: CN=dmpos-elkpoc-01.certificate.poc
X509_Subject_CommonName: dmpos-elkpoc-01.certificate.poc
X509_Subject_SerialNumber:
X509_Subject_Country: []
X509_Subject_Locality: []
X509_Subject_Organization: []
X509_Subject_OrganizationalUnit: []
X509_Subject_Province: []
X509_Subject_StreetAddress: []
X509_Subject_PostalCode: []
X509_PublicKeyAlgorithm: RSA
X509_AuthorityKeyId: 04fe3d350c2047170294cdd1419a3b2691ca77a0
X509_SubjectKeyId: fc4268c97b384a407aca2daef3fcbba2d13c269b
X509_Fingerprint: 48abf5566e20231fcca33e2ce6d151daa34aaf76
X509_ExtKeyUsage: [ServerAuth ClientAuth]

time="2019-04-22T11:47:30+02:00" level=info msg="common variables"
Hostname: FD800201
User_Name: FD800201
User_Username: FD800201
```

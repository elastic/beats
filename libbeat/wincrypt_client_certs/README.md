# TLS Client/Server Authentication for GO through wincrypt

This package adds the ability to use certificates stored in a windows certificate storage with the offical go tls package.

## Motivation

Windows allows storing x509 certificates with privates keys for client/server authentication
in the [local machine and current user certificate store](https://docs.microsoft.com/de-de/dotnet/framework/wcf/feature-details/working-with-certificates#certificate-stores).

Signed certificates can be relatively easy distributed and get automatically renewed with
[GPOs](https://docs.microsoft.com/de-de/windows-server/identity/ad-fs/deployment/distribute-certificates-to-client-computers-by-using-group-policy)
to windows machines. To improve security certificates can be marked as non-exportable which makes it harder to steal private keys,
[although these restrictions can be circumvented relatively easily by an attacker that has obtained admin rights](https://github.com/gentilkiwi/mimikatz#crypto).

These certificates can be used through the [wincrypt-API](https://docs.microsoft.com/en-us/windows/desktop/api/wincrypt/)
and [ncypt-API](https://docs.microsoft.com/en-us/windows/desktop/api/ncrypt/) for decryption and signing.

## Usage

Example: 

```
config := &wincrypt_tls_provieder.Config{
    Stores: []string{"LocalMachine/My"},
    Query: "X509_Subject_CommonName == 'nytimes.com'",
}

var wincrypt, err = wincrypt_tls_provieder.New(config)
if err != nil {
    panic(err)
}

tlsConfig := &tls.Config{
    [...]
    GetClientCertificate: cert_store.GetClientCertificate
    [...]
}
```

### Config

The `wincrypt_tls_provieder.Config` struct takes the following parameters:

#### Stores

**Example**: `[]string{'CurrentUser/My', 'LocalMachine/My'}`

A list of windows system stores to look for certificates.
A system store can be located in `LocalMachines` or `CurrentUser`.
Predefined system stores are:
 - `My`
 - `Root`
 - `Trust`
 - `CA` 

See also: https://docs.microsoft.com/en-us/windows-hardware/drivers/install/certificate-stores

#### Query

**Example**: `Subject_CommonName == 'nytimes.com'`

The query parameter is a boolean expression used to filter certificates.
The query is parsed with [govaluate](https://github.com/Knetic/govaluate)

You can list variables for available certificates and test queries with the [wincrypt client certs query util](wincrypt_client_certs_query_util/README.md).

##### Variables

**certificate related**

The following variables can be used in the query to access information about the certificate:


|  Variable                       | Type     | Example Value |
| ------------------------------- | -------- | ------------- |
| X509_Version                    | int      | 3 |
| X509_SerialNumber               | string   | "bfd6231bea59aad6cb566ed0878d60ed" |
| X509_SignatureAlgorithm         | string   | "SHA256-RSA" |
| X509_Issuer                     | string   | "CN=COMODO RSA Organization Validation Secure Server CA,O=COMODO CA Limited,L=Salford,ST=Greater Manchester,C=GB" |
| X509_Issuer_CommonName          | string   | "COMODO RSA Organization Validation Secure Server CA" |
| X509_Issuer_SerialNumber        | string   | "" |
| X509_Issuer_Country             | string   | "GB" 
| X509_Issuer_Locality            | []string | []string{"Salford"} |
| X509_Issuer_Organization        | []string | []string{"COMODO CA Limited"} |
| X509_Issuer_OrganizationalUnit  | []string | []string{"COMODO CA Limited"} |
| X509_Issuer_Province            | []string | []string{"Greater Manchester"} |
| X509_Issuer_StreetAddress       | []string | []string(nil) |
| X509_Issuer_PostalCode          | []string | []string(nil) |
| X509_Subject                    | string   | "CN=nytimes.com,OU=The New York Times+OU=Multi-Domain SSL,O=The New York Times,POSTALCODE=10018,STREET=620 8th Ave,L=New York,ST=New York,C=US" |
| X509_Subject_CommonName         | string   | "nytimes.com" |
| X509_Subject_SerialNumber       | string   | "" |
| X509_Subject_Country            | []string | []string{"US"} |
| X509_Subject_Locality           | []string | []string{"New York"} |
| X509_Subject_Organization       | []string | []string{"The New York Times"} |
| X509_Subject_OrganizationalUnit | []string | []string{"The New York Times", "Multi-Domain SSL"} |
| X509_Subject_Province           | []string | []string{"New York"} |
| X509_Subject_StreetAddress      | []string | []string{"620 8th Ave"} |
| X509_Subject_PostalCode         | []string | []string{"10018"} |
| X509_PublicKeyAlgorithm         | string   | "RSA" |
| X509_AuthorityKeyId             | string   | "9af32bdacfad4fb62fbb2a48482a12b71b42c124" |
| X509_SubjectKeyId               | string   | "8622ac0657dd85d0603d0300b7ddee16dc990534" |
| X509_Fingerprint                | string   | "309686edf9bff832c6e2ab9f0954190e10241655" |
| X509_ExtKeyUsage                | []string | []string{"ServerAuth", "ClientAuth"} |

**execution context related**

The following variables can be used in the query to access information about the current host and user:

|  Variable                       | Type     | Example Value |
| ------------------------------- | -------- | ------------- |
| Hostname                        | string   | "MyComputer"  |
| User_Username                   | string   | 'DM\John'     |
| User_Name                       | string   | "John"        |

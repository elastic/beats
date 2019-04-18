package cmd

import (
	"crypto/x509"
	"fmt"
	"github.com/elastic/beats/libbeat/wincrypt_client_certs"
	"os"

	"github.com/spf13/cobra"
	"github.com/Sirupsen/logrus"
)

var rootCmd = &cobra.Command{
	Use:   "wincrypt_client_certs_query_util",
	Short: "Util to test wincrypt_client_certs queries",
	Long: `This util dumps all variables for all certificates with an accessible private key for the given search criteria.

Read github.com/elastic/libbeat/wincrypt_client-certs/README.md for more information.`,

	Run: func(cmd *cobra.Command, args []string) {
		query, err := cmd.Flags().GetString("query")
		if err != nil { logrus.Fatal(err) }

		stores, err := cmd.Flags().GetStringSlice("stores")
		if err != nil { logrus.Fatal(err) }

		config := &wincrypt_client_certs.Config{
			Stores: stores,
			Query:  query,
		}

		var params wincrypt_client_certs.X509Parameters
		get := func(name string) interface{} {
			 result, err := params.Get(name)
			 if err != nil {
				  logrus.Error(err)
				  return ""
			 }
			 return result
		}

		store, err := wincrypt_client_certs.New(config)
		if err != nil {
			logrus.Error(err)
		} else {
			for _, c := range (store.Certificates) {
				params = wincrypt_client_certs.X509Parameters(*c.Leaf)

				logrus.Info("certificate")
				fmt.Printf("%s: %v\n", "X509_Version", get("X509_Version"))
				fmt.Printf("%s: %v\n", "X509_SerialNumber", get("X509_SerialNumber"))
				fmt.Printf("%s: %v\n", "X509_SignatureAlgorithm", get("X509_SignatureAlgorithm"))
				fmt.Printf("%s: %v\n", "X509_Issuer", get("X509_Issuer"))
				fmt.Printf("%s: %v\n", "X509_Issuer_CommonName", get("X509_Issuer_CommonName"))
				fmt.Printf("%s: %v\n", "X509_Issuer_SerialNumber", get("X509_Issuer_SerialNumber"))
				fmt.Printf("%s: %v\n", "X509_Issuer_Country", get("X509_Issuer_Country"))
				fmt.Printf("%s: %v\n", "X509_Issuer_Locality", get("X509_Issuer_Locality"))
				fmt.Printf("%s: %v\n", "X509_Issuer_Organization", get("X509_Issuer_Organization"))
				fmt.Printf("%s: %v\n", "X509_Issuer_OrganizationalUnit", get("X509_Issuer_OrganizationalUnit"))
				fmt.Printf("%s: %v\n", "X509_Issuer_Province", get("X509_Issuer_Province"))
				fmt.Printf("%s: %v\n", "X509_Issuer_StreetAddress", get("X509_Issuer_StreetAddress"))
				fmt.Printf("%s: %v\n", "X509_Issuer_PostalCode", get("X509_Issuer_PostalCode"))
				fmt.Printf("%s: %v\n", "X509_Subject", get("X509_Subject"))
				fmt.Printf("%s: %v\n", "X509_Subject_CommonName", get("X509_Subject_CommonName"))
				fmt.Printf("%s: %v\n", "X509_Subject_SerialNumber", get("X509_Subject_SerialNumber"))
				fmt.Printf("%s: %v\n", "X509_Subject_Country", get("X509_Subject_Country"))
				fmt.Printf("%s: %v\n", "X509_Subject_Locality", get("X509_Subject_Locality"))
				fmt.Printf("%s: %v\n", "X509_Subject_Organization", get("X509_Subject_Organization"))
				fmt.Printf("%s: %v\n", "X509_Subject_OrganizationalUnit", get("X509_Subject_OrganizationalUnit"))
				fmt.Printf("%s: %v\n", "X509_Subject_Province", get("X509_Subject_Province"))
				fmt.Printf("%s: %v\n", "X509_Subject_StreetAddress", get("X509_Subject_StreetAddress"))
				fmt.Printf("%s: %v\n", "X509_Subject_PostalCode", get("X509_Subject_PostalCode"))
				fmt.Printf("%s: %v\n", "X509_PublicKeyAlgorithm", get("X509_PublicKeyAlgorithm"))
				fmt.Printf("%s: %v\n", "X509_AuthorityKeyId", get("X509_AuthorityKeyId"))
				fmt.Printf("%s: %v\n", "X509_SubjectKeyId", get("X509_SubjectKeyId"))
				fmt.Printf("%s: %v\n", "X509_Fingerprint", get("X509_Fingerprint"))
				fmt.Printf("%s: %v\n", "X509_ExtKeyUsage", get("X509_ExtKeyUsage"))

				fmt.Println("")
			}
		}

		logrus.Info("common variables")
		params = wincrypt_client_certs.X509Parameters(x509.Certificate{})
		fmt.Printf("%s: %v\n", "Hostname", get("Hostname"))
		fmt.Printf("%s: %v\n", "User_Name", get("Hostname"))
		fmt.Printf("%s: %v\n", "User_Username", get("Hostname"))
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringSliceP("stores", "s", []string{"LocalMachine/My", "CurrentUser/My"}, "List of stores to search trough")
	rootCmd.Flags().StringP("query", "q", "true", "Query to filter certificates by")
}

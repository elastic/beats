package certificates

type Response struct {
	Status string `xml:"status,attr"`
	Result string `xml:"result"`
}

type Certificate struct {
	CertName          string
	Issuer            string
	IssuerSubjectHash string
	IssuerKeyHash     string
	DBType            string
	DBExpDate         string
	DBRevDate         string
	DBSerialNo        string
	DBFile            string
	DBName            string
	DBStatus          string
}

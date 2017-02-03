package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	lbeat "github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/dashboards/dashboards"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

var usage = fmt.Sprintf(`
Usage: ./import_dashboards [options]

Kibana dashboards are stored in a special index in Elasticsearch together with the searches, visualizations, and indexes that they use.

To import the official Kibana dashboards for your Beat version into a local Elasticsearch instance, use:

	./import_dashboards

To import the official Kibana dashboards for your Beat version into a remote Elasticsearch instance with Shield, use:

	./import_dashboards -es https://xyz.found.io -user user -pass password

For more details, check https://www.elastic.co/guide/en/beats/libbeat/5.0/import-dashboards.html.

`)

var beat string

type Options struct {
	KibanaIndex          string
	ES                   string
	Index                string
	Dir                  string
	File                 string
	Beat                 string
	URL                  string
	User                 string
	Pass                 string
	Certificate          string
	CertificateKey       string
	CertificateAuthority string
	Insecure             bool // Allow insecure SSL connections.
	OnlyDashboards       bool
	OnlyIndex            bool
	Snapshot             bool
	Quiet                bool
}

type CommandLine struct {
	flagSet *flag.FlagSet
	opt     Options
}

type Importer struct {
	cl     *CommandLine
	client *elasticsearch.Client
}

func DefineCommandLine() (*CommandLine, error) {
	var cl CommandLine

	cl.flagSet = flag.NewFlagSet("import", flag.ContinueOnError)

	cl.flagSet.Usage = func() {

		os.Stderr.WriteString(usage)
		cl.flagSet.PrintDefaults()
	}

	cl.flagSet.StringVar(&cl.opt.KibanaIndex, "k", ".kibana", "Kibana index")
	cl.flagSet.StringVar(&cl.opt.ES, "es", "http://127.0.0.1:9200", "Elasticsearch URL")
	cl.flagSet.StringVar(&cl.opt.User, "user", "", "Username to connect to Elasticsearch. By default no username is passed.")
	cl.flagSet.StringVar(&cl.opt.Pass, "pass", "", "Password to connect to Elasticsearch. By default no password is passed.")
	cl.flagSet.StringVar(&cl.opt.Index, "i", "", "The Elasticsearch index name. This overwrites the index name defined in the dashboards and index pattern. Example: metricbeat-*")
	cl.flagSet.StringVar(&cl.opt.Dir, "dir", "", "Directory containing the subdirectories: dashboard, visualization, search, index-pattern. Example: etc/kibana/")
	cl.flagSet.StringVar(&cl.opt.File, "file", "", "Zip archive file containing the Beats dashboards. The archive contains a directory for each Beat.")
	cl.flagSet.StringVar(&cl.opt.URL, "url",
		fmt.Sprintf("https://artifacts.elastic.co/downloads/beats/beats-dashboards/beats-dashboards-%s.zip", lbeat.GetDefaultVersion()),
		"URL to the zip archive containing the Beats dashboards")
	cl.flagSet.StringVar(&cl.opt.Beat, "beat", beat, "The Beat name that is used to select what dashboards to install from a zip. An empty string selects all.")
	cl.flagSet.BoolVar(&cl.opt.OnlyDashboards, "only-dashboards", false, "Import only dashboards together with visualizations and searches. By default import both, dashboards and the index-pattern.")
	cl.flagSet.BoolVar(&cl.opt.OnlyIndex, "only-index", false, "Import only the index-pattern. By default imports both, dashboards and the index pattern.")
	cl.flagSet.BoolVar(&cl.opt.Snapshot, "snapshot", false, "Import dashboards from snapshot builds.")
	cl.flagSet.StringVar(&cl.opt.CertificateAuthority, "cacert", "", "Certificate Authority for server verification")
	cl.flagSet.StringVar(&cl.opt.Certificate, "cert", "", "Certificate for SSL client authentication in PEM format.")
	cl.flagSet.StringVar(&cl.opt.CertificateKey, "key", "", "Client Certificate Key in PEM format.")
	cl.flagSet.BoolVar(&cl.opt.Insecure, "insecure", false, `Allows "insecure" SSL connections`)
	cl.flagSet.BoolVar(&cl.opt.Quiet, "quiet", false, "Suppresses all status messages. Error messages are still printed to stderr.")

	return &cl, nil
}

func (cl *CommandLine) ParseCommandLine() error {

	cl.opt.Beat = beat

	if err := cl.flagSet.Parse(os.Args[1:]); err != nil {
		return err
	}

	if cl.opt.URL == "" && cl.opt.File == "" && cl.opt.Dir == "" {
		return errors.New("Missing input. Please specify one of the options -file, -url or -dir")
	}

	if cl.opt.Certificate != "" && cl.opt.CertificateKey == "" {
		return errors.New("A certificate key needs to be passed as well by using the -key option.")
	}

	if cl.opt.CertificateKey != "" && cl.opt.Certificate == "" {
		return errors.New("A certificate needs to be passed as well by using the -cert option.")
	}

	return nil
}

func New() (*dashboards.Importer, error) {
	/* define the command line arguments */
	cl, err := DefineCommandLine()
	if err != nil {
		cl.flagSet.Usage()
		return nil, err
	}
	/* parse command line arguments */
	err = cl.ParseCommandLine()
	if err != nil {
		return nil, err
	}

	cfg := dashboards.DashboardsConfig{
		Enabled:        true,
		KibanaIndex:    cl.opt.KibanaIndex,
		Index:          cl.opt.Index,
		Dir:            cl.opt.Dir,
		File:           cl.opt.File,
		Beat:           cl.opt.Beat,
		URL:            cl.opt.URL,
		OnlyDashboards: cl.opt.OnlyDashboards,
		OnlyIndex:      cl.opt.OnlyIndex,
		Snapshot:       cl.opt.Snapshot,
		SnapshotURL:    fmt.Sprintf("https://beats-nightlies.s3.amazonaws.com/dashboards/beats-dashboards-%s-SNAPSHOT.zip", lbeat.GetDefaultVersion()),
	}

	/* prepare the Elasticsearch index pattern */
	fmtstr, err := fmtstr.CompileEvent(cl.opt.Index)
	if err != nil {
		return nil, fmt.Errorf("Failed to build the Elasticsearch index pattern: %s", err)
	}
	indexSel := outil.MakeSelector(outil.FmtSelectorExpr(fmtstr, ""))

	var tlsConfig outputs.TLSConfig
	var tls *transport.TLSConfig

	if cl.opt.Insecure {
		tlsConfig.VerificationMode = transport.VerifyNone
	}

	if len(cl.opt.Certificate) > 0 && len(cl.opt.CertificateKey) > 0 {
		tlsConfig.Certificate = outputs.CertificateConfig{
			Certificate: cl.opt.Certificate,
			Key:         cl.opt.CertificateKey,
		}
	}

	if len(cl.opt.CertificateAuthority) > 0 {
		tlsConfig.CAs = []string{cl.opt.CertificateAuthority}
	}

	tls, err = outputs.LoadTLSConfig(&tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to load the SSL certificate: %s", err)
	}

	/* connect to Elasticsearch */
	client, err := elasticsearch.NewClient(
		elasticsearch.ClientSettings{
			URL:      cl.opt.ES,
			Index:    indexSel,
			TLS:      tls,
			Username: cl.opt.User,
			Password: cl.opt.Pass,
			Timeout:  60 * time.Second,
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to Elasticsearch: %s", err)
	}

	statusMsg := dashboards.MessageOutputter(func(msg string, a ...interface{}) {
		if cl.opt.Quiet {
			return
		}

		if len(a) == 0 {
			fmt.Println(msg)
		} else {
			fmt.Println(fmt.Sprintf(msg, a...))
		}
	})

	return dashboards.NewImporter(&cfg, client, &statusMsg)
}

func main() {

	importer, err := New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Exiting")
		os.Exit(1)
	}
	err = importer.Import()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Exiting")
		os.Exit(1)
	}
}

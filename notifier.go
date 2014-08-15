package gobrake

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/golang/glog"
	"github.com/mreiferson/go-httpclient"
)

var client *http.Client

var GeoTrustGlobalCACert = `-----BEGIN CERTIFICATE-----
MIIDVDCCAjygAwIBAgIDAjRWMA0GCSqGSIb3DQEBBQUAMEIxCzAJBgNVBAYTAlVT
MRYwFAYDVQQKEw1HZW9UcnVzdCBJbmMuMRswGQYDVQQDExJHZW9UcnVzdCBHbG9i
YWwgQ0EwHhcNMDIwNTIxMDQwMDAwWhcNMjIwNTIxMDQwMDAwWjBCMQswCQYDVQQG
EwJVUzEWMBQGA1UEChMNR2VvVHJ1c3QgSW5jLjEbMBkGA1UEAxMSR2VvVHJ1c3Qg
R2xvYmFsIENBMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA2swYYzD9
9BcjGlZ+W988bDjkcbd4kdS8odhM+KhDtgPpTSEHCIjaWC9mOSm9BXiLnTjoBbdq
fnGk5sRgprDvgOSJKA+eJdbtg/OtppHHmMlCGDUUna2YRpIuT8rxh0PBFpVXLVDv
iS2Aelet8u5fa9IAjbkU+BQVNdnARqN7csiRv8lVK83Qlz6cJmTM386DGXHKTubU
1XupGc1V3sjs0l44U+VcT4wt/lAjNvxm5suOpDkZALeVAjmRCw7+OC7RHQWa9k0+
bw8HHa8sHo9gOeL6NlMTOdReJivbPagUvTLrGAMoUgRx5aszPeE4uwc2hGKceeoW
MPRfwCvocWvk+QIDAQABo1MwUTAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTA
ephojYn7qwVkDBF9qn1luMrMTjAfBgNVHSMEGDAWgBTAephojYn7qwVkDBF9qn1l
uMrMTjANBgkqhkiG9w0BAQUFAAOCAQEANeMpauUvXVSOKVCUn5kaFOSPeCpilKIn
Z57QzxpeR+nBsqTP3UEaBU6bS+5Kb1VSsyShNwrrZHYqLizz/Tt1kL/6cdjHPTfS
tQWVYrmm3ok9Nns4d0iXrKYgjy6myQzCsplFAMfOEVEiIuCl6rYVSAlk6l5PdPcF
PseKUgzbFbS9bZvlxrFUaKnjaZC2mqUPuLk/IH2uSrW4nOQdtqvmlKXBx4Ot2/Un
hw4EbNX/3aBd7YdStysVAq45pmp06drE57xNNB6pXE0zX5IJL4hmXXeXxx12E6nV
5fEWCRE11azbJHFwLJhWC9kXtNHjUStedejV0NxPNO3CBWaAocvmMw==
-----END CERTIFICATE-----`

func init() {
	chain := BuildRootCertificate()
	config := tls.Config{}
	config.RootCAs = x509.NewCertPool()
	for _, cert := range chain.Certificate {
		xc, err := x509.ParseCertificate(cert)
		if err != nil {
			panic(err)
		}
		config.RootCAs.AddCert(xc)
	}
	config.BuildNameToCertificate()

	transport := &httpclient.Transport{
		ConnectTimeout:        1 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		RequestTimeout:        10 * time.Second,
		TLSClientConfig:       &config,
	}
	client = &http.Client{Transport: transport}
}

type Notifier struct {
	Client      *http.Client
	StackFilter func(string, int, string, string) bool

	createNoticeURL string
	context         map[string]string
}

func NewNotifier(projectId int64, key string) *Notifier {
	n := &Notifier{
		Client:      client,
		StackFilter: stackFilter,

		createNoticeURL: getCreateNoticeURL(projectId, key),
		context:         make(map[string]string),
	}
	n.context["language"] = runtime.Version()
	n.context["os"] = runtime.GOOS
	n.context["architecture"] = runtime.GOARCH
	if hostname, err := os.Hostname(); err == nil {
		n.context["hostname"] = hostname
	}
	if wd, err := os.Getwd(); err == nil {
		n.context["rootDirectory"] = wd
	}
	return n
}

func (n *Notifier) SetContext(name, value string) {
	n.context[name] = value
}

func (n *Notifier) Notify(e interface{}, req *http.Request) error {
	notice := n.Notice(e, req, 3)
	if err := n.SendNotice(notice); err != nil {
		glog.Errorf("gobrake failed (%s) reporting error: %v", err, e)
		return err
	}
	return nil
}

func (n *Notifier) Notice(e interface{}, req *http.Request, startFrame int) *Notice {
	stack := stack(startFrame, n.StackFilter)
	notice := NewNotice(e, stack, req)
	for k, v := range n.context {
		notice.Context[k] = v
	}
	return notice
}

func (n *Notifier) SendNotice(notice *Notice) error {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(notice); err != nil {
		return err
	}

	resp, err := n.Client.Post(n.createNoticeURL, "application/json", buf)
	if err != nil {
		return err
	}

	// Read response so underlying connection can be reused.
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf(
			"gobrake: got %d response, wanted 201", resp.StatusCode)
	}

	return nil
}

func BuildRootCertificate() (cert tls.Certificate) {
	certPEMBlock := []byte(GeoTrustGlobalCACert)
	var certDERBlock *pem.Block
	for {
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		}
	}
	return
}

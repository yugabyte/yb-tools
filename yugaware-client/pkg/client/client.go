package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/juju/persistent-cookiejar"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	swaggerclient "github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/session_management"
	"golang.org/x/net/publicsuffix"
)

type YugawareClient struct {
	session   *http.Client
	cookiejar *cookiejar.Jar

	// One of these two values needs to be set when running API commands
	apiToken  string // The API key from the GUI
	authToken string // The API key from logging in - stored in a cookie

	csrfToken string // Required when using login()

	// Either set via API call (when using the API key) or from login()
	customerUUID strfmt.UUID
	userUUID     strfmt.UUID

	Ctx            context.Context
	hostname       string
	port           string
	baseURI        *url.URL
	timeoutSeconds time.Duration
	transport      *http.Transport

	tlsOptions *TLSOptions

	err error

	Log logr.Logger

	PlatformAPIs *swaggerclient.YugabytePlatformAPIs
	SwaggerAuth  runtime.ClientAuthInfoWriter
}

type TLSOptions struct {
	SkipHostVerification bool
	CaCertPath           string
	CertPath             string
	KeyPath              string
}

func (c *YugawareClient) Session() *http.Client {
	return c.session
}

func (c *YugawareClient) CustomerUUID() strfmt.UUID {
	return strfmt.UUID(c.customerUUID)
}

func New(ctx context.Context, log logr.Logger, hostname string) *YugawareClient {
	c := &YugawareClient{
		Ctx:            ctx,
		timeoutSeconds: 30,
		Log:            log.WithName("YugawareClient"),
	}

	c.hostname, c.port, c.err = splitHostPort(hostname)

	return c
}

func (c *YugawareClient) TimeoutSeconds(timeout int) *YugawareClient {
	c.timeoutSeconds = time.Duration(timeout) * time.Second

	return c
}

func (c *YugawareClient) TLSOptions(opts *TLSOptions) *YugawareClient {
	c.tlsOptions = opts

	return c
}

func (c *YugawareClient) APIToken(token string) *YugawareClient {
	c.apiToken = token

	return c
}

func (c *YugawareClient) Connect() (*YugawareClient, error) {

	// Check for previous errors
	if c.err != nil {
		return nil, c.err
	}

	var err error
	prefix := "http://"
	if c.hasTLS() {
		prefix = "https://"
	}

	if c.port == "" {
		c.port = "80"
		if c.hasTLS() {
			c.port = "443"
		}
	}

	c.transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   c.timeoutSeconds,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   c.timeoutSeconds,
		ExpectContinueTimeout: 1 * time.Second,
	}

	c.baseURI, err = url.Parse(prefix + c.hostname + ":" + c.port)
	if err != nil {
		return nil, fmt.Errorf("unable to parse uri: %w", err)
	}

	c.transport.TLSClientConfig, err = c.getTLSConfig()
	if err != nil {
		return nil, err
	}

	c.Log = c.Log.WithValues("host", c.baseURI.String())

	err = c.openCookieJar()
	if err != nil {
		return nil, err
	}

	c.session = &http.Client{
		Transport: c.transport,
		Jar:       c.cookiejar,
		Timeout:   c.timeoutSeconds,
	}

	c.Log.V(1).Info("connecting to host")

	// The csrfCookie is given to the client when connecting to this address. This cookie
	// is required to access API endpoints that are intended for the Web GUI only, such as
	// the Provider configuration interfaces.
	_, err = c.newRequest().Path("/api/v1/platform_config").Get().Do()

	if err != nil {
		return nil, fmt.Errorf("unable to connect: %w", err)
	}

	cookies := c.session.Jar.Cookies(c.baseURI)
	c.Log.V(1).Info("opened cookie jar", "cookies", cookies)
	// Get persisted credentials
	for _, cookie := range cookies {
		if cookie.Name == "csrfCookie" {
			c.csrfToken = cookie.Value
		} else if cookie.Name == "authToken" {
			c.authToken = cookie.Value
		}
	}

	// Don't use cookies beyond this point when using the API token
	if c.apiToken != "" {
		c.session.Jar = nil
	}

	if c.csrfToken == "" {
		c.err = fmt.Errorf("could not obtain csrfToken")
	}

	c.setupSwaggerClient()

	err = c.obtainSessionInfo()

	return c, err
}

func (c *YugawareClient) setupSwaggerClient() {
	var schemes []string

	if c.hasTLS() {
		schemes = []string{"https"}
	} else {
		schemes = []string{"http"}
	}

	rt := client.NewWithClient(c.hostname+":"+c.port, "/", schemes, c.session)

	c.PlatformAPIs = swaggerclient.New(rt, nil)

	// TODO: this needs to change in order to support the API token- if set, then use it instead of the cookies
	// We are relying on the cookies obtained from Login() rather than the API key
	c.SwaggerAuth = runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		if c.authToken == "" &&
			c.apiToken == "" {
			return errors.Errorf("not logged in")
		}

		if c.authToken != "" {
			err := r.SetHeaderParam("X-AUTH-TOKEN", c.authToken)
			if err != nil {
				return err
			}
		}

		if c.apiToken != "" {
			err := r.SetHeaderParam("X-AUTH-YW-API-TOKEN", c.apiToken)
			if err != nil {
				return err
			}
		}
		if c.csrfToken != "" {
			err := r.SetHeaderParam("Csrf-Token", c.csrfToken)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *YugawareClient) obtainSessionInfo() error {
	if c.apiToken != "" {
		params := session_management.NewGetSessionInfoParams()

		sessionInfo, err := c.PlatformAPIs.SessionManagement.GetSessionInfo(params, c.SwaggerAuth)
		if err != nil {
			return err
		}
		c.Log.V(1).Info("obtained session info from server", "session_info", sessionInfo.GetPayload())

		c.customerUUID = sessionInfo.GetPayload().CustomerUUID
		c.userUUID = sessionInfo.GetPayload().UserUUID
	} else {
		cookies := c.session.Jar.Cookies(c.baseURI)
		// Get persisted credentials
		for _, cookie := range cookies {
			if cookie.Name == "customerId" {
				c.customerUUID = strfmt.UUID(cookie.Value)
			} else if cookie.Name == "userId" {
				c.userUUID = strfmt.UUID(cookie.Value)
			}
		}
		c.Log.V(1).Info("using persisted session info", "CustomerUUID", c.customerUUID, "UserUUID", c.userUUID)
	}

	return nil
}

func (c *YugawareClient) openCookieJar() error {
	home, err := homedir.Dir()
	if err != nil {
		c.Log.Error(err, "unable to find $HOME directory")
		return err
	}

	c.cookiejar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
		Filename:         path.Join(home, ".yugaware-client-cookiejar"),
		NoPersist:        c.apiToken != "", // Don't persist cookies when using the API token
		Filter: cookiejar.CookieFilterFunc(func(cookie *http.Cookie) bool {
			if cookie.Name == "authToken" ||
				cookie.Name == "customerId" ||
				cookie.Name == "userId" {
				return true
			}
			return false
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create cookie jar for HTTP client: %w", err)
	}

	return nil
}

func (c *YugawareClient) expireSession() {
	cookieNames := []string{"authToken", "customerId", "userId"}
	for _, name := range cookieNames {
		cookie := &http.Cookie{
			Name:   name,
			Path:   "/",
			Domain: c.baseURI.Hostname(),
		}

		c.Log.V(1).Info("removing cookie", "cookie", cookie)
		c.cookiejar.RemoveCookie(cookie)
	}
	c.authToken = ""
	c.userUUID = ""
	c.customerUUID = ""
}

func (c *YugawareClient) persistCookies() error {
	cookies := c.cookiejar.Cookies(c.baseURI)

	c.Log.V(1).Info("persisting cookie jar", "cookies", cookies)
	return c.cookiejar.Save()
}

func splitHostPort(hostname string) (string, string, error) {
	var p string
	var err error

	h, p, err := net.SplitHostPort(hostname)
	if err != nil {
		switch x := err.(type) {
		case *net.AddrError:
			// The hostname did not contain a port
			if x.Err == "missing port in address" {
				// We will use the standard http ports if one is not specified, so don't set it here
				return hostname, "", nil
			}
		}
		return "", "", err
	}
	return h, p, nil
}

func (c *YugawareClient) newRequest() *Request {
	return NewRequest(c.Ctx, c.Log, c.session, c.baseURI, c.authToken, c.csrfToken)
}

func (c *YugawareClient) hasTLS() bool {
	if c.tlsOptions == nil {
		return false
	}

	if c.tlsOptions.SkipHostVerification ||
		c.tlsOptions.CaCertPath != "" ||
		c.tlsOptions.CertPath != "" ||
		c.tlsOptions.KeyPath != "" {
		return true
	}

	return false
}

func (c *YugawareClient) getTLSConfig() (*tls.Config, error) {
	if !c.hasTLS() {
		return nil, nil
	}

	tlsConfig := &tls.Config{}

	if c.tlsOptions.SkipHostVerification {
		tlsConfig.InsecureSkipVerify = true
	} else {
		if c.tlsOptions.CaCertPath != "" {
			f, err := os.ReadFile(c.tlsOptions.CaCertPath)
			if err != nil {
				return nil, err
			}
			if tlsConfig.RootCAs == nil {
				tlsConfig.RootCAs = x509.NewCertPool()
			}
			if ok := tlsConfig.RootCAs.AppendCertsFromPEM(f); !ok {
				return nil, errors.Errorf("unable to add %s to the CA list", c.tlsOptions.CaCertPath)
			}
		}

		if c.tlsOptions.CertPath != "" || c.tlsOptions.KeyPath != "" {
			if c.tlsOptions.KeyPath == "" || c.tlsOptions.CertPath == "" {
				return nil, errors.New("client certificate and key must both be set")
			}
			tlsCert, err := os.ReadFile(c.tlsOptions.CertPath)
			if err != nil {
				return nil, fmt.Errorf("unable to read x509 certificate: %w", err)
			}

			tlsKey, err := os.ReadFile(c.tlsOptions.KeyPath)
			if err != nil {
				return nil, fmt.Errorf("unable to read client key: %w", err)
			}

			tlsCertificate, err := tls.X509KeyPair(tlsCert, tlsKey)
			if err != nil {
				return nil, fmt.Errorf("unable to read x509 key pair: %w", err)
			}

			tlsConfig.Certificates = append(tlsConfig.Certificates, tlsCertificate)
		}

	}
	return tlsConfig, nil
}

type Request struct {
	ctx       context.Context
	session   *http.Client
	baseURI   *url.URL
	authToken string
	csrfToken string

	uri *url.URL

	method       string
	body         interface{}
	jsonResponse interface{}
	err          error

	Log logr.Logger
}

func NewRequest(ctx context.Context, log logr.Logger, session *http.Client, baseURI *url.URL, authToken string, csrfToken string) *Request {
	return &Request{
		ctx:       ctx,
		session:   session,
		baseURI:   baseURI,
		authToken: authToken,
		csrfToken: csrfToken,
		Log:       log.WithName("Request"),
	}
}

func (r *Request) Get() *Request {
	r.method = "GET"
	r.Log = r.Log.WithValues("method", r.method)

	return r
}

func (r *Request) Post() *Request {
	r.method = "POST"
	r.Log = r.Log.WithValues("method", r.method)

	return r
}

func (r *Request) Path(path string) *Request {
	r.uri, r.err = url.Parse(path)
	if r.err == nil {
		r.Log = r.Log.WithValues("path", r.uri.String())
	}

	return r
}

func (r *Request) RequestBody(message interface{}) *Request {
	r.body = message
	r.Log = r.Log.WithValues("body", message)

	return r
}

func (r *Request) DecodeResponseInto(message interface{}) *Request {
	r.jsonResponse = message

	return r
}

func (r *Request) Do() (*YugawareResponse, error) {
	var err error

	if r.err != nil {
		return nil, r.err
	}

	requestURI := r.baseURI.ResolveReference(r.uri)

	var request *http.Request
	if r.body != nil {

		body, err := json.Marshal(r.body)
		if err != nil {
			return nil, err
		}
		bodyBuf := bytes.NewBuffer(body)
		request, err = http.NewRequestWithContext(r.ctx, r.method, requestURI.String(), bodyBuf)
		if err != nil {
			return nil, err
		}

		request.Header.Set("Content-Type", "application/json")
	} else {
		request, err = http.NewRequestWithContext(r.ctx, r.method, requestURI.String(), nil)
		if err != nil {
			return nil, err
		}
	}

	if r.jsonResponse != nil {
		request.Header.Set("Accept", "application/json")
	}

	if r.authToken != "" {
		request.Header.Set("X-AUTH-TOKEN", r.authToken)
	}

	if r.csrfToken != "" {
		request.Header.Set("Csrf-Token", r.csrfToken)
	}

	yr := &YugawareResponse{}

	r.Log.V(1).Info("sending request", "headers", request.Header)
	yr.Response, err = r.session.Do(request)
	if err != nil {
		ue, ok := err.(*url.Error)
		if !ok {
			return nil, fmt.Errorf("unexpected error: %w", err)
		}
		if ue != nil {
			return nil, ue
		}
	}
	defer yr.Response.Body.Close()

	yr.Body, err = io.ReadAll(yr.Response.Body)
	if err != nil {
		return nil, err
	}

	if yr.Response.StatusCode == 200 {
		if r.jsonResponse != nil {
			err := json.Unmarshal(yr.Body, r.jsonResponse)
			if err != nil {
				return yr, err
			}
			r.Log.V(1).Info("recieved response", "response", r.jsonResponse, "headers", yr.Response.Header)
			if err != nil {
				return yr, err
			}
		} else {
			r.Log.V(1).Info("recieved response", "response", string(yr.Body), "headers", yr.Response.Header)
		}
	} else {
		r.Log.V(1).Info("unexpected response", "response", string(yr.Body), "headers", yr.Response.Header, "code", yr.Response.StatusCode)
		err = fmt.Errorf("unexpected response code: %d %s", yr.Response.StatusCode, yr.Body)
	}

	return yr, err
}

type YugawareResponse struct {
	Response *http.Response
	Body     []byte
}

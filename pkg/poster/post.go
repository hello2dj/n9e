package poster

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/toolkits/pkg/logger"
	"golang.org/x/net/http/httpproxy"
)

func PostJSON(url string, timeout time.Duration, v interface{}, retries ...int) (response []byte, code int, err error) {
	var bs []byte

	bs, err = json.Marshal(v)
	if err != nil {
		return
	}

	bf := bytes.NewBuffer(bs)

	client := http.Client{
		Transport: buildTransport(),
		Timeout:   timeout,
	}

	req, err := http.NewRequest("POST", url, bf)
	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response

	if len(retries) > 0 {
		for i := 0; i < retries[0]; i++ {
			resp, err = client.Do(req)
			if err == nil {
				break
			}

			tryagain := ""
			if i+1 < retries[0] {
				tryagain = " try again"
			}

			logger.Warningf("failed to curl %s error: %s"+tryagain, url, err)

			if i+1 < retries[0] {
				time.Sleep(time.Millisecond * 200)
			}
		}
	} else {
		resp, err = client.Do(req)
	}

	if err != nil {
		return
	}

	code = resp.StatusCode

	if resp.Body != nil {
		defer resp.Body.Close()
		response, err = ioutil.ReadAll(resp.Body)
	}

	return
}

func buildTransport() http.RoundTripper {
	return &http.Transport{
		Proxy: proxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

}

func proxyFromEnvironment(req *http.Request) (*url.URL, error) {
	return envProxyFunc()(req.URL)
}

var (
	envProxyOnce      sync.Once
	envProxyFuncValue func(*url.URL) (*url.URL, error)
)

func envProxyFunc() func(*url.URL) (*url.URL, error) {
	logger.Debug("使用代理", os.Getenv("QY_ALERT_HTTP_PROXY"))
	envProxyOnce.Do(func() {
		envProxyFuncValue = (&httpproxy.Config{
			HTTPProxy:  os.Getenv("QY_ALERT_HTTP_PROXY"),
			HTTPSProxy: os.Getenv("QY_ALERT_HTTP_PROXY"),
		}).ProxyFunc()
	})
	return envProxyFuncValue
}

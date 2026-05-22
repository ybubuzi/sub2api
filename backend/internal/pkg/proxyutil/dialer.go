// Package proxyutil 提供统一的代理配置功能
//
// 支持的代理协议：
//   - HTTP/HTTPS: 通过 Transport.Proxy 设置
//   - SOCKS5: 通过 Transport.DialContext 设置（客户端本地解析 DNS）
//   - SOCKS5H: 通过 Transport.DialContext 设置（代理端远程解析 DNS，推荐）
//
// 注意：proxyurl.Parse() 会自动将 socks5:// 升级为 socks5h://，
// 确保 DNS 也由代理端解析，防止 DNS 泄漏。
package proxyutil

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// ConfigureTransportProxy 根据代理 URL 配置 Transport
//
// 支持的协议：
//   - http/https: 设置 transport.Proxy
//   - socks5: 设置 transport.DialContext（客户端本地解析 DNS）
//   - socks5h: 设置 transport.DialContext（代理端远程解析 DNS，推荐）
//
// 参数：
//   - transport: 需要配置的 http.Transport
//   - proxyURL: 代理地址，nil 表示直连
//
// 返回：
//   - error: 代理配置错误（协议不支持或 dialer 创建失败）
func ConfigureTransportProxy(transport *http.Transport, proxyURL *url.URL) error {
	if proxyURL == nil {
		return nil
	}

	scheme := strings.ToLower(proxyURL.Scheme)
	switch scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)
		return nil

	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return fmt.Errorf("create socks5 dialer: %w", err)
		}
		// 优先使用支持 context 的 DialContext，以支持请求取消和超时
		if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = contextDialer.DialContext
		} else {
			// 回退路径：如果 dialer 不支持 ContextDialer，则包装为简单的 DialContext
			// 注意：此回退不支持请求取消和超时控制
			transport.DialContext = func(_ context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		}
		return nil

	case "chain":
		chainProxyURLs, err := chainProxyURLs(proxyURL)
		if err != nil {
			return err
		}
		transport.DialContext = chainDialContext(chainProxyURLs)
		transport.Proxy = nil
		return nil

	default:
		return fmt.Errorf("unsupported proxy scheme: %s", scheme)
	}
}

func chainProxyURLs(proxyURL *url.URL) ([]*url.URL, error) {
	upstreamRaw := strings.TrimSpace(proxyURL.Query().Get("upstream"))
	proxyRaw := strings.TrimSpace(proxyURL.Query().Get("proxy"))
	if upstreamRaw == "" || proxyRaw == "" {
		return nil, fmt.Errorf("chain proxy requires upstream and proxy")
	}
	upstreamURL, err := url.Parse(upstreamRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid chain upstream proxy")
	}
	nextProxyURL, err := url.Parse(proxyRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid chain next proxy")
	}
	if strings.EqualFold(upstreamURL.Scheme, "chain") {
		urls, err := chainProxyURLs(upstreamURL)
		if err != nil {
			return nil, err
		}
		return append(urls, nextProxyURL), nil
	}
	return []*url.URL{upstreamURL, nextProxyURL}, nil
}

func chainDialContext(proxyURLs []*url.URL) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, fmt.Errorf("unsupported chain proxy network: %s", network)
		}

		if len(proxyURLs) == 0 {
			return nil, fmt.Errorf("empty chain proxy")
		}
		conn, err := dialProxy(ctx, proxyURLs[0])
		if err != nil {
			return nil, err
		}
		for i, current := range proxyURLs {
			target := addr
			if i+1 < len(proxyURLs) {
				target = proxyHostPort(proxyURLs[i+1])
			}
			if err := connectHTTPProxy(ctx, conn, target, current.User); err != nil {
				_ = conn.Close()
				return nil, fmt.Errorf("connect chain hop %d: %w", i+1, err)
			}
		}
		return conn, nil
	}
}

func proxyHostPort(proxyURL *url.URL) string {
	if proxyURL == nil {
		return ""
	}
	if strings.Contains(proxyURL.Host, ":") {
		return proxyURL.Host
	}
	port := "80"
	if strings.EqualFold(proxyURL.Scheme, "https") {
		port = "443"
	}
	return net.JoinHostPort(proxyURL.Hostname(), port)
}

func dialProxy(ctx context.Context, proxyURL *url.URL) (net.Conn, error) {
	if proxyURL == nil {
		return nil, fmt.Errorf("proxy URL is nil")
	}
	host := proxyHostPort(proxyURL)
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return dialer.DialContext(ctx, "tcp", host)
}

func connectHTTPProxy(ctx context.Context, conn net.Conn, target string, user *url.Userinfo) error {
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		deadline = time.Now().Add(10 * time.Second)
	}
	_ = conn.SetDeadline(deadline)
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	var b strings.Builder
	if _, err := fmt.Fprintf(&b, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nConnection: keep-alive\r\n", target, target); err != nil {
		return err
	}
	if user != nil {
		username := user.Username()
		password, _ := user.Password()
		token := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		if _, err := fmt.Fprintf(&b, "Proxy-Authorization: Basic %s\r\n", token); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(&b, "\r\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(conn, b.String()); err != nil {
		return err
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, &http.Request{Method: http.MethodConnect})
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	if reader.Buffered() > 0 {
		return &bufferedConnectUnsupportedError{n: reader.Buffered()}
	}
	return nil
}

type bufferedConnectUnsupportedError struct {
	n int
}

func (e *bufferedConnectUnsupportedError) Error() string {
	return "unexpected buffered proxy data after CONNECT: " + strconv.Itoa(e.n)
}

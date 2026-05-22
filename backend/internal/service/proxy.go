package service

import (
	"net"
	"net/url"
	"strconv"
	"time"
)

type Proxy struct {
	ID              int64
	Name            string
	Protocol        string
	Host            string
	Port            int
	Username        string
	Password        string
	Status          string
	UpstreamProxyID *int64
	UpstreamProxy   *Proxy
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (p *Proxy) IsActive() bool {
	return p.Status == StatusActive
}

func (p *Proxy) URL() string {
	current := p.directURL()
	if p.UpstreamProxy == nil {
		return current
	}
	u := &url.URL{
		Scheme: "chain",
		Host:   "proxy",
	}
	q := u.Query()
	q.Set("upstream", p.UpstreamProxy.URL())
	q.Set("proxy", current)
	u.RawQuery = q.Encode()
	return u.String()
}

func (p *Proxy) directURL() string {
	u := &url.URL{
		Scheme: p.Protocol,
		Host:   net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
	}
	if p.Username != "" && p.Password != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	return u.String()
}

type ProxyWithAccountCount struct {
	Proxy
	AccountCount   int64
	LatencyMs      *int64
	LatencyStatus  string
	LatencyMessage string
	IPAddress      string
	Country        string
	CountryCode    string
	Region         string
	City           string
	QualityStatus  string
	QualityScore   *int
	QualityGrade   string
	QualitySummary string
	QualityChecked *int64
}

type ProxyAccountSummary struct {
	ID       int64
	Name     string
	Platform string
	Type     string
	Notes    *string
}

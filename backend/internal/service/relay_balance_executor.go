package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	relayBalanceDefaultPackageJSON = `{"type":"module"}`
	relayBalanceMaxOutputBytes     = 64 * 1024
	relayBalanceExecTimeout        = 30 * time.Second
)

type RelayBalanceExecutor interface {
	Execute(ctx context.Context, station *RelayBalanceStation) RelayBalanceRun
}

type NodeRelayBalanceExecutor struct{}

func NewNodeRelayBalanceExecutor() RelayBalanceExecutor {
	return &NodeRelayBalanceExecutor{}
}

func (e *NodeRelayBalanceExecutor) Execute(ctx context.Context, station *RelayBalanceStation) RelayBalanceRun {
	started := time.Now()
	run := RelayBalanceRun{StationID: station.ID, StationName: station.Name, Status: "failed", StartedAt: started}
	defer func() {
		finished := time.Now()
		run.FinishedAt = &finished
		run.DurationMs = int(finished.Sub(started).Milliseconds())
	}()

	workDir, err := os.MkdirTemp("", "sub2api-relay-balance-*")
	if err != nil {
		run.Error = err.Error()
		return run
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	if err := os.WriteFile(filepath.Join(workDir, "package.json"), []byte(normalizePackageJSON(station.PackageJSON)), 0600); err != nil {
		run.Error = err.Error()
		return run
	}
	if hasDependencies(station.PackageJSON) {
		if stdout, stderr, err := runLimitedCommand(ctx, workDir, 60*time.Second, "npm", "install", "--omit=dev", "--no-audit", "--no-fund"); err != nil {
			run.Stdout = stdout
			run.Stderr = stderr
			run.Error = fmt.Sprintf("npm install failed: %v", err)
			return run
		}
	}
	if err := os.WriteFile(filepath.Join(workDir, "station-script.mjs"), []byte(station.Script), 0600); err != nil {
		run.Error = err.Error()
		return run
	}

	wrapper := buildRelayBalanceWrapper(station)
	if err := os.WriteFile(filepath.Join(workDir, "runner.mjs"), []byte(wrapper), 0600); err != nil {
		run.Error = err.Error()
		return run
	}

	stdout, stderr, err := runLimitedCommand(ctx, workDir, relayBalanceExecTimeout, "node", "runner.mjs")
	run.Stdout = stdout
	run.Stderr = stderr
	if err != nil {
		run.Error = err.Error()
		return run
	}
	parsed, err := parseRelayBalanceOutput(stdout)
	if err != nil {
		run.Error = err.Error()
		return run
	}
	run.Balance = &parsed.Balance
	run.Currency = strings.TrimSpace(parsed.Currency)
	run.Raw = string(parsed.Raw)
	run.Status = "success"
	return run
}

type relayBalanceScriptOutput struct {
	Balance  float64         `json:"balance"`
	Currency string          `json:"currency"`
	Raw      json.RawMessage `json:"raw"`
}

func normalizePackageJSON(pkg string) string {
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return relayBalanceDefaultPackageJSON
	}
	return pkg
}

func hasDependencies(pkg string) bool {
	var obj struct {
		Dependencies map[string]any `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(normalizePackageJSON(pkg)), &obj); err != nil {
		return false
	}
	return len(obj.Dependencies) > 0
}

func buildRelayBalanceWrapper(station *RelayBalanceStation) string {
	ctxJSON, _ := json.Marshal(map[string]string{"stationName": station.Name, "baseUrl": station.BaseURL})
	return fmt.Sprintf(`
import https from 'https';
import http from 'http';
import tls from 'tls';

function parseProxy(url) {
  if (!url) return null;
  try {
    const u = new URL(url);
    return { host: u.hostname, port: parseInt(u.port, 10) || 7890 };
  } catch { return null; }
}

const proxyTarget = parseProxy(process.env.HTTPS_PROXY || process.env.https_proxy || process.env.HTTP_PROXY || process.env.http_proxy || '');

async function proxyFetch(url, opts = {}) {
  const parsed = new URL(url);
  const { hostname, pathname, search } = parsed;
  const method = (opts.method || 'GET').toUpperCase();
  const headers = opts.headers || {};
  const body = opts.body || null;
  const isHttps = parsed.protocol === 'https:';

  if (proxyTarget && isHttps) {
    return new Promise((resolve, reject) => {
      const tunnelReq = http.request({
        host: proxyTarget.host, port: proxyTarget.port,
        method: 'CONNECT', path: hostname + ':443',
        headers: { Host: hostname },
      });
      tunnelReq.on('connect', (res, socket) => {
        const tlsSocket = tls.connect({ socket, host: hostname, servername: hostname, rejectUnauthorized: true });
        const httpsReq = https.request({
          createConnection: () => tlsSocket, host: hostname,
          path: pathname + search, method, headers,
        }, (response) => {
          const chunks = [];
          response.on('data', c => chunks.push(c));
          response.on('end', () => {
            const text = Buffer.concat(chunks).toString();
            resolve({ ok: response.statusCode >= 200 && response.statusCode < 300, status: response.statusCode, statusText: response.statusMessage, headers: { get: (k) => response.headers[k.toLowerCase()] || null }, text: () => Promise.resolve(text), json: () => Promise.resolve(JSON.parse(text)) });
          });
        });
        httpsReq.on('error', reject);
        if (body) httpsReq.write(body);
        httpsReq.end();
      });
      tunnelReq.on('error', reject);
      tunnelReq.end();
    });
  }

  const mod = isHttps ? https : http;
  return new Promise((resolve, reject) => {
    const req = mod.request(
      isHttps ? { hostname, path: pathname + search, method, headers, rejectUnauthorized: true } : { hostname, path: pathname + search, method, headers },
      (res) => {
        const chunks = [];
        res.on('data', c => chunks.push(c));
        res.on('end', () => {
          const text = Buffer.concat(chunks).toString();
          resolve({ ok: res.statusCode >= 200 && res.statusCode < 300, status: res.statusCode, statusText: res.statusMessage, headers: { get: (k) => res.headers[k.toLowerCase()] || null }, text: () => Promise.resolve(text), json: () => Promise.resolve(JSON.parse(text)) });
        });
      }
    );
    req.on('error', reject);
    if (body) req.write(body);
    req.end();
  });
}

const ctx = %s;
globalThis.fetch = proxyFetch;
const mod = await import('./station-script.mjs');
const fn = mod.default || mod.run;
if (typeof fn !== 'function') throw new Error('script must export default async function run(ctx) or export function run(ctx)');
const result = await fn(ctx);
if (!result || typeof result !== 'object') throw new Error('script must return an object');
if (typeof result.balance !== 'number' || !Number.isFinite(result.balance)) throw new Error('result.balance must be a finite number');
console.log(JSON.stringify({ balance: result.balance, currency: result.currency || '', raw: result.raw ?? null }));
`, string(ctxJSON))
}

func parseRelayBalanceOutput(stdout string) (*relayBalanceScriptOutput, error) {
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var out relayBalanceScriptOutput
		if err := json.Unmarshal([]byte(line), &out); err != nil {
			return nil, fmt.Errorf("last stdout line is not valid JSON: %w", err)
		}
		if !json.Valid(out.Raw) && len(out.Raw) > 0 {
			return nil, errors.New("raw must be valid JSON")
		}
		out.Currency = strings.TrimSpace(out.Currency)
		return &out, nil
	}
	return nil, errors.New("script produced no JSON output")
}

func runLimitedCommand(parent context.Context, dir string, timeout time.Duration, name string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	env := []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "npm_config_cache=" + filepath.Join(dir, ".npm")}
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "all_proxy", "no_proxy"} {
		if v := os.Getenv(key); v != "" {
			env = append(env, key+"="+v)
		}
	}
	cmd.Env = env
	var stdout, stderr limitedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return stdout.String(), stderr.String(), fmt.Errorf("command timed out after %s", timeout)
	}
	return stdout.String(), stderr.String(), err
}

type limitedBuffer struct{ b bytes.Buffer }

func (w *limitedBuffer) Write(p []byte) (int, error) {
	remaining := relayBalanceMaxOutputBytes - w.b.Len()
	if remaining > 0 {
		if len(p) > remaining {
			_, _ = w.b.Write(p[:remaining])
		} else {
			_, _ = w.b.Write(p)
		}
	}
	return len(p), nil
}

func (w *limitedBuffer) String() string { return w.b.String() }

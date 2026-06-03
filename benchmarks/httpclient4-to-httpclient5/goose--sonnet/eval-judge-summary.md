## Eval Judge Report — goose / sonnet / httpclient4-to-httpclient5

- **14 of 24 rules passed** (report.yaml lists 28 including ruleset + dependency rules)
- **7 precision issues** (all `warn`): unqualified METHOD_CALL rules — `closeExpiredConnections`, `closeIdleConnections`, `addInterceptorLast`, `setRetryHandler`, `getAllHeaders` (high false-positive risk), `setConnectTimeout` (high), `setSocketTimeout` (high)
- **3 coherence issues**: **00260** fires on `CloseableHttpClient` import but gives async-only guidance — wrong for classic migration. **00270** same for `HttpClients`. **00250** same for `PoolingHttpClientConnectionManager` — classic users should use `PoolingHttpClientConnectionManagerBuilder`, not async replacement
- **3 cross-rule issues**: `00010`+`00260`+`00270` contradictory guidance (package rule says re-import, async rules say switch to async), `00010` overlaps with all IMPORT rules, `00280`+`00260` both push async pattern
- **8 gaps**: `HttpResponse` interface split (high), `ResponseHandler` pattern (high), `CookieSpecs` replacement, `TimeUnit`→`Timeout`/`TimeValue` migration, `HttpGet/Put/Delete/Patch` constructors (only `HttpPost` covered), `EntityUtils` changes, `BasicNameValuePair` package rename, async streaming entity consumers

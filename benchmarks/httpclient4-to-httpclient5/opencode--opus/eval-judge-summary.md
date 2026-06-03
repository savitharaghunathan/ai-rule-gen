## Eval Judge Report — opencode / opus / httpclient4-to-httpclient5

- **20 of 28 rules passed** (report.yaml lists 29 including ruleset)
- **8 precision issues** (all `warn`): unqualified METHOD_CALL rules — `setConnectTimeout`, `setSocketTimeout`, `setConnectionTimeToLive`, `addInterceptorLast`, `setRetryHandler`, `setSSLSocketFactory`, `closeExpiredConnections`, `closeIdleConnections`
- **3 coherence issues**: **00250** fires on classic `PoolingHttpClientConnectionManager` import but gives async-only guidance (should mention `PoolingHttpClientConnectionManagerBuilder` for classic path). **00260** fires on `CloseableHttpClient` import but says replace with `CloseableHttpAsyncClient` — wrong for classic users. **00270** fires on `HttpClients` import but says replace with `HttpAsyncClients` — same issue
- **2 cross-rule issues**: `00020`+`00280` overlap on SSL guidance (import + method call with near-identical messages), `00250`+`00260`+`00270` cluster assumes async migration path
- **8 gaps**: `CookieSpecs` replacement, `HttpGet/Put/Delete/Patch` constructor migration (only `HttpPost` covered), `ResponseHandler` pattern, `BasicCookieStore` package change, `BasicCredentialsProvider` package change, classic `PoolingHttpClientConnectionManagerBuilder`, async `SimpleHttpRequest`/`SimpleRequestBuilder`, `IOReactorConfig`

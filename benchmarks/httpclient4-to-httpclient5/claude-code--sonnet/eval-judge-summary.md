## Eval Judge Report — claude-code / sonnet / httpclient4-to-httpclient5

- **31 of 43 rules passed**
- **4 precision issues** (all `warn`): unqualified METHOD_CALL rules (00050, 00060, 00070, 00080) that also match 5.x replacement APIs — duplicates of more qualified rules (00200, 00210, 00180)
- **4 coherence issues** (1 `fail`, 3 `warn`): most serious is **00350** — fires on classic `CloseableHttpClient` import but tells all users to switch to async client, which is wrong for the majority following the classic migration path
- **3 cross-rule duplicates**: 00050+00200, 00060+00210, 00080+00180 — unqualified rules should be removed in favor of their qualified counterparts
- **1 gap**: no rule for `HttpContext.getAttribute(HTTP_TARGET_HOST)` → `HttpClientContext.getHttpRoute().getTargetHost()`

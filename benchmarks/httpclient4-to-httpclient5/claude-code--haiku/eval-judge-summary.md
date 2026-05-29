## Eval Judge Report — claude-code / haiku / httpclient4-to-httpclient5

- **0 of 26 rules passed** — all fail due to empty messages and wrong condition type
- **3 precision issues** (all `fail`): 2 rules target the **new** 5.x API (`ClassicRequestBuilder`), 1 rule matches JDK `SSLContext` (not HttpClient-specific), all 26 use `builtin.filecontent` instead of `java.referenced`
- **1 systemic coherence failure**: every rule has `message: ': '` (empty) — developers get zero migration guidance
- **5 duplicate groups** (11 rules): `00010+00090`, `00040+00050`, `00060+00070`, `00140+00200`, `00100+00210+00230+00240+00260`
- **Massive coverage gap**: only ~14 unique old APIs detected out of 380 ground truth entries (3.7% coverage) — missing `CloseableHttpClient`, `HttpUriRequest`, `HttpPut/Delete/Patch`, async API, auth/cookie/connection classes

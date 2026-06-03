## Eval Judge Report — opencode / opus / spring-boot3-to-spring-boot4

- **78 of 91 rules passed** eval judge review
- **7 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `alwaysApplyingWhenNonNull` unqualified METHOD_CALL, CycloneDX version-blind, `org.springframework.boot.autoconfigure.graphql*` slightly broad, MongoDB property substring risk, Jackson 2 compat properties vague, `PathRequest` import too broad for fonts change
- **4 coherence issues**: **00450** (fail) detects wrong FQN — uses SB4 package for `HttpMessageConverters` instead of SB3. **00880** fires on any `PropertyMapper` import for a narrow null-handling change. **00010** umbrella rule with effort=7 for informational notice. **00870** `WebClient` import rule doesn't account for `spring-boot-starter-webflux` users
- **2 cross-rule issues**: `00230`+`00880` overlapping PropertyMapper guidance, deprecated starter rules use `optional` category instead of `potential`
- **8 gaps**: Flyway starter requirement (high), Liquibase starter requirement (high), `@AutoConfigureMockMvc` attribute changes, `HttpMessageConverter` customizer migration, logback charset, `@AutoConfigureWebTestClient`, `spring-boot-starter-classic` path, `spring.jackson.generator.*` properties

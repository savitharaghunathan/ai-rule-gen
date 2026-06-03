## Eval Judge Report — claude-code / opus / spring-boot3-to-spring-boot4

- **56 of 74 rules passed**
- **5 precision issues** (all `warn`): `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `webClientEnabled|webDriverEnabled` via filecontent too broad, MockMvc detection fires on users already using `@AutoConfigureMockMvc`, CycloneDX matches comments, `launchScript` filecontent minor risk
- **4 coherence issues**: most impactful is **00010** — conflates system requirements (Java 17, Jakarta EE 11), modular design, and classic starters into one rule with overloaded guidance. `00500` assumes WAR deployment for all `spring-boot-starter-tomcat` users. `00120` vague on test starter replacements. `00580` broad MongoDB detection for narrow UUID/BigDecimal issue
- **3 cross-rule issues**: `00440`+`00730` duplicate (both detect `/fonts/**` static resource change), `00290`+`00420` overlap on Spring Authorization Server, `00160`+`00010` implicit ordering conflict on starter renames
- **10 gaps**: `@AutoConfigureTestRestTemplate` requirement (high), actuator `@Nullable` migration (high), Jackson annotations exception, classic starters quick path, logback charset, Jackson auto-module-registration, SAML starter rename, `@AutoConfigureRestTestClient`, package organization changes, `jackson-2-defaults` compat property

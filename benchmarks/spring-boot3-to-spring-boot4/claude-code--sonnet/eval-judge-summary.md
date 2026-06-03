## Eval Judge Report — claude-code / sonnet / spring-boot3-to-spring-boot4

- **73 of 89 rules passed**
- **3 precision issues** (all `warn`): `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `org.springframework.lang*` too broad for nullability migration, CycloneDX rule fires regardless of version
- **7 coherence issues** (all `warn`): 5 rules with **inverted detection logic** (`00260`, `00420`, `00430`, `00450`, `00730`) — detect config properties that only exist in projects already managing the setting, will produce zero incidents on SB3 codebases. `00820` fires on all `@SpringBootTest` but gives MockMVC-specific advice. `00890` duplicates `00270` (AOP starter rename)
- **2 cross-rule overlaps**: `00270`+`00890` (AOP starter rename duplicate), `00100`+`00110`+`00120` (security test triple-fire)
- **8 gaps**: Optional dependencies in Maven, WebClient/TestRestTemplate with @SpringBootTest, SAML starter rename, BootstrapRegistryInitializer/ConfigurableBootstrapContext package moves, BigDecimal representation config, modular starter rules

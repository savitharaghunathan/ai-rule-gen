## Eval Judge Report — opencode / sonnet / spring-boot3-to-spring-boot4

- **73 of 95 rules passed**
- **8 precision issues** (all `warn`): `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `spring.datasource.` too broad, `org.springframework.graphql*` fires on all GraphQL imports, CycloneDX matches comments, `@SpringBootTest` fires on all tests not just WebClient/TestRestTemplate users, jackson find-and-add-modules inverted, logback broad, properties-migrator inverted
- **5 coherence issues**: **00530** (fail) detects wrong FQN — uses SB4 package instead of SB3 for `HttpMessageConverters`. **00700** detects wrong Spring AMQP class instead of Spring Boot autoconfigure class. **00730** fires on `@AutoConfigureMockMvc` users who are already correct. **00920/00930/00940/00950** are noise rules that fire on unchanged MongoDB properties saying "no change required"
- **3 cross-rule issues**: `00620`+`00870-00910` MongoDB property rename duplication (broad rule + 5 individual rules), `00290`+`00830` AOP starter overlap, `00010`+`00140` system requirements vs starter rename overlap
- **8 gaps**: Optional dependencies in Maven uber jars, MongoDB SSL properties, individual MongoDB connection properties, actuator `@Nullable` parameter context, Jackson generator properties, package organization class relocations, `BootstrapRegistryInitializer` move, MongoDB gridfs.database

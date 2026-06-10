# Ruleset comparison: ai-generated vs handcrafted

- **A**: ai-generated (15 rules) — `evals/javaee-to-quarkus/rules`
- **B**: handcrafted (82 rules) — `evals/javaee-to-quarkus-handcrafted/rules`

## Coverage matrix

How many rules on one side are matched by a rule keyed on the same API on the other side.

| Direction | Covered | Partial | Missing |
|---|---|---|---|
| A → B (ai-generated rules covered by handcrafted) | 3 | 3 | 9 |
| B → A (handcrafted rules covered by ai-generated) | 4 | 1 | 77 |

### Rules in ai-generated with no equivalent in handcrafted (9)

- `build-pattern-00010` — Quarkus 2 requires Java 11+; raise maven.compiler.source/target from 1.8 to 11
  - keys: fc:<maven\.compiler\.(source|target)>1\.8</maven\.compiler\.(source|target)>
- `build-xml-00010` — maven-war-plugin must be removed when migrating to Quarkus 2; Quarkus produces JAR artifacts
  - keys: xml://*[local-name()='plugin'][*[local-name()='groupId' and text()='org.apache.maven.plugins'] and *[local-name()='artifactId' and text()='maven-war-plugin']]
- `build-xml-00020` — Quarkus 2 projects must use jar packaging (not war)
  - keys: xml:/m:project/m:packaging[text()='war']
- `build-xml-00030` — Quarkus 2 requires Java 11 or higher; maven.compiler.source must be at least 11
  - keys: xml:/m:project/m:properties/m:maven.compiler.source[number(text()) &lt; 11]
- `build-xml-00040` — Quarkus 2 requires Java 11 or higher; maven.compiler.target must be at least 11
  - keys: xml:/m:project/m:properties/m:maven.compiler.target[number(text()) &lt; 11]
- `build-xml-00050` — WAR packaging is not used by Quarkus; switch to JAR (or omit, default is jar)
  - keys: xml:/m:project/m:packaging[text()='war']
- `dependencies-dependency-00010` — javax:javaee-api umbrella dependency is removed; replace with targeted Quarkus extensions
  - keys: dep:javax.javaee-api
- `dependencies-dependency-00020` — javax.annotation:javax.annotation-api removed; the annotation API is provided transitively by Quarkus extensions
  - keys: dep:javax.annotation.javax.annotation-api
- `dependencies-dependency-00030` — javax.enterprise:cdi-api is removed; CDI is provided by quarkus-arc
  - keys: dep:javax.enterprise.cdi-api

### Rules in handcrafted with no equivalent in ai-generated (77)

- `cdi-to-quarkus-00000` — Replace javax.enterprise:cdi-api dependency
  - keys: xml:/m:project/m:dependencies/m:dependency[m:artifactId/text() = 'cdi-api' and m:groupId/text() = 'javax.enterprise' and (count(../m:dependency/m:groupId[contains(., 'io.quarkus')]) = 0)]
- `cdi-to-quarkus-00020` — Replace javax.inject:javax.inject dependency
  - keys: xml:/m:project/m:dependencies/m:dependency[m:artifactId/text() = 'javax.inject' and m:groupId/text() = 'javax.inject' and (count(../m:dependency/m:groupId[contains(., 'io.quarkus')]) = 0)]
- `cdi-to-quarkus-00030` — `beans.xml` descriptor content is ignored
  - keys: xml:/b:beans
- `cdi-to-quarkus-00040` — @Produces annotation no longer required
  - keys: java:javax.enterprise.inject.produces
- `dependency-removal-for-quarkus-00000` — Remove non-quarkus dependencies
  - keys: dep:org.jboss.spec.javax.annotation.jboss-annotations-api_1.3_spec, dep:org.jboss.spec.javax.ejb.jboss-ejb-api_3.2_spec, dep:org.jboss.spec.javax.xml.bind.jboss-jaxb-api_2.3_spec
- `jakarta-cdi-to-quarkus-00000` — Replace jakarta.enterprise:jakarta.enterprise.cdi-api dependency
  - keys: xml:/m:project/m:dependencies/m:dependency[m:artifactId/text() = 'jakarta.enterprise.cdi-api' and m:groupId/text() = 'jakarta.enterprise' and (count(../m:dependency/m:groupId[contains(., 'io.quarkus')]) = 0)]
- `jakarta-cdi-to-quarkus-00020` — Replace jakarta.inject:jakarta.inject-api dependency
  - keys: xml:/m:project/m:dependencies/m:dependency[m:artifactId/text() = 'jakarta.inject-api' and m:groupId/text() = 'jakarta.inject' and (count(../m:dependency/m:groupId[contains(., 'io.quarkus')]) = 0)]
- `jakarta-cdi-to-quarkus-00030` — `beans.xml` descriptor content is ignored
  - keys: xml:/b:beans
- `jakarta-cdi-to-quarkus-00040` — @Produces annotation no longer required
  - keys: java:javax.enterprise.inject.produces
- `jakarta-faces-to-quarkus-00000` — Replace Jakarta Faces Dependency with MyFaces
  - keys: xml:/m:project/m:dependencies/m:dependency[m:groupId/text() = 'jakarta.faces']
- `jakarta-faces-to-quarkus-00010` — Replace Jakarta Faces Dependency with MyFaces
  - keys: fc:artifactId>jakarta.faces<
- `jakarta-jaxrs-to-quarkus-00010` — Replace jakarta JAX-RS dependency
  - keys: dep:jakarta.ws.rs.jakarta.ws.rs-api
- `jakarta-jaxrs-to-quarkus-00020` — Jakarta JAX-RS activation is no longer necessary
  - keys: java:javax.ws.rs.applicationpath, java:javax.ws.rs.core.application
- `javaee-faces-to-quarkus-00000` — Replace JSF Dependency with MyFaces
  - keys: xml:/m:project/m:dependencies/m:dependency[m:groupId/text() = 'org.jboss.spec.javax.faces']
- `javaee-pom-to-quarkus-00000` — The expected project artifact's extension is `jar`
  - keys: xml:/m:project/m:packaging/text()[matches(self::node(), '^(pom|maven-plugin|ejb|war|ear|rar)$')]
- `javaee-pom-to-quarkus-00010` — Adopt Quarkus BOM
  - keys: xml:/m:project[not(m:dependencyManagement/m:dependencies/m:dependency/m:artifactId/text() = 'quarkus-bom') and not(m:dependencyManagement/m:dependencies/m:dependency/m:artifactId/text() = '${quarkus.platform.artifact-id}')]
- `javaee-pom-to-quarkus-00020` — Adopt Quarkus Maven plugin
  - keys: xml:/m:project[not(m:build/m:plugins/m:plugin/m:artifactId/text() = 'quarkus-maven-plugin')]
- `javaee-pom-to-quarkus-00030` — Adopt Maven Compiler plugin
  - keys: xml:/m:project[not(m:build/m:plugins/m:plugin/m:artifactId/text() = 'maven-compiler-plugin') or m:build/m:plugins/m:plugin/m:artifactId[text() = 'maven-compiler-plugin' and not(../m:configuration/m:compilerArgs/m:arg/text() = '-parameters')]]
- `javaee-pom-to-quarkus-00040` — Adopt Maven Surefire plugin
  - keys: xml:/m:project[not(m:build/m:plugins/m:plugin/m:artifactId/text() = 'maven-surefire-plugin') or m:build/m:plugins/m:plugin/m:artifactId[text() = 'maven-surefire-plugin' and not(../m:configuration/m:systemPropertyVariables/m:java.util.logging.manager/text() = 'org.jboss.logmanager.LogManager')]]
- `javaee-pom-to-quarkus-00050` — Adopt Maven Failsafe plugin
  - keys: xml:/m:project[ not(m:build/m:plugins/m:plugin/m:artifactId/text() = 'maven-failsafe-plugin') or m:build/m:plugins/m:plugin[m:artifactId[text() = 'maven-failsafe-plugin'] and not(m:executions/m:execution/m:configuration/m:systemPropertyVariables/m:native.image.path) and not(m:configuration/m:systemPropertyVariables/m:native.image.path) ] ]
- `javaee-pom-to-quarkus-00060` — Add Maven profile to run the Quarkus native build
  - keys: xml:/m:project[not(m:profiles/m:profile/m:properties/m:quarkus.package.type/text() = 'native')]
- `javaee-pom-to-quarkus-00070` — Configure Quarkus hibernate-orm
  - keys: xml:/m:project/m:dependencies/m:dependency/m:groupId[contains(text(),'org.hibernate')], xml:/m:project/m:dependencies/m:dependency[m:artifactId/text() = 'jakarta.persistence-api']
- `javaee-pom-to-quarkus-00080` — Use Quarkus junit artifact
  - keys: dep:junit.junit
- `jaxrs-to-quarkus-00000` — Replace JAX-RS dependency
  - keys: dep:org.jboss.spec.javax.ws.rs.jboss-jaxrs-api_2.1_spec
- `jaxrs-to-quarkus-00010` — Replace JAX-RS dependency
  - keys: dep:javax.ws.rs.javax.ws.rs-api
- `jaxrs-to-quarkus-00020` — JAX-RS activation is no longer necessary
  - keys: java:javax.ws.rs.applicationpath, java:javax.ws.rs.core.application
- `jdbc-jpa-mixed-to-quarkus-00001` — Mixed JDBC and JPA usage detected
  - keys: java:*entitymanager, java:java.sql.preparedstatement
- `jdbc-jpa-mixed-to-quarkus-00002` — Direct JDBC Connection usage detected
  - keys: java:java.sql.connection
- `jdbc-jpa-mixed-to-quarkus-00003` — Statement usage should be reviewed
  - keys: java:java.sql.statement
- `jms-to-reactive-quarkus-00000` — JMS is not supported in Quarkus
  - keys: dep:jakarta.jms.jakarta.jms-api, dep:javax.jms.javax.jms-api
- `jms-to-reactive-quarkus-00010` — @MessageDriven - EJBs are not supported in Quarkus
  - keys: java:javax.ejb.messagedriven
- `jms-to-reactive-quarkus-00020` — Configure message listener method with @Incoming
  - keys: java:javax.ejb.activationconfigproperty
- `jms-to-reactive-quarkus-00030` — JMS' Queue must be replaced with an Emitter
  - keys: java:javax.jms.queue
- `jms-to-reactive-quarkus-00040` — JMS' Topic must be replaced with an Emitter
  - keys: java:javax.jms.topic
- `jms-to-reactive-quarkus-00050` — JMS is not supported in Quarkus
  - keys: java:javax.jms*
- `jndi-to-quarkus-00001` — JNDI InitialContext is not supported in Quarkus
  - keys: java:javax.naming.initialcontext
- `jndi-to-quarkus-00002` — JNDI lookup() method is not supported in Quarkus
  - keys: java:javax.naming.context.lookup*
- `persistence-to-quarkus-00000` — Move persistence config to a properties file
  - keys: file:.*-ds\.xml, file:persistence\.xml
- `persistence-to-quarkus-00011` — @Produces cannot annotate an EntityManager
  - keys: java:javax.persistence.entitymanager
- `remote-ejb-to-quarkus-00000` — Remote EJBs are not supported in Quarkus
  - keys: java:javax.ejb.remote
- `springboot-actuator-to-quarkus-0100` — Replace the Spring Boot Actuator dependency with Quarkus Smallrye Health extension
  - keys: dep:org.springframework.boot.spring-boot-actuator, dep:org.springframework.boot.spring-boot-actuator-autoconfigure, dep:org.springframework.boot.spring-boot-starter-actuator
- `springboot-actuator-to-quarkus-0200` — Replace Spring Health endpoint mapping
  - keys: fc:management.endpoints.web.exposure.include.*health
- `springboot-annotations-to-quarkus-00000` — Remove the SpringBoot @SpringBootApplication annotation
  - keys: java:org.springframework.boot.autoconfigure.springbootapplication
- `springboot-cache-to-quarkus-00000` — Replace the SpringBoot cache artifact with Quarkus 'spring-cache' extension
  - keys: dep:org.springframework.boot.spring-boot-starter-cache
- `springboot-cloud-config-client-to-quarkus-00000` — Replace the Spring Cloud Config Client artifact with Quarkus 'quarkus-spring-cloud-config-client' extension
  - keys: dep:org.springframework.cloud.spring-cloud-config-client
- `springboot-devtools-to-quarkus-0000` — Remove spring-boot-devtools dependency
  - keys: xml:/m:project/m:dependencies/m:dependency[m:artifactId = 'spring-boot-devtools']
- `springboot-di-to-quarkus-00000` — Replace the SpringBoot Dependency Injection artifact with Quarkus 'spring-di' extension
  - keys: dep:org.springframework.spring-beans
- `springboot-di-to-quarkus-00001` — For Spring DI the XML-based bean configuration metadata is not supported by Quarkus 
  - keys: xml://*/b:bean/@class, xml://*/c:annotation-config
- `springboot-di-to-quarkus-00002` — Spring DI infrastructure classes not supported by Quarkus
  - keys: java:org.springframework.beans.factory.config.beanfactorypostprocessor, java:org.springframework.beans.factory.config.beanpostprocessor, java:org.springframework.beans.factory.config.destructionawarebeanpostprocessor, …+2
- `springboot-generic-catchall-00100` — Spring component requires investigation for compatibility with Quarkus extensions or possibility of code rewrite.
  - keys: dep:{group}.{artifact}
- `springboot-integration-to-quarkus-00010` — SpringBoot Integration flows are not supported.
  - keys: xml://*/int:channel
- `springboot-integration-to-quarkus-00020` — SpringBoot IntegrationFlow class usage is not supported.
  - keys: java:org.springframework.integration.dsl.integrationflow
- `springboot-jmx-to-quarkus-00000` — Spring JMX is not supported by Quarkus with GraalVM on a Native Image
  - keys: xml://*/c:bean/@class[matches(self::node(), 'org.springframework.jmx.export.MBeanExporter')]
- `springboot-jmx-to-quarkus-00001` — Spring JMX is not supported by Quarkus with GraalVM on a Native Image
  - keys: java:org.springframework.jmx.*
- `springboot-jpa-to-quarkus-00000` — Replace the SpringBoot Data JPA artifact with Quarkus 'spring-data-jpa' extension
  - keys: dep:org.springframework.boot.spring-boot-starter-data-jpa, dep:org.springframework.data.spring-data-jpa
- `springboot-metrics-to-quarkus-0100` — Replace the Micrometer dependency with Quarkus Microprofile 'metrics' extension
  - keys: dep:io.micrometer.micrometer-core
- `springboot-metrics-to-quarkus-0200` — Replace the Micrometer code with Microprofile Metrics code
  - keys: dep:io.micrometer.micrometer-core
- `springboot-metrics-to-quarkus-0300` — Replace Spring Prometheus Metrics endpoint mapping
  - keys: fc:management.endpoints.web.exposure.include.*prometheus
- `springboot-parent-pom-to-quarkus-00000` — Replace the Spring Parent POM with Quarkus BOM
  - keys: xml:/m:project/m:parent[m:groupId/text() = 'org.springframework.boot' and m:artifactId/text() = 'spring-boot-parent'], xml:/m:project/m:parent[m:groupId/text() = 'org.springframework.boot' and m:artifactId/text() = 'spring-boot-starter-parent']
- `springboot-plugins-to-quarkus-0000` — Replace the spring-boot-maven-plugin dependency
  - keys: xml:/m:project/m:build/m:plugins/m:plugin[m:artifactId = 'spring-boot-maven-plugin']
- `springboot-properties-to-quarkus-00000` — Replace the SpringBoot artifact with Quarkus 'spring-boot-properties' extension
  - keys: dep:org.springframework.boot.spring-boot
- `springboot-properties-to-quarkus-00001` — Spring property profiles in separate files must be refactored into Quarkus properties file
  - keys: file:application-.+\.(properties|yml|yaml)
- `springboot-properties-to-quarkus-00002` — Replace Spring datasource property key/value pairs with Quarkus properties
  - keys: fc:spring.datasource
- `springboot-properties-to-quarkus-00003` — Replace Spring log level property with Quarkus property
  - keys: fc:logging.level.org.springframework
- `springboot-properties-to-quarkus-00004` — Replace Spring JPA Hiberate property with Quarkus property
  - keys: fc:spring.jpa.hibernate.ddl-auto=create-drop
- `springboot-properties-to-quarkus-00005` — Replace Spring Swagger endpoint mapping
  - keys: fc:springdoc.swagger-ui.path
- `springboot-properties-to-quarkus-00006` — Replace Spring OpenAPI endpoint mapping
  - keys: fc:springdoc.api-docs.path
- `springboot-scheduled-to-quarkus-00000` — Replace the SpringBoot context artifact with Quarkus 'spring-scheduled' extension
  - keys: java:org.springframework.scheduling.annotation.scheduled
- `springboot-security-to-quarkus-00000` — Replace the SpringBoot Security artifact with Quarkus 'spring-security' extension
  - keys: dep:org.springframework.boot.spring-boot-starter-security, dep:org.springframework.security.spring-security-core
- `springboot-shell-to-quarkus-00000` — Replace the SpringBoot Shell artifact with Quarkus 'picocli' extension
  - keys: dep:org.springframework.shell.spring-shell-core
- `springboot-web-to-quarkus-00000` — Replace the Spring Web artifact with Quarkus 'spring-web' extension
  - keys: dep:org.springframework.boot.spring-boot-starter-web, dep:org.springframework.spring-web
- `springboot-web-to-quarkus-00010` — Add the Quarkus 'quarkus-resteasy-reactive-jackson' dependency
  - keys: dep:io.quarkus.quarkus-resteasy-reactive-jackson, dep:io.quarkus.quarkus-spring-web
- `springboot-webmvc-to-quarkus-00000` — Spring MVC is not supported by Quarkus
  - keys: java:org.springframework.web.servlet.mvc*
- `springboot-webmvc-to-quarkus-01000` — Spring WebFlux is not supported by Quarkus
  - keys: dep:org.springframework.boot.spring-boot-starter-webflux, dep:org.springframework.spring-webflux
- `transaction-to-quarkus-00001` — EntityManager persistence operations require @Transactional in Quarkus
  - keys: java:javax.persistence.entitymanager.persist*
- `transaction-to-quarkus-00002` — EntityManager merge operations require @Transactional in Quarkus
  - keys: java:javax.persistence.entitymanager.merge*
- `transaction-to-quarkus-00003` — EntityManager remove operations require @Transactional in Quarkus
  - keys: java:javax.persistence.entitymanager.remove*

## Kantra diff (app: /Users/fabian/scratch/eval-apps/coolstore)

| | ai-generated | handcrafted |
|---|---|---|
| Rules fired | 8 | 25 |
| Incidents | 12 | 55 |
| Files flagged only here | 0 | 6 |
| Files flagged by both | 9 | 9 |

### Files flagged only by handcrafted (6)

- `src/main/java/com/redhat/coolstore/rest/RestApplication.java`
- `src/main/java/com/redhat/coolstore/service/InventoryNotificationMDB.java`
- `src/main/java/com/redhat/coolstore/service/OrderServiceMDB.java`
- `src/main/java/com/redhat/coolstore/utils/Producers.java`
- `src/main/resources/META-INF/persistence.xml`
- `src/main/webapp/WEB-INF/beans.xml`

### Files flagged by both (9)

- `pom.xml` — A: [build-xml-00010 build-xml-00020 build-xml-00050 …+1] · B: [javaee-pom-to-quarkus-00000 javaee-pom-to-quarkus-00010 javaee-pom-to-quarkus-00020 …+4]
- `src/main/java/com/redhat/coolstore/persistence/Resources.java` — A: [persistence-annotation-00010] · B: [cdi-to-quarkus-00040 persistence-to-quarkus-00010 persistence-to-quarkus-00011]
- `src/main/java/com/redhat/coolstore/service/CatalogService.java` — A: [ejb-annotation-00010] · B: [ee-to-quarkus-00000 ee-to-quarkus-00020 transaction-to-quarkus-00002]
- `src/main/java/com/redhat/coolstore/service/OrderService.java` — A: [ejb-annotation-00010] · B: [ee-to-quarkus-00000 ee-to-quarkus-00020 transaction-to-quarkus-00001]
- `src/main/java/com/redhat/coolstore/service/ProductService.java` — A: [ejb-annotation-00010] · B: [ee-to-quarkus-00000 ee-to-quarkus-00020]
- `src/main/java/com/redhat/coolstore/service/ShippingService.java` — A: [ejb-annotation-00010] · B: [ee-to-quarkus-00000 ee-to-quarkus-00020 remote-ejb-to-quarkus-00000]
- `src/main/java/com/redhat/coolstore/service/ShoppingCartOrderProcessor.java` — A: [ejb-annotation-00010] · B: [ee-to-quarkus-00000 ee-to-quarkus-00020 jms-to-reactive-quarkus-00040 …+1]
- `src/main/java/com/redhat/coolstore/service/ShoppingCartService.java` — A: [ejb-annotation-00020] · B: [ee-to-quarkus-00010 ee-to-quarkus-00020 jndi-to-quarkus-00001 …+1]
- `src/main/java/com/redhat/coolstore/utils/DataBaseMigrationStartup.java` — A: [ejb-annotation-00030] · B: [ee-to-quarkus-00020]


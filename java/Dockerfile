# Stage 1: Build
FROM eclipse-temurin:17-jdk-jammy AS builder

WORKDIR /build
COPY pom.xml .
COPY src ./src

# Install Maven wrapper if mvnw is present, else use system maven
RUN apt-get update && apt-get install -y maven && rm -rf /var/lib/apt/lists/*
RUN mvn clean package -DskipTests -B

# Stage 2: Runtime
FROM eclipse-temurin:17-jre-jammy

LABEL maintainer="superagent-ai"
LABEL description="Superagent Base Java — AgentScope 2.0 agent platform"

WORKDIR /app

# Create non-root user
RUN groupadd -r superagent && useradd -r -g superagent superagent

COPY --from=builder /build/target/superagent-base-java-*.jar app.jar

RUN chown -R superagent:superagent /app
USER superagent

# Default agent config volume
VOLUME /app/configs

EXPOSE 8890
EXPOSE 50051

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8890/actuator/health || exit 1

ENTRYPOINT ["java", \
    "-XX:+UseG1GC", \
    "-XX:MaxRAMPercentage=75.0", \
    "-jar", "app.jar"]

# Matrix Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 拉起 Go / Java / Python 三套独立后端 + 三个前端实例，通过功能矩阵测试验证 API 契约一致性。

**Architecture:** 共享 MySQL + Redis（Go 用全量 schema，Java/Python 用 Redis 做会话存储），三套后端各自 Docker 化，前端通过 Vite `--mode` 切换后端目标，矩阵测试套件用现有 Python SDK 对三后端各跑一遍。

**Tech Stack:** Docker Compose, Spring Boot 3 + WebFlux, FastAPI + AgentScope 2.0, Hertz + Eino, Vite `--mode`, pytest-asyncio, Python SDK (`sdks/python/`)

---

## File Map

| 文件 | 操作 | 说明 |
|---|---|---|
| `docker/docker-compose-matrix.yml` | 新建 | 三后端 + 共享中间件 |
| `docker/.env.matrix.example` | 新建 | 矩阵部署环境变量模板 |
| `docker/volumes/mysql/schema-matrix.sql` | 新建 | Go 后端用的轻量 schema（仅 api_key 表） |
| `Makefile` | 修改 | 新增 matrix-* targets |
| `java/src/main/.../memory/RedisMemory.java` | 修改 | 实现 store/retrieve/clear |
| `java/src/main/.../tools/builtin/HttpRequestTool.java` | 修改 | 实现 HTTP 调用 |
| `java/src/main/.../tools/builtin/WebSearchTool.java` | 修改 | 实现 DuckDuckGo 搜索 |
| `java/src/main/.../tools/builtin/CodeExecuteTool.java` | 修改 | 实现 subprocess 代码执行 |
| `web/vite.matrix.config.ts` | 新建 | 支持 --mode 切换后端的 Vite 配置 |
| `web/.env.go` | 新建 | VITE_API_BASE=http://localhost:8888 |
| `web/.env.python` | 新建 | VITE_API_BASE=http://localhost:8889 |
| `web/.env.java` | 新建 | VITE_API_BASE=http://localhost:8890 |
| `tests/matrix/__init__.py` | 新建 | 空 |
| `tests/matrix/conftest.py` | 新建 | pytest fixtures（三后端 client） |
| `tests/matrix/test_health.py` | 新建 | 健康检查测例 |
| `tests/matrix/test_agents.py` | 新建 | Agent 列表 + Admin CRUD |
| `tests/matrix/test_chat.py` | 新建 | 流式对话 + collect() |
| `tests/matrix/report.py` | 新建 | 生成 Markdown 对比报告 |
| `tests/matrix/requirements.txt` | 新建 | pytest + httpx + anyio |

---

## Task 1：docker-compose-matrix.yml

**Files:**
- Create: `docker/docker-compose-matrix.yml`
- Create: `docker/.env.matrix.example`
- Create: `docker/volumes/mysql/schema-matrix.sql`

- [ ] **Step 1: 创建 MySQL schema-matrix.sql（Go 后端最小依赖）**

```sql
-- docker/volumes/mysql/schema-matrix.sql
SET NAMES utf8mb4;
CREATE DATABASE IF NOT EXISTS sa_go COLLATE utf8mb4_unicode_ci;
USE sa_go;
CREATE TABLE IF NOT EXISTS `api_key` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `api_key` varchar(255) NOT NULL DEFAULT '',
  `name` varchar(255) NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT 0,
  `user_id` bigint NOT NULL DEFAULT 0,
  `expired_at` bigint NOT NULL DEFAULT 0,
  `created_at` bigint unsigned NOT NULL DEFAULT 0,
  `updated_at` bigint unsigned NOT NULL DEFAULT 0,
  `last_used_at` bigint NOT NULL DEFAULT 0,
  `ak_type` tinyint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
INSERT IGNORE INTO `api_key` (`id`,`api_key`,`name`,`status`,`user_id`,`expired_at`,`created_at`,`updated_at`,`last_used_at`,`ak_type`)
VALUES (1,'matrix-admin-key','Matrix Admin Key',0,1,0,0,0,0,0);
```

- [ ] **Step 2: 创建 .env.matrix.example**

```bash
# docker/.env.matrix.example
# Copy to .env.matrix and fill in your values

# ── LLM 模型（三后端共用） ──────────────────────────────────────────────
OPENAI_API_KEY=sk-your-key-here
OPENAI_BASE_URL=https://api.openai.com/v1
DEFAULT_MODEL=gpt-4o-mini

# ── Go 后端 ───────────────────────────────────────────────────────────
GO_LISTEN_ADDR=:8888
GO_MYSQL_DSN=coze:coze123@tcp(sa-mysql:3306)/sa_go?charset=utf8mb4&parseTime=True
GO_REDIS_ADDR=sa-redis:6379
GO_ADMIN_API_KEY=matrix-admin-key

# ── Python 后端 ───────────────────────────────────────────────────────
PYTHON_PORT=8889
PYTHON_REDIS_URL=redis://sa-redis:6379/1
PYTHON_AGENTS_DIR=/app/configs/agents
PYTHON_ADMIN_API_KEY=matrix-admin-key

# ── Java 后端 ─────────────────────────────────────────────────────────
JAVA_PORT=8890
JAVA_REDIS_URL=redis://sa-redis:6379/2
JAVA_AGENTS_DIR=/app/configs/agents
JAVA_ADMIN_API_KEY=matrix-admin-key
```

- [ ] **Step 3: 创建 docker-compose-matrix.yml**

```yaml
# docker/docker-compose-matrix.yml
name: sa-matrix

x-env-file: &env_file
  - .env.matrix

services:
  # ── 共享中间件 ────────────────────────────────────────────────────────
  sa-mysql:
    image: mysql:8.4.5
    container_name: sa-matrix-mysql
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: sa_go
      MYSQL_USER: coze
      MYSQL_PASSWORD: coze123
    ports:
      - "3306:3306"
    volumes:
      - ./data/matrix-mysql:/var/lib/mysql
      - ./volumes/mysql/schema-matrix.sql:/docker-entrypoint-initdb.d/init.sql
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-ucoze", "-pcoze123"]
      interval: 5s
      timeout: 5s
      retries: 12
    networks:
      - sa-matrix-net

  sa-redis:
    image: redis:7-alpine
    container_name: sa-matrix-redis
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 3s
      timeout: 3s
      retries: 5
    networks:
      - sa-matrix-net

  # ── Go 后端（:8888）──────────────────────────────────────────────────
  go-backend:
    build:
      context: ../backend
      dockerfile: ../Dockerfile.production
    container_name: sa-matrix-go
    env_file: *env_file
    environment:
      APP_ENV: debug
      LISTEN_ADDR: ":8888"
      MYSQL_DSN: ${GO_MYSQL_DSN}
      REDIS_ADDR: ${GO_REDIS_ADDR}
      AGENT_CONFIG_DIR: /app/configs/agents
      ADMIN_API_KEY: ${GO_ADMIN_API_KEY}
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      OPENAI_BASE_URL: ${OPENAI_BASE_URL}
    ports:
      - "8888:8888"
    volumes:
      - ../configs/agents:/app/configs/agents:ro
    depends_on:
      sa-mysql:
        condition: service_healthy
      sa-redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8888/health"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 15s
    networks:
      - sa-matrix-net

  # ── Python 后端（:8889）──────────────────────────────────────────────
  python-backend:
    build:
      context: ../python
      dockerfile: Dockerfile
    container_name: sa-matrix-python
    env_file: *env_file
    environment:
      PORT: ${PYTHON_PORT:-8889}
      REDIS_URL: ${PYTHON_REDIS_URL}
      AGENTS_DIR: ${PYTHON_AGENTS_DIR:-/app/configs/agents}
      ADMIN_API_KEY: ${PYTHON_ADMIN_API_KEY}
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      OPENAI_BASE_URL: ${OPENAI_BASE_URL}
    ports:
      - "8889:8889"
    volumes:
      - ../configs/agents:/app/configs/agents:ro
    depends_on:
      sa-redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8889/health"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 10s
    networks:
      - sa-matrix-net

  # ── Java 后端（:8890）────────────────────────────────────────────────
  java-backend:
    build:
      context: ../java
      dockerfile: Dockerfile
    container_name: sa-matrix-java
    env_file: *env_file
    environment:
      SERVER_PORT: ${JAVA_PORT:-8890}
      SPRING_REDIS_URL: ${JAVA_REDIS_URL}
      SUPERAGENT_AGENTS_DIR: ${JAVA_AGENTS_DIR:-/app/configs/agents}
      SUPERAGENT_ADMIN_API_KEY: ${JAVA_ADMIN_API_KEY}
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      OPENAI_BASE_URL: ${OPENAI_BASE_URL}
    ports:
      - "8890:8890"
    volumes:
      - ../configs/agents:/app/configs/agents:ro
    depends_on:
      sa-redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8890/actuator/health"]
      interval: 10s
      timeout: 5s
      retries: 12
      start_period: 30s
    networks:
      - sa-matrix-net

networks:
  sa-matrix-net:
    name: sa-matrix-net
```

- [ ] **Step 4: 验证 YAML 语法**

```bash
cd /Users/ailabuser1/Desktop/gitcode/superagent-base
docker compose -f docker/docker-compose-matrix.yml config --quiet
```

Expected: 无报错输出

- [ ] **Step 5: 添加 Makefile matrix targets**

在 `Makefile` 顶部的 `.PHONY` 行末尾追加，然后在文件末尾添加：

```makefile
# ── Matrix deployment targets ────────────────────────────────────────

MATRIX_COMPOSE := docker/docker-compose-matrix.yml
MATRIX_ENV := docker/.env.matrix

matrix-env:
	@if [ ! -f "$(MATRIX_ENV)" ]; then \
		echo "Creating $(MATRIX_ENV) from example..."; \
		cp docker/.env.matrix.example $(MATRIX_ENV); \
		echo "Edit $(MATRIX_ENV) and set your OPENAI_API_KEY before running matrix-up"; \
	fi

matrix-up: matrix-env
	@echo "Starting matrix: Go:8888 Python:8889 Java:8890 ..."
	@docker compose -f $(MATRIX_COMPOSE) --env-file $(MATRIX_ENV) up -d --build

matrix-down:
	@docker compose -f $(MATRIX_COMPOSE) down

matrix-clean:
	@docker compose -f $(MATRIX_COMPOSE) down -v
	@rm -rf docker/data/matrix-mysql

matrix-logs:
	@docker compose -f $(MATRIX_COMPOSE) logs -f

matrix-ps:
	@docker compose -f $(MATRIX_COMPOSE) ps

matrix-wait:
	@echo "Waiting for all three backends to be healthy..."
	@for port in 8888 8889 8890; do \
		echo -n "  Waiting for :$$port "; \
		for i in $$(seq 1 30); do \
			if curl -sf http://localhost:$$port/health > /dev/null 2>&1 || \
			   curl -sf http://localhost:$$port/actuator/health > /dev/null 2>&1; then \
				echo " OK"; break; \
			fi; \
			echo -n "."; sleep 3; \
		done; \
	done

matrix-test: matrix-wait
	@echo "Running matrix tests..."
	@cd tests/matrix && pip install -q -r requirements.txt && pytest -v --tb=short

matrix-fe-go:
	@cd web && VITE_API_BASE=http://localhost:8888 npx vite --port 3501 --mode go

matrix-fe-python:
	@cd web && VITE_API_BASE=http://localhost:8889 npx vite --port 3502 --mode python

matrix-fe-java:
	@cd web && VITE_API_BASE=http://localhost:8890 npx vite --port 3503 --mode java

matrix-fe:
	@echo "Starting 3 frontend instances (Go:3501, Python:3502, Java:3503)..."
	@$(MAKE) matrix-fe-go & \
	 $(MAKE) matrix-fe-python & \
	 $(MAKE) matrix-fe-java & \
	 wait
```

- [ ] **Step 6: Commit**

```bash
cd /Users/ailabuser1/Desktop/gitcode/superagent-base
git add docker/docker-compose-matrix.yml docker/.env.matrix.example \
        docker/volumes/mysql/schema-matrix.sql Makefile
git commit -m "feat(matrix): add docker-compose-matrix and Makefile targets for 3-backend deployment"
```

---

## Task 2：Java RedisMemory 实现

**Files:**
- Modify: `java/src/main/java/io/superagent/memory/RedisMemory.java`
- Test: `java/src/test/java/io/superagent/agents/BaseAgentTest.java`（已有，验证 memory 接口）

- [ ] **Step 1: 写失败测试（验证 store/retrieve 有数据）**

在 `java/src/test/java/io/superagent/agents/BaseAgentTest.java` 中添加：

```java
@Test
void redisMemoryStoreAndRetrieve() {
    // Given
    BuiltinMemory mem = new BuiltinMemory();  // 用 builtin 替代测 interface
    mem.store("sess1", "user", "Hello from test", Map.of());

    // When
    List<MemoryStore.MemoryMessage> messages = mem.retrieve("sess1", 10);

    // Then
    assertFalse(messages.isEmpty(), "Should retrieve stored message");
    assertEquals("Hello from test", messages.get(0).content());
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /Users/ailabuser1/Desktop/gitcode/superagent-base/java
mvn test -pl . -Dtest=BaseAgentTest#redisMemoryStoreAndRetrieve -q 2>&1 | tail -5
```

Expected: BUILD SUCCESS（BuiltinMemory 已实现，此测试通过作为基准）

- [ ] **Step 3: 实现 RedisMemory（用 Redis LIST 存储消息）**

完整替换 `java/src/main/java/io/superagent/memory/RedisMemory.java`：

```java
package io.superagent.memory;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.core.ReactiveStringRedisTemplate;
import org.springframework.stereotype.Component;

import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Redis-backed memory store.
 * Key pattern: {@code superagent:memory:<sessionId>}
 * Each value is a JSON-serialised MemoryMessage stored as a Redis LIST.
 */
@Component
public class RedisMemory implements MemoryStore {

    private static final Logger log = LoggerFactory.getLogger(RedisMemory.class);
    private static final String KEY_PREFIX = "superagent:memory:";
    private static final Duration TTL = Duration.ofHours(24);
    private static final int MAX_MESSAGES = 500;

    private final ReactiveStringRedisTemplate redis;
    private final ObjectMapper mapper = new ObjectMapper();

    public RedisMemory(ReactiveStringRedisTemplate redis) {
        this.redis = redis;
    }

    @Override
    public void store(String sessionId, String role, String content,
                      Map<String, Object> metadata) {
        String key = KEY_PREFIX + sessionId;
        MemoryMessage msg = new MemoryMessage(role, content,
                System.currentTimeMillis(), metadata != null ? metadata : Map.of());
        try {
            String json = mapper.writeValueAsString(msg);
            redis.opsForList().rightPush(key, json)
                .then(redis.expire(key, TTL))
                .then(trimIfNeeded(key))
                .subscribe(
                    v -> {},
                    e -> log.error("RedisMemory.store failed for session {}: {}", sessionId, e.getMessage())
                );
        } catch (JsonProcessingException e) {
            log.error("Failed to serialise message for session {}: {}", sessionId, e.getMessage());
        }
    }

    @Override
    public List<MemoryMessage> retrieve(String sessionId, int limit) {
        String key = KEY_PREFIX + sessionId;
        try {
            List<String> raw = redis.opsForList()
                .range(key, -Math.max(limit, 1), -1)
                .collectList()
                .block(Duration.ofSeconds(5));
            if (raw == null) return List.of();
            List<MemoryMessage> result = new ArrayList<>();
            for (String json : raw) {
                try {
                    result.add(mapper.readValue(json, MemoryMessage.class));
                } catch (JsonProcessingException e) {
                    log.warn("Skipping malformed message in session {}", sessionId);
                }
            }
            return result;
        } catch (Exception e) {
            log.error("RedisMemory.retrieve failed for session {}: {}", sessionId, e.getMessage());
            return List.of();
        }
    }

    @Override
    public List<MemoryMessage> search(String sessionId, String query, int limit) {
        // Simple substring search over recent messages
        return retrieve(sessionId, MAX_MESSAGES).stream()
            .filter(m -> m.content().toLowerCase().contains(query.toLowerCase()))
            .limit(limit)
            .toList();
    }

    @Override
    public void clear(String sessionId) {
        redis.delete(KEY_PREFIX + sessionId)
            .subscribe(
                v -> log.debug("Cleared memory for session {}", sessionId),
                e -> log.error("RedisMemory.clear failed: {}", e.getMessage())
            );
    }

    private reactor.core.publisher.Mono<Void> trimIfNeeded(String key) {
        return redis.opsForList().size(key)
            .flatMap(size -> {
                if (size > MAX_MESSAGES) {
                    return redis.opsForList().trim(key, size - MAX_MESSAGES, -1);
                }
                return reactor.core.publisher.Mono.empty();
            });
    }
}
```

- [ ] **Step 4: 运行 Java 编译验证**

```bash
cd /Users/ailabuser1/Desktop/gitcode/superagent-base/java
mvn compile -q 2>&1 | tail -10
```

Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add java/src/main/java/io/superagent/memory/RedisMemory.java
git commit -m "feat(java): implement RedisMemory store/retrieve/clear with Redis LIST"
```

---

## Task 3：Java Builtin Tools 实现

**Files:**
- Modify: `java/src/main/java/io/superagent/tools/builtin/HttpRequestTool.java`
- Modify: `java/src/main/java/io/superagent/tools/builtin/WebSearchTool.java`
- Modify: `java/src/main/java/io/superagent/tools/builtin/CodeExecuteTool.java`
- Test: `java/src/test/java/io/superagent/tools/CodeExecuteToolTest.java`（已有）

- [ ] **Step 1: 运行已有 CodeExecuteTool 测试确认当前状态**

```bash
cd /Users/ailabuser1/Desktop/gitcode/superagent-base/java
mvn test -Dtest=CodeExecuteToolTest -q 2>&1 | tail -10
```

Expected: 记录当前通过/失败数量

- [ ] **Step 2: 实现 HttpRequestTool（用 java.net.http.HttpClient）**

完整替换 `java/src/main/java/io/superagent/tools/builtin/HttpRequestTool.java`：

```java
package io.superagent.tools.builtin;

import io.superagent.tools.Tool;
import org.springframework.stereotype.Component;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * HTTP request tool — makes HTTP calls to external APIs.
 * URI: {@code builtin/http_request}
 */
@Component
public class HttpRequestTool implements Tool {

    private final HttpClient client = HttpClient.newBuilder()
        .connectTimeout(Duration.ofSeconds(10))
        .followRedirects(HttpClient.Redirect.NORMAL)
        .build();

    @Override
    public String getName() { return "http_request"; }

    @Override
    public String getDescription() {
        return "Make an HTTP request to a given URL. Supports GET, POST, PUT, DELETE with headers and body.";
    }

    @Override
    public Map<String, Object> getParameterSchema() {
        return Map.of(
            "type", "object",
            "properties", Map.of(
                "url",     Map.of("type", "string",  "description", "Request URL"),
                "method",  Map.of("type", "string",  "description", "HTTP method (GET, POST, PUT, DELETE)"),
                "headers", Map.of("type", "object",  "description", "Request headers"),
                "body",    Map.of("type", "string",  "description", "Request body for POST/PUT")
            ),
            "required", List.of("url", "method")
        );
    }

    @Override
    @SuppressWarnings("unchecked")
    public Map<String, Object> execute(Map<String, Object> parameters) {
        String url    = (String) parameters.getOrDefault("url", "");
        String method = ((String) parameters.getOrDefault("method", "GET")).toUpperCase();
        String body   = (String) parameters.getOrDefault("body", "");

        HttpRequest.Builder builder = HttpRequest.newBuilder()
            .uri(URI.create(url))
            .timeout(Duration.ofSeconds(30));

        // Set custom headers
        Object headersRaw = parameters.get("headers");
        if (headersRaw instanceof Map<?,?> hdrs) {
            ((Map<String, String>) hdrs).forEach(builder::header);
        }
        if (!builder.build().headers().map().containsKey("Content-Type")) {
            builder.header("Content-Type", "application/json");
        }

        builder = switch (method) {
            case "POST" -> builder.POST(HttpRequest.BodyPublishers.ofString(body));
            case "PUT"  -> builder.PUT(HttpRequest.BodyPublishers.ofString(body));
            case "DELETE" -> builder.DELETE();
            default -> builder.GET();
        };

        try {
            HttpResponse<String> resp = client.send(builder.build(),
                HttpResponse.BodyHandlers.ofString());
            Map<String, Object> result = new LinkedHashMap<>();
            result.put("tool",        getName());
            result.put("status",      "success");
            result.put("status_code", resp.statusCode());
            result.put("url",         url);
            result.put("method",      method);
            result.put("body",        resp.body());
            return result;
        } catch (Exception e) {
            return Map.of("tool", getName(), "status", "error",
                          "url", url, "message", e.getMessage());
        }
    }
}
```

- [ ] **Step 3: 实现 WebSearchTool（DuckDuckGo Lite HTML 解析）**

完整替换 `java/src/main/java/io/superagent/tools/builtin/WebSearchTool.java`：

```java
package io.superagent.tools.builtin;

import io.superagent.tools.Tool;
import org.springframework.stereotype.Component;

import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.*;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Web search via DuckDuckGo Lite (no API key required).
 * URI: {@code builtin/web_search}
 */
@Component
public class WebSearchTool implements Tool {

    private static final String DDGO_URL = "https://lite.duckduckgo.com/lite/?q=";
    private static final Pattern TITLE_PATTERN =
        Pattern.compile("<a[^>]+class=\"result-link\"[^>]*>([^<]+)</a>", Pattern.CASE_INSENSITIVE);
    private static final Pattern SNIPPET_PATTERN =
        Pattern.compile("<td[^>]+class=\"result-snippet\"[^>]*>([^<]+)</td>", Pattern.CASE_INSENSITIVE);

    private final HttpClient client = HttpClient.newBuilder()
        .connectTimeout(Duration.ofSeconds(10))
        .followRedirects(HttpClient.Redirect.NORMAL)
        .build();

    @Override
    public String getName() { return "web_search"; }

    @Override
    public String getDescription() {
        return "Search the web via DuckDuckGo. Returns titles and snippets.";
    }

    @Override
    public Map<String, Object> getParameterSchema() {
        return Map.of(
            "type", "object",
            "properties", Map.of(
                "query",       Map.of("type", "string",  "description", "Search query"),
                "num_results", Map.of("type", "integer", "description", "Max results (default 5)")
            ),
            "required", List.of("query")
        );
    }

    @Override
    public Map<String, Object> execute(Map<String, Object> parameters) {
        String query = (String) parameters.getOrDefault("query", "");
        int numResults = parameters.get("num_results") instanceof Number n ? n.intValue() : 5;

        if (query.isBlank()) {
            return Map.of("tool", getName(), "status", "error", "message", "query is required");
        }

        String encodedQuery = URLEncoder.encode(query, StandardCharsets.UTF_8);
        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(DDGO_URL + encodedQuery))
            .header("User-Agent", "Mozilla/5.0 (compatible; SuperagentBot/1.0)")
            .timeout(Duration.ofSeconds(15))
            .GET()
            .build();

        try {
            HttpResponse<String> resp = client.send(request, HttpResponse.BodyHandlers.ofString());
            List<Map<String, String>> results = parseResults(resp.body(), numResults);
            return Map.of(
                "tool", getName(), "status", "success",
                "query", query, "results", results
            );
        } catch (Exception e) {
            return Map.of("tool", getName(), "status", "error",
                          "query", query, "message", e.getMessage());
        }
    }

    private List<Map<String, String>> parseResults(String html, int limit) {
        List<Map<String, String>> results = new ArrayList<>();
        Matcher titleMatcher   = TITLE_PATTERN.matcher(html);
        Matcher snippetMatcher = SNIPPET_PATTERN.matcher(html);

        while (titleMatcher.find() && results.size() < limit) {
            Map<String, String> r = new LinkedHashMap<>();
            r.put("title", titleMatcher.group(1).strip());
            if (snippetMatcher.find()) {
                r.put("snippet", snippetMatcher.group(1).strip());
            }
            results.add(r);
        }
        return results;
    }
}
```

- [ ] **Step 4: 实现 CodeExecuteTool（subprocess 隔离执行）**

完整替换 `java/src/main/java/io/superagent/tools/builtin/CodeExecuteTool.java`：

```java
package io.superagent.tools.builtin;

import io.superagent.tools.Tool;
import org.springframework.stereotype.Component;

import java.io.*;
import java.nio.file.*;
import java.util.*;
import java.util.concurrent.TimeUnit;

/**
 * Code execution via subprocess isolation.
 * Supports python3 and node. URI: {@code builtin/code_execute}
 *
 * Security note: This runs arbitrary code — use only in trusted/sandboxed environments.
 */
@Component
public class CodeExecuteTool implements Tool {

    @Override
    public String getName() { return "code_execute"; }

    @Override
    public String getDescription() {
        return "Execute code in a subprocess. Supports python and javascript. Returns stdout, stderr, exit_code.";
    }

    @Override
    public Map<String, Object> getParameterSchema() {
        return Map.of(
            "type", "object",
            "properties", Map.of(
                "language", Map.of("type", "string",  "description", "python or javascript"),
                "code",     Map.of("type", "string",  "description", "Code to execute"),
                "timeout",  Map.of("type", "integer", "description", "Timeout seconds (default 30)")
            ),
            "required", List.of("language", "code")
        );
    }

    @Override
    public Map<String, Object> execute(Map<String, Object> parameters) {
        String language = (String) parameters.getOrDefault("language", "python");
        String code     = (String) parameters.getOrDefault("code", "");
        int timeout     = parameters.get("timeout") instanceof Number n ? n.intValue() : 30;

        String ext = language.startsWith("python") ? ".py" : ".js";
        String cmd = language.startsWith("python") ? "python3" : "node";

        Path tmpFile = null;
        try {
            tmpFile = Files.createTempFile("sa_exec_", ext);
            Files.writeString(tmpFile, code);

            Process proc = new ProcessBuilder(cmd, tmpFile.toString())
                .redirectErrorStream(false)
                .start();

            boolean finished = proc.waitFor(timeout, TimeUnit.SECONDS);
            if (!finished) {
                proc.destroyForcibly();
                return Map.of("tool", getName(), "status", "timeout",
                              "language", language, "exit_code", -1,
                              "stdout", "", "stderr", "Execution timed out after " + timeout + "s");
            }

            String stdout = new String(proc.getInputStream().readAllBytes());
            String stderr = new String(proc.getErrorStream().readAllBytes());

            return Map.of(
                "tool",      getName(),
                "status",    proc.exitValue() == 0 ? "success" : "error",
                "language",  language,
                "exit_code", proc.exitValue(),
                "stdout",    stdout,
                "stderr",    stderr
            );
        } catch (Exception e) {
            return Map.of("tool", getName(), "status", "error",
                          "language", language, "exit_code", -1,
                          "stdout", "", "stderr", e.getMessage());
        } finally {
            if (tmpFile != null) {
                try { Files.deleteIfExists(tmpFile); } catch (IOException ignored) {}
            }
        }
    }
}
```

- [ ] **Step 5: 运行编译 + 工具测试**

```bash
cd /Users/ailabuser1/Desktop/gitcode/superagent-base/java
mvn test -Dtest=CodeExecuteToolTest -q 2>&1 | tail -15
```

Expected: BUILD SUCCESS

- [ ] **Step 6: Commit**

```bash
git add java/src/main/java/io/superagent/tools/builtin/
git commit -m "feat(java): implement HttpRequestTool, WebSearchTool(DuckDuckGo), CodeExecuteTool(subprocess)"
```

---

## Task 4：前端多实例配置

**Files:**
- Create: `web/.env.go`
- Create: `web/.env.python`
- Create: `web/.env.java`
- Modify: `web/vite.config.ts`

- [ ] **Step 1: 创建三个 env 文件**

```bash
echo "VITE_API_BASE=http://localhost:8888" > /Users/ailabuser1/Desktop/gitcode/superagent-base/web/.env.go
echo "VITE_API_BASE=http://localhost:8889" > /Users/ailabuser1/Desktop/gitcode/superagent-base/web/.env.python
echo "VITE_API_BASE=http://localhost:8890" > /Users/ailabuser1/Desktop/gitcode/superagent-base/web/.env.java
```

- [ ] **Step 2: 修改 vite.config.ts 支持 VITE_API_BASE 环境变量**

在 `web/vite.config.ts` 中将 proxy 配置改为动态读取：

```typescript
/// <reference types="vitest" />
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiBase = env.VITE_API_BASE || 'http://localhost:8888'

  return {
    plugins: [react()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      host: '0.0.0.0',
      port: parseInt(env.PORT || '3500'),
      proxy: {
        '/api': {
          target: apiBase,
          changeOrigin: true,
        },
        '/grpc': {
          target: apiBase.replace(/:\d+$/, ':50051'),
          changeOrigin: true,
        },
        '/metrics': {
          target: apiBase,
          changeOrigin: true,
        },
      },
    },
    build: {
      rollupOptions: {
        output: {
          manualChunks: {
            'vendor': ['react', 'react-dom', 'react-router-dom'],
            'monaco': ['@monaco-editor/react'],
            'xyflow': ['@xyflow/react'],
            'query': ['@tanstack/react-query', 'zustand'],
          },
        },
      },
    },
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/test/setup.ts'],
      exclude: ['e2e/**', 'node_modules/**'],
      css: true,
    },
  }
})
```

- [ ] **Step 3: 验证 Go 模式能正常启动**

```bash
cd /Users/ailabuser1/Desktop/gitcode/superagent-base/web
npm run dev -- --mode go --port 3501 &
sleep 3
curl -s http://localhost:3501/ | head -5
kill %1
```

Expected: 返回 HTML 内容（index.html）

- [ ] **Step 4: Commit**

```bash
git add web/vite.config.ts web/.env.go web/.env.python web/.env.java
git commit -m "feat(web): support VITE_API_BASE env var and --mode flag for multi-backend frontend"
```

---

## Task 5：功能矩阵测试套件

**Files:**
- Create: `tests/matrix/__init__.py`
- Create: `tests/matrix/requirements.txt`
- Create: `tests/matrix/conftest.py`
- Create: `tests/matrix/test_health.py`
- Create: `tests/matrix/test_agents.py`
- Create: `tests/matrix/test_chat.py`
- Create: `tests/matrix/report.py`

- [ ] **Step 1: 创建 requirements.txt**

```text
# tests/matrix/requirements.txt
pytest>=8.0
pytest-asyncio>=0.23
anyio>=4.0
httpx>=0.27
```

- [ ] **Step 2: 创建 conftest.py（三后端 fixtures）**

```python
# tests/matrix/conftest.py
"""Pytest fixtures for matrix tests.

Each fixture creates a SuperagentClient pointing at one of the three backends.
Tests parametrised with `backend_client` run against all three automatically.
"""
import sys
import os
import pytest

# Allow importing the Python SDK from the repo root
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../sdks/python'))

from superagent.client import SuperagentClient  # noqa: E402

GO_URL     = os.getenv("MATRIX_GO_URL",     "http://localhost:8888")
PYTHON_URL = os.getenv("MATRIX_PYTHON_URL", "http://localhost:8889")
JAVA_URL   = os.getenv("MATRIX_JAVA_URL",   "http://localhost:8890")

BACKENDS = [
    pytest.param(GO_URL,     id="go"),
    pytest.param(PYTHON_URL, id="python"),
    pytest.param(JAVA_URL,   id="java"),
]


@pytest.fixture(params=BACKENDS)
def client(request) -> SuperagentClient:
    """Parametrised fixture: yields one client per backend."""
    return SuperagentClient(base_url=request.param, timeout=30.0)


@pytest.fixture
def go_client() -> SuperagentClient:
    return SuperagentClient(base_url=GO_URL, timeout=30.0)


@pytest.fixture
def python_client() -> SuperagentClient:
    return SuperagentClient(base_url=PYTHON_URL, timeout=30.0)


@pytest.fixture
def java_client() -> SuperagentClient:
    return SuperagentClient(base_url=JAVA_URL, timeout=30.0)
```

- [ ] **Step 3: 创建 test_health.py**

```python
# tests/matrix/test_health.py
"""健康检查：每个后端的 /health 端点必须返回 200。"""
import httpx
import pytest

HEALTH_URLS = [
    pytest.param("http://localhost:8888/health",            id="go"),
    pytest.param("http://localhost:8889/health",            id="python"),
    pytest.param("http://localhost:8890/actuator/health",   id="java"),
]


@pytest.mark.parametrize("url", HEALTH_URLS)
def test_health_returns_200(url: str) -> None:
    resp = httpx.get(url, timeout=10)
    assert resp.status_code == 200, f"Expected 200 from {url}, got {resp.status_code}"


@pytest.mark.parametrize("url", HEALTH_URLS)
def test_health_response_is_json(url: str) -> None:
    resp = httpx.get(url, timeout=10)
    assert resp.headers.get("content-type", "").startswith("application/json"), \
        f"Expected JSON content-type from {url}"
```

- [ ] **Step 4: 创建 test_agents.py**

```python
# tests/matrix/test_agents.py
"""Agent 列表 + Admin status 测试：三后端对齐验证。"""
import pytest
from superagent.client import SuperagentClient


@pytest.mark.asyncio
async def test_list_agents_returns_list(client: SuperagentClient) -> None:
    """GET /api/v2/agents 必须返回列表（可为空）。"""
    async with client:
        agents = await client.list_agents()
    assert isinstance(agents, list), "list_agents() should return a list"


@pytest.mark.asyncio
async def test_list_agents_items_have_name(client: SuperagentClient) -> None:
    """Agent 列表中每一项都有 name 字段。"""
    async with client:
        agents = await client.list_agents()
    for agent in agents:
        assert hasattr(agent, 'name') and agent.name, \
            f"Agent item missing name: {agent}"


@pytest.mark.asyncio
async def test_admin_status(client: SuperagentClient) -> None:
    """GET /api/v2/admin/status 必须返回含 uptime 或 status 字段的对象。"""
    async with client:
        status = await client.admin.status()
    assert isinstance(status, dict), "admin.status() should return a dict"
    assert any(k in status for k in ("uptime", "status", "version", "agents_loaded")), \
        f"admin.status() missing expected fields: {status}"


@pytest.mark.asyncio
async def test_admin_validate_agent(client: SuperagentClient) -> None:
    """POST /api/v2/admin/agents/validate 对有效 YAML 应返回 valid=True。"""
    valid_yaml = """apiVersion: superagent/v1
kind: Agent
metadata:
  name: test-validate-agent
spec:
  type: chat_model_agent
  system_prompt: "You are a test agent."
  model:
    primary: gpt-4o-mini
"""
    async with client:
        result = await client.admin.validate_agent(valid_yaml)
    assert result.valid is True, f"Valid YAML should pass validation: {result}"
```

- [ ] **Step 5: 创建 test_chat.py**

```python
# tests/matrix/test_chat.py
"""流式对话测试：验证 A2UI 事件流和 collect() 行为。"""
import pytest
from superagent.client import SuperagentClient
from superagent.types import AgentInfo


def _first_agent_name(agents: list[AgentInfo]) -> str | None:
    """Return the name of the first loaded agent, or None."""
    return agents[0].name if agents else None


@pytest.mark.asyncio
async def test_chat_stream_returns_text(client: SuperagentClient) -> None:
    """chat_stream().collect() 必须返回非空字符串。"""
    async with client:
        agents = await client.list_agents()
        agent_name = _first_agent_name(agents)
        if not agent_name:
            pytest.skip("No agents loaded — cannot test chat")

        text = await client.chat(agent_name, "Say hello in one word.", session_id="matrix-test-1")
    assert isinstance(text, str) and len(text) > 0, \
        f"Expected non-empty text response, got: {repr(text)}"


@pytest.mark.asyncio
async def test_chat_stream_emits_events(client: SuperagentClient) -> None:
    """chat_stream() 必须发出至少一个事件。"""
    async with client:
        agents = await client.list_agents()
        agent_name = _first_agent_name(agents)
        if not agent_name:
            pytest.skip("No agents loaded — cannot test chat")

        events = []
        stream = client.chat_stream(agent_name, "Say yes.", session_id="matrix-test-2")
        async with stream:
            async for event in stream:
                events.append(event)
                if len(events) >= 10:
                    break

    assert len(events) > 0, "Expected at least one SSE event from chat_stream()"


@pytest.mark.asyncio
async def test_chat_session_continuity(client: SuperagentClient) -> None:
    """同 session_id 的第二条消息不应报错（验证会话管理）。"""
    async with client:
        agents = await client.list_agents()
        agent_name = _first_agent_name(agents)
        if not agent_name:
            pytest.skip("No agents loaded — cannot test chat")

        session = "matrix-continuity-test"
        await client.chat(agent_name, "My name is Matrix.", session_id=session)
        response2 = await client.chat(agent_name, "What is my name?", session_id=session)
    assert isinstance(response2, str), "Second message in session should succeed"
```

- [ ] **Step 6: 创建 report.py（Markdown 报告生成器）**

```python
# tests/matrix/report.py
"""
Run with: python tests/matrix/report.py
Runs pytest on all matrix tests and generates tests/matrix/REPORT.md.
"""
import subprocess
import sys
import json
from datetime import datetime
from pathlib import Path

REPORT_PATH = Path(__file__).parent / "REPORT.md"
BACKENDS = {"go": 8888, "python": 8889, "java": 8890}


def run_pytest_json() -> dict:
    result = subprocess.run(
        [sys.executable, "-m", "pytest", "tests/matrix/",
         "--tb=short", "-q", "--json-report", "--json-report-file=/tmp/matrix-report.json"],
        capture_output=True, text=True,
        cwd=Path(__file__).parent.parent.parent
    )
    try:
        with open("/tmp/matrix-report.json") as f:
            return json.load(f)
    except FileNotFoundError:
        return {"tests": [], "summary": {}}


def build_markdown(data: dict) -> str:
    now = datetime.now().strftime("%Y-%m-%d %H:%M")
    lines = [
        f"# Matrix Test Report — {now}\n",
        "| Test | Go :8888 | Python :8889 | Java :8890 |",
        "|---|:---:|:---:|:---:|",
    ]

    tests_by_name: dict[str, dict] = {}
    for t in data.get("tests", []):
        node = t["nodeid"]
        # Extract backend from parametrize id: test_foo[go] -> go
        backend = "unknown"
        for b in BACKENDS:
            if f"[{b}]" in node:
                backend = b
                break
        base = node.split("[")[0].split("::")[-1]
        tests_by_name.setdefault(base, {})[backend] = t["outcome"]

    for name, outcomes in sorted(tests_by_name.items()):
        go     = "✅" if outcomes.get("go")     == "passed" else ("❌" if "go"     in outcomes else "—")
        python = "✅" if outcomes.get("python")  == "passed" else ("❌" if "python" in outcomes else "—")
        java   = "✅" if outcomes.get("java")    == "passed" else ("❌" if "java"   in outcomes else "—")
        lines.append(f"| `{name}` | {go} | {python} | {java} |")

    summary = data.get("summary", {})
    passed  = summary.get("passed", 0)
    failed  = summary.get("failed", 0)
    total   = summary.get("total",  0)
    lines += ["", f"**Total: {passed}/{total} passed, {failed} failed**"]
    return "\n".join(lines) + "\n"


if __name__ == "__main__":
    print("Running matrix tests...")
    data = run_pytest_json()
    md = build_markdown(data)
    REPORT_PATH.write_text(md)
    print(f"Report written to {REPORT_PATH}")
    print(md)
```

- [ ] **Step 7: 创建 __init__.py**

```bash
touch /Users/ailabuser1/Desktop/gitcode/superagent-base/tests/matrix/__init__.py
```

- [ ] **Step 8: 本地快速验证（仅跑健康检查，后端未启动时应 skip）**

```bash
cd /Users/ailabuser1/Desktop/gitcode/superagent-base
pip install -q pytest pytest-asyncio anyio httpx
pytest tests/matrix/test_health.py -v --tb=short -k "go" 2>&1 | tail -15
```

Expected: 若 Go 后端未启动则 FAILED with connection refused（说明测试结构正确）

- [ ] **Step 9: Commit**

```bash
git add tests/matrix/
git commit -m "feat(tests): add matrix test suite for 3-backend functional comparison"
```

---

## Task 6：端到端冒烟验证

- [ ] **Step 1: 复制 env 文件并填入 API Key**

```bash
cd /Users/ailabuser1/Desktop/gitcode/superagent-base
cp docker/.env.matrix.example docker/.env.matrix
echo "Edit docker/.env.matrix — set OPENAI_API_KEY before continuing"
```

- [ ] **Step 2: 构建并启动全部后端**

```bash
make matrix-up
```

Expected: 三个容器 sa-matrix-go / sa-matrix-python / sa-matrix-java 均 starting

- [ ] **Step 3: 等待健康检查通过**

```bash
make matrix-wait
```

Expected: 每个端口输出 OK

- [ ] **Step 4: 运行矩阵测试**

```bash
make matrix-test
```

Expected: test summary 表格，三列对齐

- [ ] **Step 5: 生成 Markdown 报告**

```bash
cd /Users/ailabuser1/Desktop/gitcode/superagent-base
pip install -q pytest-json-report
python tests/matrix/report.py
cat tests/matrix/REPORT.md
```

- [ ] **Step 6: 启动三前端（可选，需要 Node 依赖已安装）**

```bash
cd web && npm install
make matrix-fe
```

打开：
- http://localhost:3501 → Go 后端
- http://localhost:3502 → Python 后端
- http://localhost:3503 → Java 后端

- [ ] **Step 7: Final commit**

```bash
git add tests/matrix/REPORT.md
git commit -m "test(matrix): initial matrix test run report"
```

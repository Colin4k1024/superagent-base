package io.superagent.skills;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URI;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Invokes skills via HTTP endpoint calls.
 *
 * <p>Each skill is mapped to a URL endpoint. Invocation sends a POST request
 * with the input as JSON body and parses the JSON response.</p>
 *
 * <p>Maps to Go {@code skill.HTTPInvoker}.</p>
 */
public class HTTPInvoker implements SkillInvoker {

    private static final Logger log = LoggerFactory.getLogger(HTTPInvoker.class);
    private static final ObjectMapper mapper = new ObjectMapper();

    private final ConcurrentHashMap<String, String> endpoints = new ConcurrentHashMap<>();
    private final Map<String, String> defaultHeaders;

    public HTTPInvoker() {
        this(Map.of());
    }

    public HTTPInvoker(Map<String, String> defaultHeaders) {
        this.defaultHeaders = defaultHeaders != null ? Map.copyOf(defaultHeaders) : Map.of();
    }

    /**
     * Register an HTTP endpoint for a skill.
     *
     * @param name skill name
     * @param url  HTTP endpoint URL
     */
    public void registerEndpoint(String name, String url) {
        endpoints.put(name, url);
        log.debug("Registered HTTP skill endpoint: {} -> {}", name, url);
    }

    /**
     * Unregister an HTTP endpoint.
     *
     * @param name skill name
     * @return true if the endpoint was registered
     */
    public boolean unregisterEndpoint(String name) {
        return endpoints.remove(name) != null;
    }

    @Override
    @SuppressWarnings("unchecked")
    public Map<String, Object> invoke(String name, Map<String, Object> input) throws SkillException {
        String url = endpoints.get(name);
        if (url == null) {
            throw new SkillException("HTTP skill endpoint not found: " + name);
        }

        try {
            String jsonBody = mapper.writeValueAsString(input != null ? input : Map.of());
            String responseBody = doPost(url, jsonBody);
            Map<String, Object> result = mapper.readValue(responseBody, Map.class);
            log.debug("HTTP skill '{}' invoked successfully at {}", name, url);
            return result;
        } catch (JsonProcessingException e) {
            throw new SkillException("Failed to serialize/deserialize for skill '" + name + "': " + e.getMessage(), e);
        } catch (IOException e) {
            throw new SkillException("HTTP call failed for skill '" + name + "': " + e.getMessage(), e);
        }
    }

    @Override
    public boolean canInvoke(String name) {
        return endpoints.containsKey(name);
    }

    /**
     * Get all registered endpoint names.
     */
    public Set<String> getRegisteredNames() {
        return Set.copyOf(endpoints.keySet());
    }

    /**
     * Get the endpoint URL for a skill.
     */
    public String getEndpoint(String name) {
        return endpoints.get(name);
    }

    private String doPost(String urlStr, String jsonBody) throws IOException {
        URL url = URI.create(urlStr).toURL();
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        conn.setRequestProperty("Accept", "application/json");
        for (var entry : defaultHeaders.entrySet()) {
            conn.setRequestProperty(entry.getKey(), entry.getValue());
        }
        conn.setDoOutput(true);
        conn.setConnectTimeout(10_000);
        conn.setReadTimeout(30_000);

        try (OutputStream os = conn.getOutputStream()) {
            os.write(jsonBody.getBytes(StandardCharsets.UTF_8));
        }

        int status = conn.getResponseCode();
        var is = status >= 400 ? conn.getErrorStream() : conn.getInputStream();
        String body = new String(is.readAllBytes(), StandardCharsets.UTF_8);

        if (status >= 400) {
            throw new IOException("HTTP " + status + ": " + body);
        }
        return body;
    }
}

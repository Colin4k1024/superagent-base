package io.superagent.server;

import org.springframework.stereotype.Component;
import org.springframework.web.server.ServerWebExchange;
import org.springframework.web.server.WebFilter;
import org.springframework.web.server.WebFilterChain;
import reactor.core.publisher.Mono;

/**
 * Rewrites {@code /api/v1/} requests to {@code /api/v2/} so the frontend
 * (which defaults to {@code API_BASE = '/api/v1'}) works against the Java
 * backend without any frontend changes.
 *
 * <p>This is a compatibility shim for local matrix testing only.</p>
 */
@Component
public class ApiVersionRewriteFilter implements WebFilter {

    private static final String V1_PREFIX = "/api/v1/";
    private static final String V2_PREFIX = "/api/v2/";

    @Override
    public Mono<Void> filter(ServerWebExchange exchange, WebFilterChain chain) {
        String path = exchange.getRequest().getURI().getPath();
        if (path.startsWith(V1_PREFIX)) {
            String rewritten = V2_PREFIX + path.substring(V1_PREFIX.length());
            ServerWebExchange mutated = exchange.mutate()
                .request(r -> r.path(rewritten))
                .build();
            return chain.filter(mutated);
        }
        return chain.filter(exchange);
    }
}

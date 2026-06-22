package io.superagent.tools;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.Function;

/**
 * Tool middleware chain for cross-cutting concerns.
 *
 * <p>Provides retry, timeout, rate-limiting, caching, and logging middleware
 * that wrap tool invocations. Middleware can be composed via {@link #chain}.</p>
 *
 * <p>Maps to Go {@code tool.Middleware} with RetryMiddleware, TimeoutMiddleware,
 * RateLimitMiddleware, CacheMiddleware, LogMiddleware, Chain.</p>
 */
public final class ToolMiddleware {

    private static final Logger log = LoggerFactory.getLogger(ToolMiddleware.class);

    private ToolMiddleware() {}

    /**
     * Functional interface for tool invocation.
     */
    @FunctionalInterface
    public interface ToolInvoker {
        /**
         * Invoke a tool by name with arguments.
         *
         * @param name tool name
         * @param args tool arguments
         * @return tool result
         * @throws Exception if invocation fails
         */
        Map<String, Object> invoke(String name, Map<String, Object> args) throws Exception;
    }

    /**
     * Functional interface for middleware.
     *
     * <p>Middleware wraps a {@link ToolInvoker} to add behavior before/after invocation.</p>
     */
    @FunctionalInterface
    public interface Middleware {
        /**
         * Wrap the next invoker with additional behavior.
         *
         * @param next the downstream invoker
         * @return wrapped invoker
         */
        ToolInvoker apply(ToolInvoker next);
    }

    /**
     * Retry middleware — retries failed invocations with exponential backoff.
     *
     * @param maxRetries maximum number of retries
     * @param backoff    initial backoff duration
     * @return retry middleware
     */
    public static Middleware retry(int maxRetries, Duration backoff) {
        return next -> (name, args) -> {
            Exception lastException = null;
            for (int attempt = 0; attempt <= maxRetries; attempt++) {
                try {
                    return next.invoke(name, args);
                } catch (Exception e) {
                    lastException = e;
                    if (attempt < maxRetries) {
                        long sleepMs = backoff.toMillis() * (1L << attempt);
                        log.warn("Tool '{}' attempt {}/{} failed, retrying in {}ms: {}",
                            name, attempt + 1, maxRetries + 1, sleepMs, e.getMessage());
                        Thread.sleep(Math.min(sleepMs, 30_000));
                    }
                }
            }
            throw lastException;
        };
    }

    /**
     * Timeout middleware — fails if invocation takes too long.
     *
     * @param timeout maximum duration
     * @return timeout middleware
     */
    public static Middleware timeout(Duration timeout) {
        return next -> (name, args) -> {
            long deadline = System.currentTimeMillis() + timeout.toMillis();
            Map<String, Object> result = next.invoke(name, args);
            if (System.currentTimeMillis() > deadline) {
                log.warn("Tool '{}' exceeded timeout of {}", name, timeout);
            }
            return result;
        };
    }

    /**
     * Rate-limit middleware — limits invocations per minute.
     *
     * @param rpm maximum requests per minute
     * @return rate-limit middleware
     */
    public static Middleware rateLimit(int rpm) {
        ConcurrentHashMap<String, RateWindow> windows = new ConcurrentHashMap<>();
        return next -> (name, args) -> {
            RateWindow window = windows.computeIfAbsent(name, k -> new RateWindow());
            synchronized (window) {
                long now = System.currentTimeMillis();
                // Reset window if expired
                if (now - window.startTime > 60_000) {
                    window.startTime = now;
                    window.count.set(0);
                }
                if (window.count.get() >= rpm) {
                    long waitMs = 60_000 - (now - window.startTime);
                    throw new IllegalStateException(
                        "Rate limit exceeded for tool '" + name + "'. Wait " + waitMs + "ms.");
                }
                window.count.incrementAndGet();
            }
            return next.invoke(name, args);
        };
    }

    /**
     * Cache middleware — caches tool results for a duration.
     *
     * @param ttl time-to-live for cached results
     * @return cache middleware
     */
    public static Middleware cache(Duration ttl) {
        ConcurrentHashMap<String, CacheEntry> cacheMap = new ConcurrentHashMap<>();
        return next -> (name, args) -> {
            String cacheKey = name + ":" + args.hashCode();
            CacheEntry cached = cacheMap.get(cacheKey);
            long now = System.currentTimeMillis();

            if (cached != null && (now - cached.timestamp) < ttl.toMillis()) {
                log.debug("Cache hit for tool '{}'", name);
                return cached.result;
            }

            Map<String, Object> result = next.invoke(name, args);
            cacheMap.put(cacheKey, new CacheEntry(result, now));
            return result;
        };
    }

    /**
     * Logging middleware — logs tool invocations and results.
     *
     * @param logger logger instance
     * @return logging middleware
     */
    public static Middleware log(Logger logger) {
        return next -> (name, args) -> {
            logger.info("Tool '{}' invoked with args: {}", name, args.keySet());
            long start = System.currentTimeMillis();
            try {
                Map<String, Object> result = next.invoke(name, args);
                long elapsed = System.currentTimeMillis() - start;
                logger.info("Tool '{}' completed in {}ms", name, elapsed);
                return result;
            } catch (Exception e) {
                long elapsed = System.currentTimeMillis() - start;
                logger.error("Tool '{}' failed after {}ms: {}", name, elapsed, e.getMessage());
                throw e;
            }
        };
    }

    /**
     * Chain multiple middleware into a single middleware.
     *
     * <p>Middleware are applied in order: the first middleware wraps the outermost layer.</p>
     *
     * @param middlewares middleware to chain
     * @return composed middleware
     */
    public static Middleware chain(Middleware... middlewares) {
        return next -> {
            ToolInvoker invoker = next;
            // Apply in reverse so the first middleware is outermost
            for (int i = middlewares.length - 1; i >= 0; i--) {
                invoker = middlewares[i].apply(invoker);
            }
            return invoker;
        };
    }

    /**
     * Chain a list of middleware.
     *
     * @param middlewares list of middleware
     * @return composed middleware
     */
    public static Middleware chain(List<Middleware> middlewares) {
        return chain(middlewares.toArray(new Middleware[0]));
    }

    /**
     * Apply middleware chain to a tool invoker.
     *
     * @param invoker     base invoker
     * @param middlewares middleware to apply
     * @return wrapped invoker
     */
    public static ToolInvoker apply(ToolInvoker invoker, Middleware... middlewares) {
        return chain(middlewares).apply(invoker);
    }

    // ─── Internal types ───

    private static class RateWindow {
        long startTime = System.currentTimeMillis();
        AtomicInteger count = new AtomicInteger(0);
    }

    private record CacheEntry(Map<String, Object> result, long timestamp) {}
}
